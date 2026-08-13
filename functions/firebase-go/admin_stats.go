package ipace

import (
	"context"
	"math"
	"net/http"
	"sort"
	"time"

	"google.golang.org/api/iterator"
)

// adminStatsResponse is returned by the /api/admin/stats endpoint.
type adminStatsResponse struct {
	GeneratedAt       string            `json:"generatedAt"`
	MemberStats       memberStats       `json:"memberStats"`
	VehicleStats      vehicleStats      `json:"vehicleStats"`
	ServiceEventStats serviceEventStats `json:"serviceEventStats"`
}

// memberStats holds aggregate member metrics.
type memberStats struct {
	TotalMembers     int                `json:"totalMembers"`
	VerifiedCount    int                `json:"verifiedCount"`
	CountryBreakup   []countryBreakdown `json:"countryBreakup"`
	JoinedTimeline   []timelineBucket   `json:"joinedTimeline"`
	VerifiedTimeline []timelineBucket   `json:"verifiedTimeline"`
}

// countryBreakdown shows member counts by country.
type countryBreakdown struct {
	Country    string `json:"country"`
	Joined     int    `json:"joined"`
	Registered int    `json:"registered"`
	Verified   int    `json:"verified"`
}

type memberAccountStatus struct {
	Registered bool
	VerifiedAt time.Time
}

// vehicleStats holds aggregate vehicle/SoH metrics.
type vehicleStats struct {
	TotalVehicles        int                  `json:"totalVehicles"`
	VehiclesWithSOH      int                  `json:"vehiclesWithSoh"`
	ModelYearBreakup     []modelYearBreakdown `json:"modelYearBreakup"`
	RegistrationTimeline []timelineBucket     `json:"registrationTimeline"`
	BatteryReadings      []batteryReadingRow  `json:"batteryReadings"`
}

// modelYearBreakdown shows vehicle counts by model year.
type modelYearBreakdown struct {
	ModelYear string `json:"modelYear"`
	Count     int    `json:"count"`
}

// batteryReadingRow captures per-vehicle SoH history.
type batteryReadingRow struct {
	VehicleID    string       `json:"vehicleId"`
	Registration string       `json:"registration"`
	ModelYear    string       `json:"modelYear"`
	Readings     []sohReading `json:"readings"`
	LatestSOH    *float64     `json:"latestSoh,omitempty"`
}

// sohReading captures a single SoH measurement.
type sohReading struct {
	MeasuredAt string   `json:"measuredAt"`
	SOH        *float64 `json:"soh,omitempty"`
	Mileage    *int     `json:"mileage,omitempty"`
}

// serviceEventStats holds aggregate service event metrics.
type serviceEventStats struct {
	TotalEvents        int                  `json:"totalEvents"`
	EventTypeBreakup   []eventTypeBreakdown `json:"eventTypeBreakup"`
	CategoryAggregates []categoryAggregate  `json:"categoryAggregates"`
}

// eventTypeBreakdown shows event counts by EventType.
type eventTypeBreakdown struct {
	EventType string `json:"eventType"`
	Count     int    `json:"count"`
}

// categoryAggregate holds min/max/avg for a service category.
type categoryAggregate struct {
	Category   string   `json:"category"`
	EventCount int      `json:"eventCount"`
	MinDays    *int     `json:"minDays,omitempty"`
	AvgDays    *float64 `json:"avgDays,omitempty"`
	MaxDays    *int     `json:"maxDays,omitempty"`
}

// timelineBucket represents a time bucket for compact graphs.
type timelineBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

var adminStatsRequireUser = requireUser
var adminStatsIsAdmin = isAdmin

// AdminStats serves aggregate statistics for the admin dashboard.
func AdminStats(w http.ResponseWriter, r *http.Request) {
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

	user, err := adminStatsRequireUser(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Sign in required"})
		return
	}
	if !adminStatsIsAdmin(user) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}

	db, err := firestoreClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not connect to data store"})
		return
	}

	now := time.Now().UTC()

	// Fetch all collections.
	var joins []joinRecord
	var vehicles []vehicleRecord
	var readings []batteryReadingRecord
	var services []serviceEventRecord

	if err := readCollection(r.Context(), db.Collection("joinSubmissions").Query, &joins); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load join submissions"})
		return
	}
	if err := readCollection(r.Context(), db.Collection("vehicles").Query, &vehicles); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load vehicle submissions"})
		return
	}
	if err := readCollection(r.Context(), db.Collection("batteryReadings").Query, &readings); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load battery readings"})
		return
	}
	if err := readCollection(r.Context(), db.Collection("serviceEvents").Query, &services); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load service events"})
		return
	}

	accounts, err := loadMemberAccountStatuses(r.Context(), joins)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load verified accounts"})
		return
	}

	resp := adminStatsResponse{
		GeneratedAt:       now.Format(time.RFC3339),
		MemberStats:       computeMemberStats(joins, accounts),
		VehicleStats:      computeVehicleStats(vehicles, readings),
		ServiceEventStats: computeServiceEventStats(services),
	}

	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, resp)
}

// computeMemberStats aggregates member data.
func computeMemberStats(joins []joinRecord, accounts map[string]memberAccountStatus) memberStats {
	totalMembers := 0
	countryCounts := make(map[string]countryBreakdown)
	timelineMap := make(map[string]int)
	verifiedTimelineMap := make(map[string]int)
	seenMembers := map[string]bool{}
	verifiedCount := 0

	for _, rec := range joins {
		totalMembers++

		// Country breakdown.
		if email := canonicalCampaignEmail(rec.Contact.Email); email != "" && !seenMembers[email] {
			seenMembers[email] = true
			if rec.Contact.Country != "" {
				row := countryCounts[rec.Contact.Country]
				row.Country = rec.Contact.Country
				row.Joined++
				if account := accounts[email]; account.Registered {
					row.Registered++
					if !account.VerifiedAt.IsZero() {
						row.Verified++
					}
				}
				countryCounts[rec.Contact.Country] = row
			}
		}

		// Joined timeline: bucket by day to match the other admin trends.
		if !rec.CreatedAt.IsZero() {
			key := rec.CreatedAt.Format("2006-01-02")
			timelineMap[key]++
		}
	}
	for _, account := range accounts {
		if !account.VerifiedAt.IsZero() {
			verifiedCount++
			verifiedTimelineMap[account.VerifiedAt.UTC().Format("2006-01-02")]++
		}
	}

	countryBreakup := make([]countryBreakdown, 0, len(countryCounts))
	for _, row := range countryCounts {
		countryBreakup = append(countryBreakup, row)
	}
	sort.Slice(countryBreakup, func(i, j int) bool {
		if countryBreakup[i].Joined == countryBreakup[j].Joined {
			return countryBreakup[i].Country < countryBreakup[j].Country
		}
		return countryBreakup[i].Joined > countryBreakup[j].Joined
	})

	timeline := make([]timelineBucket, 0, len(timelineMap))
	for label, count := range timelineMap {
		timeline = append(timeline, timelineBucket{Label: label, Count: count})
	}
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Label < timeline[j].Label
	})
	verifiedTimeline := make([]timelineBucket, 0, len(verifiedTimelineMap))
	for label, count := range verifiedTimelineMap {
		verifiedTimeline = append(verifiedTimeline, timelineBucket{Label: label, Count: count})
	}
	sort.Slice(verifiedTimeline, func(i, j int) bool { return verifiedTimeline[i].Label < verifiedTimeline[j].Label })

	return memberStats{
		TotalMembers:     totalMembers,
		VerifiedCount:    verifiedCount,
		CountryBreakup:   countryBreakup,
		JoinedTimeline:   timeline,
		VerifiedTimeline: verifiedTimeline,
	}
}

func loadMemberAccountStatuses(ctx context.Context, joins []joinRecord) (map[string]memberAccountStatus, error) {
	client, err := firebaseAuth(ctx)
	if err != nil {
		return nil, err
	}
	joinEmails := make(map[string]struct{}, len(joins))
	for _, join := range joins {
		if email := canonicalCampaignEmail(join.Contact.Email); email != "" {
			joinEmails[email] = struct{}{}
		}
	}
	iter := client.Users(ctx, "")
	accounts := make(map[string]memberAccountStatus)
	for {
		account, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		email := canonicalCampaignEmail(account.Email)
		if _, joined := joinEmails[email]; !joined {
			continue
		}
		status := memberAccountStatus{Registered: true}
		if account.UserMetadata != nil && account.UserMetadata.LastLogInTimestamp > 0 && account.UserMetadata.CreationTimestamp > 0 {
			// Passwordless accounts are created when their first magic link is completed.
			status.VerifiedAt = time.UnixMilli(account.UserMetadata.CreationTimestamp).UTC()
		}
		accounts[email] = status
	}
	return accounts, nil
}

// computeVehicleStats aggregates vehicle and SoH data.
func computeVehicleStats(vehicles []vehicleRecord, readings []batteryReadingRecord) vehicleStats {
	totalVehicles := 0
	vehiclesWithSOH := 0
	modelYearCounts := make(map[string]int)
	regTimelineMap := make(map[string]int)

	for _, rec := range vehicles {
		totalVehicles++

		if rec.Battery.StateOfHealth != nil {
			vehiclesWithSOH++
		}

		if rec.Vehicle.ModelYear != "" {
			modelYearCounts[rec.Vehicle.ModelYear]++
		}

		if !rec.CreatedAt.IsZero() {
			key := rec.CreatedAt.Format("2006-01")
			regTimelineMap[key]++
		}
	}

	modelYearBreakup := make([]modelYearBreakdown, 0, len(modelYearCounts))
	for modelYear, count := range modelYearCounts {
		modelYearBreakup = append(modelYearBreakup, modelYearBreakdown{ModelYear: modelYear, Count: count})
	}
	sort.Slice(modelYearBreakup, func(i, j int) bool {
		return modelYearBreakup[i].ModelYear < modelYearBreakup[j].ModelYear
	})

	regTimeline := make([]timelineBucket, 0, len(regTimelineMap))
	for label, count := range regTimelineMap {
		regTimeline = append(regTimeline, timelineBucket{Label: label, Count: count})
	}
	sort.Slice(regTimeline, func(i, j int) bool {
		return regTimeline[i].Label < regTimeline[j].Label
	})

	// Build per-vehicle SoH history.
	batteryReadings := buildBatteryReadingRows(vehicles, readings)

	return vehicleStats{
		TotalVehicles:        totalVehicles,
		VehiclesWithSOH:      vehiclesWithSOH,
		ModelYearBreakup:     modelYearBreakup,
		RegistrationTimeline: regTimeline,
		BatteryReadings:      batteryReadings,
	}
}

// buildBatteryReadingRows groups readings by vehicle.
func buildBatteryReadingRows(vehicles []vehicleRecord, readings []batteryReadingRecord) []batteryReadingRow {
	vehMap := make(map[string]vehicleRecord)
	for _, rec := range vehicles {
		vehMap[rec.ID] = rec
	}

	grouped := make(map[string][]batteryReadingRecord)
	var order []string
	for _, reading := range readings {
		if _, ok := grouped[reading.VehicleID]; !ok {
			order = append(order, reading.VehicleID)
		}
		grouped[reading.VehicleID] = append(grouped[reading.VehicleID], reading)
	}

	rows := make([]batteryReadingRow, 0, len(order))
	for _, vid := range order {
		recs := grouped[vid]
		sort.Slice(recs, func(i, j int) bool {
			return recs[i].CreatedAt.Before(recs[j].CreatedAt)
		})

		readingsArr := make([]sohReading, 0, len(recs))
		var latestSOH *float64
		for _, r := range recs {
			readingsArr = append(readingsArr, sohReading{
				MeasuredAt: r.Battery.MeasuredAt,
				SOH:        r.Battery.StateOfHealth,
				Mileage:    r.Battery.MileageAtMeasurement,
			})
			if r.Battery.StateOfHealth != nil {
				latestSOH = r.Battery.StateOfHealth
			}
		}

		var reg string
		var modelYear string
		if veh, ok := vehMap[vid]; ok {
			reg = veh.Vehicle.Registration
			modelYear = veh.Vehicle.ModelYear
		}

		rows = append(rows, batteryReadingRow{
			VehicleID:    vid,
			Registration: reg,
			ModelYear:    modelYear,
			Readings:     readingsArr,
			LatestSOH:    latestSOH,
		})
	}

	return rows
}

// computeServiceEventStats aggregates service event data.
func computeServiceEventStats(services []serviceEventRecord) serviceEventStats {
	totalEvents := len(services)
	eventTypeCounts := make(map[string]int)
	categoryMap := make(map[string][]*serviceEventRecord)

	for index := range services {
		rec := &services[index]
		eventTypeCounts[rec.EventType]++
		if rec.ServiceProviderName != "" {
			categoryMap[rec.ServiceProviderName] = append(categoryMap[rec.ServiceProviderName], rec)
		} else if rec.DisputeStatus != "" {
			categoryMap["Disputes"] = append(categoryMap["Disputes"], rec)
		} else {
			categoryMap[rec.EventType] = append(categoryMap[rec.EventType], rec)
		}
	}

	eventTypeBreakup := make([]eventTypeBreakdown, 0, len(eventTypeCounts))
	for eventType, count := range eventTypeCounts {
		eventTypeBreakup = append(eventTypeBreakup, eventTypeBreakdown{EventType: eventType, Count: count})
	}
	sort.Slice(eventTypeBreakup, func(i, j int) bool {
		return eventTypeBreakup[i].Count > eventTypeBreakup[j].Count
	})

	categoryAggregates := computeCategoryAggregates(categoryMap)

	return serviceEventStats{
		TotalEvents:        totalEvents,
		EventTypeBreakup:   eventTypeBreakup,
		CategoryAggregates: categoryAggregates,
	}
}

// computeCategoryAggregates computes min/avg/max for each category.
func computeCategoryAggregates(categoryMap map[string][]*serviceEventRecord) []categoryAggregate {
	aggregates := make([]categoryAggregate, 0, len(categoryMap))

	for category, records := range categoryMap {
		count := len(records)
		var daysSum float64
		daysCount := 0
		minDays := math.MaxInt32
		maxDays := 0

		for _, rec := range records {
			if rec.DaysToFinalFix != nil && *rec.DaysToFinalFix >= 0 {
				d := *rec.DaysToFinalFix
				daysSum += float64(d)
				daysCount++
				if d < minDays {
					minDays = d
				}
				if d > maxDays {
					maxDays = d
				}
			}
		}

		agg := categoryAggregate{
			Category:   category,
			EventCount: count,
			MinDays:    nil,
			AvgDays:    nil,
			MaxDays:    nil,
		}

		if daysCount > 0 {
			avg := math.Round((daysSum/float64(daysCount))*10) / 10
			agg.MinDays = &minDays
			agg.AvgDays = &avg
			agg.MaxDays = &maxDays
		}

		aggregates = append(aggregates, agg)
	}

	sort.Slice(aggregates, func(i, j int) bool {
		return aggregates[i].EventCount > aggregates[j].EventCount
	})

	return aggregates
}
