package ipace

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

const (
	customCampaignKind        = "custom-member"
	jlrContactCampaignKind    = "jlr-contact"
	surveyCampaignKind        = "survey-september-2026"
	customCampaignMarkdownMax = 20000
)

var customCampaignPlaceholderRegexp = regexp.MustCompile(`\{\{\s*([A-Za-z][A-Za-z0-9]*)\s*\}\}`)
var customCampaignIDRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,160}$`)

var customCampaignPlaceholders = []string{
	"membersJoined",
	"membersVerified",
	"memberFirstName",
	"memberLastName",
	"memberTittle",
	"memberTitle",
	"memberJoined",
	"memberVerified",
	"memberVehicles",
	"vehiclesRegisteredCount",
	"vehiclesSoHReadingsCount",
	"serviceFaultRecordsCount",
}

type customCampaignDraftRequest struct {
	CampaignID       string `json:"campaignId"`
	SourceCampaignID string `json:"sourceCampaignId"`
	Name             string `json:"name"`
	Subject          string `json:"subject"`
	Markdown         string `json:"markdown"`
	HeroImage        string `json:"-"`
	HeroImageAlt     string `json:"-"`
	Kind             string `json:"-"`
	CreateWithID     bool   `json:"-"`
}

type customCampaignRecord struct {
	CampaignID       string    `json:"campaignId" firestore:"campaignId"`
	Kind             string    `json:"kind" firestore:"kind"`
	Name             string    `json:"name" firestore:"name"`
	Subject          string    `json:"subject" firestore:"subject"`
	Markdown         string    `json:"markdown" firestore:"markdown"`
	HeroImage        string    `json:"-" firestore:"heroImage,omitempty"`
	HeroImageAlt     string    `json:"-" firestore:"heroImageAlt,omitempty"`
	SourceCampaignID string    `json:"sourceCampaignId,omitempty" firestore:"sourceCampaignId,omitempty"`
	Status           string    `json:"status" firestore:"status"`
	Eligible         int       `json:"eligible" firestore:"eligible"`
	Sent             int       `json:"sent" firestore:"sent"`
	Failed           int       `json:"failed" firestore:"failed"`
	Remaining        int       `json:"remaining" firestore:"remaining"`
	BatchCount       int       `json:"batchCount" firestore:"batchCount"`
	CreatedAt        time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt" firestore:"updatedAt"`
	LastSentAt       time.Time `json:"lastSentAt,omitempty" firestore:"lastSentAt,omitempty"`
	Delivered        int       `json:"delivered,omitempty" firestore:"-"`
	Opened           int       `json:"opened,omitempty" firestore:"-"`
	Clicked          int       `json:"clicked,omitempty" firestore:"-"`
	Bounced          int       `json:"bounced,omitempty" firestore:"-"`
	Delayed          int       `json:"delayed,omitempty" firestore:"-"`
	Suppressed       int       `json:"suppressed,omitempty" firestore:"-"`
	Complained       int       `json:"complained,omitempty" firestore:"-"`
	ProviderFailed   int       `json:"providerFailed,omitempty" firestore:"-"`
	Undeliverable    int       `json:"undeliverable,omitempty" firestore:"-"`
	AwaitingDelivery int       `json:"awaitingDelivery,omitempty" firestore:"-"`
}

type customCampaignDocumentWriter interface {
	Set(context.Context, any, ...firestore.SetOption) (*firestore.WriteResult, error)
}

type customCampaignHistory struct {
	Campaigns           []customCampaignRecord `json:"campaigns"`
	Placeholders        []string               `json:"placeholders"`
	FeedbackAvailable   bool                   `json:"feedbackAvailable"`
	FeedbackRefreshedAt time.Time              `json:"feedbackRefreshedAt,omitempty"`
	FeedbackMessage     string                 `json:"feedbackMessage,omitempty"`
}

type customCampaignPreviewResponse struct {
	CampaignID       string               `json:"campaignId"`
	SourceCampaignID string               `json:"sourceCampaignId,omitempty"`
	Name             string               `json:"name"`
	Eligible         int                  `json:"eligible"`
	Sent             int                  `json:"sent"`
	Failed           int                  `json:"failed"`
	BatchSent        int                  `json:"batchSent"`
	Remaining        int                  `json:"remaining"`
	EmailPreview     campaignEmailPreview `json:"emailPreview"`
}

type customCampaignSendRequest struct {
	CampaignID       string `json:"campaignId"`
	ExpectedEligible int    `json:"expectedEligible"`
	Confirmation     string `json:"confirmation"`
}

type customCampaignVehicle struct {
	ID                    string `json:"id"`
	Registration          string `json:"registration,omitempty"`
	Country               string `json:"country,omitempty"`
	ModelYear             string `json:"modelYear,omitempty"`
	Mileage               *int   `json:"mileage,omitempty"`
	OwnedSince            string `json:"ownedSince,omitempty"`
	FirstRegistrationDate string `json:"firstRegistrationDate,omitempty"`
	SoHReadingsCount      int    `json:"sohReadingsCount"`
}

type customCampaignRecipient struct {
	UID        string
	Email      string
	Name       string
	JoinedAt   time.Time
	VerifiedAt time.Time
	Vehicles   []customCampaignVehicle
}

type customCampaignData struct {
	MembersJoined            int
	MembersVerified          int
	MemberFirstName          string
	MemberLastName           string
	MemberTittle             string
	MemberJoined             string
	MemberVerified           string
	MemberVehicles           string
	VehiclesRegisteredCount  int
	VehiclesSoHReadingsCount int
	ServiceFaultRecordsCount int
}

type customCampaignAudience struct {
	Recipients               []customCampaignRecipient
	MembersJoined            int
	MembersVerified          int
	VehiclesRegisteredCount  int
	VehiclesSoHReadingsCount int
	ServiceFaultRecordsCount int
}

var (
	customCampaignHistoryLoad = loadCustomCampaignHistory
	customCampaignPreview     = previewCustomCampaign
	customCampaignSend        = sendCustomCampaignBatch
)

func AdminEmailCampaignHistory(w http.ResponseWriter, r *http.Request) {
	if !adminCampaignRequestAllowed(w, r) {
		return
	}
	history, err := customCampaignHistoryLoad(r.Context())
	if err != nil {
		logEvent("admin-email-campaign-history", "error", "history load failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load campaign history"})
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func AdminCustomCampaignPreview(w http.ResponseWriter, r *http.Request) {
	if !adminCampaignRequestAllowed(w, r) {
		return
	}
	var input customCampaignDraftRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := customCampaignPreview(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func AdminCustomCampaignSend(w http.ResponseWriter, r *http.Request) {
	if !adminCampaignRequestAllowed(w, r) {
		return
	}
	var input customCampaignSendRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := customCampaignSend(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func adminCampaignRequestAllowed(w http.ResponseWriter, r *http.Request) bool {
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

func previewCustomCampaign(ctx context.Context, input customCampaignDraftRequest) (customCampaignPreviewResponse, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Subject = strings.TrimSpace(input.Subject)
	input.Markdown = strings.TrimSpace(input.Markdown)
	input.SourceCampaignID = strings.TrimSpace(input.SourceCampaignID)
	if err := validateCustomCampaignDraft(input); err != nil {
		return customCampaignPreviewResponse{}, err
	}
	audience, err := loadCustomCampaignAudience(ctx)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	id := strings.TrimSpace(input.CampaignID)
	now := time.Now().UTC()
	createdAt := now
	failed := 0
	batchCount := 0
	lastSentAt := time.Time{}
	if id != "" {
		existing, err := loadCustomCampaignRecord(ctx, db, id)
		if err != nil {
			if !input.CreateWithID {
				return customCampaignPreviewResponse{}, fmt.Errorf("campaign draft was not found; create a new preview")
			}
		} else {
			if existing.Sent > 0 && !sameCustomCampaignDraft(existing, input) {
				var reused bool
				input, reused = staticCampaignInputForStartedRecord(input, existing)
				if !reused {
					return customCampaignPreviewResponse{}, fmt.Errorf("a campaign that has started sending cannot be edited; rerun it as a new campaign")
				}
			}
			createdAt = existing.CreatedAt
			failed = existing.Failed
			batchCount = existing.BatchCount
			lastSentAt = existing.LastSentAt
		}
	} else {
		id = submissionID("email-campaign")
	}
	sent, err := loadSentFingerprints(ctx, db, id)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	sentCount := countCustomCampaignSent(audience.Recipients, sent)
	record := customCampaignRecord{
		CampaignID:       id,
		Kind:             campaignDraftKind(input),
		Name:             input.Name,
		Subject:          input.Subject,
		Markdown:         input.Markdown,
		HeroImage:        input.HeroImage,
		HeroImageAlt:     input.HeroImageAlt,
		SourceCampaignID: input.SourceCampaignID,
		Status:           customCampaignStatus(len(audience.Recipients), sentCount),
		Eligible:         len(audience.Recipients),
		Sent:             sentCount,
		Failed:           failed,
		Remaining:        max(0, len(audience.Recipients)-sentCount),
		BatchCount:       batchCount,
		CreatedAt:        createdAt,
		UpdatedAt:        now,
		LastSentAt:       lastSentAt,
	}
	if err := saveCustomCampaignRecord(ctx, db.Collection("emailCampaigns").Doc(id), record); err != nil {
		return customCampaignPreviewResponse{}, err
	}
	return makeCustomCampaignPreview(record, audience), nil
}

func previewJLRContactCampaign(ctx context.Context) (customCampaignPreviewResponse, error) {
	return previewEmbeddedCustomCampaign(ctx, "jlr-contact", jlrContactCampaignKind)
}

func previewSurveyCampaign(ctx context.Context) (customCampaignPreviewResponse, error) {
	return previewEmbeddedCustomCampaign(ctx, "survey-september-2026", surveyCampaignKind)
}

func previewEmbeddedCustomCampaign(ctx context.Context, templateName, campaignKind string) (customCampaignPreviewResponse, error) {
	template, err := embeddedCampaignTemplate(templateName)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	if template.ID != campaignKind || template.Audience != customCampaignKind {
		return customCampaignPreviewResponse{}, fmt.Errorf("%s template has an invalid audience", templateName)
	}
	return previewCustomCampaign(ctx, customCampaignDraftRequest{
		CampaignID:   staticCustomCampaignID(template.ID),
		Name:         template.Name,
		Subject:      template.Subject,
		Markdown:     template.Markdown,
		HeroImage:    template.HeroImage,
		HeroImageAlt: template.HeroImageAlt,
		Kind:         campaignKind,
		CreateWithID: true,
	})
}

func campaignDraftKind(input customCampaignDraftRequest) string {
	if isStaticCampaignKind(input.Kind) {
		return input.Kind
	}
	return customCampaignKind
}

func isStaticCampaignKind(kind string) bool {
	return kind == jlrContactCampaignKind || kind == surveyCampaignKind
}

func staticCustomCampaignID(templateID string) string {
	environment := "production"
	if strings.Contains(strings.ToLower(projectID()), "staging") {
		environment = "staging"
	}
	return templateID + "-" + environment + "-" + time.Now().UTC().Format("2006-01-02")
}

func sameCustomCampaignDraft(record customCampaignRecord, input customCampaignDraftRequest) bool {
	return record.Name == input.Name &&
		record.Subject == input.Subject &&
		record.Markdown == input.Markdown &&
		record.HeroImage == input.HeroImage &&
		record.HeroImageAlt == input.HeroImageAlt &&
		record.SourceCampaignID == input.SourceCampaignID
}

// staticCampaignInputForStartedRecord keeps a sent static campaign immutable when its
// source Markdown is edited later. Its preview must show the version that recipients
// received, including when everyone in the current audience has already been sent.
func staticCampaignInputForStartedRecord(input customCampaignDraftRequest, record customCampaignRecord) (customCampaignDraftRequest, bool) {
	if !input.CreateWithID || !isStaticCampaignKind(input.Kind) || input.Kind != record.Kind {
		return input, false
	}
	input.Name = record.Name
	input.Subject = record.Subject
	input.Markdown = record.Markdown
	input.HeroImage = record.HeroImage
	input.HeroImageAlt = record.HeroImageAlt
	input.SourceCampaignID = record.SourceCampaignID
	return input, true
}

func sendCustomCampaignBatch(ctx context.Context, input customCampaignSendRequest) (customCampaignPreviewResponse, error) {
	if !resendEmailConfigured() {
		return customCampaignPreviewResponse{}, fmt.Errorf("email delivery is not configured")
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	record, err := loadCustomCampaignRecord(ctx, db, strings.TrimSpace(input.CampaignID))
	if err != nil || (record.Kind != customCampaignKind && !isStaticCampaignKind(record.Kind)) {
		return customCampaignPreviewResponse{}, fmt.Errorf("campaign draft was not found; preview again")
	}
	if err := validateCustomCampaignDraft(customCampaignDraftRequest{
		CampaignID: record.CampaignID,
		Name:       record.Name,
		Subject:    record.Subject,
		Markdown:   record.Markdown,
	}); err != nil {
		return customCampaignPreviewResponse{}, fmt.Errorf("saved campaign is invalid; preview again")
	}
	audience, err := loadCustomCampaignAudience(ctx)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	if input.ExpectedEligible != len(audience.Recipients) || record.Eligible != len(audience.Recipients) {
		return customCampaignPreviewResponse{}, fmt.Errorf("eligible count changed; preview again")
	}
	if input.Confirmation != fmt.Sprintf("SEND %d", len(audience.Recipients)) {
		return customCampaignPreviewResponse{}, fmt.Errorf("confirmation did not match; no emails sent")
	}
	sent, err := loadSentFingerprints(ctx, db, record.CampaignID)
	if err != nil {
		return customCampaignPreviewResponse{}, err
	}
	for _, person := range audience.Recipients {
		if _, _, _, err := renderCustomCampaignEmail(record, audience, person); err != nil {
			return customCampaignPreviewResponse{}, fmt.Errorf("campaign could not be rendered safely for every recipient; preview again")
		}
	}
	batchSent := 0
	for _, person := range audience.Recipients {
		fingerprint := campaignEmailFingerprint(person.Email)
		if sent[fingerprint] || batchSent >= emailCampaignBatchSize {
			continue
		}
		subject, htmlBody, text, err := renderCustomCampaignEmail(record, audience, person)
		if err != nil {
			return customCampaignPreviewResponse{}, err
		}
		resendID, err := sendCampaignPayload(ctx, person.Email, subject, htmlBody, text, record.Kind, record.CampaignID)
		if err != nil {
			_, _ = db.Collection("emailCampaigns").Doc(record.CampaignID).Set(ctx, map[string]any{
				"failed":    firestore.Increment(1),
				"updatedAt": time.Now().UTC(),
			}, firestore.MergeAll)
			return customCampaignPreviewResponse{}, fmt.Errorf("email provider rejected a message; retry the batch")
		}
		_, err = db.Collection("emailCampaigns").Doc(record.CampaignID).Collection("deliveries").Doc(fingerprint).Set(ctx, map[string]any{
			"status":   "sent",
			"resendId": resendID,
			"sentAt":   firestore.ServerTimestamp,
		})
		if err != nil {
			return customCampaignPreviewResponse{}, fmt.Errorf("email sent but campaign ledger update failed; retry safely")
		}
		sent[fingerprint] = true
		batchSent++
		if batchSent < emailCampaignBatchSize {
			time.Sleep(250 * time.Millisecond)
		}
	}
	sentCount := countCustomCampaignSent(audience.Recipients, sent)
	now := time.Now().UTC()
	record.Sent = sentCount
	record.Remaining = max(0, len(audience.Recipients)-sentCount)
	record.Status = customCampaignStatus(len(audience.Recipients), sentCount)
	record.BatchCount++
	record.UpdatedAt = now
	if batchSent > 0 {
		record.LastSentAt = now
	}
	if err := saveCustomCampaignRecord(ctx, db.Collection("emailCampaigns").Doc(record.CampaignID), record); err != nil {
		return customCampaignPreviewResponse{}, fmt.Errorf("campaign sent but summary update failed")
	}
	response := makeCustomCampaignPreview(record, audience)
	response.Sent = sentCount
	response.BatchSent = batchSent
	response.Remaining = record.Remaining
	return response, nil
}

func validateCustomCampaignDraft(input customCampaignDraftRequest) error {
	if input.CampaignID != "" && !customCampaignIDRegexp.MatchString(input.CampaignID) {
		return fmt.Errorf("campaign ID is invalid")
	}
	if input.SourceCampaignID != "" && !customCampaignIDRegexp.MatchString(input.SourceCampaignID) {
		return fmt.Errorf("source campaign ID is invalid")
	}
	if input.Name == "" || len(input.Name) > 100 {
		return fmt.Errorf("campaign name is required and must be 100 characters or fewer")
	}
	if input.Subject == "" || len(input.Subject) > 200 {
		return fmt.Errorf("subject is required and must be 200 characters or fewer")
	}
	if strings.ContainsAny(input.Subject, "\r\n") {
		return fmt.Errorf("subject must be a single line")
	}
	if input.Markdown == "" || len(input.Markdown) > customCampaignMarkdownMax {
		return fmt.Errorf("Markdown is required and must be %d characters or fewer", customCampaignMarkdownMax)
	}
	allowed := map[string]bool{}
	for _, name := range customCampaignPlaceholders {
		allowed[name] = true
	}
	matches := customCampaignPlaceholderRegexp.FindAllStringSubmatch(input.Subject+"\n"+input.Markdown, -1)
	for _, match := range matches {
		if !allowed[match[1]] {
			return fmt.Errorf("unsupported substitution: %s", match[1])
		}
	}
	stripped := customCampaignPlaceholderRegexp.ReplaceAllString(input.Subject+"\n"+input.Markdown, "")
	if strings.Contains(stripped, "{{") || strings.Contains(stripped, "}}") {
		return fmt.Errorf("substitutions must use the documented {{name}} format")
	}
	if err := validateEmailMarkdownLinks(input.Markdown); err != nil {
		return err
	}
	return nil
}

func renderCustomCampaignEmail(record customCampaignRecord, audience customCampaignAudience, person customCampaignRecipient) (string, string, string, error) {
	data := customCampaignSubstitutionData(audience, person)
	subject, err := applyCustomCampaignSubstitutions(record.Subject, data)
	if err != nil {
		return "", "", "", err
	}
	if len(subject) > 500 || strings.ContainsAny(subject, "\r\n") {
		return "", "", "", fmt.Errorf("rendered subject is too long or contains a line break")
	}
	text, contentHTML, err := renderCustomCampaignContent(record.Markdown, data)
	if err != nil {
		return "", "", "", err
	}
	text += "\nYou are receiving this because you registered with the group and agreed that we could contact you. Reply if you no longer wish to hear from us.\n"
	htmlBody := brandedEmailHTML(brandedEmailMessage{
		DocumentTitle: subject,
		Preheader:     subject,
		Heading:       subject,
		BodyHTML:      contentHTML,
		FooterNote:    "You are receiving this because you registered with the group and agreed that we could contact you. Reply if you no longer wish to hear from us.",
		AssetBaseURL:  emailAssetBaseURL(campaignContinueURL()),
		HeroImagePath: record.HeroImage,
		HeroImageAlt:  record.HeroImageAlt,
	})
	return subject, htmlBody, text, nil
}

func customCampaignSubstitutionData(audience customCampaignAudience, person customCampaignRecipient) customCampaignData {
	title, first, last := splitCampaignMemberName(person.Name)
	vehicles, _ := json.Marshal(person.Vehicles)
	return customCampaignData{
		MembersJoined:            audience.MembersJoined,
		MembersVerified:          audience.MembersVerified,
		MemberFirstName:          first,
		MemberLastName:           last,
		MemberTittle:             title,
		MemberJoined:             formatCampaignDate(person.JoinedAt),
		MemberVerified:           formatCampaignDate(person.VerifiedAt),
		MemberVehicles:           string(vehicles),
		VehiclesRegisteredCount:  audience.VehiclesRegisteredCount,
		VehiclesSoHReadingsCount: audience.VehiclesSoHReadingsCount,
		ServiceFaultRecordsCount: audience.ServiceFaultRecordsCount,
	}
}

func applyCustomCampaignSubstitutions(value string, data customCampaignData) (string, error) {
	values := customCampaignValues(data)
	var renderErr error
	rendered := customCampaignPlaceholderRegexp.ReplaceAllStringFunc(value, func(token string) string {
		match := customCampaignPlaceholderRegexp.FindStringSubmatch(token)
		replacement, ok := values[match[1]]
		if !ok {
			renderErr = fmt.Errorf("unsupported substitution: %s", match[1])
			return token
		}
		return replacement
	})
	if renderErr != nil {
		return "", renderErr
	}
	return rendered, nil
}

func customCampaignValues(data customCampaignData) map[string]string {
	return map[string]string{
		"membersJoined":            fmt.Sprint(data.MembersJoined),
		"membersVerified":          fmt.Sprint(data.MembersVerified),
		"memberFirstName":          data.MemberFirstName,
		"memberLastName":           data.MemberLastName,
		"memberTittle":             data.MemberTittle,
		"memberTitle":              data.MemberTittle,
		"memberJoined":             data.MemberJoined,
		"memberVerified":           data.MemberVerified,
		"memberVehicles":           data.MemberVehicles,
		"vehiclesRegisteredCount":  fmt.Sprint(data.VehiclesRegisteredCount),
		"vehiclesSoHReadingsCount": fmt.Sprint(data.VehiclesSoHReadingsCount),
		"serviceFaultRecordsCount": fmt.Sprint(data.ServiceFaultRecordsCount),
	}
}

func renderCustomCampaignContent(markdown string, data customCampaignData) (string, string, error) {
	values := customCampaignValues(data)
	markers := map[string]string{}
	prefix := "IPACE_EMAIL_SUBSTITUTION_"
	for strings.Contains(markdown, prefix) {
		prefix += "X"
	}
	var renderErr error
	index := 0
	marked := customCampaignPlaceholderRegexp.ReplaceAllStringFunc(markdown, func(token string) string {
		match := customCampaignPlaceholderRegexp.FindStringSubmatch(token)
		replacement, ok := values[match[1]]
		if !ok {
			renderErr = fmt.Errorf("unsupported substitution: %s", match[1])
			return token
		}
		marker := fmt.Sprintf("%s%d_END", prefix, index)
		index++
		markers[marker] = replacement
		return marker
	})
	if renderErr != nil {
		return "", "", renderErr
	}
	textBody := markdownToPlainText(marked)
	htmlBody := markdownToEmailHTML(marked)
	for marker, replacement := range markers {
		textBody = strings.ReplaceAll(textBody, marker, replacement)
		htmlBody = strings.ReplaceAll(htmlBody, marker, html.EscapeString(replacement))
	}
	return textBody, htmlBody, nil
}

func splitCampaignMemberName(value string) (title, first, last string) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return "", "member", ""
	}
	titles := map[string]bool{
		"mr": true, "mrs": true, "ms": true, "miss": true, "mx": true, "dr": true, "prof": true,
		"sir": true, "dame": true, "lord": true, "lady": true, "rev": true, "capt": true,
		"col": true, "major": true,
	}
	if titles[strings.ToLower(strings.TrimSuffix(fields[0], "."))] {
		title = fields[0]
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return title, "member", ""
	}
	first = fields[0]
	if len(fields) > 1 {
		last = fields[len(fields)-1]
	}
	return title, first, last
}

func formatCampaignDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2 January 2006")
}

func makeCustomCampaignPreview(record customCampaignRecord, audience customCampaignAudience) customCampaignPreviewResponse {
	person := customCampaignRecipient{
		Name:       "Dr Alex Owner",
		JoinedAt:   time.Date(2026, time.July, 17, 12, 0, 0, 0, time.UTC),
		VerifiedAt: time.Date(2026, time.July, 18, 9, 30, 0, 0, time.UTC),
		Vehicles: []customCampaignVehicle{{
			ID: "vehicle-example", Registration: "EXAMPLE", Country: "GB", ModelYear: "2021", OwnedSince: "2023-04-10", SoHReadingsCount: 2,
		}},
	}
	if len(audience.Recipients) > 0 {
		person = audience.Recipients[0]
	}
	subject, htmlBody, text, _ := renderCustomCampaignEmail(record, audience, person)
	return customCampaignPreviewResponse{
		CampaignID:       record.CampaignID,
		SourceCampaignID: record.SourceCampaignID,
		Name:             record.Name,
		Eligible:         len(audience.Recipients),
		Sent:             record.Sent,
		Failed:           record.Failed,
		Remaining:        max(0, len(audience.Recipients)-record.Sent),
		EmailPreview:     campaignEmailPreview{Subject: subject, HTML: htmlBody, Text: text},
	}
}

func countCustomCampaignSent(eligible []customCampaignRecipient, sent map[string]bool) int {
	count := 0
	for _, person := range eligible {
		if sent[campaignEmailFingerprint(person.Email)] {
			count++
		}
	}
	return count
}

func customCampaignStatus(eligible, sent int) string {
	if sent == 0 {
		return "draft"
	}
	if sent >= eligible {
		return "complete"
	}
	return "sending"
}

func loadCustomCampaignRecord(ctx context.Context, db *firestore.Client, id string) (customCampaignRecord, error) {
	if !customCampaignIDRegexp.MatchString(id) {
		return customCampaignRecord{}, fmt.Errorf("campaign ID is invalid")
	}
	doc, err := db.Collection("emailCampaigns").Doc(id).Get(ctx)
	if err != nil {
		return customCampaignRecord{}, err
	}
	var record customCampaignRecord
	if err := doc.DataTo(&record); err != nil {
		return customCampaignRecord{}, err
	}
	return record, nil
}

func loadCustomCampaignHistory(ctx context.Context) (customCampaignHistory, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return customCampaignHistory{}, err
	}
	campaigns := map[string]customCampaignRecord{}
	iter := db.Collection("emailCampaigns").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignHistory{}, err
		}
		var record customCampaignRecord
		if err := doc.DataTo(&record); err != nil {
			continue
		}
		if record.CampaignID == "" {
			record.CampaignID = doc.Ref.ID
		}
		campaigns[record.CampaignID] = record
	}
	iter.Stop()

	deliveryCounts := map[string]int{}
	deliveryLastSent := map[string]time.Time{}
	deliveryFeedback := make([]campaignDeliveryFeedback, 0)
	deliveries := db.CollectionGroup("deliveries").Documents(ctx)
	for {
		doc, err := deliveries.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignHistory{}, err
		}
		campaignRef := doc.Ref.Parent.Parent
		if campaignRef == nil || campaignRef.Parent.ID != "emailCampaigns" {
			continue
		}
		var delivery struct {
			Status         string    `firestore:"status"`
			ResendID       string    `firestore:"resendId"`
			ProviderStatus string    `firestore:"providerStatus"`
			SentAt         time.Time `firestore:"sentAt"`
		}
		if doc.DataTo(&delivery) != nil || delivery.Status != "sent" {
			continue
		}
		if _, ok := campaigns[campaignRef.ID]; !ok {
			campaigns[campaignRef.ID] = inferredLegacyCampaign(campaignRef.ID)
		}
		deliveryCounts[campaignRef.ID]++
		if delivery.SentAt.After(deliveryLastSent[campaignRef.ID]) {
			deliveryLastSent[campaignRef.ID] = delivery.SentAt
		}
		deliveryFeedback = append(deliveryFeedback, campaignDeliveryFeedback{
			CampaignID: campaignRef.ID, ResendID: delivery.ResendID,
			ProviderStatus: delivery.ProviderStatus, Ref: doc.Ref,
		})
	}
	deliveries.Stop()

	feedbackRefreshedAt, feedbackAvailable, feedbackMessage := refreshCampaignDeliveryFeedback(ctx, db, deliveryFeedback)
	feedbackTotals := map[string]customCampaignRecord{}
	for _, delivery := range deliveryFeedback {
		total := feedbackTotals[delivery.CampaignID]
		addDeliveryFeedback(&total, delivery.ProviderStatus)
		feedbackTotals[delivery.CampaignID] = total
	}

	result := make([]customCampaignRecord, 0, len(campaigns))
	for id, record := range campaigns {
		if count, ok := deliveryCounts[id]; ok {
			record.Sent = count
		}
		if sentAt := deliveryLastSent[id]; sentAt.After(record.LastSentAt) {
			record.LastSentAt = sentAt
			record.UpdatedAt = sentAt
		}
		if record.Eligible > 0 {
			record.Remaining = max(0, record.Eligible-record.Sent)
		}
		record.Status = customCampaignStatus(record.Eligible, record.Sent)
		feedback := feedbackTotals[id]
		record.Delivered = feedback.Delivered
		record.Opened = feedback.Opened
		record.Clicked = feedback.Clicked
		record.Bounced = feedback.Bounced
		record.Delayed = feedback.Delayed
		record.Suppressed = feedback.Suppressed
		record.Complained = feedback.Complained
		record.ProviderFailed = feedback.ProviderFailed
		record.Undeliverable = feedback.Undeliverable
		record.AwaitingDelivery = feedback.AwaitingDelivery
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return customCampaignHistory{
		Campaigns: result, Placeholders: append([]string(nil), customCampaignPlaceholders...),
		FeedbackAvailable: feedbackAvailable, FeedbackRefreshedAt: feedbackRefreshedAt,
		FeedbackMessage: feedbackMessage,
	}, nil
}

func inferredLegacyCampaign(id string) customCampaignRecord {
	record := customCampaignRecord{
		CampaignID: id,
		Kind:       "legacy",
		Name:       id,
		Status:     "complete",
	}
	switch {
	case strings.HasPrefix(id, "member-referral-"):
		record.Kind = "member-referral"
		record.Name = "Member referral"
		record.Subject = "Could you help one more I-PACE owner find us?"
		record.Markdown = "Hi {{memberFirstName}},\n\nThank you for joining the I-PACE Owners group. We now have {{membersJoined}} members.\n\nCould you help another I-PACE owner find us?\n\nhttps://ipace-owners.org/"
	case strings.HasPrefix(id, "join-account-verification-"):
		record.Kind = "join-reengagement"
		record.Name = "Registration reminder"
	}
	return record
}

func recordSpecializedCampaign(ctx context.Context, db *firestore.Client, summary campaignSummary, kind, name, subject, markdown string) error {
	now := time.Now().UTC()
	createdAt := now
	batchCount := 1
	failed := 0
	if existing, err := loadCustomCampaignRecord(ctx, db, summary.CampaignID); err == nil {
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
		batchCount = existing.BatchCount + 1
		failed = existing.Failed
	}
	record := customCampaignRecord{
		CampaignID: summary.CampaignID,
		Kind:       kind,
		Name:       name,
		Subject:    subject,
		Markdown:   markdown,
		Status:     customCampaignStatus(summary.Eligible, summary.Sent),
		Eligible:   summary.Eligible,
		Sent:       summary.Sent,
		Failed:     failed,
		Remaining:  summary.Remaining,
		BatchCount: batchCount,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
		LastSentAt: now,
	}
	return saveCustomCampaignRecord(ctx, db.Collection("emailCampaigns").Doc(summary.CampaignID), record)
}

func saveCustomCampaignRecord(ctx context.Context, document customCampaignDocumentWriter, record customCampaignRecord) error {
	// Firestore MergeAll accepts map data only. The record is the authoritative parent
	// document, and replacing it does not remove its deliveries subcollection.
	_, err := document.Set(ctx, record)
	return err
}

func loadCustomCampaignAudience(ctx context.Context) (customCampaignAudience, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return customCampaignAudience{}, err
	}
	authClient, err := firebaseAuth(ctx)
	if err != nil {
		return customCampaignAudience{}, err
	}
	joins, err := loadCampaignJoins(ctx, db)
	if err != nil {
		return customCampaignAudience{}, err
	}
	joinByEmail := map[string]campaignRecipient{}
	for _, person := range joins {
		joinByEmail[canonicalCampaignEmail(person.Email)] = person
	}

	accounts := map[string]*auth.ExportedUserRecord{}
	userIter := authClient.Users(ctx, "")
	verifiedCount := 0
	for {
		user, err := userIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignAudience{}, err
		}
		key := canonicalCampaignEmail(user.Email)
		if key == "" {
			continue
		}
		if user.EmailVerified {
			if _, exists := accounts[key]; !exists {
				verifiedCount++
			}
			accounts[key] = user
		}
	}

	vehiclesByUID := map[string][]customCampaignVehicle{}
	vehicleIndex := map[string]struct {
		UID   string
		Index int
	}{}
	vehicleCount := 0
	vehicleIter := db.Collection("vehicles").Documents(ctx)
	for {
		doc, err := vehicleIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignAudience{}, err
		}
		var record vehicleRecord
		if doc.DataTo(&record) != nil || record.IdentityUserID == "" {
			continue
		}
		vehicleCount++
		item := customCampaignVehicle{
			ID:                    record.ID,
			Registration:          record.Vehicle.Registration,
			Country:               record.Vehicle.Country,
			ModelYear:             record.Vehicle.ModelYear,
			Mileage:               record.Vehicle.Mileage,
			OwnedSince:            record.Vehicle.OwnedSince,
			FirstRegistrationDate: record.Vehicle.FirstRegistrationDate,
		}
		vehiclesByUID[record.IdentityUserID] = append(vehiclesByUID[record.IdentityUserID], item)
		vehicleIndex[record.ID] = struct {
			UID   string
			Index int
		}{record.IdentityUserID, len(vehiclesByUID[record.IdentityUserID]) - 1}
	}
	vehicleIter.Stop()

	readingCount := 0
	readingIter := db.Collection("batteryReadings").Documents(ctx)
	for {
		doc, err := readingIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignAudience{}, err
		}
		var record batteryReadingRecord
		if doc.DataTo(&record) != nil {
			continue
		}
		readingCount++
		if location, ok := vehicleIndex[record.VehicleID]; ok {
			vehiclesByUID[location.UID][location.Index].SoHReadingsCount++
		}
	}
	readingIter.Stop()

	serviceFaultQuery := db.Collection("serviceEvents").Where("review.status", "!=", "deleted")
	serviceFaultAggregation, err := serviceFaultQuery.
		NewAggregationQuery().
		WithCount("serviceFaultRecords").
		Get(ctx)
	if err != nil {
		return customCampaignAudience{}, err
	}
	serviceFaultRecordCount, ok := serviceFaultAggregation.Data()["serviceFaultRecords"].(int64)
	if !ok || serviceFaultRecordCount < 0 {
		return customCampaignAudience{}, fmt.Errorf("service-event count aggregation returned an invalid value")
	}

	verifiedAtByUID := map[string]time.Time{}
	memberIter := db.Collection("members").Documents(ctx)
	for {
		doc, err := memberIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return customCampaignAudience{}, err
		}
		var member struct {
			EmailVerifiedAt time.Time `firestore:"emailVerifiedAt"`
		}
		if doc.DataTo(&member) == nil && !member.EmailVerifiedAt.IsZero() {
			verifiedAtByUID[doc.Ref.ID] = member.EmailVerifiedAt
		}
	}
	memberIter.Stop()

	recipients := []customCampaignRecipient{}
	verificationBackfills := map[string]time.Time{}
	for key, person := range joinByEmail {
		account := accounts[key]
		if account == nil {
			continue
		}
		verifiedAt, inferred := customCampaignVerifiedAt(verifiedAtByUID[account.UID], account.UserMetadata)
		if inferred {
			verificationBackfills[account.UID] = verifiedAt
		}
		recipients = append(recipients, customCampaignRecipient{
			UID:        account.UID,
			Email:      strings.ToLower(strings.TrimSpace(account.Email)),
			Name:       person.Name,
			JoinedAt:   person.CreatedAt,
			VerifiedAt: verifiedAt,
			Vehicles:   vehiclesByUID[account.UID],
		})
	}
	if err := persistCampaignVerificationBackfills(ctx, db, verificationBackfills); err != nil {
		return customCampaignAudience{}, err
	}
	sort.Slice(recipients, func(i, j int) bool { return recipients[i].Email < recipients[j].Email })
	return customCampaignAudience{
		Recipients:               recipients,
		MembersJoined:            len(joinByEmail),
		MembersVerified:          verifiedCount,
		VehiclesRegisteredCount:  vehicleCount,
		VehiclesSoHReadingsCount: readingCount,
		ServiceFaultRecordsCount: int(serviceFaultRecordCount),
	}, nil
}

func customCampaignVerifiedAt(stored time.Time, metadata *auth.UserMetadata) (time.Time, bool) {
	if !stored.IsZero() {
		return stored.UTC(), false
	}
	if metadata == nil {
		return time.Time{}, false
	}
	if metadata.LastLogInTimestamp > 0 {
		return time.UnixMilli(metadata.LastLogInTimestamp).UTC(), true
	}
	if metadata.CreationTimestamp > 0 {
		return time.UnixMilli(metadata.CreationTimestamp).UTC(), true
	}
	return time.Time{}, false
}

func persistCampaignVerificationBackfills(ctx context.Context, db *firestore.Client, values map[string]time.Time) error {
	const batchLimit = 400
	batch := db.Batch()
	pending := 0
	for uid, verifiedAt := range values {
		batch.Set(db.Collection("members").Doc(uid), map[string]any{
			"emailVerifiedAt":         verifiedAt,
			"emailVerifiedAtInferred": true,
		}, firestore.MergeAll)
		pending++
		if pending == batchLimit {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = db.Batch()
			pending = 0
		}
	}
	if pending > 0 {
		_, err := batch.Commit(ctx)
		return err
	}
	return nil
}
