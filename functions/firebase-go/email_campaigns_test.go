package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClassifyCampaignRecipientsSuppressesAliasesAndNames(t *testing.T) {
	joins := []campaignRecipient{{Name: "Jane Driver", Email: "jane+owners@example.com"}, {Name: "Other Person", Email: "other@example.com"}}
	registered := map[string]bool{"jane@example.com": true}
	got := classifyCampaignRecipients(joins, registered)
	if len(got) != 1 || got[0].Email != "other@example.com" {
		t.Fatalf("unexpected eligible recipients: %#v", got)
	}
	registered = map[string]bool{"name:" + normalizedCampaignName("Other Person"): true}
	got = classifyCampaignRecipients(joins, registered)
	if len(got) != 1 || got[0].Email != "jane+owners@example.com" {
		t.Fatalf("unexpected name suppression: %#v", got)
	}
}

func TestAdminReengagementPreviewRequiresAdmin(t *testing.T) {
	original := campaignAuthorize
	t.Cleanup(func() { campaignAuthorize = original })
	campaignAuthorize = func(context.Context, *http.Request) error { return context.Canceled }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reengagement-preview", strings.NewReader(`{}`))
	res := httptest.NewRecorder()
	AdminReengagementPreview(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminReengagementSendRejectsBadBodyBeforeSending(t *testing.T) {
	originalAuth, originalSend := campaignAuthorize, campaignSend
	t.Cleanup(func() { campaignAuthorize = originalAuth; campaignSend = originalSend })
	campaignAuthorize = func(context.Context, *http.Request) error { return nil }
	called := false
	campaignSend = func(context.Context, campaignSendRequest) (campaignSummary, error) {
		called = true
		return campaignSummary{}, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/reengagement-send", strings.NewReader(`{"campaignId":`))
	res := httptest.NewRecorder()
	AdminReengagementSend(res, req)
	if res.Code != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%v", res.Code, called)
	}
}

func TestCampaignSummaryNeverReportsNegativeRemaining(t *testing.T) {
	got := makeCampaignSummary("campaign", 2, 3, 4, 0)
	if got.Remaining != 0 {
		t.Fatalf("remaining=%d", got.Remaining)
	}
}

func TestCampaignEmailPreviewUsesTheDeliveryTemplate(t *testing.T) {
	preview := makeCampaignEmailPreview(371, 12)
	if preview.Subject != "Complete your I-PACE Owners registration" {
		t.Fatalf("unexpected preview subject: %q", preview.Subject)
	}
	if !strings.Contains(preview.Text, "371 owners have joined") || !strings.Contains(preview.Text, "fresh, private sign-in link") {
		t.Fatalf("preview missing audience context: %q", preview.Text)
	}
}

func TestMemberReferralAudienceRequiresRegistrationAndContactConsent(t *testing.T) {
	joins := []campaignRecipient{{Name: "Jane Driver", Email: "jane+owners@example.com"}, {Name: "Not Registered", Email: "other@example.com"}}
	registered := map[string]string{"jane@example.com": "jane@example.com"}
	got := classifyMemberReferralRecipients(joins, registered)
	if len(got) != 1 || got[0].Email != "jane@example.com" {
		t.Fatalf("unexpected referral audience: %#v", got)
	}
}

func TestMemberReferralEmailExplainsGoalAndProvidesShares(t *testing.T) {
	preview := makeMemberReferralEmailPreview(371)
	for _, expected := range []string{"371 owners have joined", "629 members away", "grow to 742 members", "in the 700s"} {
		if !strings.Contains(preview.Text, expected) {
			t.Fatalf("preview missing %q: %s", expected, preview.Text)
		}
	}
	labels := map[string]bool{}
	for _, share := range preview.Shares {
		labels[share.Label] = true
	}
	for _, expected := range []string{"Facebook", "X", "Bluesky", "LinkedIn", "Instagram", "WhatsApp", "Email"} {
		if !labels[expected] {
			t.Fatalf("missing %s share action", expected)
		}
	}
}
