package ipace

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

const surveyReminderTemplateID = "survey-reminder-september-2026"
const septemberSurveyID = "survey_38447815d17b0e954a4edbca1b9600c9"

type surveyReminderState struct {
	Survey           surveyRecord
	Responses        int
	RespondentEmails map[string]bool
}

var marketingSurveyReminderState = loadSurveyReminderState
var marketingReminderNow = time.Now

// Load document IDs only; never read or expose members' answers for targeting.
func loadSurveyReminderState(ctx context.Context) (surveyReminderState, error) {
	state := surveyReminderState{RespondentEmails: map[string]bool{}}
	db, err := firestoreClient(ctx)
	if err != nil {
		return state, err
	}
	doc, err := db.Collection("surveys").Doc(septemberSurveyID).Get(ctx)
	if err != nil {
		return state, err
	}
	if err := doc.DataTo(&state.Survey); err != nil {
		return state, err
	}
	if !surveyIsPublished(state.Survey) {
		return state, fmt.Errorf("September survey is not published")
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

func marketingMessageAudienceFor(ctx context.Context, input marketingMessageRequest) ([]campaignRecipient, error) {
	audience, err := marketingMessageAudience(ctx)
	if err != nil || input.TemplateID != surveyReminderTemplateID {
		return audience, err
	}
	state, err := marketingSurveyReminderState(ctx)
	if err != nil {
		return nil, fmt.Errorf("Could not verify survey responses; no reminder audience is available")
	}
	if !surveyReminderIsTimely(state, marketingReminderNow()) {
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
		return input, fmt.Errorf("Could not load the September survey response count")
	}
	if !surveyReminderIsTimely(state, marketingReminderNow()) {
		return input, fmt.Errorf("The survey reminder is no longer available outside the final week before the meeting")
	}
	stats, err := marketingMessageStats(ctx)
	if err != nil {
		return input, err
	}
	source, ok := marketingMessageTemplateSource(surveyReminderTemplateID)
	if !ok {
		return input, fmt.Errorf("Survey reminder template is unavailable")
	}
	input.Name, input.Subject = source.Name, source.Subject
	input.Markdown = strings.ReplaceAll(marketingTemplateMarkdown(source.Markdown, stats), "{{surveyResponses}}", strconv.Itoa(state.Responses))
	return input, nil
}
