package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	if strings.TrimSpace(preview.Subject) == "" {
		t.Fatal("preview subject is empty")
	}
	for _, expected := range []string{"[A fresh, private sign-in link is inserted for each recipient]"} {
		if !strings.Contains(preview.Text, expected) {
			t.Fatalf("text preview missing %q: %q", expected, preview.Text)
		}
	}
	if strings.Contains(preview.Text, "{{.") {
		t.Fatalf("text preview contains an unresolved template field: %q", preview.Text)
	}
	for _, expected := range []string{"<!doctype html>", "Verify my account details", "/images/ipace-hero.png"} {
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
		"&lt;Jane",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML email missing %q", expected)
		}
	}
	for _, expected := range []string{"22 July 2026", "https://example.com/sign-in?a=1&b=2"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("plain-text email missing rendered value %q: %q", expected, text)
		}
	}
	if strings.Contains(text, "{{.") || strings.Contains(htmlBody, "{{.") || strings.Contains(htmlBody, "<Jane") {
		t.Fatalf("rendered email contains an unresolved template field")
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

func TestMemberReferralEmailRendersDynamicDataAndProvidesShares(t *testing.T) {
	preview := makeMemberReferralEmailPreview(371)
	if strings.TrimSpace(preview.Subject) == "" || strings.TrimSpace(preview.Text) == "" {
		t.Fatal("preview subject or text is empty")
	}
	if strings.Contains(preview.Text, "{{.") || strings.Contains(preview.HTML, "{{.") {
		t.Fatalf("preview contains an unresolved template field")
	}
	for _, expected := range []string{"<!doctype html>", "/images/ipace-hero.png", `href="https://ipace-owners.org/"`, "Facebook", "WhatsApp"} {
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

func TestMemberReferralShareButtonsCarrySuggestedCTAWhereSupported(t *testing.T) {
	expected := memberReferralShareMessage(412)
	if !strings.Contains(expected, "stronger together") || !strings.Contains(expected, "Own an I-PACE?") || !strings.Contains(expected, "help us reach 1,000") {
		t.Fatalf("suggested share CTA is incomplete: %q", expected)
	}
	for _, share := range memberReferralShareLinks(412) {
		parsed, err := url.Parse(share.URL)
		if err != nil {
			t.Fatalf("%s URL: %v", share.Label, err)
		}
		var actual string
		switch share.Label {
		case "Facebook":
			actual = parsed.Query().Get("quote")
		case "X", "Bluesky", "WhatsApp":
			actual = parsed.Query().Get("text")
		case "Email":
			actual = parsed.Query().Get("body")
		default:
			continue
		}
		if actual != expected {
			t.Fatalf("%s suggested text=%q", share.Label, actual)
		}
	}
}

func TestMemberReferralEmailUsesSharedBrandingAndActionButtons(t *testing.T) {
	_, htmlBody, text, _ := memberReferralEmailBodies(campaignRecipient{Name: "Jane"}, 371)
	for _, expected := range []string{
		">I-PACE Owners</div>",
		"/images/ipace-hero.png",
		`href="https://ipace-owners.org/"`,
		"www.facebook.com/sharer/sharer.php",
		"https://wa.me/",
		"www.instagram.com/ipaceowners/",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("HTML email missing %q", expected)
		}
	}
	if !strings.Contains(text, "https://ipace-owners.org/") {
		t.Fatalf("plain-text email missing share fallback: %q", text)
	}
	if strings.Contains(htmlBody, "/images/ipace-owners-logo") {
		t.Fatalf("referral email must use the text masthead, not a logo image: %q", htmlBody)
	}
}

func TestAllMembersDriveEmailUsesRequestedRecruitmentMessage(t *testing.T) {
	preview := makeAllMembersDriveEmailPreview(412)
	for _, expected := range []string{
		"412 members",
		"Thank you for joining",
		"Thank you for your support",
		"17 July—less than two weeks ago",
		"I-PACE owners are stronger together",
		"Own an I-PACE? Add your voice",
		"formally approaching Jaguar",
		"1,000 members in less than a month",
		"approximately 30,000 I-PACEs",
		"https://stillontheroad.co.uk/cars/jaguar/i-pace",
		"traction-battery faults",
		"Technical Service Bulletins",
		"Facebook",
		"WhatsApp",
	} {
		if !strings.Contains(preview.HTML, expected) && !strings.Contains(preview.Text, expected) {
			t.Fatalf("campaign preview missing %q", expected)
		}
	}
	if strings.Contains(preview.HTML, "{{.") || strings.Contains(preview.Text, "{{.") {
		t.Fatal("campaign preview contains unresolved template values")
	}
}

func TestAllMembersDrivePreviewRequiresAdmin(t *testing.T) {
	original := campaignAuthorize
	t.Cleanup(func() { campaignAuthorize = original })
	campaignAuthorize = func(context.Context, *http.Request) error { return context.Canceled }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/all-members-drive-preview", strings.NewReader(`{}`))
	res := httptest.NewRecorder()
	AdminAllMembersDrivePreview(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
