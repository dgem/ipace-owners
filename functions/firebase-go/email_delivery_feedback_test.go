package ipace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type feedbackRoundTripFunc func(*http.Request) (*http.Response, error)

func (function feedbackRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestFetchResendEmailStatusesPaginatesAndKeepsOnlyCampaignProviderIDs(t *testing.T) {
	originalEndpoint, originalClient := resendEmailsEndpoint, resendFeedbackHTTPClient
	t.Cleanup(func() {
		resendEmailsEndpoint = originalEndpoint
		resendFeedbackHTTPClient = originalClient
	})
	requests := 0
	resendEmailsEndpoint = "https://resend.test/emails"
	resendFeedbackHTTPClient = &http.Client{Transport: feedbackRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
		}
		page := resendEmailStatusPage{
			HasMore: true,
			Data: []resendEmailStatus{
				{ID: "campaign-email-1", LastEvent: "delivered"},
				{ID: "unrelated-magic-link", LastEvent: "bounced"},
			},
		}
		if r.URL.Query().Get("after") != "" {
			page.HasMore = false
			page.Data = []resendEmailStatus{{ID: "campaign-email-2", LastEvent: "bounced"}}
		}
		var body strings.Builder
		if err := json.NewEncoder(&body).Encode(page); err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body.String())),
			Header:     make(http.Header),
		}, nil
	})}

	statuses, complete, err := fetchResendEmailStatuses(context.Background(), "test-key", map[string]struct{}{
		"campaign-email-1": {},
		"campaign-email-2": {},
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !complete {
		t.Fatalf("requests=%d complete=%v", requests, complete)
	}
	if len(statuses) != 2 || statuses["campaign-email-1"] != "delivered" || statuses["campaign-email-2"] != "bounced" {
		t.Fatalf("statuses=%#v", statuses)
	}
	if _, ok := statuses["unrelated-magic-link"]; ok {
		t.Fatal("unrelated email status must not be retained")
	}
}

func TestAddDeliveryFeedbackClassifiesProviderOutcomes(t *testing.T) {
	var record customCampaignRecord
	for _, providerStatus := range []string{
		"delivered", "opened", "clicked", "complained", "delivery_delayed",
		"bounced", "failed", "suppressed", "canceled", "sent", "unknown",
	} {
		addDeliveryFeedback(&record, providerStatus)
	}
	if record.Delivered != 4 || record.Opened != 1 || record.Clicked != 1 ||
		record.Complained != 1 || record.Delayed != 1 || record.Bounced != 1 ||
		record.ProviderFailed != 2 || record.Suppressed != 1 || record.Undeliverable != 4 ||
		record.AwaitingDelivery != 1 {
		t.Fatalf("record=%#v", record)
	}
}

func TestFeedbackCoverageMessageExplainsPartialProviderHistory(t *testing.T) {
	if message := feedbackCoverageMessage(false); !strings.Contains(message, "partial") {
		t.Fatalf("message = %q", message)
	}
	if message := feedbackCoverageMessage(true); message != "" {
		t.Fatalf("message = %q", message)
	}
}

func TestMergeResendEmailStatusesMakesFreshOutcomesImmediatelyAvailable(t *testing.T) {
	deliveries := []campaignDeliveryFeedback{
		{ResendID: "email-1", ProviderStatus: "sent"},
		{ResendID: "email-2", ProviderStatus: "delivered"},
		{ResendID: "email-3", ProviderStatus: "sent"},
	}
	changed := mergeResendEmailStatuses(deliveries, map[string]string{
		"email-1": "bounced",
		"email-2": "delivered",
	})
	if len(changed) != 1 || changed[0] != 0 {
		t.Fatalf("changed=%v", changed)
	}
	if deliveries[0].ProviderStatus != "bounced" || deliveries[1].ProviderStatus != "delivered" ||
		deliveries[2].ProviderStatus != "sent" {
		t.Fatalf("deliveries=%#v", deliveries)
	}
}
