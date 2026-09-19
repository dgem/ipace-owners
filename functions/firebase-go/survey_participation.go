package ipace

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var surveyParticipationCount = loadSurveyParticipationCount

func loadSurveyParticipationCount(ctx context.Context) (int64, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return 0, err
	}
	doc := db.Collection("surveys").Doc(septemberSurveyID)
	snapshot, err := doc.Get(ctx)
	if err != nil {
		return 0, err
	}
	var survey surveyRecord
	if err := snapshot.DataTo(&survey); err != nil {
		return 0, err
	}
	if !surveyIsPublished(survey) {
		return 0, fmt.Errorf("survey is not published")
	}
	result, err := doc.Collection("responses").NewAggregationQuery().WithCount("responses").Get(ctx)
	if err != nil {
		return 0, err
	}
	count, ok := result.Data()["responses"].(int64)
	if !ok || count < 0 {
		return 0, fmt.Errorf("invalid response count")
	}
	return count, nil
}

// SurveyParticipation exposes only the participation total of the named public
// September campaign. No query can select another survey or reveal vote choices.
func SurveyParticipation(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	count, err := surveyParticipationCount(r.Context())
	if err != nil {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Participation count is temporarily unavailable"})
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, map[string]any{"responses": count, "generatedAt": time.Now().UTC().Format(time.RFC3339)})
}
