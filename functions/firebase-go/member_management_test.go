package ipace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfirmedDeleteRequestRequiresTypedConfirmation(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/delete-vehicle", strings.NewReader(`{"id":"vehicle_123","confirmation":"DELETE"}`))
	response := httptest.NewRecorder()
	got, ok := confirmedDeleteRequest(response, request)
	if !ok || got.ID != "vehicle_123" {
		t.Fatalf("confirmed delete = %#v, %t", got, ok)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/delete-vehicle", strings.NewReader(`{"id":"vehicle_123","confirmation":"delete"}`))
	response = httptest.NewRecorder()
	if _, ok := confirmedDeleteRequest(response, request); ok || response.Code != http.StatusBadRequest {
		t.Fatalf("lowercase confirmation accepted: status=%d", response.Code)
	}
}

func TestDeletedVehicleSetsSoftDeleteState(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	got := deletedVehicle(vehicleRecord{ID: "vehicle_123", Review: reviewRecord{Status: "new"}}, now)
	if got.Review.Status != "deleted" || !got.Review.DeletedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Fatalf("deleted vehicle = %#v", got)
	}
}
