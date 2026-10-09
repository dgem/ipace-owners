package ipace

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"time"
)

const nextStepsSurveyID = "survey_what_next_october_2026"

// The staging fixture uses the normal survey collection and response path. It is
// created only once, so subsequent admin edits and test responses are preserved.
func ensureNextStepsSurvey(ctx context.Context, db *firestore.Client) error {
	ref := db.Collection("surveys").Doc(nextStepsSurveyID)
	return seedNextStepsSurvey(os.Getenv("SEED_NEXT_STEPS_SURVEY") == "true", time.Now().UTC(), func() (bool, error) {
		_, err := ref.Get(ctx)
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return err == nil, err
	}, func(record surveyRecord) error {
		_, err := ref.Create(ctx, record)
		return err
	})
}

func seedNextStepsSurvey(enabled bool, now time.Time, exists func() (bool, error), create func(surveyRecord) error) error {
	if !enabled {
		return nil
	}
	found, err := exists()
	if err != nil || found {
		return err
	}
	record, err := nextStepsSurveyRecord(now)
	if err != nil {
		return err
	}
	err = create(record)
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}

func nextStepsSurveyRecord(now time.Time) (surveyRecord, error) {
	record, err := validateSurvey(surveyInput{
		ID:           nextStepsSurveyID,
		Title:        "What do you think the group should do next?",
		Description:  "JLR has offered to review owners' cases individually. Help us set the group's priorities. Choose any steps you support, then mark one as your top priority. These paths can be pursued together. This survey does not authorise the group to represent you legally or share your details or story. Any preparatory legal engagement requires a separate choice, UK eligibility and identity checks; proceedings and costs would require further authority.",
		Question:     "Which steps should the group prioritise now?",
		CallToAction: "Choose the steps you support",
		Status:       "published",
		Multiple:     true,
		StartsOn:     "2026-10-09",
		EndsOn:       "2026-11-09",
		ShowResults:  true,
		Options: []surveyOption{
			{ID: "dialogue", Name: "Give JLR time to respond", Description: "Keep dialogue open and assess whether individual case reviews produce lasting results.", AllowsPreferred: true},
			{ID: "evidence", Name: "Grow membership and evidence", Description: "Reach more I-PACE owners and strengthen verified vehicle and service records.", AllowsPreferred: true},
			{ID: "press", Name: "Measured press engagement", Description: "Explain owners' experiences publicly without undermining constructive dialogue. Anyone whose story may be used would be asked separately for consent.", AllowsPreferred: true},
			{ID: "legal-preparation", Name: "Prepare legal escalation", Description: "Assess legal routes, costs and possible funding with advisers. Selecting this is not consent to representation or litigation.", AllowsPreferred: true},
			{ID: "other", Name: "Another approach", Description: "Tell us what you would prioritise.", AllowsPreferred: true, AllowsText: true, TextPrompt: "What else should we consider?"},
		},
	})
	if err != nil {
		return surveyRecord{}, fmt.Errorf("invalid next-steps survey: %w", err)
	}
	record.CreatedAt = now.UTC()
	record.UpdatedAt = now.UTC()
	return record, nil
}
