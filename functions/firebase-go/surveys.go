package ipace

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/csv"
	firebaseauth "firebase.google.com/go/v4/auth"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const surveyOtherTextMax = 250
const surveyDescriptionMax = 4000
const surveyCallToActionMax = 1000
const surveyOptionNameMax = 120
const surveyOptionDescriptionMax = 2000
const surveyOptionTextPromptMax = 160
const adminSurveyResponsePageSize = 50

type surveyOption struct {
	ID          string `json:"id" firestore:"id"`
	Name        string `json:"name" firestore:"name"`
	Description string `json:"description" firestore:"description"`
	TextPrompt  string `json:"textPrompt,omitempty" firestore:"textPrompt,omitempty"`
	// Label is retained only to read surveys created before options had separate names and descriptions.
	Label           string `json:"label,omitempty" firestore:"label,omitempty"`
	AllowsText      bool   `json:"allowsText" firestore:"allowsText"`
	AllowsPreferred bool   `json:"allowsPreferred" firestore:"allowsPreferred"`
}
type surveyRecord struct {
	ID           string `json:"id" firestore:"id"`
	Title        string `json:"title" firestore:"title"`
	Description  string `json:"description" firestore:"description"`
	Question     string `json:"question" firestore:"question"`
	CallToAction string `json:"callToAction" firestore:"callToAction"`
	Status       string `json:"status" firestore:"status"`
	Multiple     bool   `json:"multiple" firestore:"multiple"`
	StartsOn     string `json:"startsOn" firestore:"startsOn"`
	EndsOn       string `json:"endsOn" firestore:"endsOn"`
	ShowResults  bool   `json:"showResults" firestore:"showResults"`
	// PreferredEligibilityConfigured distinguishes surveys created before option-level
	// preferred eligibility was introduced. Those legacy surveys retain their
	// previous behaviour until an admin saves them.
	PreferredEligibilityConfigured bool           `json:"preferredEligibilityConfigured" firestore:"preferredEligibilityConfigured"`
	Options                        []surveyOption `json:"options" firestore:"options"`
	CreatedAt                      time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt                      time.Time      `json:"updatedAt" firestore:"updatedAt"`
}
type surveyInput struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Question     string         `json:"question"`
	CallToAction string         `json:"callToAction"`
	Status       string         `json:"status"`
	Multiple     bool           `json:"multiple"`
	StartsOn     string         `json:"startsOn"`
	EndsOn       string         `json:"endsOn"`
	ShowResults  bool           `json:"showResults"`
	Options      []surveyOption `json:"options"`
}
type surveyResponseInput struct {
	SurveyID          string            `json:"surveyId"`
	OptionIDs         []string          `json:"optionIds"`
	TextByOption      map[string]string `json:"textByOption"`
	PreferredOptionID string            `json:"preferredOptionId"`
}
type surveyResult struct {
	SurveyRecord        surveyRecord      `json:"survey"`
	Counts              map[string]int    `json:"counts"`
	PreferredCounts     map[string]int    `json:"preferredCounts"`
	TextCounts          map[string]int    `json:"textCounts"`
	Total               int               `json:"total"`
	MyOptionIDs         []string          `json:"myOptionIds,omitempty"`
	MyTextByOption      map[string]string `json:"myTextByOption,omitempty"`
	MyPreferredOptionID string            `json:"myPreferredOptionId,omitempty"`
	CanRespond          bool              `json:"canRespond"`
}

// surveyAggregate contains only counts. It is stored separately from the
// editable survey template so an edit cannot overwrite live response totals.
type surveyAggregate struct {
	Counts          map[string]int `firestore:"counts"`
	PreferredCounts map[string]int `firestore:"preferredCounts"`
	TextCounts      map[string]int `firestore:"textCounts"`
	Total           int            `firestore:"total"`
	UpdatedAt       time.Time      `firestore:"updatedAt"`
}
type surveyResponseRecord struct {
	OptionIDs         []string          `firestore:"optionIds"`
	TextByOption      map[string]string `firestore:"textByOption"`
	PreferredOptionID string            `firestore:"preferredOptionId"`
	UpdatedAt         time.Time         `firestore:"updatedAt"`
}
type adminSurveyResponse struct {
	Respondent        string    `json:"respondent"`
	OptionIDs         []string  `json:"optionIds"`
	TextResponses     []string  `json:"textResponses"`
	PreferredOptionID string    `json:"preferredOptionId,omitempty"`
	UpdatedAt         time.Time `json:"updatedAt"`
}
type storedSurveyResponse struct {
	UID      string
	Response surveyResponseRecord
}
type adminSurveyAnalysis struct {
	Survey          surveyRecord          `json:"survey"`
	Counts          map[string]int        `json:"counts"`
	PreferredCounts map[string]int        `json:"preferredCounts"`
	Total           int                   `json:"total"`
	Responses       []adminSurveyResponse `json:"responses"`
	ResponsesOffset int                   `json:"responsesOffset"`
	HasMore         bool                  `json:"hasMore"`
}

var surveyNow = time.Now

func AdminSurveys(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not connect to data store"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		var surveys []surveyRecord
		if err := readCollection(r.Context(), db.Collection("surveys").OrderBy("createdAt", firestore.Desc), &surveys); err != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not load surveys"})
			return
		}
		if surveys == nil {
			surveys = []surveyRecord{}
		}
		writeJSON(w, 200, map[string]any{"surveys": surveys})
	case http.MethodPost, http.MethodPut:
		var input surveyInput
		if decodeJSON(r, &input) != nil {
			writeJSON(w, 400, map[string]any{"error": "Invalid request body"})
			return
		}
		record, e := validateSurvey(input)
		if e != nil {
			writeJSON(w, 400, map[string]any{"error": e.Error()})
			return
		}
		now := surveyNow().UTC()
		if r.Method == http.MethodPost {
			if record.ID == "" {
				record.ID = submissionID("survey")
			}
			record.CreatedAt = now
		} else {
			if record.ID == "" {
				writeJSON(w, 400, map[string]any{"error": "Survey ID is required"})
				return
			}
			old, e := db.Collection("surveys").Doc(record.ID).Get(r.Context())
			if e != nil {
				writeJSON(w, 404, map[string]any{"error": "Survey not found"})
				return
			}
			var existing surveyRecord
			if old.DataTo(&existing) != nil {
				writeJSON(w, 500, map[string]any{"error": "Could not load survey"})
				return
			}
			record.CreatedAt = existing.CreatedAt
		}
		record.UpdatedAt = now
		surveyRef := db.Collection("surveys").Doc(record.ID)
		if r.Method == http.MethodPut {
			batch := db.Batch()
			batch.Set(surveyRef, record)
			batch.Delete(surveyAggregateRef(db, record.ID))
			_, e = batch.Commit(r.Context())
		} else {
			_, e = surveyRef.Set(r.Context(), record)
		}
		if e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not save survey"})
			return
		}
		action := "admin-survey-created"
		if r.Method == http.MethodPut {
			action = "admin-survey-updated"
		}
		logSurveyEvent(r, action, record.ID, map[string]any{"status": record.Status})
		writeJSON(w, 200, record)
	case http.MethodDelete:
		id := cleanString(r.URL.Query().Get("id"), 160)
		if id == "" {
			writeJSON(w, 400, map[string]any{"error": "Survey ID is required"})
			return
		}
		if e := deleteSurveyResponses(r.Context(), db, id); e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not delete survey responses"})
			return
		}
		if _, e := surveyAggregateRef(db, id).Delete(r.Context()); e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not delete survey results"})
			return
		}
		if _, e := db.Collection("surveys").Doc(id).Delete(r.Context()); e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not delete survey"})
			return
		}
		logSurveyEvent(r, "admin-survey-deleted", id, nil)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
	}
}

func AdminSurveyResults(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	id := cleanString(r.URL.Query().Get("id"), 160)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Survey ID is required"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}
	snapshot, err := db.Collection("surveys").Doc(id).Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey not found"})
		return
	}
	var survey surveyRecord
	if err := snapshot.DataTo(&survey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey"})
		return
	}
	offset, err := surveyResponseOffset(r.URL.Query().Get("offset"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid response page"})
		return
	}
	pageSize := adminSurveyResponsePageSize
	if r.URL.Query().Get("format") == "csv" {
		pageSize = 0
		offset = 0
	}
	analysis, err := loadAdminSurveyAnalysis(r.Context(), db, survey, offset, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey responses"})
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		logSurveyEvent(r, "admin-survey-csv-downloaded", id, map[string]any{"responseCount": analysis.Total})
		writeAdminSurveyCSV(w, id, analysis)
		return
	}
	logSurveyEvent(r, "admin-survey-analysis-viewed", id, map[string]any{"responseCount": analysis.Total})
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, analysis)
}

// AdminSurveyPreview lets administrators validate a response against an unpublished survey
// without creating a response document or affecting member-visible results.
func AdminSurveyPreview(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	id := cleanString(r.URL.Query().Get("id"), 160)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Survey ID is required"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}
	snapshot, err := db.Collection("surveys").Doc(id).Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey not found"})
		return
	}
	var survey surveyRecord
	if err := snapshot.DataTo(&survey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey"})
		return
	}
	if r.Method == http.MethodGet {
		logSurveyEvent(r, "admin-survey-preview-viewed", id, nil)
		writeJSON(w, http.StatusOK, survey)
		return
	}
	var input surveyResponseInput
	if decodeJSON(r, &input) != nil || cleanString(input.SurveyID, 160) != id {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid test response"})
		return
	}
	if _, _, _, err := validateSurveyResponse(survey, input); err != nil {
		logSurveyEvent(r, "admin-survey-preview-rejected", id, map[string]any{"reason": "validation"})
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	logSurveyEvent(r, "admin-survey-preview-validated", id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

func surveyResponseOffset(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 || offset > 100000 {
		return 0, fmt.Errorf("invalid response offset")
	}
	return offset, nil
}

// loadAdminSurveyAnalysis scans response records for the aggregate, but only
// resolves and returns a bounded page of respondent details. Firebase Auth
// lookups are batched rather than made serially for every survey response.
func loadAdminSurveyAnalysis(ctx context.Context, db *firestore.Client, survey surveyRecord, offset, pageSize int) (adminSurveyAnalysis, error) {
	aggregate, err := loadSurveyResult(ctx, db, survey, "")
	if err != nil {
		return adminSurveyAnalysis{}, err
	}
	analysis := adminSurveyAnalysis{Survey: survey, Counts: aggregate.Counts, PreferredCounts: aggregate.PreferredCounts, Total: aggregate.Total, Responses: []adminSurveyResponse{}, ResponsesOffset: offset}
	allowed := map[string]bool{}
	for _, option := range survey.Options {
		allowed[option.ID] = true
	}
	query := db.Collection("surveys").Doc(survey.ID).Collection("responses").OrderBy("updatedAt", firestore.Desc)
	if offset > 0 {
		query = query.Offset(offset)
	}
	if pageSize > 0 {
		query = query.Limit(pageSize + 1)
	}
	iter := query.Documents(ctx)
	defer iter.Stop()
	responses := []storedSurveyResponse{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return analysis, err
		}
		var response storedSurveyResponse
		response.UID = doc.Ref.ID
		if err := doc.DataTo(&response.Response); err != nil {
			return analysis, err
		}
		responses = append(responses, response)
	}
	if pageSize > 0 && len(responses) > pageSize {
		analysis.HasMore = true
		responses = responses[:pageSize]
	}
	respondents, err := maskedSurveyRespondents(ctx, responses)
	if err != nil {
		return analysis, err
	}
	for _, response := range responses {
		item := adminSurveyResponse{Respondent: respondents[response.UID], UpdatedAt: response.Response.UpdatedAt}
		if item.Respondent == "" {
			item.Respondent = "Email unavailable"
		}
		for _, id := range response.Response.OptionIDs {
			if !allowed[id] {
				continue
			}
			item.OptionIDs = append(item.OptionIDs, id)
			if text := cleanString(response.Response.TextByOption[id], surveyOtherTextMax); text != "" {
				item.TextResponses = append(item.TextResponses, id+": "+text)
			}
		}
		if surveyResponseAllowsPreferred(survey, response.Response.OptionIDs, response.Response.PreferredOptionID) {
			item.PreferredOptionID = response.Response.PreferredOptionID
		}
		analysis.Responses = append(analysis.Responses, item)
	}
	return analysis, nil
}

func maskedSurveyRespondents(ctx context.Context, responses []storedSurveyResponse) (map[string]string, error) {
	result := map[string]string{}
	if len(responses) == 0 {
		return result, nil
	}
	client, err := firebaseAuth(ctx)
	if err != nil {
		return nil, err
	}
	for start := 0; start < len(responses); start += 100 {
		end := min(start+100, len(responses))
		identifiers := make([]firebaseauth.UserIdentifier, 0, end-start)
		for _, response := range responses[start:end] {
			identifiers = append(identifiers, firebaseauth.UIDIdentifier{UID: response.UID})
		}
		users, err := client.GetUsers(ctx, identifiers)
		if err != nil {
			return nil, err
		}
		for _, user := range users.Users {
			if masked := maskedEmail(user.Email); masked != "" {
				result[user.UID] = masked
			}
		}
	}
	return result, nil
}

func writeAdminSurveyCSV(w http.ResponseWriter, id string, analysis adminSurveyAnalysis) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=survey-results-"+id+".csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"masked_respondent", "submitted_at_utc", "selected_option_ids", "preferred_option_id", "text_responses"})
	for _, response := range analysis.Responses {
		_ = writer.Write([]string{safeCSVCell(response.Respondent), response.UpdatedAt.UTC().Format(time.RFC3339), safeCSVCell(strings.Join(response.OptionIDs, " | ")), safeCSVCell(response.PreferredOptionID), safeCSVCell(strings.Join(response.TextResponses, " | "))})
	}
	writer.Flush()
}

// safeCSVCell prevents spreadsheet applications from interpreting member-supplied text as a formula.
func safeCSVCell(value string) string {
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}
func validateSurvey(input surveyInput) (surveyRecord, error) {
	r := surveyRecord{ID: cleanString(input.ID, 160), Title: cleanString(input.Title, 120), Description: cleanString(input.Description, surveyDescriptionMax), Question: cleanString(input.Question, 500), CallToAction: cleanString(input.CallToAction, surveyCallToActionMax), Status: cleanString(input.Status, 16), Multiple: input.Multiple, StartsOn: cleanString(input.StartsOn, 10), EndsOn: cleanString(input.EndsOn, 10), ShowResults: input.ShowResults, PreferredEligibilityConfigured: true}
	if r.Title == "" {
		return r, fmt.Errorf("title is required")
	}
	if r.Status == "" {
		r.Status = "draft"
	}
	if r.Status != "draft" && r.Status != "published" {
		return r, fmt.Errorf("survey status must be draft or published")
	}
	start, e := time.Parse("2006-01-02", r.StartsOn)
	if e != nil {
		return r, fmt.Errorf("a valid start date is required")
	}
	end, e := time.Parse("2006-01-02", r.EndsOn)
	if e != nil || end.Before(start) {
		return r, fmt.Errorf("end date must be on or after the start date")
	}
	if len(input.Options) < 2 || len(input.Options) > 12 {
		return r, fmt.Errorf("add between 2 and 12 options")
	}
	seen := map[string]bool{}
	for i, o := range input.Options {
		o.ID = cleanString(o.ID, 40)
		o.Name = cleanSurveyOptionName(o.Name)
		o.Description = cleanString(o.Description, surveyOptionDescriptionMax)
		o.TextPrompt = cleanString(o.TextPrompt, surveyOptionTextPromptMax)
		o.Label = cleanString(o.Label, surveyOptionDescriptionMax)
		if o.Description == "" {
			o.Description = o.Label
		}
		if o.Name == "" {
			o.Name = surveyOptionFallbackName(o.Description)
		}
		if o.ID == "" {
			o.ID = fmt.Sprintf("option-%d", i+1)
		}
		if o.Name == "" || o.Description == "" || seen[o.ID] {
			return r, fmt.Errorf("each option needs a unique ID, name and description")
		}
		o.Label = ""
		if o.AllowsText && o.TextPrompt == "" {
			o.TextPrompt = "Optional detail"
		}
		if !o.AllowsText {
			o.TextPrompt = ""
		}
		seen[o.ID] = true
		r.Options = append(r.Options, o)
	}
	return r, nil
}

func cleanSurveyOptionName(value string) string {
	return cleanString(strings.Join(strings.Fields(value), " "), surveyOptionNameMax)
}

func surveyOptionFallbackName(description string) string {
	for _, line := range strings.Split(description, "\n") {
		line = cleanSurveyOptionName(strings.Trim(line, " -*"))
		if line != "" {
			return line
		}
	}
	return ""
}

func deleteSurveyResponses(ctx context.Context, db *firestore.Client, surveyID string) error {
	iter := db.Collection("surveys").Doc(surveyID).Collection("responses").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}
}
func MemberSurveys(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return
	}
	u, e := requireUser(r.Context(), r)
	if e != nil {
		writeJSON(w, 401, map[string]any{"error": "Sign in required"})
		return
	}
	db, e := firestoreClient(r.Context())
	if e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not connect to data store"})
		return
	}
	var surveys []surveyRecord
	if e = readCollection(r.Context(), db.Collection("surveys").OrderBy("createdAt", firestore.Desc), &surveys); e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not load surveys"})
		return
	}
	out := []surveyResult{}
	resultsVisible := 0
	for _, s := range surveys {
		if !surveyIsPublished(s) {
			continue
		}
		if result, e := loadSurveyResult(r.Context(), db, s, u.UID); e == nil {
			if !memberMayViewSurveyResults(s, result) {
				result.Counts = nil
				result.PreferredCounts = nil
				result.TextCounts = nil
				result.Total = 0
			} else {
				resultsVisible++
			}
			out = append(out, result)
		}
	}
	logSurveyEvent(r, "member-survey-list-viewed", "", map[string]any{"publishedCount": len(out), "resultsVisibleCount": resultsVisible})
	writeJSON(w, 200, map[string]any{"surveys": out})
}
func SubmitSurveyResponse(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return
	}
	u, e := requireUser(r.Context(), r)
	if e != nil {
		writeJSON(w, 401, map[string]any{"error": "Sign in required"})
		return
	}
	var input surveyResponseInput
	if decodeJSON(r, &input) != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid request body"})
		return
	}
	db, e := firestoreClient(r.Context())
	if e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not connect to data store"})
		return
	}
	snap, e := db.Collection("surveys").Doc(cleanString(input.SurveyID, 160)).Get(r.Context())
	if e != nil {
		writeJSON(w, 404, map[string]any{"error": "Survey not found"})
		return
	}
	var s surveyRecord
	if snap.DataTo(&s) != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not load survey"})
		return
	}
	if !surveyIsPublished(s) {
		writeJSON(w, 404, map[string]any{"error": "Survey not found"})
		return
	}
	if !surveyIsLive(s, surveyNow()) {
		logSurveyEvent(r, "member-survey-response-rejected", s.ID, map[string]any{"reason": "not-live"})
		writeJSON(w, 409, map[string]any{"error": "This survey is not currently open"})
		return
	}
	ids, textByOption, preferredOptionID, e := validateSurveyResponse(s, input)
	if e != nil {
		logSurveyEvent(r, "member-survey-response-rejected", s.ID, map[string]any{"reason": "validation"})
		writeJSON(w, 400, map[string]any{"error": e.Error()})
		return
	}
	responseRef := db.Collection("surveys").Doc(s.ID).Collection("responses").Doc(u.UID)
	response := surveyResponseRecord{OptionIDs: ids, TextByOption: textByOption, PreferredOptionID: preferredOptionID, UpdatedAt: surveyNow().UTC()}
	created, e := saveSurveyResponse(r.Context(), db, s, responseRef, response)
	if e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not save response"})
		return
	}
	action := "member-survey-response-created"
	if !created {
		action = "member-survey-response-updated"
	}
	logSurveyEvent(r, action, s.ID, nil)
	result, e := loadSurveyResult(r.Context(), db, s, u.UID)
	if e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not load results"})
		return
	}
	if !memberMayViewSurveyResults(s, result) {
		result.Counts = nil
		result.PreferredCounts = nil
		result.TextCounts = nil
		result.Total = 0
	}
	writeJSON(w, 200, result)
}

// logSurveyEvent records only operational metadata. Survey selections, free text,
// member identities, and administrative CSV contents must never enter Cloud Logging.
func logSurveyEvent(r *http.Request, action, surveyID string, fields map[string]any) {
	logEvent("survey", "info", action, surveyEventFields(r, surveyID, fields))
}

func surveyEventFields(r *http.Request, surveyID string, fields map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range fields {
		result[key] = value
	}
	if surveyID != "" {
		result["surveyId"] = surveyID
	}
	return addAuthTrace(result, r)
}

func memberMayViewSurveyResults(s surveyRecord, result surveyResult) bool {
	return s.ShowResults && (len(result.MyOptionIDs) > 0 || surveyIsClosed(s, surveyNow()))
}

func surveyIsPublished(s surveyRecord) bool {
	// Surveys created before the status field existed remain visible rather than disappearing.
	return s.Status != "draft"
}

func surveyOptionAllowsPreferred(s surveyRecord, optionID string) bool {
	for _, option := range s.Options {
		if option.ID == optionID {
			return !s.PreferredEligibilityConfigured || option.AllowsPreferred
		}
	}
	return false
}

func surveyResponseAllowsPreferred(s surveyRecord, optionIDs []string, preferredOptionID string) bool {
	if !s.Multiple || !surveyOptionAllowsPreferred(s, preferredOptionID) {
		return false
	}
	for _, optionID := range optionIDs {
		if optionID == preferredOptionID {
			return true
		}
	}
	return false
}

func validateSurveyResponse(s surveyRecord, input surveyResponseInput) ([]string, map[string]string, string, error) {
	allowed := map[string]surveyOption{}
	for _, o := range s.Options {
		allowed[o.ID] = o
	}
	seen := map[string]bool{}
	ids := []string{}
	textByOption := map[string]string{}
	for _, id := range input.OptionIDs {
		id = cleanString(id, 40)
		_, ok := allowed[id]
		if !ok || seen[id] {
			return nil, nil, "", fmt.Errorf("choose valid survey options")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 || (!s.Multiple && len(ids) != 1) {
		return nil, nil, "", fmt.Errorf("choose a valid number of options")
	}
	for _, id := range ids {
		if !allowed[id].AllowsText {
			continue
		}
		text := cleanString(input.TextByOption[id], surveyOtherTextMax)
		if text != "" {
			textByOption[id] = text
		}
	}
	preferred := cleanString(input.PreferredOptionID, 40)
	if preferred != "" {
		if !s.Multiple || !seen[preferred] || !surveyOptionAllowsPreferred(s, preferred) {
			return nil, nil, "", fmt.Errorf("choose one selected eligible option as preferred, or leave it blank")
		}
	}
	return ids, textByOption, preferred, nil
}
func surveyIsLive(s surveyRecord, now time.Time) bool {
	start, e := time.Parse("2006-01-02", s.StartsOn)
	if e != nil {
		return false
	}
	end, e := time.Parse("2006-01-02", s.EndsOn)
	if e != nil {
		return false
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return !today.Before(start) && !today.After(end)
}

func surveyIsClosed(s surveyRecord, now time.Time) bool {
	end, err := time.Parse("2006-01-02", s.EndsOn)
	if err != nil {
		return false
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return today.After(end)
}
func loadSurveyResult(ctx context.Context, db *firestore.Client, s surveyRecord, uid string) (surveyResult, error) {
	aggregate, err := ensureSurveyAggregate(ctx, db, s)
	if err != nil {
		return surveyResult{}, err
	}
	r := surveyResult{SurveyRecord: s, Counts: aggregate.Counts, PreferredCounts: aggregate.PreferredCounts, TextCounts: aggregate.TextCounts, Total: aggregate.Total, CanRespond: surveyIsLive(s, surveyNow())}
	if uid == "" {
		return r, nil
	}
	snapshot, err := db.Collection("surveys").Doc(s.ID).Collection("responses").Doc(uid).Get(ctx)
	if isFirestoreNotFound(err) {
		return r, nil
	}
	if err != nil {
		return surveyResult{}, err
	}
	var response surveyResponseRecord
	if err := snapshot.DataTo(&response); err != nil {
		return surveyResult{}, err
	}
	r.MyOptionIDs = response.OptionIDs
	r.MyTextByOption = response.TextByOption
	if surveyResponseAllowsPreferred(s, response.OptionIDs, response.PreferredOptionID) {
		r.MyPreferredOptionID = response.PreferredOptionID
	}
	return r, nil
}

func surveyAggregateRef(db *firestore.Client, surveyID string) *firestore.DocumentRef {
	return db.Collection("surveys").Doc(surveyID).Collection("metadata").Doc("aggregate")
}

func isFirestoreNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// ensureSurveyAggregate upgrades older surveys lazily. The first request after
// this feature scans legacy responses once; every later results request reads
// only the count-only aggregate plus the requesting member's own response.
func ensureSurveyAggregate(ctx context.Context, db *firestore.Client, s surveyRecord) (surveyAggregate, error) {
	ref := surveyAggregateRef(db, s.ID)
	snapshot, err := ref.Get(ctx)
	if err == nil {
		return decodeSurveyAggregate(snapshot)
	}
	if !isFirestoreNotFound(err) {
		return surveyAggregate{}, err
	}
	aggregate, err := surveyAggregateFromResponses(ctx, db, s)
	if err != nil {
		return surveyAggregate{}, err
	}
	if _, err := ref.Create(ctx, aggregate); err == nil {
		return aggregate, nil
	}
	snapshot, err = ref.Get(ctx)
	if err != nil {
		return surveyAggregate{}, err
	}
	return decodeSurveyAggregate(snapshot)
}

func decodeSurveyAggregate(snapshot *firestore.DocumentSnapshot) (surveyAggregate, error) {
	var aggregate surveyAggregate
	if err := snapshot.DataTo(&aggregate); err != nil {
		return surveyAggregate{}, err
	}
	return normaliseSurveyAggregate(aggregate), nil
}

func normaliseSurveyAggregate(aggregate surveyAggregate) surveyAggregate {
	if aggregate.Counts == nil {
		aggregate.Counts = map[string]int{}
	}
	if aggregate.PreferredCounts == nil {
		aggregate.PreferredCounts = map[string]int{}
	}
	if aggregate.TextCounts == nil {
		aggregate.TextCounts = map[string]int{}
	}
	return aggregate
}

func surveyAggregateFromResponses(ctx context.Context, db *firestore.Client, s surveyRecord) (surveyAggregate, error) {
	aggregate := normaliseSurveyAggregate(surveyAggregate{})
	iter := db.Collection("surveys").Doc(s.ID).Collection("responses").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return surveyAggregate{}, err
		}
		var response surveyResponseRecord
		if err := doc.DataTo(&response); err != nil {
			return surveyAggregate{}, err
		}
		applySurveyResponseDelta(&aggregate, s, surveyResponseRecord{}, response)
	}
	aggregate.UpdatedAt = surveyNow().UTC()
	return aggregate, nil
}

func saveSurveyResponse(ctx context.Context, db *firestore.Client, s surveyRecord, responseRef *firestore.DocumentRef, response surveyResponseRecord) (bool, error) {
	if _, err := ensureSurveyAggregate(ctx, db, s); err != nil {
		return false, err
	}
	created := false
	err := db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshots, err := tx.GetAll([]*firestore.DocumentRef{responseRef, surveyAggregateRef(db, s.ID)})
		if err != nil && !isFirestoreNotFound(err) {
			return err
		}
		var previous surveyResponseRecord
		created = !snapshots[0].Exists()
		if snapshots[0].Exists() {
			if err := snapshots[0].DataTo(&previous); err != nil {
				return err
			}
		}
		aggregate, err := decodeSurveyAggregate(snapshots[1])
		if err != nil {
			return err
		}
		applySurveyResponseDelta(&aggregate, s, previous, response)
		aggregate.UpdatedAt = response.UpdatedAt
		if err := tx.Set(responseRef, response); err != nil {
			return err
		}
		return tx.Set(surveyAggregateRef(db, s.ID), aggregate)
	})
	return created, err
}

func applySurveyResponseDelta(aggregate *surveyAggregate, s surveyRecord, previous, next surveyResponseRecord) {
	*aggregate = normaliseSurveyAggregate(*aggregate)
	applySurveyResponseCounts(aggregate, s, previous, -1)
	applySurveyResponseCounts(aggregate, s, next, 1)
}

func applySurveyResponseCounts(aggregate *surveyAggregate, s surveyRecord, response surveyResponseRecord, delta int) {
	if len(response.OptionIDs) == 0 {
		return
	}
	aggregate.Total += delta
	allowsText := map[string]bool{}
	for _, option := range s.Options {
		allowsText[option.ID] = option.AllowsText
	}
	for _, id := range response.OptionIDs {
		aggregate.Counts[id] += delta
		if allowsText[id] && cleanString(response.TextByOption[id], surveyOtherTextMax) != "" {
			aggregate.TextCounts[id] += delta
		}
	}
	if surveyResponseAllowsPreferred(s, response.OptionIDs, response.PreferredOptionID) {
		aggregate.PreferredCounts[response.PreferredOptionID] += delta
	}
}
