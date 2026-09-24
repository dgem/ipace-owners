package ipace

import (
	"context"
	"encoding/csv"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAdminServiceCSVPrivacyAndStructuredFields(t *testing.T) {
	record := memberExportFixture().ServiceEvents[0]
	record.IdentityUserID = "private-uid"
	record.Title = "Owner Name owner@example.test"
	record.Description = "VIN123456 registration AB12CDE telephone 07123456789"
	record.EventType = "=private-formula"
	record.Campaigns = []string{"H441", "private-campaign"}
	record.FinalFixAt = "2026-07-13"
	record.CourtesyVehicleOffered = "yes"
	record.PartsDelay = "up-to-1-week"
	record.WarrantyCover = "battery-warranty"
	record.DisputeStatus = "resolved-after-escalation"
	deleted := record
	deleted.Review.Status = "deleted"
	deletedAt := record
	deletedAt.Review.DeletedAt = time.Now()
	body, count, err := buildAdminServiceCSV(serviceExportData{
		Records: []serviceEventRecord{record, deleted, deletedAt},
		Joins:   []joinRecord{{IdentityUserID: "private-uid", Contact: contactRecord{Name: "Owner Name"}}},
		Vehicles: []vehicleRecord{{
			ID: record.VehicleID, IdentityUserID: "private-uid", Vehicle: vehicleDetails{VINLast6: "VIN123456"},
		}},
	})
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil || len(rows) != 2 || len(rows[1]) != 19 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	for _, private := range []string{"private", "Owner Name", "owner@", "VIN123456", "AB12CDE", "07123456789", "AB1 2CD", "event-one", "vehicle-one", "2026-07-01", "54000"} {
		if strings.Contains(string(body), private) {
			t.Fatalf("leaked %q", private)
		}
	}
	want := []string{"", "2026-07", "50000-54999", "resolved", "H441", "Yes", "2026-07", "12", "yes", "", "up-to-1-week", "Yes", "50000-54999", "battery-warranty", "resolved-after-escalation"}
	if rows[1][15] != "Example Jaguar Service Centre" || !strings.Contains(rows[1][16], "[redacted]") {
		t.Fatal("provider or redacted narrative missing")
	}
	for i := range want {
		if rows[1][i] != want[i] {
			t.Errorf("column %s=%q want %q", rows[0][i], rows[1][i], want[i])
		}
	}
}

func TestAdminServiceCSVEmptyAndInvalidValues(t *testing.T) {
	body, count, err := buildAdminServiceCSV(serviceExportData{})
	if err != nil || count != 0 || strings.Count(string(body), "\n") != 1 {
		t.Fatalf("empty export: %q %d %v", body, count, err)
	}
	negative := -1
	if exportMileageBand(&negative) != "" || exportMileageBand(nil) != "" || exportServiceMonth("private-name") != "" {
		t.Fatal("invalid values exported")
	}
	zero := 0
	if exportMileageBand(&zero) != "0-4999" {
		t.Fatal("zero mileage lost")
	}
}

func TestAdminServiceExportAuthorizationAndFailures(t *testing.T) {
	oldAuth, oldLoad := adminServiceExportRequireAdmin, adminServiceExportLoad
	t.Cleanup(func() { adminServiceExportRequireAdmin, adminServiceExportLoad = oldAuth, oldLoad })
	for _, tc := range []struct {
		name, method     string
		authStatus, want int
		loadErr          bool
	}{
		{"signed out", "GET", 401, 401, false},
		{"member", "GET", 403, 403, false},
		{"wrong method", "POST", 0, 405, false},
		{"admin", "GET", 0, 200, false},
		{"store unavailable", "GET", 0, 500, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loaded := false
			adminServiceExportRequireAdmin = func(context.Context, *http.Request) (*firebaseUser, error) {
				if tc.authStatus != 0 {
					return nil, authorizationFailure(tc.authStatus, "Access denied", nil)
				}
				return &firebaseUser{UID: "admin"}, nil
			}
			adminServiceExportLoad = func(context.Context) (serviceExportData, error) {
				loaded = true
				if tc.loadErr {
					return serviceExportData{}, errors.New("private database detail")
				}
				return serviceExportData{Records: []serviceEventRecord{{IdentityUserID: "member-a", EventType: "service"}, {IdentityUserID: "member-b", EventType: "fault"}}}, nil
			}
			w := httptest.NewRecorder()
			Api(w, httptest.NewRequest(tc.method, "/api/admin/service-export", nil))
			if w.Code != tc.want {
				t.Fatalf("status %d want %d: %s", w.Code, tc.want, w.Body)
			}
			if tc.want != 200 && tc.want != 500 && loaded {
				t.Fatal("unauthorized data load")
			}
			if strings.Contains(w.Body.String(), "private") || strings.Contains(w.Body.String(), "member-") {
				t.Fatal("private data leaked")
			}
			if tc.want == 200 {
				if w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Header().Get("Content-Disposition"), ".csv") {
					t.Fatal("missing download headers")
				}
				rows, err := csv.NewReader(w.Body).ReadAll()
				if err != nil || len(rows) != 3 {
					t.Fatalf("all-member rows: %v %v", rows, err)
				}
			}
		})
	}
}

func TestServiceExportRedactsNarrativeWithoutLosingAnalysis(t *testing.T) {
	data := serviceExportData{
		Joins:    []joinRecord{{IdentityUserID: "member-one", Contact: contactRecord{Name: "Ann Example", Email: "ann@example.test"}}},
		Vehicles: []vehicleRecord{{ID: "vehicle-one", IdentityUserID: "member-one", Vehicle: vehicleDetails{Registration: "P4 ANN", VINLast6: "654321"}}},
		Records:  []serviceEventRecord{{IdentityUserID: "member-one", VehicleID: "vehicle-one", EventType: "repair", ServiceProviderName: "Example Jaguar", Title: "=HYPERLINK(\"test\")", Description: "Ann cannot attend annual service. Call +44 (0) 7123 456789 or ann@example.test. P4ANN, AB12 CDE, SAJAA1B12J1F12345, 654321, SW1A 1AA. See https://example.test/private. Battery repair took 12 days, H441."}},
	}
	body, _, err := buildAdminServiceCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[1][16] != "'=HYPERLINK(\"test\")" {
		t.Fatalf("formula not neutralised: %q", rows[1][16])
	}
	description := rows[1][17]
	for _, private := range []string{"Ann", "7123", "ann@", "P4ANN", "AB12", "SAJAA", "654321", "SW1A", "https://"} {
		if strings.Contains(description, private) {
			t.Errorf("leaked %q: %s", private, description)
		}
	}
	for _, retained := range []string{"cannot", "annual service", "Battery repair took 12 days, H441"} {
		if !strings.Contains(description, retained) {
			t.Errorf("lost analysis detail %q: %s", retained, description)
		}
	}
}

func TestServiceExportRedactorUsesOnlyRelatedMemberData(t *testing.T) {
	data := serviceExportData{
		Records: []serviceEventRecord{{IdentityUserID: "included", VehicleID: "included-vehicle", Title: "Included Owner visited Unrelated Person"}},
		Joins: []joinRecord{
			{IdentityUserID: "included", Contact: contactRecord{Name: "Included Owner"}},
			{IdentityUserID: "unrelated", Contact: contactRecord{Name: "Unrelated Person"}},
		},
		Vehicles: []vehicleRecord{
			{ID: "included-vehicle", IdentityUserID: "included"},
			{ID: "unrelated-vehicle", IdentityUserID: "unrelated", Vehicle: vehicleDetails{Registration: "AB12 CDE"}},
		},
	}
	body, _, err := buildAdminServiceCSV(data)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "Included Owner") || !strings.Contains(text, "Unrelated Person") {
		t.Fatalf("redaction scope is wrong: %s", text)
	}
}

func TestServiceExportFormulaPrefixWithWhitespace(t *testing.T) {
	for _, text := range []string{" =1+1", "\t+1", "\r\n@SUM(1)", "-1+1"} {
		body, _, err := buildAdminServiceCSV(serviceExportData{Records: []serviceEventRecord{{Title: text}}})
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
		if err != nil || !strings.HasPrefix(rows[1][16], "'") {
			t.Fatalf("unsafe CSV %q: %v", body, err)
		}
	}
}
