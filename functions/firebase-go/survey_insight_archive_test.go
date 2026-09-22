package ipace

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSurveyInsightArchiveRequiresAdmin(t *testing.T) {
	for _, handler := range []http.HandlerFunc{AdminSurveyInsightArchive, AdminSurveyInsightDeck} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			request := httptest.NewRequest(method, "/api/admin/survey-insight-archive?surveyId=survey", strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			handler(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s returned %d without admin authentication", method, response.Code)
			}
		}
	}
}

func TestSurveyInsightArchiveValidatesCompletedCountsAndQuotes(t *testing.T) {
	report := surveyInsightArchiveReport{
		surveyInsightReport: surveyInsightReport{ID: "survey_test", ExpectedResponses: 2, Items: []surveyInsightItem{{OptionID: "repair", Sentiment: "negative", Themes: []string{"battery"}}}, Quotes: []surveyInsightQuote{{Kind: "bad", OptionID: "repair", Sentiment: "negative", Text: "I needed another repair visit."}}, Overview: "One owner described a battery problem.", Actions: []string{"Explain the repair plan", "Set a date", "Write to members"}},
		Survey:              surveyRecord{ID: "survey_test", Options: []surveyOption{{ID: "repair", Name: "Full HV Replacement"}}},
		Counts:              map[string]int{"repair": 1}, PreferredCounts: map[string]int{"repair": 1}, TextCounts: map[string]int{"repair": 1}, TotalResponses: 2,
	}
	if !validArchivedSurveyInsight(report) {
		t.Fatal("valid completed report rejected")
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var restored surveyInsightArchiveReport
	if json.Unmarshal(encoded, &restored) != nil || restored.ID != report.ID || len(restored.Quotes) != 1 || !validArchivedSurveyInsight(restored) {
		t.Fatal("saved analysis did not round-trip through JSON")
	}
	report.TextCounts["repair"] = 0
	if validArchivedSurveyInsight(report) {
		t.Fatal("comment counts that disagree with classifications accepted")
	}
	report.TextCounts["repair"] = 1
	report.Quotes[0].Text = "Email me at sam@example.org"
	if validArchivedSurveyInsight(report) {
		t.Fatal("unredacted quote accepted in saved report")
	}
}

func TestSurveyInsightDeckSelectsThreeOrAllAvailablePerCategory(t *testing.T) {
	quotes := []surveyInsightQuote{}
	for _, kind := range []string{"good", "bad", "ugly"} {
		for range 4 {
			quotes = append(quotes, surveyInsightQuote{Kind: kind})
		}
	}
	if !validInsightQuoteSelection(quotes, []int{0, 1, 2, 4, 5, 6, 8, 9, 10}) {
		t.Fatal("three quotes per category rejected")
	}
	for _, indexes := range [][]int{{0, 1, 4, 5, 6, 8, 9, 10}, {0, 1, 2, 3, 4, 5, 8, 9, 10}, {0, 1, 1, 4, 5, 6, 8, 9, 10}, {0, 1, 2, 4, 5, 6, 8, 9, 99}} {
		if validInsightQuoteSelection(quotes, indexes) {
			t.Fatalf("invalid quote selection accepted: %v", indexes)
		}
	}
	limited := []surveyInsightQuote{{Kind: "good"}, {Kind: "good"}, {Kind: "bad"}}
	if !validInsightQuoteSelection(limited, []int{0, 1, 2}) || !validInsightQuoteSelection(nil, nil) {
		t.Fatal("fewer than three available quotes should still allow a deck")
	}
	if validInsightQuoteSelection(limited, []int{0, 2}) || validInsightQuoteSelection(limited, []int{0, 1, 2, 2}) {
		t.Fatal("a missing or duplicate available quote was accepted")
	}
}

func TestSurveyInsightDeckRejectsNonPresentationAndZipBomb(t *testing.T) {
	if validInsightPPTX([]byte("PK invalid")) {
		t.Fatal("invalid archive accepted")
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range []string{"[Content_Types].xml", "ppt/presentation.xml"} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte("presentation"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if !validInsightPPTX(buffer.Bytes()) {
		t.Fatal("valid small presentation archive rejected")
	}
}
