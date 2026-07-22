package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

var (
	resendEndpoint   = "https://api.resend.com/emails"
	resendHTTPClient = http.DefaultClient
)

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
	authIdentities, matchedIdentities := make(map[string]bool), make(map[string]bool)
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
