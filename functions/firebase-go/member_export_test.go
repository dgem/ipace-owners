package ipace

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func memberExportFixture() memberSnapshot {
	mileage := 54000
	sohMileage := 53500
	soh := 89.5
	days := 12
	now := time.Date(2026, 7, 27, 10, 30, 0, 0, time.UTC)
	return memberSnapshot{
		IdentityUserID: "internal-uid-must-not-export",
		Email:          "owner@example.test",
		GeneratedAt:    now,
		JoinRecords: []joinRecord{{
			ID:            "join-internal",
			CreatedAt:     now.AddDate(-1, 0, 0),
			UpdatedAt:     now,
			UserEmailHash: "secret-email-hash",
			Contact:       contactRecord{Name: "=FORMULA()", Country: "GB"},
			Membership:    membershipRecord{Relationship: "current-owner-one", Skills: []string{"technical"}},
			Consents:      consentRecord{Contact: true, NotLegalClaim: true, AnonymisedAnalysis: true},
		}},
		VehicleRecords: []vehicleRecord{{
			ID:        "vehicle-one",
			CreatedAt: now.AddDate(0, -3, 0),
			UpdatedAt: now,
			Vehicle: vehicleDetails{
				VINHash: "secret-vin-hash", VINLast6: "123456", Registration: "IPACE1",
				Country: "GB", ModelYear: "2020", Mileage: &mileage,
			},
			Battery: batteryDetails{StateOfHealth: &soh, MeasuredAt: "2026-07-20", MileageAtMeasurement: &sohMileage, Source: "dealer-report"},
		}},
		BatteryReadings: []batteryReadingRecord{{
			ID:        "reading-one",
			VehicleID: "vehicle-one",
			CreatedAt: now,
			UpdatedAt: now,
			Battery:   batteryDetails{StateOfHealth: &soh, MeasuredAt: "2026-07-20", MileageAtMeasurement: &sohMileage, Source: "dealer-report"},
		}},
		ServiceEvents: []serviceEventRecord{{
			ID: "event-one", VehicleID: "vehicle-one", EventType: "fault",
			OccurredAt: "2026-07-01", Mileage: &mileage, Title: "Traction warning",
			Description: "+not a formula", Status: "resolved", DaysToFinalFix: &days,
			CreatedAt: now, UpdatedAt: now,
		}},
	}
}

func TestMemberExportRequiresAuthentication(t *testing.T) {
	original := memberExportRequireUser
	memberExportRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return nil, errors.New("no user")
	}
	t.Cleanup(func() { memberExportRequireUser = original })

	request := httptest.NewRequest(http.MethodGet, "/api/member-export?format=csv", nil)
	response := httptest.NewRecorder()
	MemberExport(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestMemberExportValidatesMethodAndFormat(t *testing.T) {
	original := memberExportRequireUser
	memberExportRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "uid", Email: "owner@example.test"}, nil
	}
	t.Cleanup(func() { memberExportRequireUser = original })

	for _, test := range []struct {
		method string
		url    string
		status int
	}{
		{http.MethodPost, "/api/member-export?format=csv", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/member-export?format=pdf", http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		MemberExport(response, httptest.NewRequest(test.method, test.url, nil))
		if response.Code != test.status {
			t.Fatalf("%s %s status = %d, want %d", test.method, test.url, response.Code, test.status)
		}
	}
}

func TestMemberCSVBundleHasSeparateSafeDatasets(t *testing.T) {
	body, err := buildMemberCSVBundle(memberExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var combined strings.Builder
	for _, file := range reader.File {
		names = append(names, file.Name)
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(data)
		if _, err := csv.NewReader(bytes.NewReader(data)).ReadAll(); err != nil {
			t.Fatalf("%s is not valid CSV: %v", file.Name, err)
		}
	}
	sort.Strings(names)
	want := []string{"membership.csv", "service-and-fault-history.csv", "soh-readings.csv", "vehicles.csv"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", names, want)
	}
	exported := combined.String()
	for _, forbidden := range []string{"internal-uid-must-not-export", "secret-email-hash", "secret-vin-hash"} {
		if strings.Contains(exported, forbidden) {
			t.Fatalf("export exposed internal value %q", forbidden)
		}
	}
	if !strings.Contains(exported, "'=FORMULA()") || !strings.Contains(exported, "'+not a formula") {
		t.Fatal("spreadsheet formula injection values were not neutralised")
	}
}

func TestMemberWorkbookContainsSheetsAndNativeCharts(t *testing.T) {
	body, err := buildMemberWorkbook(memberExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	wantSheets := []string{"Summary", "Membership", "Vehicles", "SoH History", "Service & Faults"}
	if strings.Join(book.GetSheetList(), ",") != strings.Join(wantSheets, ",") {
		t.Fatalf("sheets = %v, want %v", book.GetSheetList(), wantSheets)
	}
	value, err := book.GetCellValue("Summary", "B7")
	if err != nil || value != "1" {
		t.Fatalf("vehicle summary = %q, %v", value, err)
	}
	name, _ := book.GetCellValue("Membership", "B2")
	if name != "'=FORMULA()" {
		t.Fatalf("unsafe membership cell = %q", name)
	}
	if cellType, _ := book.GetCellType("SoH History", "C2"); cellType != excelize.CellTypeNumber && cellType != excelize.CellTypeUnset {
		t.Fatalf("SoH chart source type = %v, want number", cellType)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	chartParts := 0
	for _, file := range archive.File {
		if strings.HasPrefix(file.Name, "xl/charts/chart") {
			chartParts++
		}
	}
	if chartParts != 2 {
		t.Fatalf("chart parts = %d, want 2", chartParts)
	}
	for _, file := range archive.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"internal-uid-must-not-export", "secret-email-hash", "secret-vin-hash"} {
			if bytes.Contains(data, []byte(forbidden)) {
				t.Fatalf("workbook exposed internal value %q in %s", forbidden, file.Name)
			}
		}
	}
}

func TestMemberExportResponseHeaders(t *testing.T) {
	originalUser := memberExportRequireUser
	originalSnapshot := memberExportLoadSnapshot
	memberExportRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "uid", Email: "owner@example.test"}, nil
	}
	memberExportLoadSnapshot = func(context.Context, string, string) (memberSnapshot, error) {
		return memberExportFixture(), nil
	}
	t.Cleanup(func() {
		memberExportRequireUser = originalUser
		memberExportLoadSnapshot = originalSnapshot
	})

	response := httptest.NewRecorder()
	MemberExport(response, httptest.NewRequest(http.MethodGet, "/api/member-export?format=xlsx", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if !strings.Contains(response.Header().Get("Content-Disposition"), ".xlsx") {
		t.Fatalf("Content-Disposition = %q", response.Header().Get("Content-Disposition"))
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
}
