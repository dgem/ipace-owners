package ipace

import (
	"context"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
)

// adminStatsResponse is returned by the /api/admin/stats endpoint.
type adminStatsResponse struct {
	GeneratedAt       string               `json:"generatedAt"`
	PublicStats       publicDashboardStats `json:"publicStats"`
	MemberStats       memberStats          `json:"memberStats"`
	VehicleStats      vehicleStats         `json:"vehicleStats"`
	ServiceEventStats serviceEventStats    `json:"serviceEventStats"`
}

// publicDashboardStats mirrors the consent-filtered counters displayed on the homepage.
type publicDashboardStats struct {
	JoinedOwners        int `json:"joinedOwners"`
	VehiclesRegistered  int `json:"vehiclesRegistered"`
	SOHReadings         int `json:"sohReadings"`
	ServiceEventsLogged int `json:"serviceEventsLogged"`
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
	TotalVehicles    int                  `json:"totalVehicles"`
	VehiclesWithSOH  int                  `json:"vehiclesWithSoh"`
	ModelYearBreakup []modelYearBreakdown `json:"modelYearBreakup"`
}

// modelYearBreakdown shows vehicle counts by model year.
type modelYearBreakdown struct {
	ModelYear string `json:"modelYear"`
	Count     int    `json:"count"`
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
var ukRegistrationRE = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z]{3}$`)

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

	consented := consentedJoinHashes(joins)
	published := aggregatePublicStats(vehicles, readings, services, consented, joinedOwnerCount(joins), 0, now)
	resp := adminStatsResponse{
		GeneratedAt: now.Format(time.RFC3339),
		PublicStats: publicDashboardStats{
			JoinedOwners:        published.JoinedOwners,
			VehiclesRegistered:  published.VehiclesRegistered,
			SOHReadings:         published.SOHReadings,
			ServiceEventsLogged: published.ServiceEventsLogged,
		},
		MemberStats:       computeMemberStats(joins, accounts, vehicles),
		VehicleStats:      computeVehicleStats(vehicles),
		ServiceEventStats: computeServiceEventStats(services),
	}

	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, resp)
}

// computeMemberStats aggregates member data.
func computeMemberStats(joins []joinRecord, accounts map[string]memberAccountStatus, vehicles []vehicleRecord) memberStats {
	countryCounts := make(map[string]countryBreakdown)
	vehiclesByMember := indexVehiclesByMember(vehicles)
	timelineMap := make(map[string]int)
	verifiedTimelineMap := make(map[string]int)
	uniqueJoins := make(map[string]joinRecord)
	verifiedCount := 0

	for _, rec := range joins {
		email := canonicalCampaignEmail(rec.Contact.Email)
		if email == "" {
			continue
		}
		if existing, ok := uniqueJoins[email]; !ok || joinRecordPrecedes(rec, existing) {
			uniqueJoins[email] = rec
		}
	}
	for email, rec := range uniqueJoins {
		country := memberCountry(rec, vehiclesByMember)
		row := countryCounts[country]
		row.Country = country
		row.Joined++
		if account := accounts[email]; account.Registered {
			row.Registered++
			if !account.VerifiedAt.IsZero() {
				row.Verified++
			}
		}
		countryCounts[country] = row
		if !rec.CreatedAt.IsZero() {
			key := rec.CreatedAt.UTC().Format("2006-01-02")
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
		TotalMembers:     len(uniqueJoins),
		VerifiedCount:    verifiedCount,
		CountryBreakup:   countryBreakup,
		JoinedTimeline:   timeline,
		VerifiedTimeline: verifiedTimeline,
	}
}

func consentedJoinHashes(joins []joinRecord) map[string]bool {
	consented := make(map[string]bool)
	for _, record := range joins {
		if record.Consents.AnonymisedAnalysis && record.Review.Status != "excluded" {
			consented[record.UserEmailHash] = true
		}
	}
	return consented
}

func memberVehicleKey(identityUserID, emailHash string) string {
	if identityUserID != "" {
		return "identity:" + identityUserID
	}
	if emailHash != "" {
		return "email-hash:" + emailHash
	}
	return ""
}

func indexVehiclesByMember(vehicles []vehicleRecord) map[string][]vehicleRecord {
	indexed := make(map[string][]vehicleRecord)
	for _, vehicle := range vehicles {
		if vehicle.IdentityUserID != "" {
			key := memberVehicleKey(vehicle.IdentityUserID, "")
			indexed[key] = append(indexed[key], vehicle)
		}
		if vehicle.UserEmailHash != "" {
			key := memberVehicleKey("", vehicle.UserEmailHash)
			indexed[key] = append(indexed[key], vehicle)
		}
	}
	return indexed
}

func memberCountry(member joinRecord, vehiclesByMember map[string][]vehicleRecord) string {
	if country := strings.TrimSpace(member.Contact.Country); country != "" {
		return country
	}
	vehicles := vehiclesByMember[memberVehicleKey(member.IdentityUserID, member.UserEmailHash)]
	for _, vehicle := range vehicles {
		if country := strings.TrimSpace(vehicle.Vehicle.Country); country != "" {
			return country
		}
	}
	for _, vehicle := range vehicles {
		if ukRegistrationRE.MatchString(strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(vehicle.Vehicle.Registration), " ", ""))) {
			return "GB"
		}
	}
	return "Unknown"
}

func joinRecordPrecedes(candidate, existing joinRecord) bool {
	if existing.CreatedAt.IsZero() {
		return !candidate.CreatedAt.IsZero()
	}
	if candidate.CreatedAt.IsZero() {
		return false
	}
	return candidate.CreatedAt.Before(existing.CreatedAt)
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
	accounts := make(map[string]memberAccountStatus)
	emails := make([]string, 0, len(joinEmails))
	for email := range joinEmails {
		emails = append(emails, email)
	}
	sort.Strings(emails)
	for start := 0; start < len(emails); start += 100 {
		end := min(start+100, len(emails))
		identifiers := make([]auth.UserIdentifier, 0, end-start)
		for _, email := range emails[start:end] {
			identifiers = append(identifiers, auth.EmailIdentifier{Email: email})
		}
		result, err := client.GetUsers(ctx, identifiers)
		if err != nil {
			return nil, err
		}
		for _, account := range result.Users {
			email := canonicalCampaignEmail(account.Email)
			status := memberAccountStatus{Registered: true}
			if account.UserMetadata != nil && account.UserMetadata.LastLogInTimestamp > 0 && account.UserMetadata.CreationTimestamp > 0 {
				// Passwordless accounts are created when their first magic link is completed.
				status.VerifiedAt = time.UnixMilli(account.UserMetadata.CreationTimestamp).UTC()
			}
			accounts[email] = status
		}
	}
	return accounts, nil
}

// computeVehicleStats aggregates vehicle and SoH data.
func computeVehicleStats(vehicles []vehicleRecord) vehicleStats {
	totalVehicles := 0
	vehiclesWithSOH := 0
	modelYearCounts := make(map[string]int)

	for _, rec := range vehicles {
		totalVehicles++

		if rec.Battery.StateOfHealth != nil {
			vehiclesWithSOH++
		}

		if rec.Vehicle.ModelYear != "" {
			modelYearCounts[rec.Vehicle.ModelYear]++
		}

	}

	modelYearBreakup := make([]modelYearBreakdown, 0, len(modelYearCounts))
	for modelYear, count := range modelYearCounts {
		modelYearBreakup = append(modelYearBreakup, modelYearBreakdown{ModelYear: modelYear, Count: count})
	}
	sort.Slice(modelYearBreakup, func(i, j int) bool {
		return modelYearBreakup[i].ModelYear < modelYearBreakup[j].ModelYear
	})

	return vehicleStats{
		TotalVehicles:    totalVehicles,
		VehiclesWithSOH:  vehiclesWithSOH,
		ModelYearBreakup: modelYearBreakup,
	}
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
		} else if rec.DisputeStatus != "" && rec.DisputeStatus != "none" {
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
