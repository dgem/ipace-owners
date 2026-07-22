package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestClassifyRecipientsMatchesExactPlusAddressAndName(t *testing.T) {
	joins := []recipient{
		{Name: "Exact Match", Email: "exact@example.com"},
		{Name: "Dan Gem", Email: "dan@kanzi.co.uk"},
		{Name: "Jane D'River", Email: "another@example.com"},
		{Name: "Unregistered Owner", Email: "new@example.com"},
	}
	accounts := []authAccount{
		{Email: "exact@example.com", Name: "Someone Else"},
		{Email: "dan+ipace@kanzi.co.uk", Name: "Different Name"},
		{Email: "registered@example.net", Name: "jane driver"},
	}
	rows, eligible := classifyRecipients(joins, accounts)
	statuses := []string{"skipped_registered", "skipped_registered_alias", "skipped_registered_name", "eligible"}
	for index, expected := range statuses {
		if rows[index].Status != expected {
			t.Fatalf("row %d status = %q, want %q", index, rows[index].Status, expected)
		}
	}
	if len(eligible) != 1 || eligible[0].Email != "new@example.com" {
		t.Fatalf("eligible = %#v", eligible)
	}
}

func TestCanonicalEmailRemovesPlusTagFromEitherSide(t *testing.T) {
	for _, value := range []string{"dan@kanzi.co.uk", "dan+ipace@kanzi.co.uk", " DAN+OTHER@KANZI.CO.UK "} {
		if got := canonicalEmail(value); got != "dan@kanzi.co.uk" {
			t.Fatalf("canonicalEmail(%q) = %q", value, got)
		}
	}
}

func TestValidateConfigRequiresExplicitLiveSendConfirmation(t *testing.T) {
	config := campaignConfig{
		Environment: "production", ResultsPath: filepath.Join(t.TempDir(), "results.csv"), ProjectID: "project",
		DatabaseID: "database", ContinueURL: "https://example.com/account/", LinkDomain: "auth.example.com",
		VehicleURL: "https://example.com/vehicle/", Delay: 250 * time.Millisecond, Send: true, LogLevel: "info",
	}
	if err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "--campaign-id") {
		t.Fatalf("error = %v", err)
	}
	config.CampaignID = "join-nudge-2026-07"
	if err := validateConfig(config); err == nil || !strings.Contains(err.Error(), "--confirm-count") {
		t.Fatalf("error = %v", err)
	}
	config.Confirm = 150
	t.Setenv("RESEND_API_KEY", "test-key")
	t.Setenv("RESEND_FROM", "I-PACE Owners <members@example.com>")
	if err := validateConfig(config); err != nil {
		t.Fatal(err)
	}
}

func TestResolveEnvironmentAndConfirmation(t *testing.T) {
	config := resolveEnvironment(campaignConfig{Environment: "staging", Input: strings.NewReader("SEND staging 2\n"), Output: io.Discard})
	if config.ProjectID != "ipace-owners-staging" || config.DatabaseID != "ipace-owners-staging" || config.LinkDomain != "stage.ipace-owners.org" {
		t.Fatalf("config = %#v", config)
	}
	if err := confirmSend(config, 2); err != nil {
		t.Fatal(err)
	}
	config.Input = strings.NewReader("yes\n")
	if err := confirmSend(config, 2); err == nil {
		t.Fatal("expected mismatched confirmation to fail")
	}
}

func TestReengagementTemplateUsesPrimaryAndSecondaryActions(t *testing.T) {
	t.Setenv("RESEND_ASSET_BASE_URL", "https://ipace-owners.org")
	person := recipient{Name: "Jane <Driver>", Email: "jane@example.com", SubmittedAt: "2026-07-20T10:30:00Z"}
	htmlBody := reengagementHTML(person, "https://auth.example.com/action?a=1&b=2", "https://ipace-owners.org/member/vehicle/", 371, 150)
	for _, expected := range []string{
		"Hello Jane,", "Verify my account details", "Add my I-PACE data", "20 July 2026",
		"https://ipace-owners.org/images/ipace-hero.png", "agreed that we could contact you", "371 owners have joined us so far",
		"more than 125 additional members", "ambition of bringing together 1,000 I-PACE owners",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML missing %q", expected)
		}
	}
	if strings.Contains(htmlBody, "Jane <Driver>") || !strings.Contains(htmlBody, "a=1&amp;b=2") {
		t.Fatalf("HTML escaping failed: %s", htmlBody)
	}
	textBody := reengagementText(person, "https://auth.example.com/action", "https://ipace-owners.org/member/vehicle/", 371, 150)
	if !strings.Contains(textBody, "Verify your account details") || !strings.Contains(textBody, "Add each I-PACE") && !strings.Contains(textBody, "add each I-PACE") {
		t.Fatalf("text body missing actions: %s", textBody)
	}
}

func TestReengagementPayloadUsesCampaignCategoryAndReplyTo(t *testing.T) {
	t.Setenv("RESEND_FROM", "I-PACE Owners <members@ipace-owners.org>")
	t.Setenv("RESEND_REPLY_TO", "contact@ipace-owners.org")
	payload := reengagementPayload(recipient{Name: "Jane", Email: "jane@example.com", SubmittedAt: "2026-07-20T10:30:00Z"}, "https://auth.example.com/action", "https://ipace-owners.org/member/vehicle/", 371, 150)
	if payload["subject"] != "Complete your I-PACE Owners registration" || payload["reply_to"] != "contact@ipace-owners.org" {
		t.Fatalf("payload = %#v", payload)
	}
	tags, ok := payload["tags"].([]map[string]string)
	if !ok || len(tags) != 1 || tags[0]["value"] != "join-reengagement" {
		t.Fatalf("tags = %#v", payload["tags"])
	}
}

func TestSendReengagementEmailUsesAuthenticationAndIdempotency(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "test-key")
	t.Setenv("RESEND_FROM", "I-PACE Owners <members@ipace-owners.org>")
	person := recipient{Name: "Jane", Email: "jane@example.com", SubmittedAt: "2026-07-20T10:30:00Z"}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Idempotency-Key"); got != "join-nudge-2026-07/"+emailFingerprint(person.Email) {
			t.Errorf("Idempotency-Key = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["subject"] != "Complete your I-PACE Owners registration" {
			t.Errorf("payload = %#v", payload)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"resend-message-id"}`)),
		}, nil
	})}

	oldEndpoint, oldClient := resendEndpoint, resendHTTPClient
	resendEndpoint, resendHTTPClient = "https://resend.test/emails", client
	t.Cleanup(func() {
		resendEndpoint, resendHTTPClient = oldEndpoint, oldClient
	})

	id, err := sendReengagementEmail(context.Background(), person, "https://example.com/action", 371, 150, campaignConfig{
		VehicleURL: "https://ipace-owners.org/member/vehicle/", CampaignID: "join-nudge-2026-07",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "resend-message-id" {
		t.Fatalf("id = %q", id)
	}
}
