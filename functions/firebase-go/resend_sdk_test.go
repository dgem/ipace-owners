package ipace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/resend/resend-go/v3"
)

type resendSDKRoundTripper func(*http.Request) (*http.Response, error)

func (function resendSDKRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func useResendTestTransport(t *testing.T, handler func(*http.Request) map[string]any) {
	t.Helper()
	original := newResendEmailClient
	newResendEmailClient = func(apiKey string) *resend.Client {
		client := resend.NewCustomClient(&http.Client{Transport: resendSDKRoundTripper(func(request *http.Request) (*http.Response, error) {
			response := handler(request)
			body, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
		})}, apiKey)
		baseURL, err := url.Parse("https://resend.test/")
		if err != nil {
			t.Fatal(err)
		}
		client.BaseURL = baseURL
		return client
	}
	t.Cleanup(func() {
		newResendEmailClient = original
	})
}

func decodeResendRequest(t *testing.T, request *http.Request) map[string]any {
	t.Helper()
	if request.URL.Path != "/emails" || request.Method != http.MethodPost {
		t.Fatalf("request = %s %s", request.Method, request.URL.Path)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer test-key" {
		t.Fatalf("authorization = %q", got)
	}
	var payload map[string]any
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestFunctionResendSendPathsUseSDKAPIContract(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "test-key")
	t.Setenv("RESEND_FROM", "I-PACE Owners <members@example.com>")
	t.Setenv("RESEND_REPLY_TO", "contact@example.com")

	t.Run("magic link", func(t *testing.T) {
		useResendTestTransport(t, func(request *http.Request) map[string]any {
			payload := decodeResendRequest(t, request)
			if payload["subject"] != "Your secure sign-in link for I-PACE Owners" || payload["reply_to"] != "contact@example.com" {
				t.Fatalf("payload = %#v", payload)
			}
			return map[string]any{"id": "magic-link-id"}
		})
		if err := sendResendMagicLinkEmailRequest(context.Background(), "member@example.com", "https://example.com/action", "https://example.com"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("campaign", func(t *testing.T) {
		useResendTestTransport(t, func(request *http.Request) map[string]any {
			payload := decodeResendRequest(t, request)
			if request.Header.Get("Idempotency-Key") != "campaign-1/"+campaignEmailFingerprint("member@example.com") || payload["subject"] != "Campaign subject" {
				t.Fatalf("request=%#v payload=%#v", request.Header, payload)
			}
			return map[string]any{"id": "campaign-id"}
		})
		id, err := sendCampaignPayload(context.Background(), "member@example.com", "Campaign subject", "<p>HTML</p>", "Text", "member-update", "campaign-1")
		if err != nil || id != "campaign-id" {
			t.Fatalf("id=%q err=%v", id, err)
		}
	})

	t.Run("marketing", func(t *testing.T) {
		useResendTestTransport(t, func(request *http.Request) map[string]any {
			payload := decodeResendRequest(t, request)
			if request.Header.Get("Idempotency-Key") != "marketing-1/"+campaignEmailFingerprint("member@example.com") {
				t.Fatalf("idempotency = %q", request.Header.Get("Idempotency-Key"))
			}
			headers := payload["headers"].(map[string]any)
			if headers["List-Unsubscribe"] != "<https://example.com/unsubscribe>" || headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
				t.Fatalf("headers = %#v", headers)
			}
			return map[string]any{"id": "marketing-id"}
		})
		id, err := sendMarketingMessagePayload(context.Background(), "member@example.com", "Marketing subject", "<p>HTML</p>", "Text", "marketing-1", "https://example.com/unsubscribe")
		if err != nil || id != "marketing-id" {
			t.Fatalf("id=%q err=%v", id, err)
		}
	})
}
