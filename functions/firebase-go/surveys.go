package ipace

import (
	"cloud.google.com/go/firestore"
	"context"
	"encoding/csv"
	"fmt"
	"google.golang.org/api/iterator"
	"net/http"
	"strings"
	"time"
)

const surveyOtherTextMax = 250
const surveyDescriptionMax = 4000
const surveyCallToActionMax = 1000
const surveyOptionLabelMax = 2000

type surveyOption struct {
	ID         string `json:"id" firestore:"id"`
	Label      string `json:"label" firestore:"label"`
	AllowsText bool   `json:"allowsText" firestore:"allowsText"`
}
type surveyRecord struct {
	ID           string         `json:"id" firestore:"id"`
	Title        string         `json:"title" firestore:"title"`
	Description  string         `json:"description" firestore:"description"`
	Question     string         `json:"question" firestore:"question"`
	CallToAction string         `json:"callToAction" firestore:"callToAction"`
	Status       string         `json:"status" firestore:"status"`
	Multiple     bool           `json:"multiple" firestore:"multiple"`
	StartsOn     string         `json:"startsOn" firestore:"startsOn"`
	EndsOn       string         `json:"endsOn" firestore:"endsOn"`
	ShowResults  bool           `json:"showResults" firestore:"showResults"`
	Options      []surveyOption `json:"options" firestore:"options"`
	CreatedAt    time.Time      `json:"createdAt" firestore:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt" firestore:"updatedAt"`
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
	SurveyID     string            `json:"surveyId"`
	OptionIDs    []string          `json:"optionIds"`
	TextByOption map[string]string `json:"textByOption"`
}
type surveyResult struct {
	SurveyRecord   surveyRecord      `json:"survey"`
	Counts         map[string]int    `json:"counts"`
	Total          int               `json:"total"`
	MyOptionIDs    []string          `json:"myOptionIds,omitempty"`
	MyTextByOption map[string]string `json:"myTextByOption,omitempty"`
	CanRespond     bool              `json:"canRespond"`
}
type adminSurveyResponse struct {
	Respondent string    `json:"respondent"`
	Options    []string  `json:"options"`
	Text       []string  `json:"text"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
type adminSurveyAnalysis struct {
	Survey    surveyRecord          `json:"survey"`
	Counts    map[string]int        `json:"counts"`
	Total     int                   `json:"total"`
	Responses []adminSurveyResponse `json:"responses"`
}

var surveyNow = time.Now

func AdminSurveys(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, 403, map[string]any{"error": "Admin role required"})
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
		if _, e = db.Collection("surveys").Doc(record.ID).Set(r.Context(), record); e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not save survey"})
			return
		}
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
		if _, e := db.Collection("surveys").Doc(id).Delete(r.Context()); e != nil {
			writeJSON(w, 500, map[string]any{"error": "Could not delete survey"})
			return
		}
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
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
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
	analysis, err := loadAdminSurveyAnalysis(r.Context(), db, survey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey responses"})
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		writeAdminSurveyCSV(w, id, analysis)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, analysis)
}

func loadAdminSurveyAnalysis(ctx context.Context, db *firestore.Client, survey surveyRecord) (adminSurveyAnalysis, error) {
	analysis := adminSurveyAnalysis{Survey: survey, Counts: map[string]int{}, Responses: []adminSurveyResponse{}}
	labels := map[string]string{}
	for _, option := range survey.Options {
		labels[option.ID] = option.Label
	}
	client, err := firebaseAuth(ctx)
	if err != nil {
		return analysis, err
	}
	iter := db.Collection("surveys").Doc(survey.ID).Collection("responses").OrderBy("updatedAt", firestore.Desc).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			return analysis, nil
		}
		if err != nil {
			return analysis, err
		}
		var response struct {
			OptionIDs    []string          `firestore:"optionIds"`
			TextByOption map[string]string `firestore:"textByOption"`
			UpdatedAt    time.Time         `firestore:"updatedAt"`
		}
		if err := doc.DataTo(&response); err != nil {
			return analysis, err
		}
		respondent := "Email unavailable"
		if user, err := client.GetUser(ctx, doc.Ref.ID); err == nil {
			if masked := maskedEmail(user.Email); masked != "" {
				respondent = masked
			}
		}
		item := adminSurveyResponse{Respondent: respondent, UpdatedAt: response.UpdatedAt}
		for _, id := range response.OptionIDs {
			label := labels[id]
			if label == "" {
				continue
			}
			analysis.Counts[id]++
			item.Options = append(item.Options, label)
			if text := cleanString(response.TextByOption[id], surveyOtherTextMax); text != "" {
				item.Text = append(item.Text, label+": "+text)
			}
		}
		analysis.Total++
		analysis.Responses = append(analysis.Responses, item)
	}
}

func writeAdminSurveyCSV(w http.ResponseWriter, id string, analysis adminSurveyAnalysis) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=survey-results-"+id+".csv")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"masked_respondent", "submitted_at_utc", "selected_options", "text_responses"})
	for _, response := range analysis.Responses {
		_ = writer.Write([]string{safeCSVCell(response.Respondent), response.UpdatedAt.UTC().Format(time.RFC3339), safeCSVCell(strings.Join(response.Options, " | ")), safeCSVCell(strings.Join(response.Text, " | "))})
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
	r := surveyRecord{ID: cleanString(input.ID, 160), Title: cleanString(input.Title, 120), Description: cleanString(input.Description, surveyDescriptionMax), Question: cleanString(input.Question, 500), CallToAction: cleanString(input.CallToAction, surveyCallToActionMax), Status: cleanString(input.Status, 16), Multiple: input.Multiple, StartsOn: cleanString(input.StartsOn, 10), EndsOn: cleanString(input.EndsOn, 10), ShowResults: input.ShowResults}
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
		o.Label = cleanString(o.Label, surveyOptionLabelMax)
		if o.ID == "" {
			o.ID = fmt.Sprintf("option-%d", i+1)
		}
		if o.Label == "" || seen[o.ID] {
			return r, fmt.Errorf("each option needs a unique ID and label")
		}
		seen[o.ID] = true
		r.Options = append(r.Options, o)
	}
	return r, nil
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
	for _, s := range surveys {
		if !surveyIsPublished(s) {
			continue
		}
		if result, e := loadSurveyResult(r.Context(), db, s, u.UID); e == nil {
			if !memberMayViewSurveyResults(s, result) {
				result.Counts = nil
				result.Total = 0
			}
			out = append(out, result)
		}
	}
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
	if !surveyIsLive(s, surveyNow()) {
		writeJSON(w, 409, map[string]any{"error": "This survey is not currently open"})
		return
	}
	ids, textByOption, e := validateSurveyResponse(s, input)
	if e != nil {
		writeJSON(w, 400, map[string]any{"error": e.Error()})
		return
	}
	if _, e = db.Collection("surveys").Doc(s.ID).Collection("responses").Doc(u.UID).Set(r.Context(), map[string]any{"optionIds": ids, "textByOption": textByOption, "updatedAt": surveyNow().UTC()}); e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not save response"})
		return
	}
	result, e := loadSurveyResult(r.Context(), db, s, u.UID)
	if e != nil {
		writeJSON(w, 500, map[string]any{"error": "Could not load results"})
		return
	}
	if !memberMayViewSurveyResults(s, result) {
		result.Counts = nil
		result.Total = 0
	}
	writeJSON(w, 200, result)
}

func memberMayViewSurveyResults(s surveyRecord, result surveyResult) bool {
	return s.ShowResults && (len(result.MyOptionIDs) > 0 || surveyIsClosed(s, surveyNow()))
}

func surveyIsPublished(s surveyRecord) bool {
	// Surveys created before the status field existed remain visible rather than disappearing.
	return s.Status != "draft"
}
func validateSurveyResponse(s surveyRecord, input surveyResponseInput) ([]string, map[string]string, error) {
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
			return nil, nil, fmt.Errorf("choose valid survey options")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 || (!s.Multiple && len(ids) != 1) {
		return nil, nil, fmt.Errorf("choose a valid number of options")
	}
	for _, id := range ids {
		if !allowed[id].AllowsText {
			continue
		}
		text := cleanString(input.TextByOption[id], surveyOtherTextMax)
		if text == "" {
			return nil, nil, fmt.Errorf("please describe each selected option that requests short text")
		}
		textByOption[id] = text
	}
	return ids, textByOption, nil
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
	r := surveyResult{SurveyRecord: s, Counts: map[string]int{}, CanRespond: surveyIsLive(s, surveyNow())}
	iter := db.Collection("surveys").Doc(s.ID).Collection("responses").Documents(ctx)
	defer iter.Stop()
	for {
		doc, e := iter.Next()
		if e == iterator.Done {
			break
		}
		if e != nil {
			return r, e
		}
		var x struct {
			OptionIDs    []string          `firestore:"optionIds"`
			TextByOption map[string]string `firestore:"textByOption"`
		}
		if doc.DataTo(&x) == nil {
			r.Total++
			for _, id := range x.OptionIDs {
				r.Counts[id]++
			}
			if doc.Ref.ID == uid {
				r.MyOptionIDs = x.OptionIDs
				r.MyTextByOption = x.TextByOption
			}
		}
	}
	return r, nil
}
