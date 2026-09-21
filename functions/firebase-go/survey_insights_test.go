package ipace

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSurveyInsightRedactionKeepsIssueTextAndRemovesCommonIdentifiers(t *testing.T) {
	input := "AC failed again. Contact sam@example.org or 07700 900123 about AB12 CDE at SW1A 1AA; VIN SAJAC12AB3C456789."
	got := redactSurveyInsightText(input)
	for _, token := range []string{"AC failed again", "[email]", "[phone]", "[registration]", "[postcode]", "[VIN]"} {
		if !strings.Contains(got, token) {
			t.Errorf("missing %q in redacted comment %q", token, got)
		}
	}
	for _, secret := range []string{"sam@example.org", "07700 900123", "AB12 CDE", "SW1A 1AA", "SAJAC12AB3C456789"} {
		if strings.Contains(got, secret) {
			t.Errorf("identifier was retained: %q", secret)
		}
	}
}

func TestSurveyInsightClassificationMatchesOriginalComments(t *testing.T) {
	comments := []surveyInsightComment{
		{ID: 0, OptionID: "repair", Text: "The car is lovely to drive when it works."},
		{ID: 1, OptionID: "buyback", Text: "Battery failed twice and I waited months."},
	}
	output := surveyInsightModelOutput{Assessments: []surveyInsightAssessment{
		{ID: 1, Sentiment: "negative", Themes: []string{"battery", "parts"}, QuoteKind: "ugly"},
		{ID: 0, Sentiment: "positive", Themes: []string{"other"}, QuoteKind: "good"},
	}}
	items, quotes, err := validateSurveyInsightOutput(comments, output)
	if err != nil || len(items) != 2 || items[0].OptionID != "repair" || items[0].Sentiment != "positive" || items[1].OptionID != "buyback" || items[1].Sentiment != "negative" {
		t.Fatalf("assessments were not joined to comment source: %#v, %v", items, err)
	}
	if len(quotes) != 2 || quotes[0].Text != comments[0].Text || quotes[1].Text != comments[1].Text {
		t.Fatalf("quote candidates must be exact redacted source text: %#v", quotes)
	}
	for _, bad := range []surveyInsightModelOutput{
		{Assessments: output.Assessments[:1]},
		{Assessments: []surveyInsightAssessment{output.Assessments[0], output.Assessments[0]}},
		{Assessments: []surveyInsightAssessment{output.Assessments[0], {ID: 0, Sentiment: "negative", Themes: []string{"invented"}}}},
	} {
		if _, _, err := validateSurveyInsightOutput(comments, bad); err == nil {
			t.Fatal("incomplete, duplicate or unknown model output was accepted")
		}
	}
}

func TestSurveyInsightOffersThreeExactQuotesPerCategory(t *testing.T) {
	comments := make([]surveyInsightComment, 4)
	assessments := make([]surveyInsightAssessment, 4)
	for index := range comments {
		comments[index] = surveyInsightComment{ID: index, OptionID: "repair", Text: fmt.Sprintf("Battery repair visit %d needed a further appointment.", index)}
		assessments[index] = surveyInsightAssessment{ID: index, Sentiment: "negative", Themes: []string{"battery"}, QuoteKind: "bad"}
	}
	_, quotes, err := validateSurveyInsightOutput(comments, surveyInsightModelOutput{Assessments: assessments})
	if err != nil || len(quotes) != 3 || quotes[2].Text != comments[2].Text {
		t.Fatalf("expected three source-exact quote candidates, got %#v, %v", quotes, err)
	}
}

func TestSurveyInsightCountsAreCommentEntriesNotSurveyVotes(t *testing.T) {
	items := []surveyInsightItem{
		{OptionID: "repair", Sentiment: "negative", Themes: []string{"battery", "service"}},
		{OptionID: "repair", Sentiment: "mixed", Themes: []string{"battery", "battery"}},
		{OptionID: "buyback", Sentiment: "positive", Themes: []string{"value"}},
	}
	stats := surveyInsightSummaryInput(items)
	if stats.Sentiment["repair"]["negative"] != 1 || stats.Sentiment["repair"]["mixed"] != 1 || stats.Themes[0].Name != "battery" || stats.Themes[0].Count != 2 {
		t.Fatalf("unexpected sentiment/theme denominator: %#v", stats)
	}
}

func TestSurveyInsightRoutesRequireAdmin(t *testing.T) {
	for _, handler := range []http.HandlerFunc{AdminSurveyInsights, AdminSurveyInsightSummary} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/admin/survey-insights", strings.NewReader(`{"id":"survey"}`))
		handler(recorder, request)
		if recorder.Code != http.StatusUnauthorized && recorder.Code != http.StatusForbidden {
			t.Fatalf("unauthenticated insight request returned %d", recorder.Code)
		}
	}
}

func TestInvalidQuoteCannotEnterSummary(t *testing.T) {
	base := surveyInsightReport{ID: "survey", ExpectedResponses: 2, Items: []surveyInsightItem{{OptionID: "repair", Sentiment: "negative", Themes: []string{"battery"}}}}
	if !validSurveyInsightReport(base) {
		t.Fatal("valid aggregate rejected")
	}
	base.Quotes = []surveyInsightQuote{{Kind: "ugly", Text: "Contact me at sam@example.org"}}
	if validSurveyInsightReport(base) {
		t.Fatal("unredacted quote accepted")
	}
}

func TestServiceLocationsRequireAnalysisConsentAndHideSmallAreas(t *testing.T) {
	joins := []joinRecord{
		{UserEmailHash: "yes", Consents: consentRecord{AnonymisedAnalysis: true}},
		{UserEmailHash: "no", Consents: consentRecord{AnonymisedAnalysis: false}},
	}
	vehicles := []vehicleRecord{{ID: "v1", UserEmailHash: "yes"}, {ID: "v2", UserEmailHash: "no"}}
	services := []serviceEventRecord{}
	for i := 0; i < 5; i++ {
		services = append(services, serviceEventRecord{VehicleID: "v1", ServiceProviderPostcode: "SW1A 1AA"})
	}
	services = append(services,
		serviceEventRecord{VehicleID: "v1", ServiceProviderPostcode: "OX1 2AB"},
		serviceEventRecord{VehicleID: "v1"},
		serviceEventRecord{VehicleID: "v2", ServiceProviderPostcode: "SW1A 1AA"},
	)
	got := computeConsentServiceLocations(joins, vehicles, services)
	if got.Known != 6 || got.Unknown != 1 || len(got.Areas) != 2 || got.Areas[0] != (serviceLocationArea{Area: "SW", Count: 5}) || got.Areas[1] != (serviceLocationArea{Area: "Other areas", Count: 1}) {
		t.Fatalf("location privacy/counts = %#v", got)
	}
}

func TestConsentedGrowthTimelineUsesFirstDatedJoinPerMember(t *testing.T) {
	joins := []joinRecord{
		{Contact: contactRecord{Email: "owner+new@example.org"}, Consents: consentRecord{Contact: true}, CreatedAt: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)},
		{Contact: contactRecord{Email: "owner@example.org"}, Consents: consentRecord{Contact: true}, CreatedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
		{Contact: contactRecord{Email: "other@example.org"}, Consents: consentRecord{Contact: false}, CreatedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
	}
	got := computeConsentedJoinTimeline(joins)
	if len(got) != 1 || got[0] != (timelineBucket{Label: "2026-09-10", Count: 1}) {
		t.Fatalf("consented member growth = %#v", got)
	}
}

func TestMemberCountryBreakdownUsesConsentAndCombinesSmallGroups(t *testing.T) {
	joins := []joinRecord{}
	for i := 0; i < 5; i++ {
		joins = append(joins, joinRecord{Contact: contactRecord{Email: fmt.Sprintf("owner%d@example.org", i), Country: "United Kingdom"}, Consents: consentRecord{AnonymisedAnalysis: true}})
	}
	joins = append(joins,
		joinRecord{Contact: contactRecord{Email: "fr@example.org", Country: "France"}, Consents: consentRecord{AnonymisedAnalysis: true}},
		joinRecord{Contact: contactRecord{Email: "no@example.org", Country: "France"}, Consents: consentRecord{AnonymisedAnalysis: false}},
	)
	got := computeConsentedMemberCountries(joins)
	if len(got) != 2 || got[0] != (demographicBucket{Label: "United Kingdom", Count: 5}) || got[1] != (demographicBucket{Label: "Other / unknown", Count: 1}) {
		t.Fatalf("country breakdown = %#v", got)
	}
}
