package ipace

import (
	"testing"
	"time"
)

func TestValidateSurveyRequiresDatesAndOptions(t *testing.T) {
	_, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Question: "Choose", StartsOn: "2026-08-30", EndsOn: "2026-08-29", Options: []surveyOption{{Label: "A"}, {Label: "B"}}})
	if err == nil {
		t.Fatal("expected invalid date range to be rejected")
	}
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Question: "Choose", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Label: "A"}, {Label: "B"}}})
	if err != nil || len(record.Options) != 2 || record.Options[0].ID != "option-1" {
		t.Fatalf("valid survey = %#v, %v", record, err)
	}
}

func TestSurveyResponseValidation(t *testing.T) {
	s := surveyRecord{Multiple: false, Options: []surveyOption{{ID: "warranty", Label: "Warranty"}, {ID: "other", Label: "Other", AllowsText: true}}}
	if _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}}); err == nil {
		t.Fatal("expected missing Other detail to be rejected")
	}
	_, text, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}, OtherText: "A fair buy-back"})
	if err != nil || text != "A fair buy-back" {
		t.Fatalf("other response = %q, %v", text, err)
	}
	if _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"warranty", "other"}}); err == nil {
		t.Fatal("expected multiple response on single choice survey to be rejected")
	}
}

func TestSurveyIsLiveOnBothWholeDayBoundaries(t *testing.T) {
	s := surveyRecord{StartsOn: "2026-08-29", EndsOn: "2026-08-30"}
	if !surveyIsLive(s, time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)) || !surveyIsLive(s, time.Date(2026, 8, 30, 23, 59, 0, 0, time.UTC)) {
		t.Fatal("expected inclusive survey dates to be live")
	}
	if surveyIsLive(s, time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected survey to close after its end date")
	}
}
