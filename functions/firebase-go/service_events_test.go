package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidatedServiceEvent(t *testing.T) {
	yes := true
	event, err := validatedServiceEvent(serviceEventRequest{
		VehicleID:                 "vehicle_123",
		EventType:                 "fault",
		OccurredAt:                "2026-06-22",
		Mileage:                   "42000",
		Title:                     "Traction battery warning",
		Description:               "Warning shown while charging.",
		Status:                    "open",
		Campaigns:                 stringArray{"H441", "H448", "H570"},
		ServiceProviderID:         "J1714",
		ServiceProviderName:       "Barretts Jaguar Service Centre, Ashford",
		ServiceProviderPostcode:   "tn24 0fl",
		ServiceProviderAuthorised: &yes,
		FinalFixAt:                "2026-06-24",
		CourtesyVehicleOffered:    "yes",
		CourtesyVehicleProvided:   "no",
		PartsDelay:                "up-to-1-month",
		GoodwillPayment:           &yes,
		MilesDrivenWhilstFaulty:   "1200",
		WarrantyCover:             "battery-warranty",
		DisputeStatus:             "initially-refused",
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
	if len(event.Campaigns) != 3 || event.Campaigns[0] != "H441" || event.Campaigns[1] != "H448" || event.Campaigns[2] != "H570" {
		t.Fatalf("Campaigns = %+v", event.Campaigns)
	}
	if event.FinalFixAt != "2026-06-24" || event.DaysToFinalFix == nil || *event.DaysToFinalFix != 2 {
		t.Fatalf("fix duration fields = %+v", event)
	}
	if event.ServiceProviderID != "J1714" || event.ServiceProviderName != "Barretts Jaguar Service Centre, Ashford" || event.ServiceProviderPostcode != "TN24 0FL" || event.ServiceProviderAuthorised == nil || !*event.ServiceProviderAuthorised {
		t.Fatalf("service provider fields = %+v", event)
	}
	if event.CourtesyVehicleOffered != "yes" || event.CourtesyVehicleProvided != "no" || event.PartsDelay != "up-to-1-month" || event.GoodwillPayment == nil || !*event.GoodwillPayment || event.MilesDrivenWhilstFaulty == nil || *event.MilesDrivenWhilstFaulty != 1200 || event.WarrantyCover != "battery-warranty" || event.DisputeStatus != "initially-refused" {
		t.Fatalf("support fields = %+v", event)
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
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: "2099-06-22", Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, FinalFixAt: "not-a-date", Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, FinalFixAt: "2099-06-22", Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, FinalFixAt: "2026-06-21", Title: valid.Title, Status: valid.Status},
		{VehicleID: valid.VehicleID, EventType: valid.EventType, OccurredAt: valid.OccurredAt, MilesDrivenWhilstFaulty: "many", Title: valid.Title, Status: valid.Status},
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

func TestServiceEventResolutionDays(t *testing.T) {
	if days := serviceEventResolutionDays("2026-01-31", "2026-03-02"); days == nil || *days != 30 {
		t.Fatalf("serviceEventResolutionDays() = %v, want 30", days)
	}
	if days := serviceEventResolutionDays("2026-03-02", ""); days != nil {
		t.Fatalf("serviceEventResolutionDays() without final date = %v, want nil", days)
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
