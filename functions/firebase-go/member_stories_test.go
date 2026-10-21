package ipace

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestStoryEvidenceAcceptsOnlyOwnedLinkedRecords(t *testing.T) {
	snapshot := memberSnapshot{
		VehicleRecords: []vehicleRecord{{ID: "vehicle-1", Vehicle: vehicleDetails{Registration: "AB12 CDE", ModelYear: "2020"}}},
		ServiceEvents: []serviceEventRecord{
			{ID: "service-1", VehicleID: "vehicle-1", EventType: "repair"},
			{ID: "eo-1", VehicleID: "vehicle-1", EventType: "executive-office-contact"},
			{ID: "other-car", VehicleID: "vehicle-2", EventType: "fault"},
		},
		BatteryReadings: []batteryReadingRecord{{ID: "soh-1", VehicleID: "vehicle-1"}, {ID: "other-soh", VehicleID: "vehicle-2"}},
	}
	evidence, ok := storyEvidence(snapshot, "vehicle-1", []string{"service-1", "eo-1", "eo-1"}, []string{"soh-1"})
	if !ok || evidence.ServiceEventCount != 2 || evidence.ExecutiveContacts != 1 || evidence.SOHReadingCount != 1 {
		t.Fatalf("unexpected evidence summary: %#v, ok=%v", evidence, ok)
	}
	if _, ok = storyEvidence(snapshot, "vehicle-1", []string{"other-car"}, nil); ok {
		t.Fatal("accepted a service record belonging to another vehicle")
	}
	if _, ok = storyEvidence(snapshot, "vehicle-1", nil, []string{"other-soh"}); ok {
		t.Fatal("accepted an SoH reading belonging to another vehicle")
	}
}

func TestStoryEvidenceAcceptsRegisteredVehicleWithoutRegistration(t *testing.T) {
	snapshot := memberSnapshot{VehicleRecords: []vehicleRecord{{ID: "vehicle-1", Vehicle: vehicleDetails{VINLast6: "123456", ModelYear: "2019"}}}}
	evidence, ok := storyEvidence(snapshot, "vehicle-1", nil, nil)
	if !ok || evidence.VehicleLabel != "VIN …123456" {
		t.Fatalf("VIN-only registered vehicle was rejected: %#v, ok=%v", evidence, ok)
	}
}

func TestPublicMemberStoryOmitsPrivateReviewAndEvidenceIdentifiers(t *testing.T) {
	public := publicMemberStory{ID: "story-1", DisplayName: "Owner A", Title: "My experience", BodyMarkdown: "Story", PublishedAt: time.Now(), Evidence: publicStoryEvidence{VehicleLabel: "AB12 CDE", ServiceEventCount: 2}}
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"contactEmail", "contactName", "identityUserId", "serviceEventIds", "sohReadingIds", "adminNotes", "analysis", "consent"} {
		if containsJSONKey(text, forbidden) {
			t.Fatalf("public story exposed %q: %s", forbidden, text)
		}
	}
}

func TestStorySlugIsStableAndBounded(t *testing.T) {
	got := storySlug("Three modules replaced — and one failed again", "abcdef123456")
	if got != "three-modules-replaced-and-one-failed-again-abcdef12" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestMemberStoryResponseKeepsClarificationButOmitsPrivateReviewMaterial(t *testing.T) {
	story := memberStory{
		ContactName: "Private name", ContactEmail: "private@example.com", AdminNotes: "Internal",
		MemberMessage: "Please clarify the date.", Analysis: &memberStoryAnalysis{Sentiment: "frustrated"},
		Revisions: []memberStoryRevision{{Editor: "admin"}}, ConsentHistory: []memberStoryConsent{{Website: true}},
	}
	got := memberStoryForOwner(story)
	if got.ContactName != "" || got.ContactEmail != "" || got.AdminNotes != "" || got.Analysis != nil || got.Revisions != nil || got.ConsentHistory != nil {
		t.Fatalf("member response retained private review data: %#v", got)
	}
	if got.MemberMessage != "Please clarify the date." {
		t.Fatalf("member clarification was removed: %#v", got)
	}
}

func TestStoryApprovalRequiresPublicationConsentAndReadyMedia(t *testing.T) {
	story := memberStory{Title: "A story", DisplayName: "Owner", BodyMarkdown: strings.Repeat("experience ", 10), Consent: memberStoryConsent{Website: true, Truthful: true}}
	if message := storyApprovalError(story); message != "" {
		t.Fatalf("valid text-only story was rejected: %s", message)
	}
	story.Consent.Truthful = false
	if message := storyApprovalError(story); message == "" {
		t.Fatal("story without truthfulness confirmation was accepted")
	}
	story.Consent.Truthful = true
	story.Media = []memberStoryMedia{{Status: "pending"}}
	story.Consent.Copyright = true
	if message := storyApprovalError(story); message == "" {
		t.Fatal("story with pending media was accepted")
	}
}

func TestStoryTitleIsDerivedFromFirstMeaningfulLine(t *testing.T) {
	got := storyTitleFromMarkdown("\n\n## What happened\n\nThe battery failed without warning.")
	if got != "The battery failed without warning." {
		t.Fatalf("derived title = %q", got)
	}
}

func TestMemberStoryNamesUsePreferenceAndLegalNameFallback(t *testing.T) {
	snapshot := memberSnapshot{JoinRecords: []joinRecord{{
		Contact:    contactRecord{Name: "Join Name"},
		Membership: membershipRecord{DisplayName: "BatteryFan"},
	}}}
	contact, display := memberStoryNamesFromRecords(snapshot, "Alexandra Owner")
	if contact != "Alexandra Owner" || display != "BatteryFan" {
		t.Fatalf("names = %q, %q", contact, display)
	}

	snapshot.JoinRecords[0].Membership.DisplayName = ""
	contact, display = memberStoryNamesFromRecords(snapshot, "Alexandra Owner")
	if contact != "Alexandra Owner" || display != "Alexandra" {
		t.Fatalf("fallback names = %q, %q", contact, display)
	}
}

func containsJSONKey(value, key string) bool {
	return len(value) > 0 && stringContains(value, `"`+key+`"`)
}

func stringContains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
