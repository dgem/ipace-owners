package ipace

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"time"
)

const publicStatsSchemaVersion = 3

var publicRegistrationDate = time.Date(2026, time.July, 16, 0, 0, 0, 0, time.UTC)

func PublicStats(w http.ResponseWriter, r *http.Request) {
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
	snapshot, err := readPublicStatsObject(r.Context())
	if err == nil && snapshot.SchemaVersion < publicStatsSchemaVersion {
		err = fmt.Errorf("public statistics snapshot schema is outdated")
	}
	if err != nil {
		snapshot, err = buildPublicStatsSnapshot(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load public statistics"})
			return
		}
		_ = writePublicStatsObject(r.Context(), snapshot)
	}
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
	writeJSON(w, http.StatusOK, snapshot)
}

func regeneratePublicStatsSnapshot(ctx context.Context) error {
	snapshot, err := buildPublicStatsSnapshot(ctx)
	if err != nil {
		return err
	}
	return writePublicStatsObject(ctx, snapshot)
}

func buildPublicStatsSnapshot(ctx context.Context) (publicStatsSnapshot, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return publicStatsSnapshot{}, err
	}
	joins := []joinRecord{}
	vehicles := []vehicleRecord{}
	readings := []batteryReadingRecord{}
	if err := readCollection(ctx, db.Collection("joinSubmissions").Query, &joins); err != nil {
		return publicStatsSnapshot{}, err
	}
	if err := readCollection(ctx, db.Collection("vehicles").Query, &vehicles); err != nil {
		return publicStatsSnapshot{}, err
	}
	if err := readCollection(ctx, db.Collection("batteryReadings").Query, &readings); err != nil {
		return publicStatsSnapshot{}, err
	}
	consented := map[string]bool{}
	for _, record := range joins {
		if record.Consents.AnonymisedAnalysis && record.Review.Status != "excluded" {
			consented[record.UserEmailHash] = true
		}
	}
	registeredMembers := registeredMembersSince(joins, publicRegistrationDate)
	return aggregatePublicStats(vehicles, readings, consented, registeredMembers, time.Now().UTC()), nil
}

func registeredMembersSince(joins []joinRecord, launchDate time.Time) int {
	members := map[string]bool{}
	for _, record := range joins {
		if record.UserEmailHash == "" || record.CreatedAt.Before(launchDate) || record.Review.Status == "excluded" {
			continue
		}
		members[record.UserEmailHash] = true
	}
	return len(members)
}

func aggregatePublicStats(vehicles []vehicleRecord, readings []batteryReadingRecord, consented map[string]bool, registeredMembers int, generatedAt time.Time) publicStatsSnapshot {
	filteredVehicles := map[string]vehicleRecord{}
	owners := map[string]bool{}
	modelYears := map[string]int{}
	for _, vehicle := range vehicles {
		if vehicle.Review.Status == "excluded" || !consented[vehicle.UserEmailHash] {
			continue
		}
		filteredVehicles[vehicle.ID] = vehicle
		owners[vehicle.IdentityUserID] = true
		if vehicle.Vehicle.ModelYear != "" {
			modelYears[vehicle.Vehicle.ModelYear]++
		}
	}

	byVehicle := map[string][]batteryReadingRecord{}
	for _, reading := range readings {
		if _, ok := filteredVehicles[reading.VehicleID]; !ok || reading.Review.Status == "excluded" || reading.Battery.StateOfHealth == nil {
			continue
		}
		byVehicle[reading.VehicleID] = append(byVehicle[reading.VehicleID], reading)
	}
	for id, vehicle := range filteredVehicles {
		if len(byVehicle[id]) == 0 {
			if reading := initialBatteryReading(vehicle); reading != nil {
				byVehicle[id] = append(byVehicle[id], *reading)
			}
		}
	}

	latestValues := []float64{}
	changeValues := []float64{}
	readingCount := 0
	repeatVehicles := 0
	for _, vehicleReadings := range byVehicle {
		sort.Slice(vehicleReadings, func(i, j int) bool {
			iDate := normalisedMeasurementDate(vehicleReadings[i])
			jDate := normalisedMeasurementDate(vehicleReadings[j])
			if iDate.Equal(jDate) {
				return vehicleReadings[i].CreatedAt.Before(vehicleReadings[j].CreatedAt)
			}
			return iDate.Before(jDate)
		})
		readingCount += len(vehicleReadings)
		latest := vehicleReadings[len(vehicleReadings)-1].Battery.StateOfHealth
		if latest != nil {
			latestValues = append(latestValues, *latest)
		}
		if len(vehicleReadings) >= 2 {
			first := vehicleReadings[0].Battery.StateOfHealth
			firstDate := normalisedMeasurementDate(vehicleReadings[0])
			latestDate := normalisedMeasurementDate(vehicleReadings[len(vehicleReadings)-1])
			if first != nil && latest != nil && firstDate.Before(latestDate) {
				repeatVehicles++
				changeValues = append(changeValues, *latest-*first)
			}
		}
	}

	snapshot := publicStatsSnapshot{
		SchemaVersion:         publicStatsSchemaVersion,
		GeneratedAt:           generatedAt,
		RegisteredMembers:     registeredMembers,
		OwnersContributed:     len(owners),
		VehiclesRegistered:    len(filteredVehicles),
		VehiclesWithSOH:       len(byVehicle),
		SOHReadings:           readingCount,
		VehiclesWithRepeatSOH: repeatVehicles,
		AverageReportedSOH:    averageRounded(latestValues),
		AverageSOHChange:      averageRounded(changeValues),
		SOHDistribution: []publicDistributionBucket{
			{Label: "90-100%"}, {Label: "80-89.9%"}, {Label: "70-79.9%"}, {Label: "Below 70%"},
		},
		ModelYearDistribution: []publicDistributionBucket{},
	}
	for _, value := range latestValues {
		switch {
		case value >= 90:
			snapshot.SOHDistribution[0].Count++
		case value >= 80:
			snapshot.SOHDistribution[1].Count++
		case value >= 70:
			snapshot.SOHDistribution[2].Count++
		default:
			snapshot.SOHDistribution[3].Count++
		}
	}
	years := make([]string, 0, len(modelYears))
	for year := range modelYears {
		years = append(years, year)
	}
	sort.Strings(years)
	for _, year := range years {
		snapshot.ModelYearDistribution = append(snapshot.ModelYearDistribution, publicDistributionBucket{Label: year, Count: modelYears[year]})
	}
	return snapshot
}

func averageRounded(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	result := math.Round((total/float64(len(values)))*10) / 10
	return &result
}

func writePublicStatsObject(ctx context.Context, snapshot publicStatsSnapshot) error {
	bucketName := os.Getenv("SNAPSHOT_BUCKET")
	if bucketName == "" {
		return fmt.Errorf("SNAPSHOT_BUCKET is not configured")
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return err
	}
	writer := client.Bucket(bucketName).Object("public/stats.json").NewWriter(ctx)
	writer.ContentType = "application/json"
	writer.CacheControl = "public, max-age=300"
	if err := json.NewEncoder(writer).Encode(snapshot); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func readPublicStatsObject(ctx context.Context) (publicStatsSnapshot, error) {
	bucketName := os.Getenv("SNAPSHOT_BUCKET")
	if bucketName == "" {
		return publicStatsSnapshot{}, fmt.Errorf("SNAPSHOT_BUCKET is not configured")
	}
	client, err := gcsClient(ctx)
	if err != nil {
		return publicStatsSnapshot{}, err
	}
	reader, err := client.Bucket(bucketName).Object("public/stats.json").NewReader(ctx)
	if err != nil {
		return publicStatsSnapshot{}, err
	}
	defer reader.Close()
	var snapshot publicStatsSnapshot
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		return publicStatsSnapshot{}, err
	}
	return snapshot, nil
}
