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
