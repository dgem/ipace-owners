package ipace

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var serviceEventTypeValues = []string{"service", "fault", "repair", "recall", "inspection", "other"}
var serviceEventStatusValues = []string{"open", "monitoring", "resolved", "completed"}

func UpsertServiceEvent(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
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

	var req serviceEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	cleaned, err := validatedServiceEvent(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	vehicle, err := loadOwnedVehicle(r.Context(), cleaned.VehicleID, user.UID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Vehicle not found"})
		return
	}

	now := time.Now().UTC()
	record := serviceEventRecord{
		ID:             submissionID("event"),
		Type:           "service-event",
		CreatedAt:      now,
		UpdatedAt:      now,
		IdentityUserID: user.UID,
		VehicleID:      vehicle.ID,
		EventType:      cleaned.EventType,
		OccurredAt:     cleaned.OccurredAt,
		Mileage:        cleaned.Mileage,
		Title:          cleaned.Title,
		Description:    cleaned.Description,
		Status:         cleaned.Status,
		Review:         reviewRecord{Status: "new", VerificationLevel: "self-reported"},
	}
	if cleaned.ID != "" {
		existing, err := loadOwnedServiceEvent(r.Context(), cleaned.ID, user.UID, vehicle.ID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Service event not found"})
			return
		}
		record.ID = existing.ID
		record.CreatedAt = existing.CreatedAt
		record.Review = existing.Review
	}
	if err := saveServiceEvent(r.Context(), record); err != nil {
		logEvent("upsert-service-event", "error", "record save failed", map[string]any{"uid": user.UID, "vehicleId": vehicle.ID, "error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "Could not save service event"})
		return
	}
	if err := regenerateMemberSnapshot(r.Context(), user.UID, user.Email); err != nil {
		logEvent("upsert-service-event", "warn", "member snapshot regeneration failed", map[string]any{"uid": user.UID, "error": err.Error()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": record.ID})
}

type cleanedServiceEvent struct {
	ID          string
	VehicleID   string
	EventType   string
	OccurredAt  string
	Mileage     *int
	Title       string
	Description string
	Status      string
}

func validatedServiceEvent(req serviceEventRequest) (cleanedServiceEvent, error) {
	cleaned := cleanedServiceEvent{
		ID:          cleanString(req.ID, 100),
		VehicleID:   cleanString(req.VehicleID, 100),
		EventType:   cleanEnum(req.EventType, serviceEventTypeValues),
		OccurredAt:  cleanDate(req.OccurredAt),
		Mileage:     cleanInt(req.Mileage, 0, 500000),
		Title:       cleanString(req.Title, 160),
		Description: cleanString(req.Description, 4000),
		Status:      cleanEnum(req.Status, serviceEventStatusValues),
	}
	if cleaned.VehicleID == "" {
		return cleanedServiceEvent{}, fmt.Errorf("vehicle is required")
	}
	if cleaned.EventType == "" {
		return cleanedServiceEvent{}, fmt.Errorf("event type is required")
	}
	if cleaned.OccurredAt == "" {
		return cleanedServiceEvent{}, fmt.Errorf("event date is required")
	}
	if cleaned.Title == "" {
		return cleanedServiceEvent{}, fmt.Errorf("title is required")
	}
	if req.Mileage != "" && cleaned.Mileage == nil {
		return cleanedServiceEvent{}, fmt.Errorf("mileage must be between 0 and 500000")
	}
	if cleaned.Status == "" {
		return cleanedServiceEvent{}, fmt.Errorf("status is required")
	}
	return cleaned, nil
}

func loadOwnedServiceEvent(ctx context.Context, id string, uid string, vehicleID string) (serviceEventRecord, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return serviceEventRecord{}, err
	}
	doc, err := db.Collection("serviceEvents").Doc(id).Get(ctx)
	if err != nil {
		return serviceEventRecord{}, err
	}
	var record serviceEventRecord
	if err := doc.DataTo(&record); err != nil || !serviceEventOwnedBy(record, uid, vehicleID) {
		return serviceEventRecord{}, fmt.Errorf("service event not found")
	}
	return record, nil
}

func serviceEventOwnedBy(record serviceEventRecord, uid string, vehicleID string) bool {
	return uid != "" && vehicleID != "" && record.IdentityUserID == uid && record.VehicleID == vehicleID
}

func saveServiceEvent(ctx context.Context, record serviceEventRecord) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = db.Collection("serviceEvents").Doc(record.ID).Set(ctx, record)
	return err
}
