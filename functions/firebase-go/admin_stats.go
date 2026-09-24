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
	GeneratedAt              string               `json:"generatedAt"`
	PublicStats              publicDashboardStats `json:"publicStats"`
	MemberStats              memberStats          `json:"memberStats"`
	VehicleStats             vehicleStats         `json:"vehicleStats"`
	ServiceEventStats        serviceEventStats    `json:"serviceEventStats"`
	ServiceLocations         serviceLocationStats `json:"serviceLocations"`
	ConsentedJoinTimeline    []timelineBucket     `json:"consentedJoinTimeline"`
	ConsentedMemberCountries []demographicBucket  `json:"consentedMemberCountries"`
}

type demographicBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// serviceLocationStats counts consent-eligible service records, not unique owners.
// Small postcode areas are combined so the meeting deck cannot single out a member.
type serviceLocationStats struct {
	Known   int                   `json:"known"`
	Unknown int                   `json:"unknown"`
	Areas   []serviceLocationArea `json:"areas"`
}

type serviceLocationArea struct {
	Area  string `json:"area"`
	Count int    `json:"count"`
}

// publicDashboardStats mirrors the consent-filtered counters displayed on the homepage.
type publicDashboardStats struct {
	JoinedOwners        int `json:"joinedOwners"`
	OwnersContributed   int `json:"ownersContributed"`
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
	TotalEvents          int                   `json:"totalEvents"`
	EventsWithFinalFix   int                   `json:"eventsWithFinalFix"`
	EventTypeBreakup     []eventTypeBreakdown  `json:"eventTypeBreakup"`
	EventTypeAggregates  []resolutionAggregate `json:"eventTypeAggregates"`
	ProviderAggregates   []resolutionAggregate `json:"providerAggregates"`
	DisputeStatusBreakup []demographicBucket   `json:"disputeStatusBreakup"`
}

// eventTypeBreakdown shows event counts by EventType.
type eventTypeBreakdown struct {
	EventType string `json:"eventType"`
	Count     int    `json:"count"`
}

// resolutionAggregate keeps the record count distinct from the smaller
// denominator that has both an event date and a final-fix date.
type resolutionAggregate struct {
	Label         string   `json:"label"`
	EventCount    int      `json:"eventCount"`
	DurationCount int      `json:"durationCount"`
	MinDays       *int     `json:"minDays,omitempty"`
	MedianDays    *float64 `json:"medianDays,omitempty"`
	AvgDays       *float64 `json:"avgDays,omitempty"`
	MaxDays       *int     `json:"maxDays,omitempty"`
}

// timelineBucket represents a time bucket for compact graphs.
type timelineBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

var adminStatsRequireAdmin = requireAdmin
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

	_, err := adminStatsRequireAdmin(r.Context(), r)
	if err != nil {
		writeAdminAuthorizationError(w, err)
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

	published := publishedDashboardStats(joins, vehicles, readings, services, now)
	resp := adminStatsResponse{
		GeneratedAt: now.Format(time.RFC3339),
		PublicStats: publicDashboardStats{
			JoinedOwners:        published.JoinedOwners,
			OwnersContributed:   published.OwnersContributed,
			VehiclesRegistered:  published.VehiclesRegistered,
			SOHReadings:         published.SOHReadings,
			ServiceEventsLogged: published.ServiceEventsLogged,
		},
		MemberStats:              computeMemberStats(joins, accounts, vehicles),
		VehicleStats:             computeVehicleStats(vehicles),
		ServiceEventStats:        computeServiceEventStats(services),
		ServiceLocations:         computeConsentServiceLocations(joins, vehicles, services),
		ConsentedJoinTimeline:    computeConsentedJoinTimeline(joins),
		ConsentedMemberCountries: computeConsentedMemberCountries(joins),
	}

	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, resp)
}

func publishedDashboardStats(joins []joinRecord, vehicles []vehicleRecord, readings []batteryReadingRecord, services []serviceEventRecord, now time.Time) publicStatsSnapshot {
	consented := consentedJoinHashes(joins)
	return aggregatePublicStats(vehicles, readings, services, consented, joinedOwnerCount(joins), 0, now)
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
		if record.Consents.AnonymisedAnalysis && record.Review.Status != "excluded" && !recordDeleted(record.Review) {
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
	vehicleCountries := make(map[string]struct{})
	for _, vehicle := range vehicles {
		if country := strings.TrimSpace(vehicle.Vehicle.Country); country != "" {
			vehicleCountries[country] = struct{}{}
		}
	}
	if len(vehicleCountries) == 1 {
		for country := range vehicleCountries {
			return country
		}
	}
	if len(vehicleCountries) > 1 {
		return "Unknown"
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
	eventTypeCounts := make(map[string]int)
	eventTypeRecords := make(map[string][]*serviceEventRecord)
	providerRecords := make(map[string][]*serviceEventRecord)
	providerNames := make(map[string]map[string]int)
	disputeCounts := make(map[string]int)
	totalEvents := 0
	eventsWithFinalFix := 0

	for index := range services {
		rec := &services[index]
		if recordDeleted(rec.Review) {
			continue
		}
		totalEvents++
		eventType := cleanEnum(rec.EventType, serviceEventTypeValues)
		if eventType == "" {
			eventType = "unknown"
		}
		eventTypeCounts[eventType]++
		eventTypeRecords[eventType] = append(eventTypeRecords[eventType], rec)
		if serviceEventResolutionDays(rec.OccurredAt, rec.FinalFixAt) != nil {
			eventsWithFinalFix++
		}
		if providerKey := normalisedServiceProviderKey(*rec); providerKey != "" {
			providerRecords[providerKey] = append(providerRecords[providerKey], rec)
			name := strings.TrimSpace(rec.ServiceProviderName)
			if name != "" {
				if providerNames[providerKey] == nil {
					providerNames[providerKey] = map[string]int{}
				}
				providerNames[providerKey][name]++
			}
		}
		if dispute := cleanEnum(rec.DisputeStatus, serviceEventDisputeStatusValues); dispute != "" && dispute != "none" {
			disputeCounts[dispute]++
		}
	}

	eventTypeBreakup := make([]eventTypeBreakdown, 0, len(eventTypeCounts))
	for eventType, count := range eventTypeCounts {
		eventTypeBreakup = append(eventTypeBreakup, eventTypeBreakdown{EventType: eventType, Count: count})
	}
	sort.Slice(eventTypeBreakup, func(i, j int) bool {
		return eventTypeBreakup[i].Count > eventTypeBreakup[j].Count
	})

	eventTypeAggregates := make([]resolutionAggregate, 0, len(eventTypeRecords))
	for eventType, records := range eventTypeRecords {
		eventTypeAggregates = append(eventTypeAggregates, computeResolutionAggregate(serviceEventTypeLabel(eventType), records))
	}
	sortResolutionAggregates(eventTypeAggregates)

	providerAggregates := make([]resolutionAggregate, 0, len(providerRecords))
	for key, records := range providerRecords {
		providerAggregates = append(providerAggregates, computeResolutionAggregate(preferredServiceProviderName(providerNames[key]), records))
	}
	sortResolutionAggregates(providerAggregates)

	disputeStatusBreakup := make([]demographicBucket, 0, len(disputeCounts))
	for status, count := range disputeCounts {
		disputeStatusBreakup = append(disputeStatusBreakup, demographicBucket{Label: serviceDisputeStatusLabel(status), Count: count})
	}
	sort.Slice(disputeStatusBreakup, func(i, j int) bool {
		if disputeStatusBreakup[i].Count == disputeStatusBreakup[j].Count {
			return disputeStatusBreakup[i].Label < disputeStatusBreakup[j].Label
		}
		return disputeStatusBreakup[i].Count > disputeStatusBreakup[j].Count
	})

	return serviceEventStats{
		TotalEvents:          totalEvents,
		EventsWithFinalFix:   eventsWithFinalFix,
		EventTypeBreakup:     eventTypeBreakup,
		EventTypeAggregates:  eventTypeAggregates,
		ProviderAggregates:   providerAggregates,
		DisputeStatusBreakup: disputeStatusBreakup,
	}
}

var serviceProviderPostcodeRE = regexp.MustCompile(`(?i)\b[A-Z]{1,2}[0-9][A-Z0-9]?\s*[0-9][A-Z]{2}\b`)
var serviceProviderNonWordRE = regexp.MustCompile(`[^a-z0-9]+`)

func normalisedServiceProviderKey(record serviceEventRecord) string {
	if id := strings.ToLower(strings.TrimSpace(record.ServiceProviderID)); id != "" {
		return "id:" + id
	}
	name := strings.ToLower(strings.TrimSpace(record.ServiceProviderName))
	if name == "" {
		return ""
	}
	name = serviceProviderPostcodeRE.ReplaceAllString(name, " ")
	name = strings.NewReplacer(
		"jaguar land rover", " ",
		"service centre", " ",
		"service center", " ",
		"jaguar", " ",
		"land rover", " ",
		"jlr", " ",
		"marshalls", "marshall",
	).Replace(name)
	name = strings.Join(strings.Fields(serviceProviderNonWordRE.ReplaceAllString(name, " ")), " ")
	if name == "" {
		name = strings.Join(strings.Fields(serviceProviderNonWordRE.ReplaceAllString(strings.ToLower(record.ServiceProviderName), " ")), " ")
	}
	return "name:" + name
}

func preferredServiceProviderName(names map[string]int) string {
	best := ""
	bestCount := 0
	for name, count := range names {
		if count > bestCount || count == bestCount && (best == "" || len(name) < len(best) || len(name) == len(best) && name < best) {
			best, bestCount = name, count
		}
	}
	if best == "" {
		return "Provider name unavailable"
	}
	return best
}

func serviceEventTypeLabel(value string) string {
	labels := map[string]string{"service": "Service", "fault": "Fault", "repair": "Repair", "recall": "Recall", "inspection": "Inspection", "other": "Other", "unknown": "Unknown"}
	if label := labels[value]; label != "" {
		return label
	}
	return value
}

func serviceDisputeStatusLabel(value string) string {
	labels := map[string]string{
		"initially-refused":         "Initially refused",
		"partially-accepted":        "Partially accepted",
		"still-disputed":            "Still disputed",
		"resolved-after-escalation": "Resolved after escalation",
		"unsure":                    "Unsure",
	}
	if label := labels[value]; label != "" {
		return label
	}
	return value
}

var servicePostcodeAreaRE = regexp.MustCompile(`^[A-Z]{1,2}`)
var servicePostcodeValidRE = regexp.MustCompile(`^[A-Z]{1,2}[0-9][A-Z0-9]?\s*[0-9][A-Z]{2}$`)

func computeConsentServiceLocations(joins []joinRecord, vehicles []vehicleRecord, services []serviceEventRecord) serviceLocationStats {
	consented := consentedJoinHashes(joins)
	eligibleVehicles := map[string]bool{}
	for _, vehicle := range vehicles {
		if vehicle.Review.Status != "excluded" && !recordDeleted(vehicle.Review) && consented[vehicle.UserEmailHash] {
			eligibleVehicles[vehicle.ID] = true
		}
	}
	counts := map[string]int{}
	result := serviceLocationStats{Areas: []serviceLocationArea{}}
	for _, event := range services {
		if !eligibleVehicles[event.VehicleID] || event.Review.Status == "excluded" || recordDeleted(event.Review) {
			continue
		}
		postcode := strings.ToUpper(strings.TrimSpace(event.ServiceProviderPostcode))
		area := ""
		if servicePostcodeValidRE.MatchString(postcode) {
			area = servicePostcodeAreaRE.FindString(postcode)
		}
		if area == "" {
			result.Unknown++
			continue
		}
		result.Known++
		counts[area]++
	}
	other := 0
	for area, count := range counts {
		if count < 5 {
			other += count
		} else {
			result.Areas = append(result.Areas, serviceLocationArea{Area: area, Count: count})
		}
	}
	if other > 0 {
		result.Areas = append(result.Areas, serviceLocationArea{Area: "Other areas", Count: other})
	}
	sort.Slice(result.Areas, func(i, j int) bool {
		if result.Areas[i].Count == result.Areas[j].Count {
			return result.Areas[i].Area < result.Areas[j].Area
		}
		return result.Areas[i].Count > result.Areas[j].Count
	})
	return result
}

func computeConsentedJoinTimeline(joins []joinRecord) []timelineBucket {
	firstByEmail := map[string]time.Time{}
	for _, join := range joins {
		if !join.Consents.Contact || join.CreatedAt.IsZero() {
			continue
		}
		email := canonicalJoinEmail(join.Contact.Email)
		if email == "" {
			continue
		}
		if first, found := firstByEmail[email]; !found || join.CreatedAt.Before(first) {
			firstByEmail[email] = join.CreatedAt
		}
	}
	counts := map[string]int{}
	for _, joined := range firstByEmail {
		counts[joined.UTC().Format("2006-01-02")]++
	}
	result := make([]timelineBucket, 0, len(counts))
	for day, count := range counts {
		result = append(result, timelineBucket{Label: day, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Label < result[j].Label })
	return result
}

func computeConsentedMemberCountries(joins []joinRecord) []demographicBucket {
	firstByEmail := map[string]joinRecord{}
	for _, join := range joins {
		if !join.Consents.AnonymisedAnalysis {
			continue
		}
		email := canonicalJoinEmail(join.Contact.Email)
		if email == "" {
			continue
		}
		if previous, found := firstByEmail[email]; !found || joinRecordPrecedes(join, previous) {
			firstByEmail[email] = join
		}
	}
	counts := map[string]int{}
	for _, join := range firstByEmail {
		country := strings.TrimSpace(join.Contact.Country)
		if country == "" {
			country = "Unknown"
		}
		counts[country]++
	}
	result := []demographicBucket{}
	other := 0
	for country, count := range counts {
		if count < 5 {
			other += count
		} else {
			result = append(result, demographicBucket{Label: country, Count: count})
		}
	}
	if other > 0 {
		result = append(result, demographicBucket{Label: "Other / unknown", Count: other})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Label < result[j].Label
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func computeResolutionAggregate(label string, records []*serviceEventRecord) resolutionAggregate {
	aggregate := resolutionAggregate{Label: label, EventCount: len(records)}
	durations := make([]int, 0, len(records))
	for _, record := range records {
		if days := serviceEventResolutionDays(record.OccurredAt, record.FinalFixAt); days != nil {
			durations = append(durations, *days)
		}
	}
	if len(durations) == 0 {
		return aggregate
	}
	sort.Ints(durations)
	sum := 0
	for _, days := range durations {
		sum += days
	}
	minDays, maxDays := durations[0], durations[len(durations)-1]
	median := float64(durations[len(durations)/2])
	if len(durations)%2 == 0 {
		median = float64(durations[len(durations)/2-1]+durations[len(durations)/2]) / 2
	}
	average := math.Round((float64(sum)/float64(len(durations)))*10) / 10
	aggregate.DurationCount = len(durations)
	aggregate.MinDays = &minDays
	aggregate.MedianDays = &median
	aggregate.AvgDays = &average
	aggregate.MaxDays = &maxDays
	return aggregate
}

func sortResolutionAggregates(aggregates []resolutionAggregate) {
	sort.Slice(aggregates, func(i, j int) bool {
		if aggregates[i].EventCount == aggregates[j].EventCount {
			return strings.ToLower(aggregates[i].Label) < strings.ToLower(aggregates[j].Label)
		}
		return aggregates[i].EventCount > aggregates[j].EventCount
	})
}
