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
