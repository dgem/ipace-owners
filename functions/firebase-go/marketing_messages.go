package ipace

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	resend "github.com/resend/resend-go/v4"
)

const marketingMessageMarkdownMax = 20000

var marketingMessagePlaceholderRegexp = regexp.MustCompile(`\{\{[^}]+\}\}`)

type marketingMessageRequest struct {
	TemplateID       string `json:"templateId"`
	Name             string `json:"name"`
	Subject          string `json:"subject"`
	Markdown         string `json:"markdown"`
	ExpectedEligible int    `json:"expectedEligible"`
	Confirmation     string `json:"confirmation"`
}

type marketingMessageTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Subject      string `json:"subject"`
	Markdown     string `json:"markdown"`
	HeroImage    string `json:"heroImage,omitempty"`
	HeroImageAlt string `json:"heroImageAlt,omitempty"`
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
var marketingMessageStats = buildPublicStatsSnapshot

func AdminMarketingMessageTemplates(w http.ResponseWriter, r *http.Request) {
	if !adminMarketingMessageRequestAllowed(w, r) {
		return
	}
	templates, err := marketingMessageTemplates(r.Context())
	if err != nil {
		logEvent("admin-marketing-message-templates", "error", "template load failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load marketing message templates"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates})
}

func AdminMarketingMessagePreview(w http.ResponseWriter, r *http.Request) {
	if !adminMarketingMessageRequestAllowed(w, r) {
		return
	}
	var input marketingMessageRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	resolved, err := resolvedMarketingMessage(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	input = resolved
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
	resolved, err := resolvedMarketingMessage(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	input = resolved
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
	var err error
	input, err = resolvedMarketingMessage(ctx, input)
	if err != nil {
		return marketingMessagePreview{}, err
	}
	if err := validateMarketingMessage(input); err != nil {
		return marketingMessagePreview{}, err
	}
	audience, err := marketingMessageAudience(ctx)
	if err != nil {
		return marketingMessagePreview{}, err
	}
	markdown := marketingMessageMarkdown(input.Markdown)
	return marketingMessagePreview{Eligible: len(audience), Subject: strings.TrimSpace(input.Subject), HTML: marketingMessageHTML(markdown, input.TemplateID), Text: marketingMessageText(markdown), Confirmation: fmt.Sprintf("SEND %d", len(audience)), Notice: "Only members who opted in to group communications are included. Previewing never sends email."}, nil
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
func marketingMessageHTML(markdown, templateID string) string {
	body := markdownToEmailHTML(markdown)
	hero := ""
	if template, ok := marketingMessageTemplateSource(templateID); ok && template.HeroImage != "" {
		hero = `<img src="https://ipace-owners.org` + template.HeroImage + `" alt="` + htmlEscape(template.HeroImageAlt) + `" style="display:block;width:100%;height:auto;border:0;border-radius:10px;margin:0 0 24px;">`
	}
	return `<!doctype html><html><body style="margin:0;padding:24px;background:#f7f8fb;font-family:Arial,sans-serif;"><main style="max-width:640px;margin:auto;background:#fff;padding:32px;border-radius:12px;">` + hero + body + `<hr style="border:0;border-top:1px solid #dbe3ec;margin:28px 0 16px;"><p style="font-size:13px;color:#4b5563;">You are receiving this because you chose to receive group communications. <a href="{{{RESEND_UNSUBSCRIBE_URL}}}">Unsubscribe</a>.</p></main></body></html>`
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
	req := &resend.CreateBroadcastRequest{SegmentId: segment.Id, Name: strings.TrimSpace(input.Name), From: strings.TrimSpace(os.Getenv("RESEND_FROM")), Subject: strings.TrimSpace(input.Subject), Html: marketingMessageHTML(marketingMessageMarkdown(input.Markdown), input.TemplateID), Text: marketingMessageText(marketingMessageMarkdown(input.Markdown)), Send: true}
	if reply := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); reply != "" {
		req.ReplyTo = []string{reply}
	}
	broadcast, err := client.Broadcasts.CreateWithContext(ctx, req)
	if err != nil {
		return "", err
	}
	return broadcast.Id, nil
}

func htmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(value)
}

// marketingMessageTemplates provides the group-wide messages that used to be
// dispatched through the resumable per-recipient campaign workspace.  They now
// share the consented-audience and provider-managed unsubscribe behaviour of a
// Resend Broadcast.  Registration reminders deliberately remain separate: each
// one needs a fresh private Firebase sign-in link.
func marketingMessageTemplates(ctx context.Context) ([]marketingMessageTemplate, error) {
	stats, err := marketingMessageStats(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]marketingMessageTemplate, 0, 4)
	for _, id := range []string{"survey-september-2026", "jlr-contact", "find-members", "reach-1000"} {
		template, ok := marketingMessageTemplateSource(id)
		if !ok {
			continue
		}
		template.Markdown = marketingTemplateMarkdown(template.Markdown, stats)
		result = append(result, template)
	}
	return result, nil
}

func marketingMessageTemplateSource(id string) (marketingMessageTemplate, bool) {
	file := map[string]string{
		"survey-september-2026": "survey-september-2026",
		"jlr-contact":           "jlr-contact",
		"find-members":          "member-referral",
		"reach-1000":            "all-members-drive",
	}[id]
	if file == "" {
		return marketingMessageTemplate{}, false
	}
	source, err := embeddedCampaignTemplate(file)
	if err != nil {
		return marketingMessageTemplate{}, false
	}
	description := map[string]string{
		"survey-september-2026": "Invite every consented member to the September preferred-outcomes survey.",
		"jlr-contact":           "Share the JLR meeting update and ask members to strengthen the evidence.",
		"find-members":          "Ask members to help another I-PACE owner find the group.",
		"reach-1000":            "Ask all consented members to share the group and grow the evidence base.",
	}[id]
	return marketingMessageTemplate{ID: id, Name: source.Name, Description: description, Subject: source.Subject, Markdown: source.Markdown, HeroImage: source.HeroImage, HeroImageAlt: source.HeroImageAlt}, true
}

func resolvedMarketingMessage(ctx context.Context, input marketingMessageRequest) (marketingMessageRequest, error) {
	input.TemplateID = strings.TrimSpace(input.TemplateID)
	if input.TemplateID == "" {
		return input, nil
	}
	templates, err := marketingMessageTemplates(ctx)
	if err != nil {
		return marketingMessageRequest{}, err
	}
	for _, template := range templates {
		if template.ID == input.TemplateID {
			input.Name = template.Name
			input.Subject = template.Subject
			input.Markdown = template.Markdown
			return input, nil
		}
	}
	return marketingMessageRequest{}, fmt.Errorf("Unknown marketing message template")
}

func marketingTemplateMarkdown(markdown string, stats publicStatsSnapshot) string {
	replacements := map[string]string{
		"{{memberFirstName}}":          "{{firstName}}",
		"{{.FirstName}}":               "{{firstName}}",
		"{{.MemberCount}}":             strconv.Itoa(stats.JoinedOwners),
		"{{membersJoined}}":            strconv.Itoa(stats.JoinedOwners),
		"{{vehiclesRegisteredCount}}":  strconv.Itoa(stats.VehiclesRegistered),
		"{{vehiclesSoHReadingsCount}}": strconv.Itoa(stats.SOHReadings),
		"{{serviceFaultRecordsCount}}": strconv.Itoa(stats.ServiceEventsLogged),
		"{{.Projection}}":              "If every member helps one more I-PACE owner find us, our voice could reach " + strconv.Itoa(stats.JoinedOwners*2) + " owners.",
		"{{.SuggestedShareText}}":      "I-PACE owners are stronger together. Join the I-PACE Owners' Advocacy Group: https://ipace-owners.org/join/",
		"{{.InstagramURL}}":            "https://www.instagram.com/ipaceowners/",
	}
	for token, replacement := range replacements {
		markdown = strings.ReplaceAll(markdown, token, replacement)
	}
	return markdown
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
