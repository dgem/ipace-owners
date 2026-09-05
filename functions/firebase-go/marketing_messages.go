package ipace

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	resend "github.com/resend/resend-go/v4"
)

const marketingMessageMarkdownMax = 20000

var marketingMessagePlaceholderRegexp = regexp.MustCompile(`\{\{[^}]+\}\}`)

type marketingMessageRequest struct {
	Name             string `json:"name"`
	Subject          string `json:"subject"`
	Markdown         string `json:"markdown"`
	ExpectedEligible int    `json:"expectedEligible"`
	Confirmation     string `json:"confirmation"`
}

type marketingMessagePreview struct {
	Eligible     int    `json:"eligible"`
	Subject      string `json:"subject"`
	HTML         string `json:"html"`
	Text         string `json:"text"`
	Confirmation string `json:"confirmation"`
	Notice       string `json:"notice"`
}

type marketingMessageSent struct {
	Eligible    int    `json:"eligible"`
	BroadcastID string `json:"broadcastId"`
	Message     string `json:"message"`
}

var marketingMessageAudience = loadMarketingMessageAudience
var marketingMessageDeliver = sendMarketingMessage

func AdminMarketingMessagePreview(w http.ResponseWriter, r *http.Request) {
	if !adminMarketingMessageRequestAllowed(w, r) {
		return
	}
	var input marketingMessageRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := previewMarketingMessage(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func AdminMarketingMessageSend(w http.ResponseWriter, r *http.Request) {
	if !adminMarketingMessageRequestAllowed(w, r) {
		return
	}
	var input marketingMessageRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := previewMarketingMessage(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if input.ExpectedEligible != preview.Eligible {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "The consented audience changed; preview again before sending."})
		return
	}
	if input.Confirmation != preview.Confirmation {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Confirmation did not match; no message was sent."})
		return
	}
	audience, err := marketingMessageAudience(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not calculate the consented audience"})
		return
	}
	id, err := marketingMessageDeliver(r.Context(), input, audience)
	if err != nil {
		logEvent("admin-marketing-message-send", "error", "broadcast failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Resend could not create the broadcast; no message was sent."})
		return
	}
	logEvent("admin-marketing-message-send", "info", "broadcast created", map[string]any{"eligible": len(audience), "broadcastId": id})
	writeJSON(w, http.StatusOK, marketingMessageSent{Eligible: len(audience), BroadcastID: id, Message: "The broadcast has been handed to Resend for delivery."})
}

func adminMarketingMessageRequestAllowed(w http.ResponseWriter, r *http.Request) bool {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return false
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return false
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return false
	}
	return true
}

func previewMarketingMessage(ctx context.Context, input marketingMessageRequest) (marketingMessagePreview, error) {
	if err := validateMarketingMessage(input); err != nil {
		return marketingMessagePreview{}, err
	}
	audience, err := marketingMessageAudience(ctx)
	if err != nil {
		return marketingMessagePreview{}, err
	}
	markdown := marketingMessageMarkdown(input.Markdown)
	return marketingMessagePreview{Eligible: len(audience), Subject: strings.TrimSpace(input.Subject), HTML: marketingMessageHTML(markdown), Text: marketingMessageText(markdown), Confirmation: fmt.Sprintf("SEND %d", len(audience)), Notice: "Only members who opted in to group communications are included. Previewing never sends email."}, nil
}

func validateMarketingMessage(input marketingMessageRequest) error {
	if len(strings.TrimSpace(input.Name)) < 3 || len(strings.TrimSpace(input.Name)) > 160 {
		return fmt.Errorf("A message name between 3 and 160 characters is required")
	}
	if len(strings.TrimSpace(input.Subject)) < 3 || len(strings.TrimSpace(input.Subject)) > 200 {
		return fmt.Errorf("A subject between 3 and 200 characters is required")
	}
	if len(strings.TrimSpace(input.Markdown)) == 0 || len(input.Markdown) > marketingMessageMarkdownMax {
		return fmt.Errorf("Message content is required and must be %d characters or fewer", marketingMessageMarkdownMax)
	}
	if err := validateEmailMarkdownLinks(input.Markdown); err != nil {
		return err
	}
	if strings.Contains(input.Subject, "{{") {
		return fmt.Errorf("Personalisation is supported in the message body only")
	}
	for _, token := range marketingMessagePlaceholderRegexp.FindAllString(input.Markdown, -1) {
		if token != "{{firstName}}" {
			return fmt.Errorf("Only {{firstName}} is supported for personalisation")
		}
	}
	return nil
}

func loadMarketingMessageAudience(ctx context.Context) ([]campaignRecipient, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return nil, err
	}
	// loadCampaignJoins is the canonical, consent-only Join audience. It does not require an account sign-in.
	return loadCampaignJoins(ctx, db)
}

func marketingMessageMarkdown(markdown string) string {
	return strings.ReplaceAll(markdown, "{{firstName}}", "{{{contact.first_name|member}}}")
}
func marketingMessageHTML(markdown string) string {
	body := markdownToEmailHTML(markdown)
	return `<!doctype html><html><body style="margin:0;padding:24px;background:#f7f8fb;font-family:Arial,sans-serif;"><main style="max-width:640px;margin:auto;background:#fff;padding:32px;border-radius:12px;">` + body + `<hr style="border:0;border-top:1px solid #dbe3ec;margin:28px 0 16px;"><p style="font-size:13px;color:#4b5563;">You are receiving this because you chose to receive group communications. <a href="{{{RESEND_UNSUBSCRIBE_URL}}}">Unsubscribe</a>.</p></main></body></html>`
}
func marketingMessageText(markdown string) string {
	return markdownToPlainText(markdown) + "\nYou are receiving this because you chose to receive group communications. Unsubscribe: {{{RESEND_UNSUBSCRIBE_URL}}}\n"
}

func sendMarketingMessage(ctx context.Context, input marketingMessageRequest, audience []campaignRecipient) (string, error) {
	if !resendEmailConfigured() {
		return "", fmt.Errorf("Resend delivery is not configured")
	}
	client := resend.NewClient(strings.TrimSpace(os.Getenv("RESEND_API_KEY")))
	segment, err := client.Segments.CreateWithContext(ctx, &resend.CreateSegmentRequest{Name: "I-PACE consented members " + time.Now().UTC().Format("2006-01-02 15:04")})
	if err != nil {
		return "", err
	}
	if err := importMarketingContacts(ctx, client, segment.Id, audience); err != nil {
		return "", err
	}
	req := &resend.CreateBroadcastRequest{SegmentId: segment.Id, Name: strings.TrimSpace(input.Name), From: strings.TrimSpace(os.Getenv("RESEND_FROM")), Subject: strings.TrimSpace(input.Subject), Html: marketingMessageHTML(marketingMessageMarkdown(input.Markdown)), Text: marketingMessageText(marketingMessageMarkdown(input.Markdown)), Send: true}
	if reply := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); reply != "" {
		req.ReplyTo = []string{reply}
	}
	broadcast, err := client.Broadcasts.CreateWithContext(ctx, req)
	if err != nil {
		return "", err
	}
	return broadcast.Id, nil
}

func importMarketingContacts(ctx context.Context, client *resend.Client, segmentID string, audience []campaignRecipient) error {
	var csvData bytes.Buffer
	writer := csv.NewWriter(&csvData)
	if err := writer.Write([]string{"email", "first_name", "last_name"}); err != nil {
		return err
	}
	for _, person := range audience {
		_, first, last := splitCampaignMemberName(person.Name)
		if err := writer.Write([]string{person.Email, first, last}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	imported, err := client.Contacts.Imports.CreateWithContext(ctx, &resend.CreateContactImportRequest{File: csvData.Bytes(), Filename: "consented-members.csv", ColumnMap: map[string]any{"email": "email", "first_name": "first_name", "last_name": "last_name"}, OnConflict: "upsert", Segments: []resend.ContactImportSegment{{Id: segmentID}}})
	if err != nil {
		return err
	}
	deadline := time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		status, err := client.Contacts.Imports.GetWithContext(ctx, imported.Id)
		if err != nil {
			return err
		}
		switch status.Status {
		case resend.ContactImportStatusCompleted:
			return nil
		case resend.ContactImportStatusFailed:
			return fmt.Errorf("Resend contact import failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("Resend contact import did not complete in time")
}
