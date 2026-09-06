package ipace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	marketingMessageMarkdownMax = 20000
	marketingMessageBatchSize   = 100
	marketingMessageKind        = "marketing-message"
)

var marketingMessagePlaceholderRegexp = regexp.MustCompile(`\{\{[^}]+\}\}`)

type marketingMessageRequest struct {
	CampaignID       string `json:"campaignId"`
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
	CampaignID   string `json:"campaignId"`
	Eligible     int    `json:"eligible"`
	Subject      string `json:"subject"`
	HTML         string `json:"html"`
	Text         string `json:"text"`
	Confirmation string `json:"confirmation"`
	Notice       string `json:"notice"`
}

type marketingMessageSent struct {
	CampaignID string `json:"campaignId"`
	Eligible   int    `json:"eligible"`
	Sent       int    `json:"sent"`
	BatchSent  int    `json:"batchSent"`
	Remaining  int    `json:"remaining"`
	Message    string `json:"message"`
}

type marketingMessageRecord struct {
	CampaignID string    `firestore:"campaignId"`
	Kind       string    `firestore:"kind"`
	TemplateID string    `firestore:"templateId,omitempty"`
	Name       string    `firestore:"name"`
	Subject    string    `firestore:"subject"`
	Markdown   string    `firestore:"markdown"`
	Eligible   int       `firestore:"eligible"`
	Sent       int       `firestore:"sent"`
	Failed     int       `firestore:"failed"`
	Remaining  int       `firestore:"remaining"`
	BatchCount int       `firestore:"batchCount"`
	Status     string    `firestore:"status"`
	CreatedAt  time.Time `firestore:"createdAt"`
	UpdatedAt  time.Time `firestore:"updatedAt"`
	LastSentAt time.Time `firestore:"lastSentAt,omitempty"`
}

var marketingMessageAudience = loadMarketingMessageAudience
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
	if err := validateMarketingMessage(input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	audience, err := marketingMessageAudience(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not calculate the consented audience"})
		return
	}
	if input.ExpectedEligible != len(audience) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "The consented audience changed; preview again before sending."})
		return
	}
	if input.Confirmation != fmt.Sprintf("SEND %d", len(audience)) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Confirmation did not match; no message was sent."})
		return
	}
	sent, err := sendMarketingMessageBatch(r.Context(), input, audience)
	if err != nil {
		logEvent("admin-marketing-message-send", "error", "batch failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	logEvent("admin-marketing-message-send", "info", "batch sent", map[string]any{"eligible": len(audience), "campaignId": sent.CampaignID, "batchSent": sent.BatchSent})
	writeJSON(w, http.StatusOK, sent)
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
	previewRecipient := campaignRecipient{Name: "Member", Email: "member@example.com"}
	if len(audience) > 0 {
		previewRecipient = audience[0]
	}
	markdown := renderMarketingMessageMarkdown(input.Markdown, previewRecipient)
	previewUnsubscribeURL := "https://ipace-owners.org/api/email-unsubscribe?campaign=preview&token=preview-token"
	return marketingMessagePreview{CampaignID: strings.TrimSpace(input.CampaignID), Eligible: len(audience), Subject: strings.TrimSpace(input.Subject), HTML: marketingMessageHTML(markdown, input.TemplateID, previewUnsubscribeURL), Text: marketingMessageText(markdown, previewUnsubscribeURL), Confirmation: fmt.Sprintf("SEND %d", len(audience)), Notice: fmt.Sprintf("Only members who opted in to group communications are included. Emails are sent in resumable batches of %d; previewing never sends email.", marketingMessageBatchSize)}, nil
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

func renderMarketingMessageMarkdown(markdown string, person campaignRecipient) string {
	_, first, _ := splitCampaignMemberName(person.Name)
	if first == "" {
		first = "member"
	}
	return strings.ReplaceAll(markdown, "{{firstName}}", first)
}
func marketingMessageHTML(markdown, templateID, unsubscribeURL string) string {
	body := markdownToEmailHTML(markdown)
	hero := ""
	if template, ok := marketingMessageTemplateSource(templateID); ok && template.HeroImage != "" {
		hero = `<img src="https://ipace-owners.org` + template.HeroImage + `" alt="` + htmlEscape(template.HeroImageAlt) + `" style="display:block;width:100%;height:auto;border:0;border-radius:10px;margin:0 0 24px;">`
	}
	return `<!doctype html><html><body style="margin:0;padding:24px;background:#f7f8fb;font-family:Arial,sans-serif;"><main style="max-width:640px;margin:auto;background:#fff;padding:32px;border-radius:12px;">` + hero + body + `<hr style="border:0;border-top:1px solid #dbe3ec;margin:28px 0 16px;"><p style="font-size:13px;color:#4b5563;">You are receiving this because you chose to receive group communications. <a href="` + htmlEscape(unsubscribeURL) + `">Unsubscribe</a>.</p></main></body></html>`
}
func marketingMessageText(markdown, unsubscribeURL string) string {
	return markdownToPlainText(markdown) + "\nYou are receiving this because you chose to receive group communications. Unsubscribe: " + unsubscribeURL + "\n"
}

func sendMarketingMessageBatch(ctx context.Context, input marketingMessageRequest, audience []campaignRecipient) (marketingMessageSent, error) {
	if !resendEmailConfigured() {
		return marketingMessageSent{}, fmt.Errorf("email delivery is not configured")
	}
	if !customCampaignIDRegexp.MatchString(strings.TrimSpace(input.CampaignID)) {
		return marketingMessageSent{}, fmt.Errorf("campaign ID is invalid; preview again")
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return marketingMessageSent{}, err
	}
	record, err := loadOrCreateMarketingMessageRecord(ctx, db, input, len(audience))
	if err != nil {
		return marketingMessageSent{}, err
	}
	sent, err := loadSentFingerprints(ctx, db, record.CampaignID)
	if err != nil {
		return marketingMessageSent{}, err
	}
	batchSent := 0
	for _, person := range audience {
		fingerprint := campaignEmailFingerprint(person.Email)
		if sent[fingerprint] || batchSent >= marketingMessageBatchSize {
			continue
		}
		token := submissionID("unsubscribe")
		unsubscribeURL := marketingUnsubscribeURL(record.CampaignID, token)
		markdown := renderMarketingMessageMarkdown(record.Markdown, person)
		htmlBody := marketingMessageHTML(markdown, record.TemplateID, unsubscribeURL)
		textBody := marketingMessageText(markdown, unsubscribeURL)
		resendID, err := sendMarketingMessagePayload(ctx, person.Email, record.Subject, htmlBody, textBody, record.CampaignID, unsubscribeURL)
		if err != nil {
			_, _ = db.Collection("emailCampaigns").Doc(record.CampaignID).Set(ctx, map[string]any{"failed": firestore.Increment(1), "updatedAt": time.Now().UTC()}, firestore.MergeAll)
			return marketingMessageSent{}, fmt.Errorf("email provider rejected a message; retry the batch")
		}
		_, err = db.Collection("emailCampaigns").Doc(record.CampaignID).Collection("deliveries").Doc(fingerprint).Set(ctx, map[string]any{"status": "sent", "resendId": resendID, "sentAt": firestore.ServerTimestamp, "emailHash": emailFingerprint(person.Email), "unsubscribeTokenHash": marketingUnsubscribeTokenHash(token)})
		if err != nil {
			return marketingMessageSent{}, fmt.Errorf("email sent but campaign ledger update failed; retry safely")
		}
		sent[fingerprint] = true
		batchSent++
		if batchSent < marketingMessageBatchSize {
			time.Sleep(250 * time.Millisecond)
		}
	}
	sentCount := countMarketingMessageSent(audience, sent)
	record.Sent = sentCount
	record.Remaining = max(0, len(audience)-sentCount)
	record.Status = customCampaignStatus(len(audience), sentCount)
	record.BatchCount++
	record.UpdatedAt = time.Now().UTC()
	if batchSent > 0 {
		record.LastSentAt = record.UpdatedAt
	}
	if _, err := db.Collection("emailCampaigns").Doc(record.CampaignID).Set(ctx, record); err != nil {
		return marketingMessageSent{}, err
	}
	return marketingMessageSent{CampaignID: record.CampaignID, Eligible: len(audience), Sent: sentCount, BatchSent: batchSent, Remaining: record.Remaining, Message: marketingMessageBatchMessage(batchSent, record.Remaining)}, nil
}

func loadOrCreateMarketingMessageRecord(ctx context.Context, db *firestore.Client, input marketingMessageRequest, eligible int) (marketingMessageRecord, error) {
	id := strings.TrimSpace(input.CampaignID)
	doc := db.Collection("emailCampaigns").Doc(id)
	snapshot, err := doc.Get(ctx)
	if err == nil {
		var record marketingMessageRecord
		if err := snapshot.DataTo(&record); err != nil || record.Kind != marketingMessageKind || record.Name != strings.TrimSpace(input.Name) || record.Subject != strings.TrimSpace(input.Subject) || record.Markdown != input.Markdown || record.TemplateID != strings.TrimSpace(input.TemplateID) {
			return marketingMessageRecord{}, fmt.Errorf("campaign changed; preview again")
		}
		if record.Eligible != eligible {
			return marketingMessageRecord{}, fmt.Errorf("The consented audience changed; preview again before sending.")
		}
		return record, nil
	}
	if status.Code(err) != codes.NotFound {
		return marketingMessageRecord{}, err
	}
	now := time.Now().UTC()
	record := marketingMessageRecord{CampaignID: id, Kind: marketingMessageKind, TemplateID: strings.TrimSpace(input.TemplateID), Name: strings.TrimSpace(input.Name), Subject: strings.TrimSpace(input.Subject), Markdown: input.Markdown, Eligible: eligible, Remaining: eligible, Status: "draft", CreatedAt: now, UpdatedAt: now}
	if _, err := doc.Set(ctx, record); err != nil {
		return marketingMessageRecord{}, err
	}
	return record, nil
}

func countMarketingMessageSent(audience []campaignRecipient, sent map[string]bool) int {
	count := 0
	for _, person := range audience {
		if sent[campaignEmailFingerprint(person.Email)] {
			count++
		}
	}
	return count
}
func marketingMessageBatchMessage(batchSent, remaining int) string {
	if remaining == 0 {
		return "All consented members have now been emailed."
	}
	return fmt.Sprintf("Sent %d email(s). %d remain; confirm again to send the next batch of up to %d.", batchSent, remaining, marketingMessageBatchSize)
}
func marketingUnsubscribeTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func marketingUnsubscribeURL(campaignID, token string) string {
	return strings.TrimRight(emailAssetBaseURL("https://ipace-owners.org"), "/") + "/api/email-unsubscribe?campaign=" + campaignID + "&token=" + token
}

func sendMarketingMessagePayload(ctx context.Context, email, subject, htmlBody, text, campaignID, unsubscribeURL string) (string, error) {
	payload := map[string]any{
		"from": strings.TrimSpace(os.Getenv("RESEND_FROM")), "to": []string{email}, "subject": subject,
		"html": htmlBody, "text": text, "tags": []map[string]string{{"name": "category", "value": marketingMessageKind}},
		"headers": map[string]string{"List-Unsubscribe": "<" + unsubscribeURL + ">", "List-Unsubscribe-Post": "List-Unsubscribe=One-Click"},
	}
	if reply := strings.TrimSpace(os.Getenv("RESEND_REPLY_TO")); reply != "" {
		payload["reply_to"] = reply
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("RESEND_API_KEY")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", campaignID+"/"+campaignEmailFingerprint(email))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4096))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", fmt.Errorf("resend returned %d", res.StatusCode)
	}
	var output struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return "", err
	}
	return output.ID, nil
}

// MarketingMessageUnsubscribe is deliberately unauthenticated: a recipient must be
// able to withdraw communications consent without signing in. The opaque, per-email
// token is stored only as a hash in the campaign delivery ledger.
func MarketingMessageUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	campaignID := strings.TrimSpace(r.URL.Query().Get("campaign"))
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if r.Method == http.MethodPost && token == "" {
		var input struct {
			Campaign string `json:"campaign"`
			Token    string `json:"token"`
		}
		if decodeJSON(r, &input) == nil {
			campaignID, token = strings.TrimSpace(input.Campaign), strings.TrimSpace(input.Token)
		}
	}
	if !customCampaignIDRegexp.MatchString(campaignID) || len(token) < 16 {
		marketingUnsubscribeResponse(w, r, http.StatusBadRequest, "This unsubscribe link is invalid or has expired.")
		return
	}
	if r.Method == http.MethodGet {
		marketingUnsubscribeConfirmation(w, campaignID, token)
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		marketingUnsubscribeResponse(w, r, http.StatusServiceUnavailable, "We could not update your preferences. Please try again shortly.")
		return
	}
	iter := db.Collection("emailCampaigns").Doc(campaignID).Collection("deliveries").Where("unsubscribeTokenHash", "==", marketingUnsubscribeTokenHash(token)).Limit(1).Documents(r.Context())
	doc, err := iter.Next()
	iter.Stop()
	if err != nil {
		marketingUnsubscribeResponse(w, r, http.StatusNotFound, "This unsubscribe link is invalid or has expired.")
		return
	}
	var delivery struct {
		EmailHash string `firestore:"emailHash"`
	}
	if err := doc.DataTo(&delivery); err != nil || delivery.EmailHash == "" {
		marketingUnsubscribeResponse(w, r, http.StatusNotFound, "This unsubscribe link is invalid or has expired.")
		return
	}
	joinIter := db.Collection("joinSubmissions").Where("userEmailHash", "==", delivery.EmailHash).Documents(r.Context())
	updated := 0
	for {
		join, err := joinIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			joinIter.Stop()
			marketingUnsubscribeResponse(w, r, http.StatusServiceUnavailable, "We could not update your preferences. Please try again shortly.")
			return
		}
		if _, err := join.Ref.Update(r.Context(), []firestore.Update{{Path: "consents.contact", Value: false}, {Path: "updatedAt", Value: time.Now().UTC()}}); err != nil {
			joinIter.Stop()
			marketingUnsubscribeResponse(w, r, http.StatusServiceUnavailable, "We could not update your preferences. Please try again shortly.")
			return
		}
		updated++
	}
	joinIter.Stop()
	if updated == 0 {
		marketingUnsubscribeResponse(w, r, http.StatusNotFound, "This unsubscribe link is invalid or has expired.")
		return
	}
	logEvent("marketing-message-unsubscribe", "info", "communications consent withdrawn", map[string]any{"campaignId": campaignID, "emailHash": delivery.EmailHash})
	marketingUnsubscribeResponse(w, r, http.StatusOK, "You have been unsubscribed from I-PACE Owners group communications.")
}

func marketingUnsubscribeConfirmation(w http.ResponseWriter, campaignID, token string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width, initial-scale=1"><title>Unsubscribe | I-PACE Owners</title></head><body style="margin:0;padding:32px;background:#f7f8fb;font:16px system-ui,-apple-system,sans-serif;color:#111827"><main style="max-width:560px;margin:auto;background:#fff;padding:32px;border-radius:12px"><h1 style="color:#12324a">Stop group emails?</h1><p>This will stop future I-PACE Owners group communications to this email address. You can re-enable them from your member preferences at any time.</p><form method="post" action="/api/email-unsubscribe?campaign=` + htmlEscape(campaignID) + `&amp;token=` + htmlEscape(token) + `"><button type="submit" style="background:#0f766e;border:0;border-radius:6px;color:#fff;padding:12px 18px;font:inherit;font-weight:700;cursor:pointer">Unsubscribe</button></form><p><a href="https://ipace-owners.org/">Keep receiving emails</a></p></main></body></html>`))
}

func marketingUnsubscribeResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	if r.Method == http.MethodPost {
		writeJSON(w, status, map[string]any{"message": message})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta name="viewport" content="width=device-width, initial-scale=1"><title>Email preferences | I-PACE Owners</title></head><body style="margin:0;padding:32px;background:#f7f8fb;font:16px system-ui,-apple-system,sans-serif;color:#111827"><main style="max-width:560px;margin:auto;background:#fff;padding:32px;border-radius:12px"><h1 style="color:#12324a">Email preferences</h1><p>` + htmlEscape(message) + `</p><p><a href="https://ipace-owners.org/">Return to I-PACE Owners</a></p></main></body></html>`))
}

func htmlEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(value)
}

// marketingMessageTemplates provides the group-wide messages that used to be
// dispatched through the resumable per-recipient campaign workspace.  They now
// use the same canonical, consented Join audience and direct-email delivery ledger.
// Registration reminders deliberately remain separate: each one needs a fresh private
// Firebase sign-in link.
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
