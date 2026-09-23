package ipace

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

const surveyReminderTemplateID = "survey-reminder-september-2026"
const surveyClosingReminderTemplateID = "survey-closing-reminder-september-2026"
const septemberSurveyID = "survey_38447815d17b0e954a4edbca1b9600c9"

var errSeptemberSurveyMissing = errors.New("The September survey is not available in this environment")

type surveyReminderState struct {
	Survey           surveyRecord
	Responses        int
	RespondentEmails map[string]bool
}

var marketingSurveyReminderState = loadSurveyReminderState
var marketingReminderSurvey = loadReminderSurvey
var marketingReminderNow = time.Now

func isSurveyReminderTemplate(id string) bool {
	return id == surveyReminderTemplateID || id == surveyClosingReminderTemplateID
}

func loadReminderSurvey(ctx context.Context) (surveyRecord, error) {
	var survey surveyRecord
	db, err := firestoreClient(ctx)
	if err != nil {
		return survey, err
	}
	doc, err := db.Collection("surveys").Doc(septemberSurveyID).Get(ctx)
	if err != nil {
		return survey, err
	}
	err = doc.DataTo(&survey)
	return survey, err
}

// Load document IDs only; never read or expose members' answers for targeting.
func loadSurveyReminderState(ctx context.Context) (surveyReminderState, error) {
	state := surveyReminderState{RespondentEmails: map[string]bool{}}
	var err error
	state.Survey, err = marketingReminderSurvey(ctx)
	if err != nil {
		if isFirestoreNotFound(err) {
			return state, errSeptemberSurveyMissing
		}
		return state, err
	}
	if !surveyIsPublished(state.Survey) {
		return state, fmt.Errorf("September survey is not published")
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return state, err
	}
	iter := db.Collection("surveys").Doc(septemberSurveyID).Collection("responses").Select().Documents(ctx)
	defer iter.Stop()
	ids := []auth.UserIdentifier{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return state, err
		}
		ids = append(ids, auth.UIDIdentifier{UID: doc.Ref.ID})
	}
	state.Responses = len(ids)
	if len(ids) == 0 {
		return state, nil
	}
	client, err := firebaseAuth(ctx)
	if err != nil {
		return state, err
	}
	for start := 0; start < len(ids); start += 100 {
		result, err := client.GetUsers(ctx, ids[start:min(start+100, len(ids))])
		if err != nil {
			return state, err
		}
		for _, user := range result.Users {
			if key := canonicalCampaignEmail(user.Email); key != "" {
				state.RespondentEmails[key] = true
			}
		}
	}
	return state, nil
}

func surveyReminderIsTimely(state surveyReminderState, now time.Time) bool {
	// The "less than a week" copy is valid only in the final lead-up to this meeting.
	return !now.Before(time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)) &&
		now.Before(time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)) &&
		surveyIsPublished(state.Survey) && surveyIsLive(state.Survey, now)
}

func surveyReminderIsTimelyForTemplate(state surveyReminderState, templateID string, now time.Time) bool {
	if !surveyReminderIsTimely(state, now) {
		return false
	}
	return templateID != surveyClosingReminderTemplateID || !now.Before(time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC))
}

func marketingMessageAudienceFor(ctx context.Context, input marketingMessageRequest) ([]campaignRecipient, error) {
	audience, err := marketingMessageAudience(ctx)
	if err != nil || !isSurveyReminderTemplate(input.TemplateID) {
		return audience, err
	}
	state, err := marketingSurveyReminderState(ctx)
	if err != nil {
		return nil, fmt.Errorf("Could not verify survey responses; no reminder audience is available")
	}
	if !surveyReminderIsTimelyForTemplate(state, input.TemplateID, marketingReminderNow()) {
		return nil, fmt.Errorf("The survey reminder is only available before the survey closes and the 24 September meeting")
	}
	return surveyReminderNonrespondents(audience, state.RespondentEmails), nil
}

func surveyReminderNonrespondents(audience []campaignRecipient, respondents map[string]bool) []campaignRecipient {
	result := []campaignRecipient{}
	for _, person := range audience {
		if !respondents[canonicalCampaignEmail(person.Email)] {
			result = append(result, person)
		}
	}
	return result
}

func resolvedSurveyReminder(ctx context.Context, input marketingMessageRequest) (marketingMessageRequest, error) {
	state, err := marketingSurveyReminderState(ctx)
	if err != nil {
		if errors.Is(err, errSeptemberSurveyMissing) {
			return input, err
		}
		return input, fmt.Errorf("Could not load the September survey response count")
	}
	if !surveyReminderIsTimelyForTemplate(state, input.TemplateID, marketingReminderNow()) {
		return input, fmt.Errorf("The survey reminder is no longer available outside the final week before the meeting")
	}
	stats, err := marketingMessageStats(ctx)
	if err != nil {
		return input, err
	}
	source, ok := marketingMessageTemplateSource(input.TemplateID)
	if !ok {
		return input, fmt.Errorf("Survey reminder template is unavailable")
	}
	input.Name, input.Subject = source.Name, source.Subject
	input.Markdown = strings.ReplaceAll(marketingTemplateMarkdown(source.Markdown, stats), "{{surveyResponses}}", strconv.Itoa(state.Responses))
	return input, nil
}

// A missing environment-local survey permits design review only. This path is
// never used by sending or by audience calculation, and reads no member data.
func surveyReminderLayoutPreview(input marketingMessageRequest) (marketingMessagePreview, error) {
	source, ok := marketingMessageTemplateSource(input.TemplateID)
	if !ok {
		return marketingMessagePreview{}, fmt.Errorf("Survey reminder template is unavailable")
	}
	stats := publicStatsSnapshot{JoinedOwners: 1477, VehiclesRegistered: 721, SOHReadings: 130, ServiceEventsLogged: 182}
	responses, snapshotDate := "540", "19 September 2026"
	if input.TemplateID == surveyClosingReminderTemplateID {
		stats = publicStatsSnapshot{JoinedOwners: 1550, VehiclesRegistered: 786, SOHReadings: 144, ServiceEventsLogged: 203}
		responses, snapshotDate = "738", "23 September 2026"
	}
	markdown := strings.ReplaceAll(marketingTemplateMarkdown(source.Markdown, stats), "{{surveyResponses}}", responses)
	notice := "Layout preview only: the September survey is not available in this environment. All figures are the dated " + snapshotDate + " snapshot, not live counts. No audience has been calculated and sending is disabled."
	markdown = "**" + notice + "**\n\n" + renderMarketingMessageMarkdown(markdown, campaignRecipient{Name: "Preview Member"})
	const unsubscribeURL = "https://ipace-owners.org/api/email-unsubscribe?campaign=preview&token=preview-token"
	return marketingMessagePreview{
		CampaignID: marketingMessageCampaignID(input), PreviewOnly: true, Subject: source.Subject,
		HTML: marketingMessageHTML(markdown, input.TemplateID, unsubscribeURL),
		Text: marketingMessageText(markdown, unsubscribeURL), Notice: notice,
	}, nil
}
