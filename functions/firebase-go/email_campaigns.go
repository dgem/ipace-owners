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
	CampaignID string `json:"campaignId"`
	Eligible   int    `json:"eligible"`
	Registered int    `json:"registered"`
	Sent       int    `json:"sent"`
	BatchSent  int    `json:"batchSent"`
	Remaining  int    `json:"remaining"`
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

func campaignID() string {
	environment := "production"
	if strings.Contains(strings.ToLower(projectID()), "staging") {
		environment = "staging"
	}
	return "join-account-verification-" + environment + "-" + time.Now().UTC().Format("2006-01-02")
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
	return makeCampaignSummary(id, len(eligible), len(joins)-len(eligible), countCampaignSent(eligible, sent), 0), nil
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
	return makeCampaignSummary(preview.CampaignID, len(eligible), len(joins)-len(eligible), countCampaignSent(eligible, sent), batchSent), nil
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
	first := "there"
	if fields := strings.Fields(person.Name); len(fields) > 0 {
		first = fields[0]
	}
	text := fmt.Sprintf("Hello %s,\n\nPlease complete your I-PACE Owners registration\n\nYou asked to join on %s, but your secure sign-in was not completed. %d owners have joined.\n\nVerify your account details using this fresh, time-limited link:\n%s\n\nYou are receiving this because you submitted the Join form and agreed that we could contact you. Reply if you no longer wish to hear from us.\n", first, person.CreatedAt.Format("2 January 2006"), memberCount, link)
	htmlBody := "<p>Hello " + html.EscapeString(first) + ",</p><h1>Please complete your I-PACE Owners registration</h1><p>You asked to join on " + html.EscapeString(person.CreatedAt.Format("2 January 2006")) + ", but your secure sign-in was not completed.</p><p><strong>" + fmt.Sprint(memberCount) + " owners have joined.</strong> If everyone receiving this reminder verifies, " + fmt.Sprint(eligibleCount) + " additional members will have registered accounts.</p><p><a href=\"" + html.EscapeString(link) + "\">Verify my account details</a></p><p>You are receiving this because you submitted the Join form and agreed that we could contact you. Reply if you no longer wish to hear from us.</p>"
	payload := map[string]any{"from": strings.TrimSpace(os.Getenv("RESEND_FROM")), "to": []string{person.Email}, "subject": "Complete your I-PACE Owners registration", "html": htmlBody, "text": text, "tags": []map[string]string{{"name": "category", "value": "join-reengagement"}}}
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
	req.Header.Set("Idempotency-Key", id+"/"+campaignEmailFingerprint(person.Email))
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
