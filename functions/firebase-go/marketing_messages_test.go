package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarketingMessagePreviewUsesConsentedAudienceAndPersonalisation(t *testing.T) {
	original := marketingMessageAudience
	t.Cleanup(func() { marketingMessageAudience = original })
	marketingMessageAudience = func(context.Context) ([]campaignRecipient, error) {
		return []campaignRecipient{{Name: "Jane Driver", Email: "jane@example.com"}, {Name: "Sam Owner", Email: "sam@example.com"}}, nil
	}
	preview, err := previewMarketingMessage(context.Background(), marketingMessageRequest{Name: "September update", Subject: "Hello", Markdown: "Hello {{firstName}}\n\n[Read more](https://ipace-owners.org/)"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Eligible != 2 || preview.Confirmation != "SEND 2" {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if !strings.Contains(preview.HTML, "Hello Jane") || !strings.Contains(preview.HTML, "/api/email-unsubscribe?campaign=preview") {
		t.Fatalf("missing direct-send personalisation or unsubscribe link: %s", preview.HTML)
	}
	if !strings.Contains(preview.Text, "/api/email-unsubscribe?campaign=preview") {
		t.Fatalf("missing plain-text unsubscribe link: %s", preview.Text)
	}
}

func TestMarketingMessageBatchUsesOpaqueUnsubscribeTokens(t *testing.T) {
	token := "unsubscribe_0123456789abcdef"
	hash := marketingUnsubscribeTokenHash(token)
	if hash == token || len(hash) != 64 {
		t.Fatalf("unsubscribe token hash = %q", hash)
	}
	url := marketingUnsubscribeURL("marketing_123", token)
	if !strings.Contains(url, "campaign=marketing_123") || !strings.Contains(url, "token="+token) {
		t.Fatalf("unsubscribe URL = %q", url)
	}
	if message := marketingMessageBatchMessage(100, 0, 12); !strings.Contains(message, "12 remain") {
		t.Fatalf("batch continuation message = %q", message)
	}
	if message := marketingMessageBatchMessage(99, 1, 0); !strings.Contains(message, "excluded from automatic retries") {
		t.Fatalf("failed-recipient message = %q", message)
	}
	if got := renderMarketingMessageMarkdown("Hello {{firstName}}", campaignRecipient{Email: "noname@example.com"}); got != "Hello member" {
		t.Fatalf("missing-name personalisation = %q", got)
	}
}

func TestMarketingMessageBatchLogFieldsAreAggregateOnly(t *testing.T) {
	fields := marketingMessageBatchLogFields(marketingMessageSent{
		CampaignID:  "marketing_123",
		Eligible:    1300,
		BatchSent:   100,
		BatchFailed: 1,
		Sent:        400,
		Failed:      2,
		Remaining:   898,
	})
	if fields["campaignId"] != "marketing_123" || fields["eligible"] != 1300 || fields["batchSent"] != 100 || fields["batchFailed"] != 1 || fields["sent"] != 400 || fields["failed"] != 2 || fields["remaining"] != 898 {
		t.Fatalf("unexpected batch log fields: %#v", fields)
	}
	if len(fields) != 7 {
		t.Fatalf("batch log fields must not contain recipient data: %#v", fields)
	}
}

func TestMarketingMessageCampaignIDIsStableForIdenticalContent(t *testing.T) {
	input := marketingMessageRequest{Name: "September survey", Subject: "Have your say", Markdown: "Hi {{firstName}}"}
	first := marketingMessageCampaignID(input)
	if first == "" || first != marketingMessageCampaignID(input) {
		t.Fatalf("campaign ID is not stable: %q", first)
	}
	input.Markdown += " now"
	if first == marketingMessageCampaignID(input) {
		t.Fatal("campaign ID did not change when the message changed")
	}
	prepared := marketingMessageRequest{TemplateID: "survey-september-2026", Name: "September survey", Subject: "Have your say", Markdown: "1299 members"}
	preparedID := marketingMessageCampaignID(prepared)
	prepared.Markdown = "1300 members"
	if preparedID != marketingMessageCampaignID(prepared) {
		t.Fatal("prepared campaign ID changed with live aggregate values")
	}
	legacyPrepared := marketingMessageRecord{TemplateID: prepared.TemplateID, Name: prepared.Name, Subject: prepared.Subject, Markdown: "1299 members"}
	if !marketingMessageRecordsMatch(legacyPrepared, prepared) {
		t.Fatal("prepared campaign did not match its legacy rendered copy")
	}
	merged := map[string]string{"sent": "sent", "failed": "failed"}
	mergeMarketingMessageDeliveries(merged, map[string]string{"sent": "attempting", "failed": "sent", "new": "attempting"})
	if merged["sent"] != "sent" || merged["failed"] != "sent" || merged["new"] != "attempting" {
		t.Fatalf("merged delivery states = %#v", merged)
	}
	if !shouldReplaceMarketingMessageDelivery(marketingMessageDelivery{Status: "attempting"}, marketingMessageDelivery{Status: "sent"}) {
		t.Fatal("sent delivery did not replace an uncertain legacy delivery")
	}
	if shouldReplaceMarketingMessageDelivery(marketingMessageDelivery{Status: "sent"}, marketingMessageDelivery{Status: "failed"}) {
		t.Fatal("failed delivery replaced a sent delivery")
	}
	statuses := map[string]string{
		campaignEmailFingerprint("sent@example.com"):   "sent",
		campaignEmailFingerprint("failed@example.com"): "failed",
		campaignEmailFingerprint("held@example.com"):   "attempting",
	}
	sent, failed := countMarketingMessageDeliveries([]campaignRecipient{{Email: "sent@example.com"}, {Email: "failed@example.com"}, {Email: "held@example.com"}}, statuses)
	if sent != 1 || failed != 2 {
		t.Fatalf("delivery count = %d sent, %d failed; want 1, 2", sent, failed)
	}
	if failures := countRecordedMarketingMessageFailures(statuses); failures != 2 {
		t.Fatalf("recorded delivery failures = %d, want 2", failures)
	}
}

func TestMarketingAudienceIncludesLegacyVerifiedMembersAndHonoursOptOuts(t *testing.T) {
	joins := map[string]marketingJoinConsent{
		"current@example.com":   {Recipient: campaignRecipient{Name: "Current Member", Email: "current@example.com"}, Contact: true},
		"opted-out@example.com": {Contact: false},
	}
	legacy := []campaignRecipient{
		{Name: "Current Account", Email: "current@example.com"},
		{Name: "Legacy Member", Email: "legacy@example.com"},
		{Name: "Opted Out", Email: "opted-out@example.com"},
		{Name: "Preference Opt Out", Email: "preference@example.com"},
	}
	preferences := map[string]communicationsConsentRecord{
		emailFingerprint("preference@example.com"): {Contact: false, Source: "unsubscribe"},
	}
	audience := marketingAudienceFromSources(joins, legacy, preferences)
	if len(audience) != 2 {
		t.Fatalf("audience = %#v, want current and legacy members", audience)
	}
	if audience[0].Email != "current@example.com" || audience[1].Email != "legacy@example.com" {
		t.Fatalf("audience = %#v, want sorted current and legacy members", audience)
	}
}

func TestMarketingMessageUnsubscribeGETRequiresConfirmation(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/email-unsubscribe?campaign=marketing_123&token=unsubscribe_0123456789abcdef", nil)
	response := httptest.NewRecorder()
	MarketingMessageUnsubscribe(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !strings.Contains(body, "Stop group emails?") || !strings.Contains(body, "method=\"post\"") {
		t.Fatalf("missing unsubscribe confirmation: %s", body)
	}
}

func TestMarketingMessageTemplatesMoveSeptemberSurveyToBroadcasts(t *testing.T) {
	originalStats, originalAudience := marketingMessageStats, marketingMessageAudience
	t.Cleanup(func() {
		marketingMessageStats = originalStats
		marketingMessageAudience = originalAudience
	})
	marketingMessageStats = func(context.Context) (publicStatsSnapshot, error) {
		return publicStatsSnapshot{JoinedOwners: 1299, VehiclesRegistered: 568, SOHReadings: 96, ServiceEventsLogged: 135}, nil
	}
	marketingMessageAudience = func(context.Context) ([]campaignRecipient, error) {
		return []campaignRecipient{{Email: "jane@example.com"}}, nil
	}
	templates, err := marketingMessageTemplates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != 4 {
		t.Fatalf("template count = %d, want 4", len(templates))
	}
	var survey marketingMessageTemplate
	for _, template := range templates {
		if template.ID == "survey-september-2026" {
			survey = template
		}
	}
	if survey.ID == "" {
		t.Fatal("September survey template is missing")
	}
	for _, expected := range []string{
		"{{firstName}}",
		"survey_38447815d17b0e954a4edbca1b9600c9",
		"Have your say — tell us what you need",
		"1299 members",
		"568 cars registered",
		"96 State of Health readings",
		"135 service and fault records",
	} {
		if !strings.Contains(survey.Markdown, expected) {
			t.Fatalf("survey template is missing %q", expected)
		}
	}
	if survey.HeroImage != "/images/september-survey-2026-hero.jpg" {
		t.Fatalf("survey hero = %q", survey.HeroImage)
	}
	preview, err := previewMarketingMessage(context.Background(), marketingMessageRequest{TemplateID: survey.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.HTML, "https://ipace-owners.org/images/september-survey-2026-hero.jpg") {
		t.Fatalf("template preview omitted hero: %s", preview.HTML)
	}
}

func TestValidateMarketingMessageRestrictsPersonalisation(t *testing.T) {
	valid := marketingMessageRequest{Name: "September update", Subject: "Hello", Markdown: "Hello {{firstName}}"}
	if err := validateMarketingMessage(valid); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}
	invalid := valid
	invalid.Markdown = "Hello {{vehicleRegistration}}"
	if err := validateMarketingMessage(invalid); err == nil {
		t.Fatal("unexpected placeholder was accepted")
	}
	invalid.Markdown = "Hello {{firstName}} {{vehicleRegistration}}"
	if err := validateMarketingMessage(invalid); err == nil {
		t.Fatal("mixed placeholders were accepted")
	}
}
