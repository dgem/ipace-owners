package ipace

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"cloud.google.com/go/firestore"
)

var adminServiceExportRequireAdmin = requireAdmin
var adminServiceExportLoad = loadServiceExportRecords

type serviceExportData struct {
	Records  []serviceEventRecord
	Joins    []joinRecord
	Vehicles []vehicleRecord
}

func loadServiceExportRecords(ctx context.Context) (serviceExportData, error) {
	data := serviceExportData{}
	db, err := firestoreClient(ctx)
	if err != nil {
		return data, err
	}
	// Service records are the only collection scan. Related vehicles and member
	// redaction terms are fetched directly so export latency does not grow with
	// every member and vehicle in the system.
	if err = readCollection(ctx, db.Collection("serviceEvents").Query, &data.Records); err != nil {
		return data, err
	}
	vehicleIDs := map[string]bool{}
	memberUIDs := map[string]bool{}
	for _, record := range data.Records {
		if recordDeleted(record.Review) {
			continue
		}
		if record.VehicleID != "" {
			vehicleIDs[record.VehicleID] = true
		}
		if record.IdentityUserID != "" {
			memberUIDs[record.IdentityUserID] = true
		}
	}
	data.Vehicles, err = loadServiceExportVehicles(ctx, db, vehicleIDs)
	if err != nil {
		return data, err
	}
	for _, vehicle := range data.Vehicles {
		if vehicle.IdentityUserID != "" {
			memberUIDs[vehicle.IdentityUserID] = true
		}
	}
	data.Joins, err = loadServiceExportMembers(ctx, db, memberUIDs)
	return data, err
}

const serviceExportBatchGetSize = 100

func sortedServiceExportKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		if value = strings.TrimSpace(value); value != "" {
			keys = append(keys, value)
		}
	}
	sort.Strings(keys)
	return keys
}

func loadServiceExportVehicles(ctx context.Context, db *firestore.Client, ids map[string]bool) ([]vehicleRecord, error) {
	keys := sortedServiceExportKeys(ids)
	records := make([]vehicleRecord, 0, len(keys))
	for start := 0; start < len(keys); start += serviceExportBatchGetSize {
		end := min(start+serviceExportBatchGetSize, len(keys))
		refs := make([]*firestore.DocumentRef, 0, end-start)
		for _, id := range keys[start:end] {
			refs = append(refs, db.Collection("vehicles").Doc(id))
		}
		documents, err := db.GetAll(ctx, refs)
		if err != nil {
			return nil, err
		}
		for _, document := range documents {
			if !document.Exists() {
				continue
			}
			var record vehicleRecord
			if err := document.DataTo(&record); err != nil {
				return nil, err
			}
			if record.ID == "" {
				record.ID = document.Ref.ID
			}
			records = append(records, record)
		}
	}
	return records, nil
}

type serviceExportMemberRecord struct {
	IdentityUserID string `firestore:"identityUserId"`
	EmailHash      string `firestore:"emailHash"`
	DisplayName    string `firestore:"displayName"`
}

func loadServiceExportMembers(ctx context.Context, db *firestore.Client, ids map[string]bool) ([]joinRecord, error) {
	keys := sortedServiceExportKeys(ids)
	records := make([]joinRecord, 0, len(keys))
	for start := 0; start < len(keys); start += serviceExportBatchGetSize {
		end := min(start+serviceExportBatchGetSize, len(keys))
		refs := make([]*firestore.DocumentRef, 0, end-start)
		for _, id := range keys[start:end] {
			refs = append(refs, db.Collection("members").Doc(id))
		}
		documents, err := db.GetAll(ctx, refs)
		if err != nil {
			return nil, err
		}
		for _, document := range documents {
			if !document.Exists() {
				continue
			}
			var member serviceExportMemberRecord
			if err := document.DataTo(&member); err != nil {
				return nil, err
			}
			if member.IdentityUserID == "" {
				member.IdentityUserID = document.Ref.ID
			}
			records = append(records, joinRecord{
				IdentityUserID: member.IdentityUserID,
				UserEmailHash:  member.EmailHash,
				Contact:        contactRecord{Name: member.DisplayName},
			})
		}
	}
	return records, nil
}

// AdminServiceExport exports allowlisted fields with narrative redaction. It
// uses member and vehicle data only to redact text, never as output columns.
func AdminServiceExport(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if _, err := adminServiceExportRequireAdmin(r.Context(), r); err != nil {
		writeAdminAuthorizationError(w, err)
		return
	}
	data, err := adminServiceExportLoad(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not prepare service export. Please try again."})
		return
	}
	body, count, err := buildAdminServiceCSV(data)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not prepare service export. Please try again."})
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"ipace-service-data-redacted-%s.csv\"", time.Now().UTC().Format("2006-01-02")))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	logEvent("admin-service-export", "info", "CSV prepared", map[string]any{"recordCount": count})
	_, _ = w.Write(body)
}

func exportServiceMonth(value string) string {
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return ""
	}
	return date.Format("2006-01")
}

func exportMileageBand(value *int) string {
	if value == nil || *value < 0 || *value > 500000 {
		return ""
	}
	start := *value / 5000 * 5000
	return fmt.Sprintf("%d-%d", start, start+4999)
}

func buildAdminServiceCSV(data serviceExportData) ([]byte, int, error) {
	rows := [][]string{}
	redact := serviceExportRedactor(data)
	for _, record := range data.Records {
		if recordDeleted(record.Review) {
			continue
		}
		// Validate enums again at the export boundary, including legacy stored data.
		row := []string{
			cleanEnum(record.EventType, serviceEventTypeValues),
			exportServiceMonth(record.OccurredAt),
			exportMileageBand(record.Mileage),
			cleanEnum(record.Status, serviceEventStatusValues),
			strings.Join(cleanEnums(record.Campaigns, serviceEventCampaignValues), " | "),
			exportBool(record.ServiceProviderAuthorised),
			exportServiceMonth(record.FinalFixAt),
			exportInt(serviceEventResolutionDays(record.OccurredAt, record.FinalFixAt)),
			cleanEnum(record.CourtesyVehicleOffered, serviceEventYesNoValues),
			cleanEnum(record.CourtesyVehicleProvided, serviceEventYesNoValues),
			cleanEnum(record.PartsDelay, serviceEventPartsDelayValues),
			exportBool(record.GoodwillPayment),
			exportMileageBand(record.MilesDrivenWhilstFaulty),
			cleanEnum(record.WarrantyCover, serviceEventWarrantyCoverValues),
			cleanEnum(record.DisputeStatus, serviceEventDisputeStatusValues),
			redact(record.ServiceProviderName),
			redact(record.Title),
			redact(record.Description),
			"Private admin analysis; automated redaction",
		}
		for i := range row {
			row[i] = safeSpreadsheetText(row[i])
		}
		rows = append(rows, row)
	}
	// Sort only exported values, never identifiers or private timestamps.
	sort.Slice(rows, func(i, j int) bool { return strings.Join(rows[i], "\x00") < strings.Join(rows[j], "\x00") })
	var body bytes.Buffer
	writer := csv.NewWriter(&body)
	_ = writer.Write([]string{"event_type", "event_month", "mileage_band_miles", "status", "campaigns", "authorised_service_provider", "final_fix_month", "days_to_final_fix", "courtesy_vehicle_offered", "courtesy_vehicle_provided", "parts_delay", "goodwill_payment", "miles_driven_whilst_faulty_band", "warranty_cover", "dispute_status", "service_provider_name", "title", "description", "privacy_review"})
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
	return body.Bytes(), len(rows), writer.Error()
}

// This is an aid to human review, not a guarantee of narrative anonymisation.
// Unknown people, addresses and identifying circumstances may remain.
func serviceExportRedactor(data serviceExportData) func(string) string {
	known := map[string]bool{}
	relevantUIDs := map[string]bool{}
	relevantVehicleIDs := map[string]bool{}
	relevantEmailHashes := map[string]bool{}
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			known[value] = true
		}
	}
	for _, record := range data.Records {
		if recordDeleted(record.Review) {
			continue
		}
		if record.IdentityUserID != "" {
			relevantUIDs[record.IdentityUserID] = true
		}
		if record.VehicleID != "" {
			relevantVehicleIDs[record.VehicleID] = true
		}
		add(record.ID)
		add(record.IdentityUserID)
		add(record.VehicleID)
	}
	for _, vehicle := range data.Vehicles {
		if !relevantVehicleIDs[vehicle.ID] && !relevantUIDs[vehicle.IdentityUserID] {
			continue
		}
		if vehicle.IdentityUserID != "" {
			relevantUIDs[vehicle.IdentityUserID] = true
		}
		if vehicle.UserEmailHash != "" {
			relevantEmailHashes[vehicle.UserEmailHash] = true
		}
		add(vehicle.ID)
		add(vehicle.IdentityUserID)
		add(vehicle.UserEmailHash)
		add(vehicle.Vehicle.VINHash)
		add(vehicle.Vehicle.VINLast6)
		add(vehicle.Vehicle.Registration)
	}
	for _, join := range data.Joins {
		if !relevantUIDs[join.IdentityUserID] && !relevantEmailHashes[join.UserEmailHash] {
			continue
		}
		add(join.Contact.Name)
		add(join.Contact.Email)
		add(join.IdentityUserID)
		add(join.UserEmailHash)
		add(join.ID)
		for _, part := range strings.Fields(join.Contact.Name) {
			if len([]rune(part)) >= 3 {
				add(part)
			}
		}
	}
	values := make([]string, 0, len(known))
	for value := range known {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	patterns := make([]string, 0, len(values))
	for _, value := range values {
		patterns = append(patterns, regexp.QuoteMeta(value))
	}
	var names *regexp.Regexp
	if len(patterns) > 0 {
		names = regexp.MustCompile("(?i)(?:" + strings.Join(patterns, "|") + ")")
	}
	// Registrations may appear with different spacing from the stored form.
	registrationPatterns := []string{}
	for _, vehicle := range data.Vehicles {
		if !relevantVehicleIDs[vehicle.ID] && !relevantUIDs[vehicle.IdentityUserID] {
			continue
		}
		registration := strings.Map(func(r rune) rune {
			if r == ' ' || r == '-' {
				return -1
			}
			return r
		}, vehicle.Vehicle.Registration)
		if len(registration) < 2 {
			continue
		}
		letters := []string{}
		for _, r := range registration {
			letters = append(letters, regexp.QuoteMeta(string(r)))
		}
		registrationPatterns = append(registrationPatterns, strings.Join(letters, "[ -]*"))
	}
	var registrations *regexp.Regexp
	if len(registrationPatterns) > 0 {
		registrations = regexp.MustCompile("(?i)(?:" + strings.Join(registrationPatterns, "|") + ")")
	}
	return func(value string) string {
		for _, pattern := range serviceExportPrivatePatterns {
			value = pattern.ReplaceAllString(value, "[redacted]")
		}
		if registrations != nil {
			value = registrations.ReplaceAllString(value, "[redacted]")
		}
		if names != nil {
			value = redactKnownServiceTerms(value, names)
		}
		return value
	}
}

var serviceExportPrivatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9.-]+\.[a-z]{2,}`),
	regexp.MustCompile(`(?i)https?://[^\s<>]+|www\.[^\s<>]+`),
	regexp.MustCompile(`(?i)\b[A-HJ-NPR-Z0-9]{17}\b`),
	regexp.MustCompile(`(?i)\b[A-Z]{2}[ -]*[0-9]{2}[ -]*[A-Z]{3}\b`),
	regexp.MustCompile(`(?i)\b[A-Z]{1,2}[0-9][A-Z0-9]?\s*[0-9][A-Z]{2}\b`),
	regexp.MustCompile(`\+?[0-9][0-9 ().-]{6,}[0-9]`),
}

// Match whole terms so a member named Ann does not corrupt "cannot" or "annual".
func redactKnownServiceTerms(value string, pattern *regexp.Regexp) string {
	var out strings.Builder
	start := 0
	for _, match := range pattern.FindAllStringIndex(value, -1) {
		before, _ := utf8.DecodeLastRuneInString(value[:match[0]])
		after, _ := utf8.DecodeRuneInString(value[match[1]:])
		if unicode.IsLetter(before) || unicode.IsDigit(before) || unicode.IsLetter(after) || unicode.IsDigit(after) {
			continue
		}
		out.WriteString(value[start:match[0]])
		out.WriteString("[redacted]")
		start = match[1]
	}
	out.WriteString(value[start:])
	return out.String()
}
