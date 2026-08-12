package ipace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminStatsRejectsNonGETRequests(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/stats", nil)
	res := httptest.NewRecorder()
	AdminStats(res, req)
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminStatsRequiresAdministrator(t *testing.T) {
	originalRequireUser, originalIsAdmin := adminStatsRequireUser, adminStatsIsAdmin
	t.Cleanup(func() { adminStatsRequireUser, adminStatsIsAdmin = originalRequireUser, originalIsAdmin })
	adminStatsRequireUser = func(context.Context, *http.Request) (*firebaseUser, error) {
		return &firebaseUser{UID: "member"}, nil
	}
	adminStatsIsAdmin = func(*firebaseUser) bool { return false }

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	res := httptest.NewRecorder()
	AdminStats(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestComputeServiceEventStatsKeepsEachRecord(t *testing.T) {
	firstDays, secondDays := 3, 9
	stats := computeServiceEventStats([]serviceEventRecord{
		{EventType: "repair", ServiceProviderName: "Provider A", DaysToFinalFix: &firstDays},
		{EventType: "repair", ServiceProviderName: "Provider A", DaysToFinalFix: &secondDays},
	})
	if len(stats.CategoryAggregates) != 1 {
		t.Fatalf("aggregates=%#v", stats.CategoryAggregates)
	}
	agg := stats.CategoryAggregates[0]
	if agg.EventCount != 2 || agg.MinDays == nil || *agg.MinDays != 3 || agg.MaxDays == nil || *agg.MaxDays != 9 || agg.AvgDays == nil || *agg.AvgDays != 6 {
		t.Fatalf("aggregate=%#v", agg)
	}
}
