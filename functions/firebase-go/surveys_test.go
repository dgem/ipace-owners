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

func TestValidateSurveyAllowsMarkdownFieldsAndMultipleTextOptions(t *testing.T) {
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Description: "**Context**", CallToAction: "Choose all that apply", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Label: "Other A", AllowsText: true}, {Label: "Other B", AllowsText: true}}})
	if err != nil || record.Description != "**Context**" || record.CallToAction != "Choose all that apply" || !record.Options[0].AllowsText || !record.Options[1].AllowsText {
		t.Fatalf("valid markdown survey = %#v, %v", record, err)
	}
}

func TestSurveyResponseValidation(t *testing.T) {
	s := surveyRecord{Multiple: false, Options: []surveyOption{{ID: "warranty", Label: "Warranty"}, {ID: "other", Label: "Other", AllowsText: true}}}
	if _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}}); err == nil {
		t.Fatal("expected missing Other detail to be rejected")
	}
	_, textByOption, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}, TextByOption: map[string]string{"other": "A fair buy-back"}})
	if err != nil || textByOption["other"] != "A fair buy-back" {
		t.Fatalf("other response = %#v, %v", textByOption, err)
	}
	if _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"warranty", "other"}}); err == nil {
		t.Fatal("expected multiple response on single choice survey to be rejected")
	}
}

func TestSurveyResponseRequiresTextForEverySelectedTextOption(t *testing.T) {
	s := surveyRecord{Multiple: true, Options: []surveyOption{{ID: "problem", Label: "Problem", AllowsText: true}, {ID: "something", Label: "Something", AllowsText: true}}}
	if _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"problem", "something"}, TextByOption: map[string]string{"problem": "Battery fault"}}); err == nil {
		t.Fatal("expected text to be required for every selected text option")
	}
	_, texts, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"problem", "something"}, TextByOption: map[string]string{"problem": "Battery fault", "something": "A fair settlement"}})
	if err != nil || len(texts) != 2 {
		t.Fatalf("multiple text response = %#v, %v", texts, err)
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
