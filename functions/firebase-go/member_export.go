package ipace

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

var memberExportRequireUser = requireUser
var memberExportLoadSnapshot = loadMemberSnapshot

func MemberExport(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) {
		return
	}
	if rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	user, err := memberExportRequireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format != "csv" && format != "xlsx" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Choose csv or xlsx"})
		return
	}
	snapshot, err := memberExportLoadSnapshot(r.Context(), user.UID, user.Email)
	if err != nil {
		logEvent("member-export", "error", "snapshot load failed", map[string]any{})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not prepare your export"})
		return
	}

	var body []byte
	var contentType string
	var extension string
	if format == "csv" {
		body, err = buildMemberCSVBundle(snapshot)
		contentType = "application/zip"
		extension = "zip"
	} else {
		body, err = buildMemberWorkbook(snapshot)
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		extension = "xlsx"
	}
	if err != nil {
		logEvent("member-export", "error", "export generation failed", map[string]any{})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not prepare your export"})
		return
	}

	filename := fmt.Sprintf("ipace-owner-data-%s.%s", time.Now().UTC().Format("2006-01-02"), extension)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func buildMemberCSVBundle(snapshot memberSnapshot) ([]byte, error) {
	files := map[string][][]string{
		"membership.csv":                membershipRows(snapshot),
		"vehicles.csv":                  vehicleRows(snapshot),
		"soh-readings.csv":              sohRows(snapshot),
		"service-and-fault-history.csv": serviceRows(snapshot),
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	order := []string{"membership.csv", "vehicles.csv", "soh-readings.csv", "service-and-fault-history.csv"}
	for _, name := range order {
		writer, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		csvWriter := csv.NewWriter(writer)
		for _, row := range files[name] {
			if err := csvWriter.Write(safeSpreadsheetRow(row)); err != nil {
				return nil, err
			}
		}
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeSpreadsheetRow(row []string) []string {
	safe := make([]string, len(row))
	for index, value := range row {
		safe[index] = safeSpreadsheetText(value)
	}
	return safe
}

func safeSpreadsheetText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

func membershipRows(snapshot memberSnapshot) [][]string {
	rows := [][]string{{"Email", "Name", "Country", "Relationship", "Skills", "Contact consent", "Anonymised analysis", "Participation acknowledged", "Joined", "Updated"}}
	for _, record := range snapshot.JoinRecords {
		rows = append(rows, []string{
			snapshot.Email,
			record.Contact.Name,
			record.Contact.Country,
			record.Membership.Relationship,
			strings.Join(record.Membership.Skills, "; "),
			strconv.FormatBool(record.Consents.Contact),
			strconv.FormatBool(record.Consents.AnonymisedAnalysis),
			strconv.FormatBool(record.Consents.NotLegalClaim),
			exportTime(record.CreatedAt),
			exportTime(record.UpdatedAt),
		})
	}
	if len(snapshot.JoinRecords) == 0 {
		rows = append(rows, []string{snapshot.Email, "", "", "", "", "", "", "", "", ""})
	}
	return rows
}

func vehicleRows(snapshot memberSnapshot) [][]string {
	rows := [][]string{{"Vehicle", "Registration", "VIN last 6", "Country", "Model year", "Mileage", "Owned since", "First registered", "Latest SoH %", "SoH measured", "SoH mileage", "SoH source", "Created", "Updated"}}
	for index, record := range snapshot.VehicleRecords {
		rows = append(rows, []string{
			vehicleLabel(record, index),
			record.Vehicle.Registration,
			record.Vehicle.VINLast6,
			record.Vehicle.Country,
			record.Vehicle.ModelYear,
			exportInt(record.Vehicle.Mileage),
			record.Vehicle.OwnedSince,
			record.Vehicle.FirstRegistrationDate,
			exportFloat(record.Battery.StateOfHealth),
			record.Battery.MeasuredAt,
			exportInt(record.Battery.MileageAtMeasurement),
			record.Battery.Source,
			exportTime(record.CreatedAt),
			exportTime(record.UpdatedAt),
		})
	}
	return rows
}

func sohRows(snapshot memberSnapshot) [][]string {
	rows := [][]string{{"Vehicle", "Measured", "State of health %", "Mileage", "Source", "Recorded", "Updated"}}
	readings := append([]batteryReadingRecord(nil), snapshot.BatteryReadings...)
	sort.SliceStable(readings, func(i, j int) bool {
		return readings[i].Battery.MeasuredAt < readings[j].Battery.MeasuredAt
	})
	for _, record := range readings {
		rows = append(rows, []string{
			vehicleLabelByID(snapshot, record.VehicleID),
			record.Battery.MeasuredAt,
			exportFloat(record.Battery.StateOfHealth),
			exportInt(record.Battery.MileageAtMeasurement),
			record.Battery.Source,
			exportTime(record.CreatedAt),
			exportTime(record.UpdatedAt),
		})
	}
	return rows
}

func serviceRows(snapshot memberSnapshot) [][]string {
	rows := [][]string{{"Vehicle", "Type", "Date", "Mileage", "Title", "Description", "Status", "Campaigns", "Final fix date", "Days to final fix", "Courtesy vehicle offered", "Courtesy vehicle provided", "Parts delay", "Warranty cover", "Dispute status", "Created", "Updated"}}
	for _, record := range snapshot.ServiceEvents {
		rows = append(rows, []string{
			vehicleLabelByID(snapshot, record.VehicleID),
			record.EventType,
			record.OccurredAt,
			exportInt(record.Mileage),
			record.Title,
			record.Description,
			record.Status,
			strings.Join(record.Campaigns, "; "),
			record.FinalFixAt,
			exportInt(record.DaysToFinalFix),
			record.CourtesyVehicleOffered,
			record.CourtesyVehicleProvided,
			record.PartsDelay,
			record.WarrantyCover,
			record.DisputeStatus,
			exportTime(record.CreatedAt),
			exportTime(record.UpdatedAt),
		})
	}
	return rows
}

func vehicleLabel(record vehicleRecord, index int) string {
	if record.Vehicle.Registration != "" {
		return record.Vehicle.Registration
	}
	if record.Vehicle.VINLast6 != "" {
		return "VIN …" + record.Vehicle.VINLast6
	}
	return fmt.Sprintf("Vehicle %d", index+1)
}

func vehicleLabelByID(snapshot memberSnapshot, id string) string {
	for index, vehicle := range snapshot.VehicleRecords {
		if vehicle.ID == id {
			return vehicleLabel(vehicle, index)
		}
	}
	return "Vehicle"
}

func exportTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func exportInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func exportFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func buildMemberWorkbook(snapshot memberSnapshot) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	if err := book.SetSheetName("Sheet1", "Summary"); err != nil {
		return nil, err
	}
	for _, name := range []string{"Membership", "Vehicles", "SoH History", "Service & Faults"} {
		if _, err := book.NewSheet(name); err != nil {
			return nil, err
		}
	}

	styles, err := memberWorkbookStyles(book)
	if err != nil {
		return nil, err
	}
	if err := writeSummarySheet(book, snapshot, styles); err != nil {
		return nil, err
	}
	sheets := []struct {
		name string
		rows [][]string
	}{
		{"Membership", membershipRows(snapshot)},
		{"Vehicles", vehicleRows(snapshot)},
		{"SoH History", sohRows(snapshot)},
		{"Service & Faults", serviceRows(snapshot)},
	}
	for _, sheet := range sheets {
		if err := writeDataSheet(book, sheet.name, sheet.rows, styles); err != nil {
			return nil, err
		}
	}
	if err := addMemberWorkbookCharts(book, snapshot); err != nil {
		return nil, err
	}
	book.SetActiveSheet(0)
	var buffer bytes.Buffer
	if err := book.Write(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type memberWorkbookStyleSet struct {
	title  int
	header int
	label  int
	note   int
}

func memberWorkbookStyles(book *excelize.File) (memberWorkbookStyleSet, error) {
	title, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 18},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"12324A"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return memberWorkbookStyleSet{}, err
	}
	header, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"0F766E"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return memberWorkbookStyleSet{}, err
	}
	label, err := book.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "12324A"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E8F3F1"}, Pattern: 1},
	})
	if err != nil {
		return memberWorkbookStyleSet{}, err
	}
	note, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "12324A", Size: 10},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E8F3F1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		return memberWorkbookStyleSet{}, err
	}
	return memberWorkbookStyleSet{title: title, header: header, label: label, note: note}, nil
}

func writeSummarySheet(book *excelize.File, snapshot memberSnapshot, styles memberWorkbookStyleSet) error {
	sheet := "Summary"
	generatedAt := snapshot.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}
	falseValue := false
	_ = book.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: &falseValue})
	_ = book.MergeCell(sheet, "A1", "F1")
	_ = book.SetCellValue(sheet, "A1", "My I-PACE owner data")
	_ = book.SetCellStyle(sheet, "A1", "F1", styles.title)
	_ = book.SetRowHeight(sheet, 1, 32)
	_ = book.SetCellValue(sheet, "A3", "Generated")
	_ = book.SetCellValue(sheet, "B3", generatedAt.UTC().Format("2 January 2006 15:04 UTC"))
	_ = book.SetCellValue(sheet, "A4", "Member email")
	_ = book.SetCellValue(sheet, "B4", safeSpreadsheetText(snapshot.Email))
	_ = book.SetCellValue(sheet, "A6", "Membership records")
	_ = book.SetCellValue(sheet, "B6", len(snapshot.JoinRecords))
	_ = book.SetCellValue(sheet, "A7", "Vehicles")
	_ = book.SetCellValue(sheet, "B7", len(snapshot.VehicleRecords))
	_ = book.SetCellValue(sheet, "A8", "SoH readings")
	_ = book.SetCellValue(sheet, "B8", len(snapshot.BatteryReadings))
	_ = book.SetCellValue(sheet, "A9", "Service and fault events")
	_ = book.SetCellValue(sheet, "B9", len(snapshot.ServiceEvents))
	_ = book.SetCellStyle(sheet, "A3", "A9", styles.label)
	_ = book.SetColWidth(sheet, "A", "A", 27)
	_ = book.SetColWidth(sheet, "B", "B", 34)
	_ = book.SetColWidth(sheet, "D", "K", 15)
	_ = book.SetCellValue(sheet, "A12", "This workbook is a portable copy of the data currently saved in your member account. It excludes internal identity and security hashes.")
	_ = book.MergeCell(sheet, "A12", "B14")
	_ = book.SetCellStyle(sheet, "A12", "B14", styles.note)
	_ = book.SetRowHeight(sheet, 12, 24)
	_ = book.SetRowHeight(sheet, 13, 24)
	_ = book.SetRowHeight(sheet, 14, 24)

	eventCounts := map[string]int{}
	for _, event := range snapshot.ServiceEvents {
		label := event.EventType
		if label == "" {
			label = "Other"
		}
		eventCounts[label]++
	}
	keys := make([]string, 0, len(eventCounts))
	for key := range eventCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_ = book.SetCellValue(sheet, "J1", "Event type")
	_ = book.SetCellValue(sheet, "K1", "Count")
	for index, key := range keys {
		row := index + 2
		_ = book.SetCellValue(sheet, fmt.Sprintf("J%d", row), safeSpreadsheetText(key))
		_ = book.SetCellValue(sheet, fmt.Sprintf("K%d", row), eventCounts[key])
	}
	_ = book.SetColVisible(sheet, "J", false)
	_ = book.SetColVisible(sheet, "K", false)
	return nil
}

func writeDataSheet(book *excelize.File, sheet string, rows [][]string, styles memberWorkbookStyleSet) error {
	falseValue := false
	_ = book.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: &falseValue})
	if len(rows) == 0 {
		return nil
	}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			_ = book.SetCellValue(sheet, cell, memberWorkbookValue(sheet, rowIndex, colIndex, value))
		}
	}
	lastColumn, _ := excelize.ColumnNumberToName(len(rows[0]))
	_ = book.SetCellStyle(sheet, "A1", lastColumn+"1", styles.header)
	_ = book.SetRowHeight(sheet, 1, 30)
	_ = book.SetPanes(sheet, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	_ = book.SetColWidth(sheet, "A", lastColumn, 18)
	if len(rows) > 1 {
		showRows := true
		_ = book.AddTable(sheet, &excelize.Table{
			Range:          fmt.Sprintf("A1:%s%d", lastColumn, len(rows)),
			Name:           strings.ReplaceAll(strings.ReplaceAll(sheet, " ", ""), "&", "And") + "Table",
			StyleName:      "TableStyleMedium2",
			ShowRowStripes: &showRows,
		})
	}
	return nil
}

func memberWorkbookValue(sheet string, rowIndex int, colIndex int, value string) any {
	if rowIndex == 0 || value == "" {
		return safeSpreadsheetText(value)
	}
	numericColumns := map[string]map[int]bool{
		"Vehicles":         {5: true, 8: true, 10: true},
		"SoH History":      {2: true, 3: true},
		"Service & Faults": {3: true, 9: true},
	}
	if numericColumns[sheet][colIndex] {
		if number, err := strconv.ParseFloat(value, 64); err == nil {
			return number
		}
	}
	return safeSpreadsheetText(value)
}

func addMemberWorkbookCharts(book *excelize.File, snapshot memberSnapshot) error {
	if len(snapshot.BatteryReadings) > 0 {
		end := len(snapshot.BatteryReadings) + 1
		if err := book.AddChart("Summary", "D3", &excelize.Chart{
			Type: excelize.Line,
			Series: []excelize.ChartSeries{{
				Name:       "State of health %",
				Categories: fmt.Sprintf("'SoH History'!$B$2:$B$%d", end),
				Values:     fmt.Sprintf("'SoH History'!$C$2:$C$%d", end),
				Line:       excelize.LineOptions{Fill: excelize.Fill{Color: []string{"0F766E"}}, Width: 2},
				Marker:     excelize.ChartMarker{Symbol: "circle", Size: 6},
			}},
			Title:     excelize.ChartTitle{Paragraph: []excelize.RichTextRun{{Text: "State of health history"}}},
			Legend:    excelize.ChartLegend{Position: "bottom"},
			Dimension: excelize.ChartDimension{Width: 640, Height: 300},
			YAxis:     excelize.ChartAxis{Minimum: float64Pointer(0), Maximum: float64Pointer(100)},
		}); err != nil {
			return err
		}
	}
	if len(snapshot.ServiceEvents) > 0 {
		counts := map[string]bool{}
		for _, event := range snapshot.ServiceEvents {
			counts[event.EventType] = true
		}
		end := len(counts) + 1
		if err := book.AddChart("Summary", "D19", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{{
				Name:       "Events",
				Categories: fmt.Sprintf("Summary!$J$2:$J$%d", end),
				Values:     fmt.Sprintf("Summary!$K$2:$K$%d", end),
				Fill:       excelize.Fill{Color: []string{"12324A"}},
			}},
			Title:     excelize.ChartTitle{Paragraph: []excelize.RichTextRun{{Text: "Service and fault events"}}},
			Legend:    excelize.ChartLegend{Position: "none"},
			Dimension: excelize.ChartDimension{Width: 640, Height: 300},
		}); err != nil {
			return err
		}
	}
	return nil
}

func float64Pointer(value float64) *float64 {
	return &value
}
