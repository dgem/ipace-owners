package ipace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAdminStatsRejectsNonGETRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/stats", nil)
	res := httptest.NewRecorder()
	AdminStats(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestComputeMemberStatsBucketsJoinsByDay(t *testing.T) {
	stats := computeMemberStats([]joinRecord{
		{CreatedAt: time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC), Contact: contactRecord{Email: "first@example.com", Country: "United Kingdom"}},
		{CreatedAt: time.Date(2026, time.August, 12, 18, 0, 0, 0, time.UTC), Contact: contactRecord{Email: "second@example.com", Country: "United Kingdom"}},
		{CreatedAt: time.Date(2026, time.August, 13, 9, 0, 0, 0, time.UTC), Contact: contactRecord{Email: "third@example.com", Country: "France"}},
	}, map[string]memberAccountStatus{
		"first@example.com":  {Registered: true, VerifiedAt: time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)},
		"second@example.com": {Registered: true, VerifiedAt: time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)},
	}, nil)
	if len(stats.JoinedTimeline) != 2 || stats.JoinedTimeline[0] != (timelineBucket{Label: "2026-08-12", Count: 2}) || stats.JoinedTimeline[1] != (timelineBucket{Label: "2026-08-13", Count: 1}) {
		t.Fatalf("timeline=%#v", stats.JoinedTimeline)
	}
	if stats.VerifiedCount != 2 || len(stats.VerifiedTimeline) != 2 || stats.VerifiedTimeline[0] != (timelineBucket{Label: "2026-08-12", Count: 1}) {
		t.Fatalf("verified=%#v", stats)
	}
	if len(stats.CountryBreakup) != 2 || stats.CountryBreakup[0] != (countryBreakdown{Country: "United Kingdom", Joined: 2, Registered: 2, Verified: 2}) || stats.CountryBreakup[1] != (countryBreakdown{Country: "France", Joined: 1}) {
		t.Fatalf("countries=%#v", stats.CountryBreakup)
	}
}

func TestComputeMemberStatsCountsEachEmailOnceAtItsFirstJoin(t *testing.T) {
	stats := computeMemberStats([]joinRecord{
		{ID: "later", CreatedAt: time.Date(2026, time.August, 13, 9, 0, 0, 0, time.UTC), Contact: contactRecord{Email: "owner@example.com", Country: "France"}},
		{ID: "first", CreatedAt: time.Date(2026, time.August, 12, 9, 0, 0, 0, time.UTC), Contact: contactRecord{Email: "OWNER@example.com", Country: "United Kingdom"}},
	}, map[string]memberAccountStatus{
		"owner@example.com": {Registered: true},
	}, nil)
	if stats.TotalMembers != 1 {
		t.Fatalf("total members=%d, want 1", stats.TotalMembers)
	}
	if got := stats.JoinedTimeline; len(got) != 1 || got[0] != (timelineBucket{Label: "2026-08-12", Count: 1}) {
		t.Fatalf("timeline=%#v", got)
	}
	if got := stats.CountryBreakup; len(got) != 1 || got[0] != (countryBreakdown{Country: "United Kingdom", Joined: 1, Registered: 1}) {
		t.Fatalf("countries=%#v", got)
	}
}

func TestComputeMemberStatsUsesVehicleCountryAndConservativeUKPlateInference(t *testing.T) {
	stats := computeMemberStats([]joinRecord{
		{IdentityUserID: "vehicle-country", Contact: contactRecord{Email: "country@example.com"}},
		{IdentityUserID: "uk-plate", Contact: contactRecord{Email: "plate@example.com"}},
		{IdentityUserID: "unknown", Contact: contactRecord{Email: "unknown@example.com"}},
	}, nil, []vehicleRecord{
		{IdentityUserID: "vehicle-country", Vehicle: vehicleDetails{Country: "IE"}},
		{IdentityUserID: "uk-plate", Vehicle: vehicleDetails{Registration: "AB12 CDE"}},
		{IdentityUserID: "unknown", Vehicle: vehicleDetails{Registration: "not a plate"}},
	})
	got := map[string]countryBreakdown{}
	for _, row := range stats.CountryBreakup {
		got[row.Country] = row
	}
	for _, country := range []string{"IE", "GB", "Unknown"} {
		if got[country].Joined != 1 {
			t.Fatalf("%s row = %#v", country, got[country])
		}
	}
}

func TestComputeMemberStatsUsesUnknownForConflictingVehicleCountries(t *testing.T) {
	stats := computeMemberStats([]joinRecord{
		{IdentityUserID: "multi-country", Contact: contactRecord{Email: "owner@example.com"}},
	}, nil, []vehicleRecord{
		{IdentityUserID: "multi-country", Vehicle: vehicleDetails{Country: "GB"}},
		{IdentityUserID: "multi-country", Vehicle: vehicleDetails{Country: "IE"}},
	})
	if got := stats.CountryBreakup; len(got) != 1 || got[0] != (countryBreakdown{Country: "Unknown", Joined: 1}) {
		t.Fatalf("countries=%#v", got)
	}
}

func TestPublishedDashboardStatsUsesOnlyEligibleEvidenceConsent(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	joins := []joinRecord{
		{Contact: contactRecord{Email: "included@example.com"}, UserEmailHash: "included", Consents: consentRecord{Contact: true, AnonymisedAnalysis: true}},
		{Contact: contactRecord{Email: "opted-out@example.com"}, UserEmailHash: "opted-out", Consents: consentRecord{Contact: true}},
		{Contact: contactRecord{Email: "excluded@example.com"}, UserEmailHash: "excluded", Consents: consentRecord{Contact: true, AnonymisedAnalysis: true}, Review: reviewRecord{Status: "excluded"}},
	}
	vehicles := []vehicleRecord{
		{ID: "included", UserEmailHash: "included", Review: reviewRecord{Status: "new"}},
		{ID: "opted-out", UserEmailHash: "opted-out", Review: reviewRecord{Status: "new"}},
		{ID: "excluded", UserEmailHash: "excluded", Review: reviewRecord{Status: "new"}},
	}
	got := publishedDashboardStats(joins, vehicles, nil, nil, now)
	if got.JoinedOwners != 3 || got.OwnersContributed != 1 || got.VehiclesRegistered != 1 {
		t.Fatalf("published dashboard stats = %+v", got)
	}
}

func TestIndexVehiclesByMemberUsesIdentityBeforeEmailHash(t *testing.T) {
	vehicles := []vehicleRecord{
		{IdentityUserID: "member-1", UserEmailHash: "member-1-hash"},
		{UserEmailHash: "member-2-hash"},
	}
	indexed := indexVehiclesByMember(vehicles)
	if got := len(indexed["identity:member-1"]); got != 1 {
		t.Fatalf("identity index contains %d vehicles, want 1", got)
	}
	if got := len(indexed["email-hash:member-2-hash"]); got != 1 {
		t.Fatalf("email-hash index contains %d vehicles, want 1", got)
	}
	if got := len(indexed["email-hash:member-1-hash"]); got != 1 {
		t.Fatalf("identity-linked vehicle's email-hash index contains %d vehicles, want 1", got)
	}
}

func TestAdminStatsRequiresSignIn(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	res := httptest.NewRecorder()
	AdminStats(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminStatsRequiresAdministrator(t *testing.T) {
	originalRequireAdmin := adminStatsRequireAdmin
	t.Cleanup(func() { adminStatsRequireAdmin = originalRequireAdmin })
	adminStatsRequireAdmin = func(context.Context, *http.Request) (*firebaseUser, error) {
		return nil, authorizationFailure(http.StatusForbidden, "Admin role required", errors.New("admin role required"))
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	res := httptest.NewRecorder()
	AdminStats(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestComputeServiceEventStatsKeepsEachRecord(t *testing.T) {
	stats := computeServiceEventStats([]serviceEventRecord{
		{EventType: "repair", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-04", ServiceProviderName: "Provider A"},
		{EventType: "repair", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-10", ServiceProviderName: "Provider A"},
	})
	if stats.TotalEvents != 2 || stats.EventsWithFinalFix != 2 || len(stats.EventTypeAggregates) != 1 || len(stats.ProviderAggregates) != 1 {
		t.Fatalf("stats=%#v", stats)
	}
	agg := stats.ProviderAggregates[0]
	if agg.Label != "Provider A" || agg.EventCount != 2 || agg.DurationCount != 2 || agg.MinDays == nil || *agg.MinDays != 3 || agg.MaxDays == nil || *agg.MaxDays != 9 || agg.MedianDays == nil || *agg.MedianDays != 6 || agg.AvgDays == nil || *agg.AvgDays != 6 {
		t.Fatalf("aggregate=%#v", agg)
	}
}

func TestComputeServiceEventStatsDoesNotClassifyNoDisputeAsDisputed(t *testing.T) {
	stats := computeServiceEventStats([]serviceEventRecord{
		{EventType: "inspection", DisputeStatus: "none"},
		{EventType: "repair", DisputeStatus: "still-disputed"},
	})
	if len(stats.EventTypeAggregates) != 2 || len(stats.DisputeStatusBreakup) != 1 {
		t.Fatalf("stats=%#v", stats)
	}
	if stats.DisputeStatusBreakup[0] != (demographicBucket{Label: "Still disputed", Count: 1}) {
		t.Fatalf("disputes=%#v", stats.DisputeStatusBreakup)
	}
}

func TestComputeServiceEventStatsNormalisesProvidersAndUsesVerifiedDates(t *testing.T) {
	staleDays := 999
	stats := computeServiceEventStats([]serviceEventRecord{
		{EventType: "fault", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-01", ServiceProviderName: "Sytner Jaguar, Northampton"},
		{EventType: "fault", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-11", ServiceProviderName: "sytner JLR Northampton — NN1 2AB"},
		{EventType: "fault", DaysToFinalFix: &staleDays, ServiceProviderName: "Sytner Northampton"},
		{EventType: "fault", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-06", ServiceProviderID: "provider-1", ServiceProviderName: "Canonical Provider"},
		{EventType: "fault", OccurredAt: "2026-09-01", FinalFixAt: "2026-09-08", ServiceProviderID: "PROVIDER-1", ServiceProviderName: "Old Provider Spelling"},
		{EventType: "fault", Review: reviewRecord{Status: "deleted"}, ServiceProviderName: "Deleted Provider"},
	})
	if stats.TotalEvents != 5 || stats.EventsWithFinalFix != 4 || len(stats.ProviderAggregates) != 2 {
		t.Fatalf("stats=%#v", stats)
	}
	byProvider := map[string]resolutionAggregate{}
	for _, aggregate := range stats.ProviderAggregates {
		byProvider[aggregate.Label] = aggregate
	}
	northampton := byProvider["Sytner Northampton"]
	if northampton.EventCount != 3 || northampton.DurationCount != 2 || northampton.MedianDays == nil || *northampton.MedianDays != 5 || northampton.AvgDays == nil || *northampton.AvgDays != 5 {
		t.Fatalf("northampton=%#v providers=%#v", northampton, stats.ProviderAggregates)
	}
	canonical := byProvider["Canonical Provider"]
	if canonical.EventCount != 2 || canonical.DurationCount != 2 || canonical.MedianDays == nil || *canonical.MedianDays != 6 {
		t.Fatalf("canonical=%#v providers=%#v", canonical, stats.ProviderAggregates)
	}
}
