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
	"os"
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
	yes := true
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
			Membership:    membershipRecord{Relationship: "current-owner-one", Skills: []string{"technical"}, DisplayName: "=ALIAS()"},
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
			ServiceProviderName: "Example Jaguar Service Centre", ServiceProviderPostcode: "AB1 2CD",
			ServiceProviderAuthorised: &yes, GoodwillPayment: &yes, MilesDrivenWhilstFaulty: &mileage,
			ExecutiveOfficeAcknowledgedAt: "2026-07-02", ExecutiveOfficeRepliedAt: "2026-07-04",
			ExecutiveOfficeOutcome: "Further inspection offered", ExecutiveOfficeSatisfied: "awaiting-outcome",
			CreatedAt: now, UpdatedAt: now,
		}},
	}
}

func memberExperienceExportFixture() memberStory {
	now := time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC)
	return memberStory{
		ID: "story-internal", IdentityUserID: "internal-uid-must-not-export", ContactName: "Private Name", ContactEmail: "private@example.test",
		Title: "Three modules and still waiting", DisplayName: "Alex Owner", BodyMarkdown: "The **battery failed** without warning.", Status: "submitted",
		Consent:    memberStoryConsent{Website: true, ThirdParty: true, Copyright: true, Truthful: true, Version: storyConsentVersion, RecordedAt: now},
		Evidence:   memberStoryEvidence{VehicleLabel: "IPACE1", ModelYear: "2020", ServiceEventCount: 3, SOHReadingCount: 2, ExecutiveContacts: 1},
		Media:      []memberStoryMedia{{ID: "media-internal", Object: "private/member/story/photo.jpg", Name: "battery-warning.jpg", ContentType: "image/jpeg", Size: 1024, Status: "ready"}},
		AdminNotes: "private admin note", MemberMessage: "Please add the date of the repeat failure.", CreatedAt: now, UpdatedAt: now,
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
	body, err := buildMemberCSVBundle(memberExportFixture(), memberExperienceExportFixture())
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
	want := []string{"member-experiences.csv", "membership.csv", "service-and-fault-history.csv", "soh-readings.csv", "vehicles.csv"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("files = %v, want %v", names, want)
	}
	exported := combined.String()
	for _, forbidden := range []string{"internal-uid-must-not-export", "secret-email-hash", "secret-vin-hash", "private admin note", "private/member/story/photo.jpg", "private@example.test"} {
		if strings.Contains(exported, forbidden) {
			t.Fatalf("export exposed internal value %q", forbidden)
		}
	}
	if !strings.Contains(exported, "'=FORMULA()") || !strings.Contains(exported, "'+not a formula") {
		t.Fatal("spreadsheet formula injection values were not neutralised")
	}
	if !strings.Contains(exported, "Executive Office acknowledgement") || !strings.Contains(exported, "Further inspection offered") {
		t.Fatal("Executive Office history was not included in the member export")
	}
	if !strings.Contains(exported, "Experience (Markdown)") || !strings.Contains(exported, "The **battery failed** without warning.") || !strings.Contains(exported, "battery-warning.jpg") {
		t.Fatal("member experience was not included in the member export")
	}
}

func TestMemberWorkbookContainsSheetsAndNativeCharts(t *testing.T) {
	body, err := buildMemberWorkbook(memberExportFixture(), memberExperienceExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	wantSheets := []string{"Summary", "Membership", "Vehicles", "SoH History", "Service & Faults", "Experiences"}
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
	displayName, _ := book.GetCellValue("Membership", "C2")
	if displayName != "'=ALIAS()" {
		t.Fatalf("unsafe display-name cell = %q", displayName)
	}
	experience, _ := book.GetCellValue("Experiences", "D2")
	if experience != "The **battery failed** without warning." {
		t.Fatalf("experience body = %q", experience)
	}
	count, _ := book.GetCellValue("Summary", "B10")
	if count != "1" {
		t.Fatalf("experience summary = %q", count)
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

func TestPublicSampleWorkbookUsesFictionalDataAndProductionLayout(t *testing.T) {
	body, err := os.ReadFile("../../public/downloads/sample-ipace-owner-data.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()

	wantSheets := []string{"Summary", "Membership", "Vehicles", "SoH History", "Service & Faults", "Experiences"}
	if strings.Join(book.GetSheetList(), ",") != strings.Join(wantSheets, ",") {
		t.Fatalf("sample sheets = %v, want %v", book.GetSheetList(), wantSheets)
	}
	for cell, want := range map[string]string{"B3": "28 July 2026 09:00 UTC", "B7": "2", "B8": "5", "B9": "4"} {
		got, err := book.GetCellValue("Summary", cell)
		if err != nil || got != want {
			t.Fatalf("Summary!%s = %q, %v; want %q", cell, got, err, want)
		}
	}
	email, _ := book.GetCellValue("Membership", "A2")
	if email != "alex.owner@example.test" {
		t.Fatalf("sample email = %q", email)
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
		t.Fatalf("sample chart parts = %d, want 2", chartParts)
	}
}

func TestMemberExportResponseHeaders(t *testing.T) {
	originalUser := memberExportRequireUser
	originalSnapshot := memberExportLoadSnapshot
	originalStories := memberExportLoadStories
	memberExportRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "uid", Email: "owner@example.test"}, nil
	}
	memberExportLoadSnapshot = func(context.Context, string, string) (memberSnapshot, error) {
		return memberExportFixture(), nil
	}
	memberExportLoadStories = func(context.Context, string) ([]memberStory, error) {
		return []memberStory{memberExperienceExportFixture()}, nil
	}
	t.Cleanup(func() {
		memberExportRequireUser = originalUser
		memberExportLoadSnapshot = originalSnapshot
		memberExportLoadStories = originalStories
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

func TestMemberWorkbookUsesWorksheetFiltersAndExplicitBanding(t *testing.T) {
	snapshot := memberExportFixture()
	snapshot.ServiceEvents = append(snapshot.ServiceEvents, snapshot.ServiceEvents[0])
	body, err := buildMemberWorkbook(snapshot, memberExperienceExportFixture())
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	firstStyle, err := book.GetCellStyle("Service & Faults", "A2")
	if err != nil {
		t.Fatal(err)
	}
	secondStyle, err := book.GetCellStyle("Service & Faults", "A3")
	if err != nil || firstStyle == secondStyle {
		t.Fatal("alternating row styles missing")
	}
	style, err := book.GetStyle(firstStyle)
	if err != nil || len(style.Fill.Color) == 0 || style.Fill.Color[0] != "E8F3F1" {
		t.Fatalf("stripe style=%v err=%v", style, err)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	filters := 0
	for _, file := range archive.File {
		if strings.HasPrefix(file.Name, "xl/tables/") {
			t.Fatal("unexpected structured table part")
		}
		if !strings.HasPrefix(file.Name, "xl/worksheets/sheet") || !strings.HasSuffix(file.Name, ".xml") {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		xml, err := io.ReadAll(stream)
		stream.Close()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(xml, []byte("<autoFilter ")) {
			filters++
		}
	}
	if filters != 5 {
		t.Fatalf("filterable sheets=%d want 5", filters)
	}
}
