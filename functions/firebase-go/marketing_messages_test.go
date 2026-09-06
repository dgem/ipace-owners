package ipace

import (
	"context"
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
	if !strings.Contains(preview.HTML, "{{{contact.first_name|member}}}") || !strings.Contains(preview.HTML, "{{{RESEND_UNSUBSCRIBE_URL}}}") {
		t.Fatalf("missing Resend substitutions: %s", preview.HTML)
	}
	if !strings.Contains(preview.Text, "{{{RESEND_UNSUBSCRIBE_URL}}}") {
		t.Fatalf("missing plain-text unsubscribe: %s", preview.Text)
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
