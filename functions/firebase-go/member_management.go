package ipace

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const deleteConfirmation = "DELETE"

func UpdateMemberPreferences(w http.ResponseWriter, r *http.Request) {
	if !memberMutationRequest(w, r) {
		return
	}
	user, _ := requireUser(r.Context(), r)
	var req preferenceRequest
	if err := decodeJSON(r, &req); err != nil || req.Contact == nil || req.AnonymisedAnalysis == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Both preferences are required"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not update preferences"})
		return
	}
	iter := db.Collection("joinSubmissions").Where("userEmailHash", "==", emailFingerprint(user.Email)).Documents(r.Context())
	batch := db.Batch()
	count := 0
	for {
		doc, nextErr := iter.Next()
		if nextErr == iterator.Done {
			break
		}
		if nextErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not update preferences"})
			return
		}
		batch.Update(doc.Ref, []firestore.Update{
			{Path: "consents.contact", Value: *req.Contact},
			{Path: "consents.anonymisedAnalysis", Value: *req.AnonymisedAnalysis},
			{Path: "updatedAt", Value: time.Now().UTC()},
		})
		count++
	}
	if count == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Membership record not found"})
		return
	}
	if _, err := batch.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not update preferences"})
		return
	}
	regenerateMemberAndPublicSnapshots(r, user.UID, user.Email)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	if !memberMutationRequest(w, r) {
		return
	}
	user, _ := requireUser(r.Context(), r)
	req, ok := confirmedDeleteRequest(w, r)
	if !ok {
		return
	}
	vehicle, err := loadOwnedVehicle(r.Context(), req.ID, user.UID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Vehicle not found"})
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete vehicle"})
		return
	}
	now := time.Now().UTC()
	batch := db.Batch()
	batch.Set(db.Collection("vehicles").Doc(vehicle.ID), deletedVehicle(vehicle, now))
	if err := addDeletedVehicleEvidence(r, batch, db, vehicle.ID, user.UID, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete vehicle"})
		return
	}
	if _, err := batch.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete vehicle"})
		return
	}
	regenerateMemberAndPublicSnapshots(r, user.UID, user.Email)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func DeleteSOH(w http.ResponseWriter, r *http.Request) {
	if !memberMutationRequest(w, r) {
		return
	}
	user, _ := requireUser(r.Context(), r)
	req, ok := confirmedDeleteRequest(w, r)
	if !ok {
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete State of Health reading"})
		return
	}
	doc, err := db.Collection("batteryReadings").Doc(req.ID).Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "State of Health reading not found"})
		return
	}
	var reading batteryReadingRecord
	if err := doc.DataTo(&reading); err != nil || reading.IdentityUserID != user.UID || recordDeleted(reading.Review) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "State of Health reading not found"})
		return
	}
	reading.Review.Status = "deleted"
	reading.Review.DeletedAt = time.Now().UTC()
	reading.UpdatedAt = reading.Review.DeletedAt
	if _, err := doc.Ref.Set(r.Context(), reading); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete State of Health reading"})
		return
	}
	if vehicle, err := loadOwnedVehicle(r.Context(), reading.VehicleID, user.UID); err == nil {
		_ = refreshVehicleBattery(r.Context(), vehicle, reading.ID)
	}
	regenerateMemberAndPublicSnapshots(r, user.UID, user.Email)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func DeleteServiceEvent(w http.ResponseWriter, r *http.Request) {
	if !memberMutationRequest(w, r) {
		return
	}
	user, _ := requireUser(r.Context(), r)
	req, ok := confirmedDeleteRequest(w, r)
	if !ok {
		return
	}
	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete service event"})
		return
	}
	doc, err := db.Collection("serviceEvents").Doc(req.ID).Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Service event not found"})
		return
	}
	var record serviceEventRecord
	if err := doc.DataTo(&record); err != nil || record.IdentityUserID != user.UID || recordDeleted(record.Review) {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Service event not found"})
		return
	}
	record.Review.Status = "deleted"
	record.Review.DeletedAt = time.Now().UTC()
	record.UpdatedAt = record.Review.DeletedAt
	if _, err := doc.Ref.Set(r.Context(), record); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not delete service event"})
		return
	}
	regenerateMemberAndPublicSnapshots(r, user.UID, user.Email)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func memberMutationRequest(w http.ResponseWriter, r *http.Request) bool {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return false
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return false
	}
	if _, err := requireUser(r.Context(), r); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return false
	}
	return true
}

func confirmedDeleteRequest(w http.ResponseWriter, r *http.Request) (deleteRequest, bool) {
	var req deleteRequest
	if err := decodeJSON(r, &req); err != nil || cleanString(req.ID, 100) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "A record is required"})
		return deleteRequest{}, false
	}
	req.ID = cleanString(req.ID, 100)
	if req.Confirmation != deleteConfirmation {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Type DELETE to confirm"})
		return deleteRequest{}, false
	}
	return req, true
}

func regenerateMemberAndPublicSnapshots(r *http.Request, uid, email string) {
	if err := regenerateMemberSnapshot(r.Context(), uid, email); err != nil {
		logEvent("member-data-management", "warn", "member snapshot regeneration failed", map[string]any{"uid": uid, "error": err.Error()})
	}
	if err := regeneratePublicStatsSnapshot(r.Context()); err != nil {
		logEvent("member-data-management", "warn", "public snapshot regeneration failed", map[string]any{"error": err.Error()})
	}
}

func deletedVehicle(record vehicleRecord, now time.Time) vehicleRecord {
	record.Review.Status = "deleted"
	record.Review.DeletedAt = now
	record.UpdatedAt = now
	return record
}

func addDeletedVehicleEvidence(r *http.Request, batch *firestore.WriteBatch, db *firestore.Client, vehicleID, uid string, now time.Time) error {
	readings := db.Collection("batteryReadings").Where("identityUserId", "==", uid).Documents(r.Context())
	for {
		doc, err := readings.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var record batteryReadingRecord
		if doc.DataTo(&record) == nil && record.VehicleID == vehicleID && !recordDeleted(record.Review) {
			record.Review.Status, record.Review.DeletedAt, record.UpdatedAt = "deleted", now, now
			batch.Set(doc.Ref, record)
		}
	}
	events := db.Collection("serviceEvents").Where("identityUserId", "==", uid).Documents(r.Context())
	for {
		doc, err := events.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var record serviceEventRecord
		if doc.DataTo(&record) == nil && record.VehicleID == vehicleID && !recordDeleted(record.Review) {
			record.Review.Status, record.Review.DeletedAt, record.UpdatedAt = "deleted", now, now
			batch.Set(doc.Ref, record)
		}
	}
	return nil
}

func refreshVehicleBattery(ctx context.Context, vehicle vehicleRecord, deletedReadingID string) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	iter := db.Collection("batteryReadings").Where("identityUserId", "==", vehicle.IdentityUserID).Documents(ctx)
	var latest *batteryReadingRecord
	for {
		doc, nextErr := iter.Next()
		if nextErr == iterator.Done {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		var record batteryReadingRecord
		if doc.DataTo(&record) == nil && record.ID != deletedReadingID && record.VehicleID == vehicle.ID && !recordDeleted(record.Review) && (latest == nil || record.Battery.MeasuredAt > latest.Battery.MeasuredAt) {
			copy := record
			latest = &copy
		}
	}
	vehicle.Battery = batteryDetails{}
	if latest != nil {
		vehicle.Battery = latest.Battery
		if latest.Battery.MileageAtMeasurement != nil {
			vehicle.Vehicle.Mileage = latest.Battery.MileageAtMeasurement
		}
	}
	vehicle.UpdatedAt = time.Now().UTC()
	_, err = db.Collection("vehicles").Doc(vehicle.ID).Set(ctx, vehicle)
	return err
}
