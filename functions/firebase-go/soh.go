package ipace

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func SubmitSOH(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}

	user, err := requireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}

	var req batteryReadingRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}

	vehicleID := cleanString(req.VehicleID, 100)
	battery, err := validatedBatteryReading(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	vehicle, err := loadOwnedVehicle(r.Context(), vehicleID, user.UID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Vehicle not found"})
		return
	}

	now := time.Now().UTC()
	record := batteryReadingRecord{
		ID:             submissionID("soh"),
		Type:           "battery-reading",
		CreatedAt:      now,
		UpdatedAt:      now,
		IdentityUserID: user.UID,
		VehicleID:      vehicle.ID,
		Battery:        battery,
		Review:         reviewRecord{Status: "new", VerificationLevel: "self-reported"},
	}
	if err := saveBatteryReading(r.Context(), vehicle, record); err != nil {
		logEvent("submit-soh", "error", "record save failed", map[string]any{"uid": user.UID, "vehicleId": vehicle.ID, "error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Could not save State of Health reading"})
		return
	}
	if err := regenerateMemberSnapshot(r.Context(), user.UID, user.Email); err != nil {
		logEvent("submit-soh", "warn", "member snapshot regeneration failed", map[string]any{"uid": user.UID, "error": err.Error()})
	}
	if err := regeneratePublicStatsSnapshot(r.Context()); err != nil {
		logEvent("submit-soh", "warn", "public snapshot regeneration failed", map[string]any{"error": err.Error()})
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": record.ID})
}

func validatedBatteryReading(req batteryReadingRequest) (batteryDetails, error) {
	if cleanString(req.VehicleID, 100) == "" {
		return batteryDetails{}, fmt.Errorf("vehicle is required")
	}
	soh := cleanDecimal(req.SOH, 0, 100)
	if soh == nil {
		return batteryDetails{}, fmt.Errorf("State of Health must be between 0 and 100")
	}
	measuredAt := cleanDate(req.SOHDate)
	if measuredAt == "" {
		return batteryDetails{}, fmt.Errorf("measurement date is required")
	}
	source := cleanEnum(req.SOHSource, sohSourceValues)
	if source == "" {
		return batteryDetails{}, fmt.Errorf("measurement source is required")
	}
	return batteryDetails{
		StateOfHealth:        soh,
		MeasuredAt:           measuredAt,
		MileageAtMeasurement: cleanInt(req.SOHMileage, 0, 500000),
		Source:               source,
	}, nil
}

func loadOwnedVehicle(ctx context.Context, vehicleID string, uid string) (vehicleRecord, error) {
	if vehicleID == "" || uid == "" {
		return vehicleRecord{}, fmt.Errorf("vehicle not found")
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return vehicleRecord{}, err
	}
	doc, err := db.Collection("vehicles").Doc(vehicleID).Get(ctx)
	if err != nil {
		return vehicleRecord{}, err
	}
	var record vehicleRecord
	if err := doc.DataTo(&record); err != nil || !vehicleOwnedBy(record, uid) {
		return vehicleRecord{}, fmt.Errorf("vehicle not found")
	}
	return record, nil
}

func vehicleOwnedBy(record vehicleRecord, uid string) bool {
	return uid != "" && record.IdentityUserID == uid
}

func saveBatteryReading(ctx context.Context, vehicle vehicleRecord, reading batteryReadingRecord) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	vehicle.UpdatedAt = reading.UpdatedAt
	if readingIsLatest(vehicle.Battery, reading.Battery) {
		vehicle.Battery = reading.Battery
		if reading.Battery.MileageAtMeasurement != nil {
			vehicle.Vehicle.Mileage = reading.Battery.MileageAtMeasurement
		}
	}
	batch := db.Batch()
	batch.Set(db.Collection("batteryReadings").Doc(reading.ID), reading)
	batch.Set(db.Collection("vehicles").Doc(vehicle.ID), vehicle)
	_, err = batch.Commit(ctx)
	return err
}

func readingIsLatest(current batteryDetails, candidate batteryDetails) bool {
	if current.StateOfHealth == nil || current.MeasuredAt == "" {
		return true
	}
	currentDate, currentErr := time.Parse("2006-01-02", current.MeasuredAt)
	candidateDate, candidateErr := time.Parse("2006-01-02", candidate.MeasuredAt)
	if candidateErr != nil {
		return false
	}
	if currentErr != nil {
		return true
	}
	return !candidateDate.Before(currentDate)
}

func initialBatteryReading(record vehicleRecord) *batteryReadingRecord {
	if record.Battery.StateOfHealth == nil {
		return nil
	}
	return &batteryReadingRecord{
		ID:             submissionID("soh"),
		Type:           "battery-reading",
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
		IdentityUserID: record.IdentityUserID,
		VehicleID:      record.ID,
		Battery:        record.Battery,
		Review:         record.Review,
	}
}

func normalisedMeasurementDate(reading batteryReadingRecord) time.Time {
	if parsed, err := time.Parse("2006-01-02", strings.TrimSpace(reading.Battery.MeasuredAt)); err == nil {
		return parsed
	}
	return reading.CreatedAt
}
