package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	for _, expected := range []string{
		"You asked to join on",
		"your email address has not yet been verified",
		"fresh, private sign-in link",
	} {
		if !strings.Contains(preview.Text, expected) {
			t.Fatalf("text preview missing %q: %q", expected, preview.Text)
		}
	}
	for _, expected := range []string{"<!doctype html>", "Please complete your registration", "fresh, private sign-in link", "/images/ipace-hero.png"} {
		if !strings.Contains(preview.HTML, expected) {
			t.Fatalf("HTML preview missing %q: %q", expected, preview.HTML)
		}
	}
}

func TestCampaignEmailUsesMarkdownContentAndSharedBranding(t *testing.T) {
	_, htmlBody, text := campaignEmailBodies(campaignRecipient{
		Name:      `<Jane & Co>`,
		CreatedAt: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
	}, "https://example.com/sign-in?a=1&b=2", 371, 12)
	for _, expected := range []string{
		"<!doctype html>",
		">I-PACE Owners</div>",
		"/images/ipace-hero.png",
		"Verify my account details",
		"https://example.com/sign-in?a=1&amp;b=2",
		"Hello &lt;Jane,",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML email missing %q", expected)
		}
	}
	if !strings.Contains(text, "You asked to join on 22 July 2026") {
		t.Fatalf("plain-text email did not render Markdown template: %q", text)
	}
	if strings.Contains(htmlBody, "/images/ipace-owners-logo") || strings.Count(htmlBody, "agreed that we could contact you") != 1 {
		t.Fatalf("HTML email contains a logo or duplicate consent footer: %q", htmlBody)
	}
	if strings.Index(text, "Verify your account details:") > strings.Index(text, "You are receiving this because") {
		t.Fatalf("plain-text action must appear before the consent footer: %q", text)
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
	for _, expected := range []string{"<!doctype html>", "/images/ipace-hero.png", "Visit I-PACE Owners", "Facebook", "WhatsApp"} {
		if !strings.Contains(preview.HTML, expected) {
			t.Fatalf("HTML preview missing %q: %s", expected, preview.HTML)
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
	instagramURL := strings.Join([]string{"https:", "", "www.instagram.com", "ipaceowners", ""}, "/")
	for _, share := range preview.Shares {
		if share.Label == "Instagram" && share.URL != instagramURL {
			t.Fatalf("unexpected Instagram profile: %q", share.URL)
		}
	}
}

func TestMemberReferralEmailUsesSharedBrandingAndActionButtons(t *testing.T) {
	_, htmlBody, text, _ := memberReferralEmailBodies(campaignRecipient{Name: "Jane"}, 371)
	for _, expected := range []string{
		">I-PACE Owners</div>",
		"/images/ipace-hero.png",
		"Visit I-PACE Owners",
		"www.facebook.com/sharer/sharer.php",
		"https://wa.me/",
		"@ipaceowners profile",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML email missing %q", expected)
		}
	}
	if !strings.Contains(text, "Share the group: https://ipace-owners.org/") {
		t.Fatalf("plain-text email missing share fallback: %q", text)
	}
	if strings.Contains(htmlBody, "/images/ipace-owners-logo") {
		t.Fatalf("referral email must use the text masthead, not a logo image: %q", htmlBody)
	}
}
