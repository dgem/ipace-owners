package ipace

import (
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func TestNextStepsSurveyIsAStagingPreferenceSurvey(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	record, err := nextStepsSurveyRecord(now)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != nextStepsSurveyID || record.Status != "published" || !record.Multiple || !surveyIsLive(record, now) || len(record.Options) != 5 {
		t.Fatalf("unexpected next-steps survey: %#v", record)
	}
	if record.Title != "What do you think the group should do next?" {
		t.Fatalf("survey title = %q", record.Title)
	}
	for _, option := range record.Options {
		if !option.AllowsText || option.TextPrompt == "" {
			t.Fatalf("%s must invite an optional summary: %#v", option.ID, option)
		}
	}
	if !strings.Contains(record.Description, "does not authorise") || !strings.Contains(record.Description, "separate choice") {
		t.Fatal("survey must distinguish a preference from legal consent")
	}
	ids, _, preferred, err := validateSurveyResponse(record, surveyResponseInput{
		OptionIDs:         []string{"evidence", "legal-preparation"},
		PreferredOptionID: "legal-preparation",
	})
	if err != nil || len(ids) != 2 || preferred != "legal-preparation" {
		t.Fatalf("member preference was not accepted: %v, %#v, %q", err, ids, preferred)
	}
}

func TestStagingSurveySeedPreservesExistingRecordAndHandlesRaces(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	readCount, createCount := 0, 0
	read := func() (bool, error) { readCount++; return true, nil }
	create := func(record surveyRecord) error { createCount++; return nil }
	if err := seedNextStepsSurvey(false, now, read, create); err != nil || readCount != 0 || createCount != 0 {
		t.Fatalf("disabled seed touched the store: %v, %d, %d", err, readCount, createCount)
	}
	if err := seedNextStepsSurvey(true, now, read, create); err != nil || readCount != 1 || createCount != 0 {
		t.Fatalf("existing admin-edited survey was overwritten: %v, %d, %d", err, readCount, createCount)
	}
	read = func() (bool, error) { return false, nil }
	create = func(record surveyRecord) error {
		createCount++
		if record.ID != nextStepsSurveyID || record.CreatedAt != now {
			t.Fatalf("wrong staging record: %#v", record)
		}
		return status.Error(codes.AlreadyExists, "concurrent seed")
	}
	if err := seedNextStepsSurvey(true, now, read, create); err != nil || createCount != 1 {
		t.Fatalf("concurrent seed failed: %v, %d", err, createCount)
	}
	want := errors.New("datastore unavailable")
	read = func() (bool, error) { return false, want }
	if err := seedNextStepsSurvey(true, now, read, create); !errors.Is(err, want) {
		t.Fatalf("read error lost: %v", err)
	}
}

func TestNextStepsPresentationMigratesOnlyLegacyPrompts(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	existing := surveyRecord{
		Title: "What should the group do next?",
		Options: []surveyOption{
			{ID: "press", Name: "Measured press engagement"},
			{ID: "other", Name: "Another approach", AllowsText: true, TextPrompt: "What else should we consider?"},
			{ID: "evidence", Name: "Grow membership and evidence", AllowsText: true, TextPrompt: "Keep my custom prompt"},
		},
	}
	updated, changed, err := nextStepsSurveyPresentation(existing, now)
	if err != nil || !changed {
		t.Fatalf("legacy presentation was not updated: %v, %#v", err, updated)
	}
	if updated.Title != "What do you think the group should do next?" || !updated.Options[0].AllowsText || updated.Options[0].TextPrompt == "" || updated.Options[1].TextPrompt == "What else should we consider?" {
		t.Fatalf("legacy values not migrated: %#v", updated)
	}
	if updated.Options[2].TextPrompt != "Keep my custom prompt" {
		t.Fatalf("custom administrator prompt was overwritten: %#v", updated.Options[2])
	}
}
