package ipace

import (
	"archive/zip"
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

const surveyInsightArchiveLimit = 2 << 20
const surveyInsightDeckLimit = 12 << 20

var surveyInsightArchiveID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,160}$`)

type surveyInsightArchiveReport struct {
	surveyInsightReport
	Survey          surveyRecord   `json:"survey"`
	Counts          map[string]int `json:"counts"`
	PreferredCounts map[string]int `json:"preferredCounts"`
	TextCounts      map[string]int `json:"textCounts"`
	TotalResponses  int            `json:"totalResponses"`
}

type surveyInsightAnalysisMeta struct {
	ID            string                  `json:"id" firestore:"id"`
	SurveyID      string                  `json:"surveyId" firestore:"surveyId"`
	CreatedAt     time.Time               `json:"createdAt" firestore:"createdAt"`
	ResponseCount int                     `json:"responseCount" firestore:"responseCount"`
	CommentCount  int                     `json:"commentCount" firestore:"commentCount"`
	Decks         []surveyInsightDeckMeta `json:"decks,omitempty" firestore:"-"`
}

type surveyInsightDeckMeta struct {
	ID           string    `json:"id" firestore:"id"`
	CreatedAt    time.Time `json:"createdAt" firestore:"createdAt"`
	QuoteIndexes []int     `json:"quoteIndexes" firestore:"quoteIndexes"`
	Overview     string    `json:"overview" firestore:"overview"`
	Actions      []string  `json:"actions" firestore:"actions"`
}

type surveyInsightDeckInput struct {
	SurveyID     string   `json:"surveyId"`
	AnalysisID   string   `json:"analysisId"`
	QuoteIndexes []int    `json:"quoteIndexes"`
	Overview     string   `json:"overview"`
	Actions      []string `json:"actions"`
	PPTX         string   `json:"pptx"`
}

func validArchivedSurveyInsight(report surveyInsightArchiveReport) bool {
	if !surveyInsightArchiveID.MatchString(report.ID) || report.TotalResponses != report.ExpectedResponses || report.TotalResponses < 0 || report.Survey.ID != report.ID || len(report.Survey.Options) == 0 || len(report.Survey.Options) > 20 || len(report.Quotes) > 2000 || len(report.Actions) != 3 || cleanString(report.Overview, 650) != report.Overview || report.Overview == "" {
		return false
	}
	for _, action := range report.Actions {
		if action == "" || cleanString(action, 200) != action {
			return false
		}
	}
	// The summary endpoint normally sends no quote candidates. Archive validation
	// accepts the complete candidate set while reusing its classification checks.
	withoutQuotes := report.surveyInsightReport
	withoutQuotes.Quotes = nil
	if !validSurveyInsightReport(withoutQuotes) {
		return false
	}
	allowed := map[string]bool{}
	itemCounts := map[string]int{}
	for _, item := range report.Items {
		itemCounts[item.OptionID]++
	}
	for _, option := range report.Survey.Options {
		if !surveyInsightArchiveID.MatchString(option.ID) || allowed[option.ID] {
			return false
		}
		allowed[option.ID] = true
		if report.Counts[option.ID] < 0 || report.PreferredCounts[option.ID] < 0 || report.TextCounts[option.ID] < 0 || report.Counts[option.ID] > report.TotalResponses || report.PreferredCounts[option.ID] > report.Counts[option.ID] || report.TextCounts[option.ID] > report.Counts[option.ID] || itemCounts[option.ID] != report.TextCounts[option.ID] {
			return false
		}
	}
	for _, item := range report.Items {
		if !allowed[item.OptionID] {
			return false
		}
	}
	for _, quote := range report.Quotes {
		if !allowed[quote.OptionID] || !validSurveyInsightReport(surveyInsightReport{ID: report.ID, Quotes: []surveyInsightQuote{quote}}) {
			return false
		}
	}
	return true
}

func validInsightQuoteSelection(quotes []surveyInsightQuote, indexes []int) bool {
	counts := map[string]int{"good": 0, "bad": 0, "ugly": 0}
	available := map[string]int{"good": 0, "bad": 0, "ugly": 0}
	for _, quote := range quotes {
		available[quote.Kind]++
	}
	seen := map[int]bool{}
	for _, index := range indexes {
		if index < 0 || index >= len(quotes) || seen[index] {
			return false
		}
		seen[index] = true
		counts[quotes[index].Kind]++
	}
	for _, kind := range []string{"good", "bad", "ugly"} {
		if counts[kind] != min(3, available[kind]) {
			return false
		}
	}
	return true
}

func validInsightPPTX(data []byte) bool {
	if len(data) == 0 || len(data) > surveyInsightDeckLimit {
		return false
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) > 300 {
		return false
	}
	foundTypes, foundPresentation := false, false
	var unpacked uint64
	for _, file := range archive.File {
		unpacked += file.UncompressedSize64
		if unpacked > 50<<20 {
			return false
		}
		foundTypes = foundTypes || file.Name == "[Content_Types].xml"
		foundPresentation = foundPresentation || file.Name == "ppt/presentation.xml"
	}
	return foundTypes && foundPresentation
}

func insightObjectPath(surveyID, analysisID, deckID string) string {
	base := "survey-insights/" + surveyID + "/" + analysisID
	if deckID == "" {
		return base + "/analysis.json"
	}
	return base + "/decks/" + deckID + ".pptx"
}

func writeInsightObject(ctx context.Context, path, contentType string, data []byte) error {
	bucket := os.Getenv("SNAPSHOT_BUCKET")
	if bucket == "" {
		return errors.New("snapshot bucket missing")
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return err
	}
	writer := client.Bucket(bucket).Object(path).NewWriter(ctx)
	writer.ContentType = contentType
	writer.CacheControl = "private, no-store"
	if _, err = writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func readInsightObject(ctx context.Context, path string, limit int64) ([]byte, error) {
	bucket := os.Getenv("SNAPSHOT_BUCKET")
	if bucket == "" {
		return nil, errors.New("snapshot bucket missing")
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return nil, err
	}
	reader, err := client.Bucket(bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, errors.New("archived file unavailable or too large")
	}
	return data, nil
}

func analysisCollection(db *firestore.Client, surveyID string) *firestore.CollectionRef {
	return db.Collection("surveys").Doc(surveyID).Collection("insightAnalyses")
}

// AdminSurveyInsightArchive lists, saves and restores private, completed AI reports.
func AdminSurveyInsightArchive(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, surveyInsightArchiveLimit)
		var report surveyInsightArchiveReport
		if decodeJSON(r, &report) != nil || !validArchivedSurveyInsight(report) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid completed analysis"})
			return
		}
		snapshot, surveyErr := db.Collection("surveys").Doc(report.ID).Get(r.Context())
		if surveyErr != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey not found"})
			return
		}
		var survey surveyRecord
		if snapshot.DataTo(&survey) != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey"})
			return
		}
		totals, totalsErr := loadSurveyResult(r.Context(), db, survey, "")
		if totalsErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey totals"})
			return
		}
		if totals.Total != report.TotalResponses {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Survey responses changed; re-run AI on latest responses"})
			return
		}
		for _, option := range survey.Options {
			if totals.Counts[option.ID] != report.Counts[option.ID] || totals.PreferredCounts[option.ID] != report.PreferredCounts[option.ID] || totals.TextCounts[option.ID] != report.TextCounts[option.ID] {
				writeJSON(w, http.StatusConflict, map[string]any{"error": "Survey choices changed; re-run AI on latest responses"})
				return
			}
		}
		id := submissionID("analysis")
		data, _ := json.Marshal(report)
		if err = writeInsightObject(r.Context(), insightObjectPath(report.ID, id, ""), "application/json", data); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not save analysis"})
			return
		}
		meta := surveyInsightAnalysisMeta{ID: id, SurveyID: report.ID, CreatedAt: time.Now().UTC(), ResponseCount: report.TotalResponses, CommentCount: len(report.Items)}
		if _, err = analysisCollection(db, report.ID).Doc(id).Set(r.Context(), meta); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not list saved analysis"})
			return
		}
		writeJSON(w, http.StatusCreated, meta)
		return
	}
	surveyID, analysisID := r.URL.Query().Get("surveyId"), r.URL.Query().Get("analysisId")
	if !surveyInsightArchiveID.MatchString(surveyID) || analysisID != "" && !surveyInsightArchiveID.MatchString(analysisID) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid saved analysis request"})
		return
	}
	if analysisID != "" {
		if _, err = analysisCollection(db, surveyID).Doc(analysisID).Get(r.Context()); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Saved analysis not found"})
			return
		}
		data, readErr := readInsightObject(r.Context(), insightObjectPath(surveyID, analysisID, ""), surveyInsightArchiveLimit)
		if readErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load saved analysis"})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
		return
	}
	analyses := []surveyInsightAnalysisMeta{}
	iter := analysisCollection(db, surveyID).OrderBy("createdAt", firestore.Desc).Documents(r.Context())
	defer iter.Stop()
	for {
		doc, nextErr := iter.Next()
		if nextErr == iterator.Done {
			break
		}
		if nextErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not list saved analyses"})
			return
		}
		var meta surveyInsightAnalysisMeta
		if doc.DataTo(&meta) != nil {
			continue
		}
		decks := analysisCollection(db, surveyID).Doc(meta.ID).Collection("decks").OrderBy("createdAt", firestore.Desc).Documents(r.Context())
		for {
			deck, deckErr := decks.Next()
			if deckErr == iterator.Done {
				break
			}
			if deckErr != nil {
				decks.Stop()
				writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not list saved decks"})
				return
			}
			var deckMeta surveyInsightDeckMeta
			if deck.DataTo(&deckMeta) == nil {
				meta.Decks = append(meta.Decks, deckMeta)
			}
		}
		decks.Stop()
		analyses = append(analyses, meta)
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyses": analyses})
}

// AdminSurveyInsightDeck saves or downloads one immutable deck version.
func AdminSurveyInsightDeck(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := campaignAuthorize(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}
	if r.Method == http.MethodGet {
		surveyID, analysisID, deckID := r.URL.Query().Get("surveyId"), r.URL.Query().Get("analysisId"), r.URL.Query().Get("deckId")
		if !surveyInsightArchiveID.MatchString(surveyID) || !surveyInsightArchiveID.MatchString(analysisID) || !surveyInsightArchiveID.MatchString(deckID) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid deck request"})
			return
		}
		if _, err = analysisCollection(db, surveyID).Doc(analysisID).Collection("decks").Doc(deckID).Get(r.Context()); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Saved deck not found"})
			return
		}
		data, readErr := readInsightObject(r.Context(), insightObjectPath(surveyID, analysisID, deckID), surveyInsightDeckLimit)
		if readErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load saved deck"})
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ipace-survey-%s.pptx"`, deckID))
		_, _ = w.Write(data)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, surveyInsightDeckLimit*2)
	var input surveyInsightDeckInput
	if decodeJSON(r, &input) != nil || !surveyInsightArchiveID.MatchString(input.SurveyID) || !surveyInsightArchiveID.MatchString(input.AnalysisID) || len(input.Actions) != 3 || input.Overview == "" || cleanString(input.Overview, 650) != input.Overview {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid deck version"})
		return
	}
	for _, action := range input.Actions {
		if action == "" || cleanString(action, 200) != action {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid deck action"})
			return
		}
	}
	if _, err = analysisCollection(db, input.SurveyID).Doc(input.AnalysisID).Get(r.Context()); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Saved analysis not found"})
		return
	}
	reportBytes, err := readInsightObject(r.Context(), insightObjectPath(input.SurveyID, input.AnalysisID, ""), surveyInsightArchiveLimit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load saved analysis"})
		return
	}
	var report surveyInsightArchiveReport
	if json.Unmarshal(reportBytes, &report) != nil || !validInsightQuoteSelection(report.Quotes, input.QuoteIndexes) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Choose up to three quotes per section, or all available quotes when fewer than three exist"})
		return
	}
	data, err := base64.StdEncoding.DecodeString(input.PPTX)
	if err != nil || !validInsightPPTX(data) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid PowerPoint file"})
		return
	}
	deckID := submissionID("deck")
	if err = writeInsightObject(r.Context(), insightObjectPath(input.SurveyID, input.AnalysisID, deckID), "application/vnd.openxmlformats-officedocument.presentationml.presentation", data); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not save PowerPoint"})
		return
	}
	meta := surveyInsightDeckMeta{ID: deckID, CreatedAt: time.Now().UTC(), QuoteIndexes: input.QuoteIndexes, Overview: input.Overview, Actions: input.Actions}
	if _, err = analysisCollection(db, input.SurveyID).Doc(input.AnalysisID).Collection("decks").Doc(deckID).Set(r.Context(), meta); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not list saved PowerPoint"})
		return
	}
	writeJSON(w, http.StatusCreated, meta)
}
