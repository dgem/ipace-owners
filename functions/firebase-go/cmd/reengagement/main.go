package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

const defaultDelay = 250 * time.Millisecond

var (
	resendEndpoint   = "https://api.resend.com/emails"
	resendHTTPClient = http.DefaultClient
)

type recipient struct {
	Name        string
	Email       string
	SubmittedAt string
}

type campaignConfig struct {
	Environment string
	ResultsPath string
	ProjectID   string
	DatabaseID  string
	ContinueURL string
	LinkDomain  string
	VehicleURL  string
	CampaignID  string
	Send        bool
	Confirm     int
	Delay       time.Duration
	LogLevel    string
	Input       io.Reader
	Output      io.Writer
}

type resultRow struct {
	Recipient recipient
	Status    string
	Detail    string
	ResendID  string
}

func main() {
	config := campaignConfig{}
	flag.StringVar(&config.Environment, "env", "", "Firebase environment: staging or production")
	flag.StringVar(&config.ResultsPath, "results", "", "new CSV path for per-recipient results")
	flag.StringVar(&config.ProjectID, "project", "", "Firebase project override")
	flag.StringVar(&config.DatabaseID, "database", "", "Firestore database override")
	flag.StringVar(&config.ContinueURL, "continue-url", "", "post-sign-in account URL override")
	flag.StringVar(&config.LinkDomain, "link-domain", "", "Firebase email action link domain override")
	flag.StringVar(&config.VehicleURL, "vehicle-url", "", "secondary vehicle-data URL override")
	flag.StringVar(&config.CampaignID, "campaign-id", "", "stable campaign identifier used for Resend idempotency")
	flag.BoolVar(&config.Send, "send", false, "send emails; omitted means preflight-only dry run")
	flag.IntVar(&config.Confirm, "confirm-count", 0, "exact eligible recipient count required with --send")
	flag.DurationVar(&config.Delay, "delay", defaultDelay, "delay between sends (default 250ms)")
	flag.StringVar(&config.LogLevel, "log-level", "info", "logging level: info or debug (debug prints personal data)")
	flag.Parse()
	config.Input = os.Stdin
	config.Output = os.Stdout

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
	// Refresh the full account index before confirmation so email aliases and names created
	// since preflight are suppressed too.
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

	results := make([]resultRow, 0, len(joins))
	for _, row := range preflight {
		if row.Status != "eligible" {
			results = append(results, row)
		}
	}
	for index, person := range eligible {
		// Recheck immediately before generating a live link in case registration completed after preflight.
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
			URL:             config.ContinueURL,
			HandleCodeInApp: true,
			LinkDomain:      config.LinkDomain,
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

	sent := 0
	failed := 0
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

func validateConfig(config campaignConfig) error {
	if config.Environment != "staging" && config.Environment != "production" {
		return errors.New("--env must be staging or production")
	}
	for label, value := range map[string]string{
		"--env": config.Environment, "--results": config.ResultsPath, "--project": config.ProjectID,
		"--database": config.DatabaseID, "--continue-url": config.ContinueURL, "--link-domain": config.LinkDomain,
		"--vehicle-url": config.VehicleURL,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	if _, err := os.Stat(config.ResultsPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing results file %s", config.ResultsPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	for label, rawURL := range map[string]string{"--continue-url": config.ContinueURL, "--vehicle-url": config.VehicleURL} {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return fmt.Errorf("%s must be an absolute HTTPS URL", label)
		}
	}
	if config.Send {
		if strings.TrimSpace(config.CampaignID) == "" {
			return errors.New("--campaign-id is required with --send")
		}
		if config.Confirm < 1 {
			return errors.New("--confirm-count must be positive with --send")
		}
		if strings.TrimSpace(os.Getenv("RESEND_API_KEY")) == "" || strings.TrimSpace(os.Getenv("RESEND_FROM")) == "" {
			return errors.New("RESEND_API_KEY and RESEND_FROM are required with --send")
		}
	}
	if config.Delay < 200*time.Millisecond {
		return errors.New("--delay must be at least 200ms to respect Resend's default 5 requests/second limit")
	}
	if config.LogLevel != "info" && config.LogLevel != "debug" {
		return errors.New("--log-level must be info or debug")
	}
	return nil
}

func resolveEnvironment(config campaignConfig) campaignConfig {
	host := ""
	switch strings.ToLower(strings.TrimSpace(config.Environment)) {
	case "staging":
		config.Environment, host = "staging", "stage.ipace-owners.org"
	case "production":
		config.Environment, host = "production", "ipace-owners.org"
	default:
		return config
	}
	if config.ProjectID == "" {
		config.ProjectID = "ipace-owners-" + config.Environment
	}
	if config.DatabaseID == "" {
		config.DatabaseID = "ipace-owners-" + config.Environment
	}
	if config.ContinueURL == "" {
		config.ContinueURL = "https://" + host + "/member/account/"
	}
	if config.VehicleURL == "" {
		config.VehicleURL = "https://" + host + "/member/vehicle/"
	}
	if config.LinkDomain == "" {
		config.LinkDomain = host
	}
	if config.Input == nil {
		config.Input = os.Stdin
	}
	if config.Output == nil {
		config.Output = os.Stdout
	}
	return config
}

type authAccount struct {
	Email string
	Name  string
}

func loadJoinSubmissions(ctx context.Context, db *firestore.Client) ([]recipient, error) {
	unique := make(map[string]recipient)
	iter := db.Collection("joinSubmissions").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var record struct {
			CreatedAt time.Time `firestore:"createdAt"`
			Contact   struct {
				Name  string `firestore:"name"`
				Email string `firestore:"email"`
			} `firestore:"contact"`
			Consents struct {
				Contact bool `firestore:"contact"`
			} `firestore:"consents"`
		}
		if err := doc.DataTo(&record); err != nil {
			return nil, err
		}
		email := strings.ToLower(strings.TrimSpace(record.Contact.Email))
		name := strings.TrimSpace(record.Contact.Name)
		if email == "" || name == "" || !record.Consents.Contact {
			continue
		}
		person := recipient{Name: name, Email: email, SubmittedAt: record.CreatedAt.UTC().Format(time.RFC3339)}
		key := canonicalEmail(email)
		previous, exists := unique[key]
		if !exists || person.SubmittedAt < previous.SubmittedAt {
			unique[key] = person
		}
	}
	rows := make([]recipient, 0, len(unique))
	for _, person := range unique {
		rows = append(rows, person)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Email < rows[j].Email })
	return rows, nil
}

func loadAuthAccounts(ctx context.Context, client *auth.Client) ([]authAccount, error) {
	iter := client.Users(ctx, "")
	accounts := make([]authAccount, 0)
	for {
		user, err := iter.Next()
		if err == iterator.Done {
			return accounts, nil
		}
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, authAccount{Email: strings.ToLower(strings.TrimSpace(user.Email)), Name: strings.TrimSpace(user.DisplayName)})
	}
}

func classifyRecipients(joins []recipient, accounts []authAccount) ([]resultRow, []recipient) {
	exactEmails, canonicalEmails, names := make(map[string]bool), make(map[string]bool), make(map[string]bool)
	for _, account := range accounts {
		if account.Email != "" {
			exactEmails[account.Email] = true
			canonicalEmails[canonicalEmail(account.Email)] = true
		}
		if key := normalizedName(account.Name); key != "" {
			names[key] = true
		}
	}
	rows := make([]resultRow, 0, len(joins))
	eligible := make([]recipient, 0, len(joins))
	for _, person := range joins {
		status, detail := "eligible", "no Firebase Auth match by email alias or name"
		switch {
		case exactEmails[person.Email]:
			status, detail = "skipped_registered", "exact Firebase Auth email match"
		case canonicalEmails[canonicalEmail(person.Email)]:
			status, detail = "skipped_registered_alias", "Firebase Auth email match after removing plus addressing"
		case names[normalizedName(person.Name)]:
			status, detail = "skipped_registered_name", "Firebase Auth display-name match"
		}
		rows = append(rows, resultRow{Recipient: person, Status: status, Detail: detail})
		if status == "eligible" {
			eligible = append(eligible, person)
		}
	}
	return rows, eligible
}

func reconcileAuthCoverage(joins []recipient, accounts []authAccount) (int, int) {
	joinEmails := make(map[string]bool)
	for _, person := range joins {
		joinEmails[canonicalEmail(person.Email)] = true
	}
	authIdentities := make(map[string]bool)
	matchedIdentities := make(map[string]bool)
	for _, account := range accounts {
		identity := canonicalEmail(account.Email)
		if identity == "" {
			continue
		}
		authIdentities[identity] = true
		if joinEmails[identity] {
			matchedIdentities[identity] = true
		}
	}
	return len(authIdentities), len(authIdentities) - len(matchedIdentities)
}

func canonicalEmail(value string) string {
	email := strings.ToLower(strings.TrimSpace(value))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	if plus := strings.IndexByte(parts[0], '+'); plus >= 0 {
		parts[0] = parts[0][:plus]
	}
	return parts[0] + "@" + parts[1]
}

func normalizedName(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			return unicode.ToLower(character)
		}
		return -1
	}, value)
}

func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])[:16]
}

func logCandidates(config campaignConfig, eligible []recipient) {
	fmt.Fprintf(config.Output, "eligible recipients: %d (use --log-level=debug to print personal data)\n", len(eligible))
	if config.LogLevel != "debug" {
		return
	}
	for _, person := range eligible {
		fmt.Fprintf(config.Output, "candidate name=%q email=%q submitted=%s\n", person.Name, person.Email, person.SubmittedAt)
	}
}

func confirmSend(config campaignConfig, count int) error {
	expected := fmt.Sprintf("SEND %s %d", config.Environment, count)
	fmt.Fprintf(config.Output, "Type %q to send %d emails: ", expected, count)
	response, err := bufio.NewReader(config.Input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if strings.TrimSpace(response) != expected {
		return errors.New("confirmation did not match; no emails sent")
	}
	return nil
}

func sendReengagementEmail(ctx context.Context, person recipient, actionLink string, memberCount int, eligibleCount int, config campaignConfig) (string, error) {
	payload := reengagementPayload(person, actionLink, config.VehicleURL, memberCount, eligibleCount)
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("RESEND_API_KEY")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ipace-owners-reengagement/1.0")
	req.Header.Set("Idempotency-Key", config.CampaignID+"/"+emailFingerprint(person.Email))
	response, err := resendHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(response.Body, 4096))
	if readErr != nil {
		return "", readErr
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("Resend returned %d", response.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func reengagementPayload(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) map[string]any {
	payload := map[string]any{
		"from": strings.TrimSpace(os.Getenv("RESEND_FROM")), "to": []string{person.Email},
		"subject": "Complete your I-PACE Owners registration",
		"html":    reengagementHTML(person, actionLink, vehicleURL, memberCount, eligibleCount), "text": reengagementText(person, actionLink, vehicleURL, memberCount, eligibleCount),
		"tags": []map[string]string{{"name": "category", "value": "join-reengagement"}},
	}
	if replyTo := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); replyTo != "" {
		payload["reply_to"] = replyTo
	}
	return payload
}

func reengagementHTML(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) string {
	name := html.EscapeString(firstName(person.Name))
	link := html.EscapeString(actionLink)
	vehicle := html.EscapeString(vehicleURL)
	image := html.EscapeString(strings.TrimRight(assetBaseURL(vehicleURL), "/") + "/images/ipace-hero.png")
	date := submittedDate(person.SubmittedAt)
	progress := html.EscapeString(registrationProgress(eligibleCount))
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Complete your I-PACE Owners registration</title></head>
<body style="margin:0;background:#f7f8fb;color:#111827;font-family:Arial,Helvetica,sans-serif;"><div style="display:none;max-height:0;overflow:hidden;">One quick step completes your I-PACE Owners registration.</div>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="background:#f7f8fb;padding:24px 12px;"><tr><td align="center"><table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#fff;border-radius:20px;overflow:hidden;border:1px solid #dbe3ea;">
<tr><td style="background:#12324a;padding:24px 28px;color:#fff;"><div style="font-size:22px;font-weight:700;line-height:1.2;">I-PACE Owners</div><div style="font-size:14px;color:#c9d7e3;margin-top:4px;">Advocacy Group</div></td></tr>
<tr><td><img src="` + image + `" width="640" alt="Jaguar I-PACE" style="display:block;width:100%;height:auto;border:0;"></td></tr>
<tr><td style="padding:32px 28px;"><p style="margin:0 0 10px;font-size:16px;line-height:1.6;color:#374151;">Hello ` + name + `,</p><h1 style="margin:0 0 14px;color:#12324a;font-size:28px;line-height:1.2;">Please complete your registration</h1>
<p style="margin:0 0 18px;font-size:16px;line-height:1.6;color:#374151;">You asked to join the I-PACE Owners Advocacy Group on ` + html.EscapeString(date) + `. Your details were saved, but your secure sign-in was not completed.</p>
<div style="margin:0 0 22px;padding:18px 20px;background:#eef6f5;border-left:4px solid #0f766e;border-radius:10px;"><p style="margin:0 0 6px;font-size:16px;line-height:1.5;color:#12324a;font-weight:700;">` + fmt.Sprintf("%d", memberCount) + ` owners have joined us so far.</p><p style="margin:0;font-size:14px;line-height:1.55;color:#374151;">` + progress + ` That is a meaningful step toward our ambition of bringing together 1,000 I-PACE owners.</p></div>
<p style="margin:0 0 26px;font-size:16px;line-height:1.6;color:#374151;">Use this fresh, time-limited link to verify your account details and open your private member area.</p>
<p style="margin:0 0 24px;"><a href="` + link + `" style="display:inline-block;background:#0f766e;color:#fff;text-decoration:none;font-weight:700;border-radius:999px;padding:14px 22px;font-size:16px;">Verify my account details</a></p>
<p style="margin:0 0 8px;font-size:14px;line-height:1.5;color:#4b5563;">After signing in, you can strengthen the shared evidence by adding each I-PACE you own and any battery-health or service information you wish to contribute.</p>
<p style="margin:0 0 26px;"><a href="` + vehicle + `" style="display:inline-block;color:#0f766e;text-decoration:underline;font-weight:700;font-size:14px;">Add my I-PACE data</a></p>
<p style="margin:0 0 8px;font-size:12px;line-height:1.5;color:#6b7280;">If the main button does not work, copy this link into your browser:</p><p style="margin:0;word-break:break-all;font-size:12px;line-height:1.5;"><a href="` + link + `" style="color:#0f766e;">` + link + `</a></p></td></tr>
<tr><td style="background:#eef6f5;padding:20px 28px;color:#374151;font-size:13px;line-height:1.5;">You are receiving this because you submitted the Join form and agreed that we could contact you. If you no longer wish to hear from us, reply to this email and let us know.</td></tr>
</table></td></tr></table></body></html>`
}

func reengagementText(person recipient, actionLink string, vehicleURL string, memberCount int, eligibleCount int) string {
	return "Hello " + firstName(person.Name) + ",\n\n" +
		"Please complete your I-PACE Owners registration\n\n" +
		"You asked to join the I-PACE Owners Advocacy Group on " + submittedDate(person.SubmittedAt) + ". Your details were saved, but your secure sign-in was not completed.\n\n" +
		fmt.Sprintf("%d owners have joined us so far. %s That is a meaningful step toward our ambition of bringing together 1,000 I-PACE owners.\n\n", memberCount, registrationProgress(eligibleCount)) +
		"Verify your account details and sign in using this fresh, time-limited link:\n" + actionLink + "\n\n" +
		"After signing in, you can add each I-PACE you own and any battery-health or service information you wish to contribute:\n" + vehicleURL + "\n\n" +
		"You are receiving this because you submitted the Join form and agreed that we could contact you. If you no longer wish to hear from us, reply to this email and let us know.\n"
}

func registrationProgress(eligibleCount int) string {
	if eligibleCount > 125 {
		return "If everyone receiving this reminder completes verification, more than 125 additional members will have registered accounts."
	}
	return fmt.Sprintf("If everyone receiving this reminder completes verification, %d additional members will have registered accounts.", eligibleCount)
}

func firstName(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

func submittedDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Format("2 January 2006")
}

func assetBaseURL(fallbackURL string) string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("RESEND_ASSET_BASE_URL")), "/"); value != "" {
		return value
	}
	if parsed, err := url.Parse(fallbackURL); err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" {
		return parsed.Scheme + "://" + parsed.Host
	}
	return "https://ipace-owners.org"
}

func maskedEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return "invalid-email"
	}
	return parts[0][:1] + "***@" + parts[1]
}

func writeResults(path string, rows []resultRow) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"name", "email", "submitted_at", "status", "detail", "resend_id"}); err != nil {
		file.Close()
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Recipient.Name, row.Recipient.Email, row.Recipient.SubmittedAt, row.Status, row.Detail, row.ResendID}); err != nil {
			file.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
