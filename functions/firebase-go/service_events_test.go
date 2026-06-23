package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidatedServiceEvent(t *testing.T) {
	event, err := validatedServiceEvent(serviceEventRequest{
		VehicleID:   "vehicle_123",
		EventType:   "fault",
		OccurredAt:  "2026-06-22",
		Mileage:     "42000",
		Title:       "Traction battery warning",
		Description: "Warning shown while charging.",
		Status:      "open",
	})
	if err != nil {
		t.Fatalf("validatedServiceEvent() error = %v", err)
	}
	if event.Mileage == nil || *event.Mileage != 42000 {
		t.Fatalf("Mileage = %v", event.Mileage)
	}
	if event.EventType != "fault" || event.Status != "open" {
		t.Fatalf("event = %+v", event)
	}
}

func TestValidatedServiceEventRejectsInvalidInput(t *testing.T) {
	valid := serviceEventRequest{
		VehicleID: "vehicle_123", EventType: "service", OccurredAt: "2026-06-22",
		Title: "Annual service", Status: "completed",
	}
	cases := []serviceEventRequest{
		{EventType: valid.EventType, OccurredAt: valid.OccurredAt, Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: "accident", OccurredAt: valid.OccurredAt, Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: "not-a-date", Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, Title: valid.Title, Status: "unknown"},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, Mileage: "many", Title: valid.Title, Status: valid.Status},
	}
	for _, request := range cases {
		if _, err := validatedServiceEvent(request); err == nil {
			t.Fatalf("validatedServiceEvent(%+v) unexpectedly succeeded", request)
		}
	}
}

func TestServiceEventOwnership(t *testing.T) {
	record := serviceEventRecord{IdentityUserID: "member-1", VehicleID: "vehicle-1"}
	if !serviceEventOwnedBy(record, "member-1", "vehicle-1") {
		t.Fatal("owner and vehicle were rejected")
	}
	if serviceEventOwnedBy(record, "member-2", "vehicle-1") || serviceEventOwnedBy(record, "member-1", "vehicle-2") {
		t.Fatal("non-owner or different vehicle was accepted")
	}
}

func TestUpsertServiceEventRequiresAuthentication(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/upsert-service-event", strings.NewReader(`{"vehicleId":"vehicle_123","eventType":"fault","occurredAt":"2026-06-22","title":"Battery warning","status":"open"}`))
	rec := httptest.NewRecorder()

	UpsertServiceEvent(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
