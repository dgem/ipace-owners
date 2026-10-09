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
	if os.Getenv("SEED_NEXT_STEPS_SURVEY") != "true" {
		return nil
	}
	now := time.Now().UTC()
	snapshot, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		record, recordErr := nextStepsSurveyRecord(now)
		if recordErr != nil {
			return recordErr
		}
		_, err := ref.Create(ctx, record)
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
		return err
	}
	if err != nil {
		return err
	}
	var existing surveyRecord
	if err := snapshot.DataTo(&existing); err != nil {
		return err
	}
	updated, changed, err := nextStepsSurveyPresentation(existing, now)
	if err != nil || !changed {
		return err
	}
	_, err = ref.Update(ctx, []firestore.Update{
		{Path: "title", Value: updated.Title},
		{Path: "options", Value: updated.Options},
		{Path: "startsOn", Value: updated.StartsOn},
		{Path: "endsOn", Value: updated.EndsOn},
		{Path: "updatedAt", Value: now},
	})
	return err
}

func nextStepsSurveyPresentation(existing surveyRecord, now time.Time) (surveyRecord, bool, error) {
	desired, err := nextStepsSurveyRecord(now)
	if err != nil {
		return surveyRecord{}, false, err
	}
	changed := false
	if existing.Title == "What should the group do next?" {
		existing.Title = desired.Title
		changed = true
	}
	if existing.StartsOn == "2026-10-09" && existing.EndsOn == "2026-11-09" {
		existing.StartsOn = desired.StartsOn
		existing.EndsOn = desired.EndsOn
		changed = true
	}
	desiredOptions := make(map[string]surveyOption, len(desired.Options))
	for _, option := range desired.Options {
		desiredOptions[option.ID] = option
	}
	for i := range existing.Options {
		current := &existing.Options[i]
		wanted, ok := desiredOptions[current.ID]
		if !ok {
			continue
		}
		if !current.AllowsText {
			current.AllowsText = true
			changed = true
		}
		if nextStepsLegacyPrompt(current.TextPrompt) {
			current.TextPrompt = wanted.TextPrompt
			changed = true
		}
	}
	return existing, changed, nil
}

func nextStepsLegacyPrompt(prompt string) bool {
	switch prompt {
	case "", "What else should we consider?",
		"Please share a brief summary of what you think JLR should do next.",
		"Please share a brief summary of the evidence or member growth that would help most.",
		"Please share a brief summary that could inform measured press engagement. We would ask separately before using your words publicly.",
		"Please share a brief summary of what legal preparation you think the group should explore.",
		"Please share a brief summary of the approach you would like the group to consider.":
		return true
	default:
		return false
	}
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
		Status:       nextStepsSurveyStatus(),
		Multiple:     true,
		StartsOn:     "2026-10-10",
		EndsOn:       "2026-10-23",
		ShowResults:  true,
		Options: []surveyOption{
			{ID: "dialogue", Name: "Give JLR time to respond", Description: "Keep dialogue open and assess whether individual case reviews produce lasting results.", AllowsPreferred: true, AllowsText: true, TextPrompt: "How long should we wait, and what are you hoping will happen in that time?"},
			{ID: "evidence", Name: "Grow membership and evidence", Description: "Reach more I-PACE owners and strengthen verified vehicle and service records.", AllowsPreferred: true, AllowsText: true, TextPrompt: "Can you help us? How can we attract more I-PACE owners?"},
			{ID: "press", Name: "Measured press engagement", Description: "Explain owners' experiences publicly without undermining constructive dialogue. Anyone whose story may be used would be asked separately for consent.", AllowsPreferred: true, AllowsText: true, TextPrompt: "Do you have a story to share, or experience or contacts that could help?"},
			{ID: "legal-preparation", Name: "Prepare legal escalation", Description: "Assess legal routes, costs and possible funding with advisers. Selecting this is not consent to representation or litigation.", AllowsPreferred: true, AllowsText: true, TextPrompt: "Have you tried this avenue already, or do you have relevant expertise to offer?"},
			{ID: "other", Name: "Another approach", Description: "Tell us what you would prioritise.", AllowsPreferred: true, AllowsText: true, TextPrompt: "What other approach should the group consider?"},
		},
	})
	if err != nil {
		return surveyRecord{}, fmt.Errorf("invalid next-steps survey: %w", err)
	}
	record.CreatedAt = now.UTC()
	record.UpdatedAt = now.UTC()
	return record, nil
}

func nextStepsSurveyStatus() string {
	if os.Getenv("NEXT_STEPS_SURVEY_STATUS") == "draft" {
		return "draft"
	}
	return "published"
}
