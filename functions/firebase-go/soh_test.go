package ipace

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
)

type fakeFirebaseAuthUserIterator struct {
	remaining int
	err       error
}

func (it *fakeFirebaseAuthUserIterator) Next() (*auth.ExportedUserRecord, error) {
	if it.remaining > 0 {
		it.remaining--
		return &auth.ExportedUserRecord{}, nil
	}
	if it.err != nil {
		return nil, it.err
	}
	return nil, iterator.Done
}

func TestValidatedBatteryReading(t *testing.T) {
	reading, err := validatedBatteryReading(batteryReadingRequest{
		VehicleID:  "vehicle_123",
		SOH:        "87.64",
		SOHDate:    "2026-06-22",
		SOHMileage: "42000",
		SOHSource:  "diagnostic-app",
	})
	if err != nil {
		t.Fatalf("validatedBatteryReading() error = %v", err)
	}
	if reading.StateOfHealth == nil || *reading.StateOfHealth != 87.6 {
		t.Fatalf("StateOfHealth = %v", reading.StateOfHealth)
	}
	if reading.MileageAtMeasurement == nil || *reading.MileageAtMeasurement != 42000 {
		t.Fatalf("MileageAtMeasurement = %v", reading.MileageAtMeasurement)
	}
}

func TestValidatedBatteryReadingRejectsIncompleteMeasurement(t *testing.T) {
	cases := []batteryReadingRequest{
		{SOH: "90", SOHDate: "2026-06-22", SOHSource: "diagnostic-app"},
		{VehicleID: "vehicle_123", SOH: "101", SOHDate: "2026-06-22", SOHSource: "diagnostic-app"},
		{VehicleID: "vehicle_123", SOH: "90", SOHSource: "diagnostic-app"},
		{VehicleID: "vehicle_123", SOH: "90", SOHDate: "2026-06-22"},
		{VehicleID: "vehicle_123", SOH: "90", SOHDate: "2099-06-22", SOHSource: "diagnostic-app"},
	}
	for _, request := range cases {
		if _, err := validatedBatteryReading(request); err == nil {
			t.Fatalf("validatedBatteryReading(%+v) unexpectedly succeeded", request)
		}
	}
}

func TestVehicleOwnership(t *testing.T) {
	record := vehicleRecord{IdentityUserID: "member-1"}
	if !vehicleOwnedBy(record, "member-1") {
		t.Fatal("owner was rejected")
	}
	if vehicleOwnedBy(record, "member-2") || vehicleOwnedBy(record, "") {
		t.Fatal("non-owner was accepted")
	}
}

func TestReadingIsLatestUsesMeasurementDate(t *testing.T) {
	value := 90.0
	current := batteryDetails{StateOfHealth: &value, MeasuredAt: "2026-06-22"}
	if readingIsLatest(current, batteryDetails{StateOfHealth: &value, MeasuredAt: "2025-06-22"}) {
		t.Fatal("older historical reading replaced the latest reading")
	}
	if !readingIsLatest(current, batteryDetails{StateOfHealth: &value, MeasuredAt: "2026-07-22"}) {
		t.Fatal("newer reading was not treated as latest")
	}
}

func TestSubmitSOHRequiresAuthentication(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/submit-soh", strings.NewReader(`{"vehicleId":"vehicle_123","soh":"90","sohDate":"2026-06-22","sohSource":"diagnostic-app"}`))
	rec := httptest.NewRecorder()

	SubmitSOH(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAggregatePublicStatsUsesLatestConsentedReadings(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	soh := func(value float64) *float64 { return &value }
	vehicles := []vehicleRecord{
		{ID: "v1", IdentityUserID: "u1", UserEmailHash: "consented", Vehicle: vehicleDetails{ModelYear: "2019"}, Review: reviewRecord{Status: "new"}},
		{ID: "v2", IdentityUserID: "u1", UserEmailHash: "consented", Vehicle: vehicleDetails{ModelYear: "2020"}, Review: reviewRecord{Status: "new"}},
		{ID: "v3", IdentityUserID: "u2", UserEmailHash: "not-consented", Vehicle: vehicleDetails{ModelYear: "2021"}, Review: reviewRecord{Status: "new"}},
	}
	readings := []batteryReadingRecord{
		{ID: "r1", VehicleID: "v1", Battery: batteryDetails{StateOfHealth: soh(90), MeasuredAt: "2025-06-22"}, Review: reviewRecord{Status: "new"}},
		{ID: "r2", VehicleID: "v1", Battery: batteryDetails{StateOfHealth: soh(80), MeasuredAt: "2026-06-22"}, Review: reviewRecord{Status: "new"}},
		{ID: "r3", VehicleID: "v2", Battery: batteryDetails{StateOfHealth: soh(90), MeasuredAt: "2026-06-22"}, Review: reviewRecord{Status: "new"}},
		{ID: "r4", VehicleID: "v3", Battery: batteryDetails{StateOfHealth: soh(10), MeasuredAt: "2026-06-22"}, Review: reviewRecord{Status: "new"}},
	}

	got := aggregatePublicStats(vehicles, readings, map[string]bool{"consented": true}, 12, now)
	if got.RegisteredMembers != 12 || got.SchemaVersion != publicStatsSchemaVersion {
		t.Fatalf("membership aggregate = %+v", got)
	}
	if got.OwnersContributed != 1 || got.VehiclesRegistered != 2 || got.SOHReadings != 3 || got.VehiclesWithRepeatSOH != 1 {
		t.Fatalf("unexpected counts: %+v", got)
	}
	if got.AverageReportedSOH == nil || *got.AverageReportedSOH != 85 {
		t.Fatalf("AverageReportedSOH = %v", got.AverageReportedSOH)
	}
	if got.AverageSOHChange == nil || *got.AverageSOHChange != -10 {
		t.Fatalf("AverageSOHChange = %v", got.AverageSOHChange)
	}
	if got.SOHDistribution[0].Count != 1 || got.SOHDistribution[1].Count != 1 {
		t.Fatalf("SOHDistribution = %+v", got.SOHDistribution)
	}
}

func TestAggregatePublicStatsUsesLegacyEmbeddedReading(t *testing.T) {
	value := 88.0
	vehicle := vehicleRecord{
		ID: "legacy", IdentityUserID: "u1", UserEmailHash: "consented",
		Battery: batteryDetails{StateOfHealth: &value, MeasuredAt: "2025-01-01"},
		Review:  reviewRecord{Status: "new"},
	}
	got := aggregatePublicStats([]vehicleRecord{vehicle}, nil, map[string]bool{"consented": true}, 1, time.Now())
	if got.SOHReadings != 1 || got.VehiclesWithSOH != 1 || got.AverageReportedSOH == nil || *got.AverageReportedSOH != 88 {
		t.Fatalf("legacy aggregate = %+v", got)
	}
}

func TestRegisteredFirebaseAuthUsersCountsEveryPaginatedUser(t *testing.T) {
	original := firebaseAuthUsers
	t.Cleanup(func() { firebaseAuthUsers = original })
	firebaseAuthUsers = func(context.Context) (firebaseAuthUserIterator, error) {
		return &fakeFirebaseAuthUserIterator{remaining: 209}, nil
	}

	got, err := registeredFirebaseAuthUsers(context.Background())
	if err != nil {
		t.Fatalf("registeredFirebaseAuthUsers() error = %v", err)
	}
	if got != 209 {
		t.Fatalf("registeredFirebaseAuthUsers() = %d, want 209", got)
	}
}

func TestRegisteredFirebaseAuthUsersReturnsIteratorError(t *testing.T) {
	original := firebaseAuthUsers
	t.Cleanup(func() { firebaseAuthUsers = original })
	want := errors.New("list users failed")
	firebaseAuthUsers = func(context.Context) (firebaseAuthUserIterator, error) {
		return &fakeFirebaseAuthUserIterator{remaining: 2, err: want}, nil
	}

	got, err := registeredFirebaseAuthUsers(context.Background())
	if !errors.Is(err, want) || got != 0 {
		t.Fatalf("registeredFirebaseAuthUsers() = (%d, %v), want (0, %v)", got, err, want)
	}
}

func TestRefreshRegisteredMemberCountUpdatesChangedAuthTotal(t *testing.T) {
	original := firebaseAuthUsers
	t.Cleanup(func() { firebaseAuthUsers = original })
	firebaseAuthUsers = func(context.Context) (firebaseAuthUserIterator, error) {
		return &fakeFirebaseAuthUserIterator{remaining: 209}, nil
	}
	generatedAt := time.Date(2026, time.July, 18, 15, 0, 0, 0, time.UTC)

	got, updated := refreshRegisteredMemberCount(context.Background(), publicStatsSnapshot{RegisteredMembers: 119}, generatedAt)
	if !updated || got.RegisteredMembers != 209 || !got.GeneratedAt.Equal(generatedAt) {
		t.Fatalf("refreshRegisteredMemberCount() = (%+v, %t), want total 209 at %v", got, updated, generatedAt)
	}
}

func TestRefreshRegisteredMemberCountKeepsSnapshotWhenAuthFails(t *testing.T) {
	original := firebaseAuthUsers
	t.Cleanup(func() { firebaseAuthUsers = original })
	firebaseAuthUsers = func(context.Context) (firebaseAuthUserIterator, error) {
		return nil, errors.New("auth unavailable")
	}
	want := publicStatsSnapshot{RegisteredMembers: 119}

	got, updated := refreshRegisteredMemberCount(context.Background(), want, time.Now())
	if updated || got.RegisteredMembers != want.RegisteredMembers {
		t.Fatalf("refreshRegisteredMemberCount() = (%+v, %t), want unchanged snapshot", got, updated)
	}
}

func TestPublicStatsJSONContainsNoMemberOrVehicleIdentifiers(t *testing.T) {
	snapshot := publicStatsSnapshot{OwnersContributed: 2, VehiclesRegistered: 3}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, privateField := range []string{"identityUserId", "email", "registration", "vinHash", "vehicleId"} {
		if strings.Contains(string(encoded), privateField) {
			t.Fatalf("public snapshot contains private field %q: %s", privateField, encoded)
		}
	}
}
