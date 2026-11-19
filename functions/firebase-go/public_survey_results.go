package ipace

import "net/http"

// PublicSurveyResults exposes aggregate counts for the explicitly published,
// closed next-steps survey. It never returns member identities or free text.
func PublicSurveyResults(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if cleanString(r.URL.Query().Get("id"), 160) != nextStepsSurveyID {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey results not found"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey results"})
		return
	}
	snapshot, err := db.Collection("surveys").Doc(nextStepsSurveyID).Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey results not found"})
		return
	}
	var survey surveyRecord
	if snapshot.DataTo(&survey) != nil || !surveyIsPublished(survey) || !survey.ShowResults || !surveyIsClosed(survey, surveyNow()) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Survey results not found"})
		return
	}
	result, err := loadSurveyResult(r.Context(), db, survey, "")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load survey results"})
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, publicSurveyResult{ID: survey.ID, Title: survey.Title, Question: survey.Question, Options: publicSurveyOptions(survey.Options), Counts: result.Counts, PreferredCounts: result.PreferredCounts, Total: result.Total})
}

func publicSurveyOptions(options []surveyOption) []surveyOption {
	public := make([]surveyOption, 0, len(options))
	for _, option := range options {
		public = append(public, surveyOption{ID: option.ID, Name: option.Name, Description: option.Description, AllowsPreferred: option.AllowsPreferred})
	}
	return public
}
