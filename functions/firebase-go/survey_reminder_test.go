package ipace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func reminderTestState() surveyReminderState {
	return surveyReminderState{
		Survey:           surveyRecord{Status: "published", StartsOn: "2026-09-04", EndsOn: "2026-09-23"},
		Responses:        540,
		RespondentEmails: map[string]bool{"answered@example.com": true},
	}
}

func setupReminderTest(t *testing.T) {
	t.Helper()
	oldState, oldStats, oldAudience, oldNow := marketingSurveyReminderState, marketingMessageStats, marketingMessageAudience, marketingReminderNow
	t.Cleanup(func() {
		marketingSurveyReminderState, marketingMessageStats, marketingMessageAudience, marketingReminderNow = oldState, oldStats, oldAudience, oldNow
	})
	marketingSurveyReminderState = func(context.Context) (surveyReminderState, error) { return reminderTestState(), nil }
	marketingMessageStats = func(context.Context) (publicStatsSnapshot, error) {
		return publicStatsSnapshot{JoinedOwners: 1477, VehiclesRegistered: 721, SOHReadings: 130, ServiceEventsLogged: 182}, nil
	}
	marketingMessageAudience = func(context.Context) ([]campaignRecipient, error) {
		return []campaignRecipient{{Email: "ANSWERED+tag@example.com"}, {Name: "New Owner", Email: "new@example.com"}}, nil
	}
	marketingReminderNow = func() time.Time { return time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC) }
}

func TestSurveyReminderPreviewSubstitutesCountsAndTargetsNonrespondents(t *testing.T) {
	setupReminderTest(t)
	preview, err := previewMarketingMessage(context.Background(), marketingMessageRequest{TemplateID: surveyReminderTemplateID})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Eligible != 1 || preview.Confirmation != "SEND 1" {
		t.Fatalf("wrong audience: %+v", preview)
	}
	for _, value := range []string{"540", "1477", "721", "130", "182", "Hi New", "racing-laurel-email.png", "september-survey-reminder-2026-hero.jpg", "border-radius:999px", "facebook.com/sharer", "wa.me/", "twitter.com/intent", "linkedin.com/sharing", "24 September", "23 September", "Unsubscribe"} {
		if !strings.Contains(preview.HTML, value) {
			t.Errorf("missing %q", value)
		}
	}
	if strings.Contains(preview.HTML, "{{") || strings.Contains(preview.HTML, "[[") {
		t.Fatal("unresolved substitutions in HTML")
	}
	if strings.Contains(preview.Text, "[[") || !strings.Contains(preview.Text, "540 survey responses") {
		t.Fatal("invalid plain-text counters")
	}
	if !strings.Contains(preview.Notice, "not submitted") {
		t.Fatal("targeting not explained")
	}
}

func TestSurveyReminderAudienceFailsClosedAndRefreshes(t *testing.T) {
	setupReminderTest(t)
	input := marketingMessageRequest{TemplateID: surveyReminderTemplateID}
	audience, err := marketingMessageAudienceFor(context.Background(), input)
	if err != nil || len(audience) != 1 {
		t.Fatalf("%v %v", audience, err)
	}
	marketingSurveyReminderState = func(context.Context) (surveyReminderState, error) {
		state := reminderTestState()
		state.RespondentEmails["new@example.com"] = true
		return state, nil
	}
	audience, err = marketingMessageAudienceFor(context.Background(), input)
	if err != nil || len(audience) != 0 {
		t.Fatal("new respondents were not excluded")
	}
	marketingSurveyReminderState = func(context.Context) (surveyReminderState, error) {
		return surveyReminderState{}, errors.New("private failure")
	}
	if _, err := marketingMessageAudienceFor(context.Background(), input); err == nil {
		t.Fatal("failed lookup widened audience")
	}
	// An unavailable survey must not break unrelated campaigns.
	if audience, err = marketingMessageAudienceFor(context.Background(), marketingMessageRequest{}); err != nil || len(audience) != 2 {
		t.Fatal("unrelated campaign changed")
	}
}

func TestSurveyReminderWindowAndDraftProtection(t *testing.T) {
	state := reminderTestState()
	for _, tc := range []struct {
		day  int
		want bool
	}{{17, false}, {18, true}, {19, true}, {23, true}, {24, false}, {25, false}} {
		if got := surveyReminderIsTimely(state, time.Date(2026, 9, tc.day, 12, 0, 0, 0, time.UTC)); got != tc.want {
			t.Errorf("day %d = %v", tc.day, got)
		}
	}
	state.Survey.Status = "draft"
	if surveyReminderIsTimely(state, time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("draft reminder allowed")
	}
}

func TestSurveyReminderSendRejectsChangedAudienceBeforeDelivery(t *testing.T) {
	setupReminderTest(t)
	old := campaignAuthorize
	t.Cleanup(func() { campaignAuthorize = old })
	campaignAuthorize = func(context.Context, *http.Request) error { return nil }
	r := httptest.NewRequest("POST", "/api/admin/marketing-message-send", strings.NewReader(`{"templateId":" survey-reminder-september-2026 ","expectedEligible":2,"confirmation":"SEND 2"}`))
	w := httptest.NewRecorder()
	AdminMarketingMessageSend(w, r)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "SEND 1") {
		t.Fatalf("%d: %s", w.Code, w.Body)
	}
}

func TestSurveyParticipationOnlyPublishesCountAndTimestamp(t *testing.T) {
	old := surveyParticipationCount
	t.Cleanup(func() { surveyParticipationCount = old })
	surveyParticipationCount = func(context.Context) (int64, error) { return 540, nil }
	w := httptest.NewRecorder()
	Api(w, httptest.NewRequest("GET", "/api/survey-participation?id=private-survey", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "\"responses\":540") || strings.Contains(w.Body.String(), "private-survey") {
		t.Fatalf("%d: %s", w.Code, w.Body)
	}
	if w.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatal("missing cache policy")
	}
	w = httptest.NewRecorder()
	Api(w, httptest.NewRequest("POST", "/api/survey-participation", nil))
	if w.Code != 405 {
		t.Fatal("wrong method accepted")
	}
	surveyParticipationCount = func(context.Context) (int64, error) { return 0, errors.New("private datastore detail") }
	w = httptest.NewRecorder()
	SurveyParticipation(w, httptest.NewRequest("GET", "/api/survey-participation", nil))
	if w.Code != 503 || strings.Contains(w.Body.String(), "datastore") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("unsafe failure response")
	}
}

// Opt-in local visual fixture; never sends mail or uses real member information.
func TestRenderSurveyReminderEmailFixture(t *testing.T) {
	target := os.Getenv("SURVEY_REMINDER_PREVIEW")
	if target == "" {
		t.Skip("set SURVEY_REMINDER_PREVIEW for local visual QA")
	}
	setupReminderTest(t)
	preview, err := previewMarketingMessage(context.Background(), marketingMessageRequest{TemplateID: surveyReminderTemplateID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(preview.HTML), 0600); err != nil {
		t.Fatal(err)
	}
}
