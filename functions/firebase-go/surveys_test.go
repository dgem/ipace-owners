package ipace

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateSurveyRequiresDatesAndOptions(t *testing.T) {
	_, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Question: "Choose", StartsOn: "2026-08-30", EndsOn: "2026-08-29", Options: []surveyOption{{Name: "A", Description: "A"}, {Name: "B", Description: "B"}}})
	if err == nil {
		t.Fatal("expected invalid date range to be rejected")
	}
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Question: "Choose", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Name: "A", Description: "A"}, {Name: "B", Description: "B"}}})
	if err != nil || len(record.Options) != 2 || record.Options[0].ID != "option-1" {
		t.Fatalf("valid survey = %#v, %v", record, err)
	}
}

func TestValidateSurveyAllowsMarkdownFieldsAndMultipleTextOptions(t *testing.T) {
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", Description: "**Context**", CallToAction: "Choose all that apply", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Name: "Other A", Description: "**Other** A", AllowsText: true, AllowsPreferred: true, TextPrompt: "How should we address this?"}, {Name: "Other B", Description: "Other B", AllowsText: true}}})
	if err != nil || record.Status != "draft" || record.Description != "**Context**" || record.CallToAction != "Choose all that apply" || record.Options[0].Name != "Other A" || record.Options[0].Description != "**Other** A" || record.Options[0].TextPrompt != "How should we address this?" || record.Options[1].TextPrompt != "Optional detail" || !record.Options[0].AllowsText || !record.Options[1].AllowsText || !record.Options[0].AllowsPreferred || record.Options[1].AllowsPreferred {
		t.Fatalf("valid markdown survey = %#v, %v", record, err)
	}
}

func TestValidateSurveyMigratesLegacyOptionLabel(t *testing.T) {
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Label: "**Replace all modules**\n\nLonger explanation."}, {Label: "Fair buy back"}}})
	if err != nil || record.Options[0].Name != "Replace all modules" || record.Options[0].Description == "" || record.Options[0].Label != "" {
		t.Fatalf("legacy option migration = %#v, %v", record.Options[0], err)
	}
}

func TestValidateSurveyNormalisesOptionNameAndPreservesMarkdownDescription(t *testing.T) {
	record, err := validateSurvey(surveyInput{Title: "Preferred outcomes", StartsOn: "2026-08-29", EndsOn: "2026-08-30", Options: []surveyOption{{Name: " Replace\nall HV modules ", Description: "**Replace every recalled module**\n\n- Warranty support"}, {Name: "Fair buy back", Description: "A transparent age-based value."}}})
	if err != nil || record.Options[0].Name != "Replace all HV modules" || record.Options[0].Description != "**Replace every recalled module**\n\n- Warranty support" {
		t.Fatalf("normalised option = %#v, %v", record.Options[0], err)
	}
}

func TestSurveyResponseValidation(t *testing.T) {
	s := surveyRecord{Multiple: false, Options: []surveyOption{{ID: "warranty", Name: "Warranty"}, {ID: "other", Name: "Other", AllowsText: true}}}
	_, emptyTexts, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}})
	if err != nil || len(emptyTexts) != 0 {
		t.Fatalf("empty optional text response = %#v, %v", emptyTexts, err)
	}
	_, textByOption, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"other"}, TextByOption: map[string]string{"other": "A fair buy-back"}})
	if err != nil || textByOption["other"] != "A fair buy-back" {
		t.Fatalf("other response = %#v, %v", textByOption, err)
	}
	if _, _, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"warranty", "other"}}); err == nil {
		t.Fatal("expected multiple response on single choice survey to be rejected")
	}
}

func TestSurveyResponseStoresOnlyProvidedOptionalText(t *testing.T) {
	s := surveyRecord{Multiple: true, Options: []surveyOption{{ID: "problem", Name: "Problem", AllowsText: true, AllowsPreferred: true}, {ID: "something", Name: "Something", AllowsText: true}}}
	_, texts, _, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"problem", "something"}, TextByOption: map[string]string{"problem": "Battery fault"}})
	if err != nil || len(texts) != 1 || texts["problem"] != "Battery fault" {
		t.Fatalf("optional text response = %#v, %v", texts, err)
	}
	_, texts, preferred, err := validateSurveyResponse(s, surveyResponseInput{OptionIDs: []string{"problem", "something"}, PreferredOptionID: "problem", TextByOption: map[string]string{"problem": "Battery fault", "something": "A fair settlement"}})
	if err != nil || len(texts) != 2 || preferred != "problem" {
		t.Fatalf("multiple text response = %#v, %q, %v", texts, preferred, err)
	}
}

func TestSurveyResponsePreferredOptionMustBeSelectedOnMultipleChoice(t *testing.T) {
	multiple := surveyRecord{Multiple: true, PreferredEligibilityConfigured: true, Options: []surveyOption{{ID: "first", AllowsPreferred: true}, {ID: "second"}}}
	if _, _, _, err := validateSurveyResponse(multiple, surveyResponseInput{OptionIDs: []string{"first"}, PreferredOptionID: "second"}); err == nil {
		t.Fatal("expected an unselected preferred option to be rejected")
	}
	single := surveyRecord{Multiple: false, Options: []surveyOption{{ID: "first"}, {ID: "second"}}}
	if _, _, _, err := validateSurveyResponse(single, surveyResponseInput{OptionIDs: []string{"first"}, PreferredOptionID: "first"}); err == nil {
		t.Fatal("expected a preferred option on a single-choice survey to be rejected")
	}
	if _, _, _, err := validateSurveyResponse(multiple, surveyResponseInput{OptionIDs: []string{"second"}, PreferredOptionID: "second"}); err == nil {
		t.Fatal("expected an option not marked as preferred-eligible to be rejected")
	}
}

func TestSurveyResultsIgnorePreferredChoicesThatAreNoLongerEligible(t *testing.T) {
	s := surveyRecord{Multiple: true, PreferredEligibilityConfigured: true, Options: []surveyOption{{ID: "eligible", AllowsPreferred: true}, {ID: "ineligible"}}}
	if !surveyOptionAllowsPreferred(s, "eligible") || surveyOptionAllowsPreferred(s, "ineligible") || surveyOptionAllowsPreferred(s, "missing") {
		t.Fatal("only explicitly eligible survey options may receive preferred counts")
	}
	if !surveyResponseAllowsPreferred(s, []string{"eligible"}, "eligible") || surveyResponseAllowsPreferred(s, []string{"ineligible"}, "eligible") {
		t.Fatal("preferred result must be both eligible and selected")
	}
	if surveyResponseAllowsPreferred(surveyRecord{Options: []surveyOption{{ID: "eligible", AllowsPreferred: true}}}, []string{"eligible"}, "eligible") {
		t.Fatal("single-choice survey results must not count a preferred option")
	}

	legacy := surveyRecord{Options: []surveyOption{{ID: "previously-allowed"}}}
	if !surveyOptionAllowsPreferred(legacy, "previously-allowed") {
		t.Fatal("existing surveys retain their previous preferred-option behaviour until saved")
	}
}

func TestAggregateSurveyResultsNeverSerialiseFreeText(t *testing.T) {
	result := surveyResult{Counts: map[string]int{"option-1": 3}, PreferredCounts: map[string]int{"option-1": 2}, Total: 3}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "myTextByOption") || strings.Contains(string(encoded), "textByOption") {
		t.Fatalf("aggregate survey result exposed free text: %s", encoded)
	}
}

func TestMemberMayViewResultsOnlyAfterSubmittingResponse(t *testing.T) {
	s := surveyRecord{ShowResults: true, StartsOn: "2026-08-29", EndsOn: "2026-08-30"}
	previousNow := surveyNow
	surveyNow = func() time.Time { return time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC) }
	defer func() { surveyNow = previousNow }()
	if memberMayViewSurveyResults(s, surveyResult{}) {
		t.Fatal("expected results to stay private until the member has responded")
	}
	if !memberMayViewSurveyResults(s, surveyResult{MyOptionIDs: []string{"option-1"}}) {
		t.Fatal("expected a responding member to see enabled results")
	}
	if memberMayViewSurveyResults(surveyRecord{ShowResults: false}, surveyResult{MyOptionIDs: []string{"option-1"}}) {
		t.Fatal("expected disabled results to stay hidden")
	}
	surveyNow = func() time.Time { return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC) }
	if !memberMayViewSurveyResults(s, surveyResult{}) {
		t.Fatal("expected closed published survey results to be visible to every member")
	}
}

func TestSurveyPublicationStatus(t *testing.T) {
	if surveyIsPublished(surveyRecord{Status: "draft"}) {
		t.Fatal("draft survey must not be published")
	}
	if !surveyIsPublished(surveyRecord{Status: "published"}) || !surveyIsPublished(surveyRecord{}) {
		t.Fatal("published and legacy surveys must be visible")
	}
}

func TestAdminSurveyCSVUsesMaskedRespondentsOnly(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeAdminSurveyCSV(recorder, "survey-1", adminSurveyAnalysis{Responses: []adminSurveyResponse{{Respondent: "d***@k***.uk", OptionIDs: []string{"=buy-back"}, PreferredOptionID: "=buy-back", TextResponses: []string{"=buy-back: Fair amount"}}}})
	body := recorder.Body.String()
	if !strings.Contains(body, "selected_option_ids") || !strings.Contains(body, "preferred_option_id") || !strings.Contains(body, "d***@k***.uk") || !strings.Contains(body, "'=buy-back") || !strings.Contains(body, "=buy-back: Fair amount") || strings.Contains(body, "Other:") {
		t.Fatalf("unexpected CSV: %s", body)
	}
	if strings.Contains(body, "uid") || strings.Contains(body, "dan@kanzi.co.uk") {
		t.Fatalf("CSV leaked an identifier: %s", body)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control = %q", got)
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
