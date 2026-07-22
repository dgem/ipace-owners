package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

func main() {
	config := commandConfig()
	if err := run(context.Background(), config); err != nil {
		fmt.Fprintln(os.Stderr, "re-engagement campaign failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, config campaignConfig) error {
	config = resolveEnvironment(config)
	if err := validateConfig(config); err != nil {
		return err
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: config.ProjectID})
	if err != nil {
		return fmt.Errorf("Firebase client: %w", err)
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("Firebase Auth client: %w", err)
	}
	db, err := firestore.NewClientWithDatabase(ctx, config.ProjectID, config.DatabaseID)
	if err != nil {
		return fmt.Errorf("Firestore client: %w", err)
	}
	defer db.Close()

	joins, err := loadJoinSubmissions(ctx, db)
	if err != nil {
		return fmt.Errorf("load Join submissions: %w", err)
	}
	accounts, err := loadAuthAccounts(ctx, authClient)
	if err != nil {
		return fmt.Errorf("load Firebase Auth users: %w", err)
	}
	preflight, eligible := classifyRecipients(joins, accounts)
	memberCount := len(joins)
	authIdentities, authWithoutJoin := reconcileAuthCoverage(joins, accounts)
	logCandidates(config, eligible)

	if !config.Send {
		if err := writeResults(config.ResultsPath, preflight); err != nil {
			return err
		}
		if err := writeCampaignManifest(config, memberCount, len(accounts), authIdentities, len(preflight)-len(eligible), len(eligible)); err != nil {
			return err
		}
		fmt.Fprintf(config.Output, "dry run complete: env=%s, %d unique Join submissions, %d total Auth accounts (%d canonical identities), %d Join identities matched to Auth, %d Auth identities without Join, %d eligible; no emails sent\n", config.Environment, memberCount, len(accounts), authIdentities, len(preflight)-len(eligible), authWithoutJoin, len(eligible))
		if authWithoutJoin > 0 {
			return fmt.Errorf("privacy invariant failed: %d canonical Auth identities have no Join submission", authWithoutJoin)
		}
		return nil
	}
	if authWithoutJoin > 0 {
		return fmt.Errorf("refusing to send: %d canonical Auth identities have no Join submission", authWithoutJoin)
	}
	if config.Confirm != len(eligible) {
		return fmt.Errorf("refusing to send: --confirm-count=%d, current eligible count=%d; rerun dry mode and confirm the current count", config.Confirm, len(eligible))
	}
	accounts, err = loadAuthAccounts(ctx, authClient)
	if err != nil {
		return fmt.Errorf("refresh Firebase Auth users: %w", err)
	}
	preflight, eligible = classifyRecipients(joins, accounts)
	if config.Confirm != len(eligible) {
		return fmt.Errorf("refusing to send: eligible count changed to %d after refresh", len(eligible))
	}
	logCandidates(config, eligible)
	if err := confirmSend(config, len(eligible)); err != nil {
		return err
	}
	if err := writeCampaignManifest(config, memberCount, len(accounts), authIdentities, len(preflight)-len(eligible), len(eligible)); err != nil {
		return err
	}

	results := make([]resultRow, 0, len(joins))
	for _, row := range preflight {
		if row.Status != "eligible" {
			results = append(results, row)
		}
	}
	for index, person := range eligible {
		_, err := authClient.GetUserByEmail(ctx, person.Email)
		if err == nil {
			results = append(results, resultRow{Recipient: person, Status: "skipped_registered", Detail: "Firebase Auth user now exists"})
			if writeErr := writeResults(config.ResultsPath, results); writeErr != nil {
				return writeErr
			}
			continue
		}
		if !auth.IsUserNotFound(err) {
			results = append(results, resultRow{Recipient: person, Status: "error", Detail: err.Error()})
			if writeErr := writeResults(config.ResultsPath, results); writeErr != nil {
				return writeErr
			}
			continue
		}
		link, err := authClient.EmailSignInLink(ctx, person.Email, &auth.ActionCodeSettings{
			URL: config.ContinueURL, HandleCodeInApp: true, LinkDomain: config.LinkDomain,
		})
		if err != nil {
			results = append(results, resultRow{Recipient: person, Status: "error", Detail: "Firebase link generation failed"})
			if writeErr := writeResults(config.ResultsPath, results); writeErr != nil {
				return writeErr
			}
			continue
		}
		resendID, err := sendReengagementEmail(ctx, person, link, memberCount, len(eligible), config)
		if err != nil {
			results = append(results, resultRow{Recipient: person, Status: "error", Detail: err.Error()})
		} else {
			results = append(results, resultRow{Recipient: person, Status: "sent", Detail: "accepted by Resend", ResendID: resendID})
		}
		if err := writeResults(config.ResultsPath, results); err != nil {
			return err
		}
		if index < len(eligible)-1 {
			time.Sleep(config.Delay)
		}
	}

	sent, failed := 0, 0
	for _, row := range results {
		switch row.Status {
		case "sent":
			sent++
		case "error":
			failed++
		}
	}
	fmt.Fprintf(config.Output, "campaign complete: %d sent, %d failed, %d skipped; results: %s\n", sent, failed, len(results)-sent-failed, config.ResultsPath)
	if failed > 0 {
		return fmt.Errorf("%d recipient sends failed; inspect the results CSV", failed)
	}
	return nil
}
