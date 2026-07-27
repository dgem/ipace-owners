package ipace

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBuildCampaignSummaryAggregatesRecordedChannels(t *testing.T) {
	originalEmail, originalInstagram := campaignSummaryEmailHistory, campaignSummaryInstagramHistory
	t.Cleanup(func() {
		campaignSummaryEmailHistory = originalEmail
		campaignSummaryInstagramHistory = originalInstagram
	})
	now := time.Now().UTC()
	campaignSummaryEmailHistory = func(context.Context) (customCampaignHistory, error) {
		return customCampaignHistory{Campaigns: []customCampaignRecord{
			{Sent: 12, Failed: 1, Remaining: 2, UpdatedAt: now},
			{Sent: 3, Remaining: 4, UpdatedAt: now.Add(-time.Hour)},
		}}, nil
	}
	campaignSummaryInstagramHistory = func(context.Context) (instagramCampaignHistory, error) {
		return instagramCampaignHistory{InsightsConfigured: true, Campaigns: []instagramCampaignRecord{
			{Status: "published", UpdatedAt: now, Insights: instagramInsights{Available: true, Views: 100, Reach: 80, TotalInteractions: 7}},
			{Status: "draft", UpdatedAt: now.Add(-time.Hour)},
		}}, nil
	}
	result := buildCampaignSummary(context.Background())
	if !result.Email.Available || result.Email.Runs != 2 || result.Email.Sent != 15 || result.Email.Failed != 1 || result.Email.Remaining != 6 {
		t.Fatalf("email=%#v", result.Email)
	}
	if !result.Instagram.Available || result.Instagram.Runs != 2 || result.Instagram.Published != 1 || result.Instagram.Drafts != 1 || result.Instagram.Views != 100 || result.Instagram.Reach != 80 || result.Instagram.TotalInteractions != 7 {
		t.Fatalf("instagram=%#v", result.Instagram)
	}
	if result.Facebook.Available || result.Facebook.Message == "" {
		t.Fatalf("facebook=%#v", result.Facebook)
	}
}

func TestBuildCampaignSummaryKeepsPartialResults(t *testing.T) {
	originalEmail, originalInstagram := campaignSummaryEmailHistory, campaignSummaryInstagramHistory
	t.Cleanup(func() {
		campaignSummaryEmailHistory = originalEmail
		campaignSummaryInstagramHistory = originalInstagram
	})
	campaignSummaryEmailHistory = func(context.Context) (customCampaignHistory, error) {
		return customCampaignHistory{}, errors.New("email unavailable")
	}
	campaignSummaryInstagramHistory = func(context.Context) (instagramCampaignHistory, error) {
		return instagramCampaignHistory{Campaigns: []instagramCampaignRecord{{Status: "published"}}}, nil
	}
	result := buildCampaignSummary(context.Background())
	if result.Email.Available || result.Email.Message == "" || !result.Instagram.Available {
		t.Fatalf("result=%#v", result)
	}
}
