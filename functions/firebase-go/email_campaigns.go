package ipace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

const emailCampaignBatchSize = 10

type campaignRecipient struct {
	Name      string
	Email     string
	CreatedAt time.Time
}

type campaignSummary struct {
	CampaignID   string               `json:"campaignId"`
	Eligible     int                  `json:"eligible"`
	Registered   int                  `json:"registered"`
	Sent         int                  `json:"sent"`
	BatchSent    int                  `json:"batchSent"`
	Remaining    int                  `json:"remaining"`
	EmailPreview campaignEmailPreview `json:"emailPreview"`
}

type campaignEmailPreview struct {
	Subject string              `json:"subject"`
	HTML    string              `json:"html"`
	Text    string              `json:"text"`
	Shares  []campaignShareLink `json:"shares,omitempty"`
}

type campaignShareLink struct {
	Label string `json:"label"`
	Mark  string `json:"mark"`
	URL   string `json:"url"`
}

type campaignSendRequest struct {
	CampaignID       string `json:"campaignId"`
	ExpectedEligible int    `json:"expectedEligible"`
	Confirmation     string `json:"confirmation"`
}

var campaignAuthorize = func(ctx context.Context, r *http.Request) error {
	user, err := requireUser(ctx, r)
	if err != nil {
		return err
	}
	if !isAdmin(user) {
		return fmt.Errorf("admin role required")
	}
	return nil
}
var campaignPreview = previewReengagementCampaign
var campaignSend = sendReengagementCampaignBatch
var memberReferralPreview = previewMemberReferralCampaign
var memberReferralSend = sendMemberReferralCampaignBatch

func AdminReengagementPreview(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	summary, err := campaignPreview(r.Context())
	if err != nil {
		logEvent("admin-reengagement-preview", "error", "preview failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not calculate the campaign audience"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func AdminReengagementSend(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	var input campaignSendRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	summary, err := campaignSend(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func AdminMemberReferralPreview(w http.ResponseWriter, r *http.Request) {
	adminCampaignPreviewHandler(w, r, "admin-member-referral-preview", memberReferralPreview)
}

func AdminMemberReferralSend(w http.ResponseWriter, r *http.Request) {
	adminCampaignSendHandler(w, r, memberReferralSend)
}

func adminCampaignPreviewHandler(w http.ResponseWriter, r *http.Request, logName string, preview func(context.Context) (campaignSummary, error)) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	summary, err := preview(r.Context())
	if err != nil {
		logEvent(logName, "error", "preview failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not calculate the campaign audience"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func adminCampaignSendHandler(w http.ResponseWriter, r *http.Request, send func(context.Context, campaignSendRequest) (campaignSummary, error)) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	var input campaignSendRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	summary, err := send(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func campaignID() string {
	environment := "production"
	if strings.Contains(strings.ToLower(projectID()), "staging") {
		environment = "staging"
	}
	return "join-account-verification-" + environment + "-" + time.Now().UTC().Format("2006-01-02")
}

func memberReferralCampaignID() string {
	environment := "production"
	if strings.Contains(strings.ToLower(projectID()), "staging") {
		environment = "staging"
	}
	return "member-referral-" + environment + "-" + time.Now().UTC().Format("2006-01-02")
}

func previewReengagementCampaign(ctx context.Context) (campaignSummary, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	authClient, err := firebaseAuth(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	joins, err := loadCampaignJoins(ctx, db)
	if err != nil {
		return campaignSummary{}, err
	}
	registered, err := loadRegisteredIdentities(ctx, authClient)
	if err != nil {
		return campaignSummary{}, err
	}
	eligible := classifyCampaignRecipients(joins, registered)
	id := campaignID()
	sent, err := loadSentFingerprints(ctx, db, id)
	if err != nil {
		return campaignSummary{}, err
	}
	summary := makeCampaignSummary(id, len(eligible), len(joins)-len(eligible), countCampaignSent(eligible, sent), 0)
	summary.EmailPreview = makeCampaignEmailPreview(len(joins), len(eligible))
	return summary, nil
}

func sendReengagementCampaignBatch(ctx context.Context, input campaignSendRequest) (campaignSummary, error) {
	if !resendEmailConfigured() {
		return campaignSummary{}, fmt.Errorf("email delivery is not configured")
	}
	preview, err := previewReengagementCampaign(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	if input.CampaignID != preview.CampaignID {
		return campaignSummary{}, fmt.Errorf("campaign changed; preview again")
	}
	if input.ExpectedEligible != preview.Eligible {
		return campaignSummary{}, fmt.Errorf("eligible count changed; preview again")
	}
	if input.Confirmation != fmt.Sprintf("SEND %d", preview.Eligible) {
		return campaignSummary{}, fmt.Errorf("confirmation did not match; no emails sent")
	}

	db, _ := firestoreClient(ctx)
	authClient, _ := firebaseAuth(ctx)
	joins, err := loadCampaignJoins(ctx, db)
	if err != nil {
		return campaignSummary{}, err
	}
	registered, err := loadRegisteredIdentities(ctx, authClient)
	if err != nil {
		return campaignSummary{}, err
	}
	eligible := classifyCampaignRecipients(joins, registered)
	sent, err := loadSentFingerprints(ctx, db, preview.CampaignID)
	if err != nil {
		return campaignSummary{}, err
	}
	batchSent := 0
	for _, person := range eligible {
		fingerprint := campaignEmailFingerprint(person.Email)
		if sent[fingerprint] || batchSent >= emailCampaignBatchSize {
			continue
		}
		if _, err := authClient.GetUserByEmail(ctx, person.Email); err == nil {
			continue
		} else if !auth.IsUserNotFound(err) {
			return campaignSummary{}, err
		}
		link, err := generateFirebaseEmailSignInLink(ctx, person.Email, campaignContinueURL(), campaignLinkDomain())
		if err != nil {
			return campaignSummary{}, fmt.Errorf("could not generate sign-in link")
		}
		resendID, err := sendCampaignEmail(ctx, person, link, len(joins), len(eligible), preview.CampaignID)
		if err != nil {
			return campaignSummary{}, fmt.Errorf("email provider rejected a message; retry the batch")
		}
		_, err = db.Collection("emailCampaigns").Doc(preview.CampaignID).Collection("deliveries").Doc(fingerprint).Set(ctx, map[string]any{"status": "sent", "resendId": resendID, "sentAt": firestore.ServerTimestamp})
		if err != nil {
			return campaignSummary{}, fmt.Errorf("email sent but campaign ledger update failed; retry safely")
		}
		sent[fingerprint] = true
		batchSent++
		if batchSent < emailCampaignBatchSize {
			time.Sleep(250 * time.Millisecond)
		}
	}
	summary := makeCampaignSummary(preview.CampaignID, len(eligible), len(joins)-len(eligible), countCampaignSent(eligible, sent), batchSent)
	summary.EmailPreview = makeCampaignEmailPreview(len(joins), len(eligible))
	return summary, nil
}

func previewMemberReferralCampaign(ctx context.Context) (campaignSummary, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	authClient, err := firebaseAuth(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	joins, err := loadCampaignJoins(ctx, db)
	if err != nil {
		return campaignSummary{}, err
	}
	accounts, err := loadRegisteredEmailMap(ctx, authClient)
	if err != nil {
		return campaignSummary{}, err
	}
	eligible := classifyMemberReferralRecipients(joins, accounts)
	id := memberReferralCampaignID()
	sent, err := loadSentFingerprints(ctx, db, id)
	if err != nil {
		return campaignSummary{}, err
	}
	summary := makeCampaignSummary(id, len(eligible), len(accounts)-len(eligible), countCampaignSent(eligible, sent), 0)
	summary.EmailPreview = makeMemberReferralEmailPreview(len(joins))
	return summary, nil
}

func sendMemberReferralCampaignBatch(ctx context.Context, input campaignSendRequest) (campaignSummary, error) {
	if !resendEmailConfigured() {
		return campaignSummary{}, fmt.Errorf("email delivery is not configured")
	}
	preview, err := previewMemberReferralCampaign(ctx)
	if err != nil {
		return campaignSummary{}, err
	}
	if input.CampaignID != preview.CampaignID {
		return campaignSummary{}, fmt.Errorf("campaign changed; preview again")
	}
	if input.ExpectedEligible != preview.Eligible {
		return campaignSummary{}, fmt.Errorf("eligible count changed; preview again")
	}
	if input.Confirmation != fmt.Sprintf("SEND %d", preview.Eligible) {
		return campaignSummary{}, fmt.Errorf("confirmation did not match; no emails sent")
	}

	db, _ := firestoreClient(ctx)
	authClient, _ := firebaseAuth(ctx)
	joins, err := loadCampaignJoins(ctx, db)
	if err != nil {
		return campaignSummary{}, err
	}
	accounts, err := loadRegisteredEmailMap(ctx, authClient)
	if err != nil {
		return campaignSummary{}, err
	}
	eligible := classifyMemberReferralRecipients(joins, accounts)
	if len(eligible) != input.ExpectedEligible {
		return campaignSummary{}, fmt.Errorf("eligible count changed; preview again")
	}
	sent, err := loadSentFingerprints(ctx, db, preview.CampaignID)
	if err != nil {
		return campaignSummary{}, err
	}
	batchSent := 0
	for _, person := range eligible {
		fingerprint := campaignEmailFingerprint(person.Email)
		if sent[fingerprint] || batchSent >= emailCampaignBatchSize {
			continue
		}
		if _, err := authClient.GetUserByEmail(ctx, person.Email); err != nil {
			if auth.IsUserNotFound(err) {
				continue
			}
			return campaignSummary{}, err
		}
		resendID, err := sendMemberReferralEmail(ctx, person, len(joins), preview.CampaignID)
		if err != nil {
			return campaignSummary{}, fmt.Errorf("email provider rejected a message; retry the batch")
		}
		_, err = db.Collection("emailCampaigns").Doc(preview.CampaignID).Collection("deliveries").Doc(fingerprint).Set(ctx, map[string]any{"status": "sent", "resendId": resendID, "sentAt": firestore.ServerTimestamp})
		if err != nil {
			return campaignSummary{}, fmt.Errorf("email sent but campaign ledger update failed; retry safely")
		}
		sent[fingerprint] = true
		batchSent++
		if batchSent < emailCampaignBatchSize {
			time.Sleep(250 * time.Millisecond)
		}
	}
	summary := makeCampaignSummary(preview.CampaignID, len(eligible), len(accounts)-len(eligible), countCampaignSent(eligible, sent), batchSent)
	summary.EmailPreview = makeMemberReferralEmailPreview(len(joins))
	return summary, nil
}

func countCampaignSent(eligible []campaignRecipient, sent map[string]bool) int {
	count := 0
	for _, person := range eligible {
		if sent[campaignEmailFingerprint(person.Email)] {
			count++
		}
	}
	return count
}

func makeCampaignSummary(id string, eligible, registered, sent, batchSent int) campaignSummary {
	remaining := eligible - sent
	if remaining < 0 {
		remaining = 0
	}
	return campaignSummary{CampaignID: id, Eligible: eligible, Registered: registered, Sent: sent, BatchSent: batchSent, Remaining: remaining}
}

func loadCampaignJoins(ctx context.Context, db *firestore.Client) ([]campaignRecipient, error) {
	unique := map[string]campaignRecipient{}
	iter := db.Collection("joinSubmissions").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var row struct {
			CreatedAt time.Time `firestore:"createdAt"`
			Contact   struct {
				Name  string `firestore:"name"`
				Email string `firestore:"email"`
			} `firestore:"contact"`
			Consents struct {
				Contact bool `firestore:"contact"`
			} `firestore:"consents"`
		}
		if err := doc.DataTo(&row); err != nil {
			return nil, err
		}
		email := strings.ToLower(strings.TrimSpace(row.Contact.Email))
		name := strings.TrimSpace(row.Contact.Name)
		if email == "" || name == "" || !row.Consents.Contact {
			continue
		}
		key := canonicalCampaignEmail(email)
		previous, ok := unique[key]
		if !ok || row.CreatedAt.Before(previous.CreatedAt) {
			unique[key] = campaignRecipient{Name: name, Email: email, CreatedAt: row.CreatedAt}
		}
	}
	result := make([]campaignRecipient, 0, len(unique))
	for _, item := range unique {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Email < result[j].Email })
	return result, nil
}

func loadRegisteredIdentities(ctx context.Context, client *auth.Client) (map[string]bool, error) {
	identities := map[string]bool{}
	names := map[string]bool{}
	iter := client.Users(ctx, "")
	for {
		user, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		identities[canonicalCampaignEmail(user.Email)] = true
		if key := normalizedCampaignName(user.DisplayName); key != "" {
			names[key] = true
		}
	}
	for name := range names {
		identities["name:"+name] = true
	}
	return identities, nil
}

func loadRegisteredEmailMap(ctx context.Context, client *auth.Client) (map[string]string, error) {
	accounts := map[string]string{}
	iter := client.Users(ctx, "")
	for {
		user, err := iter.Next()
		if err == iterator.Done {
			return accounts, nil
		}
		if err != nil {
			return nil, err
		}
		email := strings.ToLower(strings.TrimSpace(user.Email))
		if email != "" {
			accounts[canonicalCampaignEmail(email)] = email
		}
	}
}

func classifyCampaignRecipients(joins []campaignRecipient, registered map[string]bool) []campaignRecipient {
	result := []campaignRecipient{}
	for _, person := range joins {
		if registered[canonicalCampaignEmail(person.Email)] || registered["name:"+normalizedCampaignName(person.Name)] {
			continue
		}
		result = append(result, person)
	}
	return result
}

func classifyMemberReferralRecipients(joins []campaignRecipient, registered map[string]string) []campaignRecipient {
	result := []campaignRecipient{}
	for _, person := range joins {
		if email := registered[canonicalCampaignEmail(person.Email)]; email != "" {
			person.Email = email
			result = append(result, person)
		}
	}
	return result
}

func loadSentFingerprints(ctx context.Context, db *firestore.Client, id string) (map[string]bool, error) {
	result := map[string]bool{}
	iter := db.Collection("emailCampaigns").Doc(id).Collection("deliveries").Where("status", "==", "sent").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		result[doc.Ref.ID] = true
	}
}

func canonicalCampaignEmail(value string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(value)), "@")
	if len(parts) != 2 {
		return strings.ToLower(strings.TrimSpace(value))
	}
	if at := strings.IndexByte(parts[0], '+'); at >= 0 {
		parts[0] = parts[0][:at]
	}
	return parts[0] + "@" + parts[1]
}
func normalizedCampaignName(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
func campaignEmailFingerprint(value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:])[:16]
}
func campaignContinueURL() string {
	return strings.TrimRight(emailAssetBaseURL("https://ipace-owners.org"), "/") + "/member/account/"
}
func campaignLinkDomain() string {
	if value := strings.TrimSpace(os.Getenv("FIREBASE_EMAIL_LINK_DOMAIN")); value != "" {
		return value
	}
	return firebaseEmailLinkDomainForContinueURL(campaignContinueURL())
}

func sendCampaignEmail(ctx context.Context, person campaignRecipient, link string, memberCount, eligibleCount int, id string) (string, error) {
	subject, htmlBody, text := campaignEmailBodies(person, link, memberCount, eligibleCount)
	return sendCampaignPayload(ctx, person.Email, subject, htmlBody, text, "join-reengagement", id)
}

func sendCampaignPayload(ctx context.Context, email, subject, htmlBody, text, category, id string) (string, error) {
	payload := map[string]any{"from": strings.TrimSpace(os.Getenv("RESEND_FROM")), "to": []string{email}, "subject": subject, "html": htmlBody, "text": text, "tags": []map[string]string{{"name": "category", "value": category}}}
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
	req.Header.Set("Idempotency-Key", id+"/"+campaignEmailFingerprint(email))
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

func campaignEmailBodies(person campaignRecipient, link string, memberCount, eligibleCount int) (string, string, string) {
	first := "there"
	if fields := strings.Fields(person.Name); len(fields) > 0 {
		first = fields[0]
	}
	subject := "Complete your I-PACE Owners registration"
	text, bodyHTML := mustRenderCampaignTemplate("campaign-reengagement.md.tmpl", struct {
		FirstName     string
		JoinedDate    string
		MemberCount   int
		EligibleCount int
	}{first, person.CreatedAt.Format("2 January 2006"), memberCount, eligibleCount})
	text += "\nVerify your account details: " + link + "\n\nYou are receiving this because you submitted the Join form and agreed that we could contact you. Reply if you no longer wish to hear from us.\n"
	htmlBody := brandedEmailHTML(brandedEmailMessage{
		DocumentTitle:      subject,
		Preheader:          "Complete your registration with a fresh, secure sign-in link.",
		Heading:            "Please complete your registration",
		BodyHTML:           bodyHTML,
		PrimaryActionLabel: "Verify my account details",
		PrimaryActionURL:   link,
		FallbackURL:        link,
		FooterNote:         "You are receiving this because you submitted the Join form and agreed that we could contact you. Reply if you no longer wish to hear from us.",
		AssetBaseURL:       emailAssetBaseURL(campaignContinueURL()),
	})
	return subject, htmlBody, text
}

func makeCampaignEmailPreview(memberCount, eligibleCount int) campaignEmailPreview {
	person := campaignRecipient{Name: "I-PACE owner", CreatedAt: time.Now().UTC()}
	subject, htmlBody, text := campaignEmailBodies(person, "[A fresh, private sign-in link is inserted for each recipient]", memberCount, eligibleCount)
	return campaignEmailPreview{Subject: subject, HTML: htmlBody, Text: text}
}

func memberReferralShareLinks(memberCount int) []campaignShareLink {
	shareURL := "https://ipace-owners.org/"
	instagramURL := "https://www.instagram.com/ipaceowners/"
	message := fmt.Sprintf("I-PACE owners are working together for fair outcomes. %d owners have already joined — could you help another owner find the group?", memberCount)
	return []campaignShareLink{
		{Label: "Facebook", Mark: "f", URL: "https://www.facebook.com/sharer/sharer.php?u=" + url.QueryEscape(shareURL)},
		{Label: "X", Mark: "𝕏", URL: "https://twitter.com/intent/tweet?text=" + url.QueryEscape(message) + "&url=" + url.QueryEscape(shareURL)},
		{Label: "Bluesky", Mark: "B", URL: "https://bsky.app/intent/compose?text=" + url.QueryEscape(message+" "+shareURL)},
		{Label: "LinkedIn", Mark: "in", URL: "https://www.linkedin.com/sharing/share-offsite/?url=" + url.QueryEscape(shareURL)},
		{Label: "Instagram", Mark: "◎", URL: instagramURL},
		{Label: "WhatsApp", Mark: "W", URL: "https://wa.me/?text=" + url.QueryEscape(message+" "+shareURL)},
		{Label: "Email", Mark: "@", URL: "mailto:?subject=" + url.QueryEscape("Will you join the I-PACE Owners group?") + "&body=" + url.QueryEscape(message+"\n\n"+shareURL)},
	}
}

func memberReferralEmailBodies(person campaignRecipient, memberCount int) (string, string, string, []campaignShareLink) {
	first := "there"
	if fields := strings.Fields(person.Name); len(fields) > 0 {
		first = fields[0]
	}
	goal := 1000
	remaining := goal - memberCount
	if remaining < 0 {
		remaining = 0
	}
	projected := memberCount * 2
	projection := fmt.Sprintf("If every one of our %d owners found just one more I-PACE owner, we would grow to %d members", memberCount, projected)
	if projected >= 700 && projected < 800 {
		projection += " — putting us in the 700s"
	}
	subject := "Could you help one more I-PACE owner find us?"
	shares := memberReferralShareLinks(memberCount)
	instagramURL := "https://www.instagram.com/ipaceowners/"
	text, bodyHTML := mustRenderCampaignTemplate("member-referral.md.tmpl", struct {
		FirstName      string
		MemberCount    int
		RemainingCount int
		Projection     string
		InstagramURL   string
	}{first, memberCount, remaining, projection, instagramURL})
	text += "\nShare the group: https://ipace-owners.org/\n\nYou are receiving this because you registered with the group and agreed that we could contact you. Reply if you no longer wish to hear from us.\n"
	buttons := ""
	for _, share := range shares {
		buttons += `<a href="` + html.EscapeString(share.URL) + `" style="display:inline-block;margin:0 8px 10px 0;padding:10px 14px;border:1px solid #0f766e;border-radius:999px;color:#0f766e;text-decoration:none;font-weight:700;"><span style="display:inline-block;min-width:18px;text-align:center;">` + html.EscapeString(share.Mark) + `</span> ` + html.EscapeString(share.Label) + `</a>`
	}
	htmlBody := brandedEmailHTML(brandedEmailMessage{
		DocumentTitle:      subject,
		Preheader:          fmt.Sprintf("%d owners have joined. Help one more I-PACE owner find the group.", memberCount),
		Heading:            "Could you help one more owner find us?",
		BodyHTML:           bodyHTML,
		PrimaryActionLabel: "Visit I-PACE Owners",
		PrimaryActionURL:   "https://ipace-owners.org/",
		SupplementHTML:     buttons,
		FooterNote:         "You are receiving this because you registered with the group and agreed that we could contact you. Reply if you no longer wish to hear from us.",
		AssetBaseURL:       emailAssetBaseURL(campaignContinueURL()),
	})
	return subject, htmlBody, text, shares
}

func makeMemberReferralEmailPreview(memberCount int) campaignEmailPreview {
	subject, htmlBody, text, shares := memberReferralEmailBodies(campaignRecipient{Name: "I-PACE owner"}, memberCount)
	return campaignEmailPreview{Subject: subject, HTML: htmlBody, Text: text, Shares: shares}
}

func sendMemberReferralEmail(ctx context.Context, person campaignRecipient, memberCount int, id string) (string, error) {
	subject, htmlBody, text, _ := memberReferralEmailBodies(person, memberCount)
	return sendCampaignPayload(ctx, person.Email, subject, htmlBody, text, "member-referral", id)
}
