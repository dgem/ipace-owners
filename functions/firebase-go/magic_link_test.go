package ipace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMagicLinkSuppressesUnregisteredEmail(t *testing.T) {
	restore := stubMagicLinkDependencies(t, 0, nil, nil)
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/send-magic-link", strings.NewReader(`{"email":"driver@example.com"}`))
	req.Header.Set("Origin", "https://ipace-owners.org")
	rec := httptest.NewRecorder()

	SendMagicLink(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("ok = %v, want true", body["ok"])
	}
}

func TestSendMagicLinkSendsForRegisteredEmail(t *testing.T) {
	sent := ""
	restore := stubMagicLinkDependencies(t, 2, nil, func(_ context.Context, email string, origin string) error {
		sent = email
		if origin != "https://ipace-owners.org" {
			t.Fatalf("origin = %q, want request origin", origin)
		}
		return nil
	})
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/send-magic-link", strings.NewReader(`{"email":" DRIVER@example.com "}`))
	req.Header.Set("Origin", "https://ipace-owners.org")
	rec := httptest.NewRecorder()

	SendMagicLink(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if sent != "driver@example.com" {
		t.Fatalf("sent email = %q, want cleaned registered email", sent)
	}
}

func TestSendMagicLinkSuppressesWhenRegistrationCheckFails(t *testing.T) {
	restore := stubMagicLinkDependencies(t, 0, errors.New("firestore unavailable"), nil)
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/send-magic-link", strings.NewReader(`{"email":"driver@example.com"}`))
	req.Header.Set("Origin", "https://ipace-owners.org")
	rec := httptest.NewRecorder()

	SendMagicLink(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func stubMagicLinkDependencies(t *testing.T, joinCount int, joinErr error, sender func(context.Context, string, string) error) func() {
	t.Helper()
	originalCount := joinSubmissionCount
	originalSender := sendFirebaseEmailLink
	sendCalled := false

	joinSubmissionCount = func(_ context.Context, emailHash string) (int, error) {
		if emailHash == "" {
			t.Fatal("joinSubmissionCount received empty email hash")
		}
		return joinCount, joinErr
	}
	sendFirebaseEmailLink = func(ctx context.Context, email string, origin string) error {
		sendCalled = true
		if sender != nil {
			return sender(ctx, email, origin)
		}
		t.Fatalf("sendFirebaseEmailLink was called for %q", email)
		return nil
	}

	return func() {
		joinSubmissionCount = originalCount
		sendFirebaseEmailLink = originalSender
		if sender == nil && sendCalled {
			t.Fatal("sendFirebaseEmailLink was unexpectedly called")
		}
	}
}
