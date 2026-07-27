package ipace

import (
	"context"
	"net/http"
	"time"
)

type campaignChannelSummary struct {
	Available         bool      `json:"available"`
	Runs              int       `json:"runs"`
	Drafts            int       `json:"drafts,omitempty"`
	Published         int       `json:"published,omitempty"`
	Sent              int       `json:"sent,omitempty"`
	Failed            int       `json:"failed,omitempty"`
	Remaining         int       `json:"remaining,omitempty"`
	Views             int64     `json:"views,omitempty"`
	Reach             int64     `json:"reach,omitempty"`
	TotalInteractions int64     `json:"totalInteractions,omitempty"`
	LatestAt          time.Time `json:"latestAt,omitempty"`
	Message           string    `json:"message,omitempty"`
}

type campaignSummaryResponse struct {
	Email     campaignChannelSummary `json:"email"`
	Instagram campaignChannelSummary `json:"instagram"`
	Facebook  campaignChannelSummary `json:"facebook"`
}

var campaignSummaryEmailHistory = loadCustomCampaignHistory
var campaignSummaryInstagramHistory = loadInstagramCampaignHistory

func AdminCampaignSummary(w http.ResponseWriter, r *http.Request) {
	if !adminCampaignRequestAllowed(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, buildCampaignSummary(r.Context()))
}

func buildCampaignSummary(ctx context.Context) campaignSummaryResponse {
	result := campaignSummaryResponse{
		Facebook: campaignChannelSummary{
			Available: false,
			Message:   "Manual outreach only. Facebook Page Insights are not connected.",
		},
	}

	emailHistory, err := campaignSummaryEmailHistory(ctx)
	if err != nil {
		result.Email.Message = "Email campaign totals are temporarily unavailable."
	} else {
		result.Email.Available = true
		result.Email.Runs = len(emailHistory.Campaigns)
		for _, campaign := range emailHistory.Campaigns {
			result.Email.Sent += campaign.Sent
			result.Email.Failed += campaign.Failed
			result.Email.Remaining += campaign.Remaining
			if campaign.UpdatedAt.After(result.Email.LatestAt) {
				result.Email.LatestAt = campaign.UpdatedAt
			}
		}
	}

	instagramHistory, err := campaignSummaryInstagramHistory(ctx)
	if err != nil {
		result.Instagram.Message = "Instagram campaign totals are temporarily unavailable."
	} else {
		result.Instagram.Available = true
		result.Instagram.Runs = len(instagramHistory.Campaigns)
		for _, campaign := range instagramHistory.Campaigns {
			switch campaign.Status {
			case "draft":
				result.Instagram.Drafts++
			case "published":
				result.Instagram.Published++
			case "failed":
				result.Instagram.Failed++
			}
			if campaign.Insights.Available {
				result.Instagram.Views += campaign.Insights.Views
				result.Instagram.Reach += campaign.Insights.Reach
				result.Instagram.TotalInteractions += campaign.Insights.TotalInteractions
			}
			if campaign.UpdatedAt.After(result.Instagram.LatestAt) {
				result.Instagram.LatestAt = campaign.UpdatedAt
			}
		}
		if !instagramHistory.InsightsConfigured {
			result.Instagram.Message = "Publishing history is available; Instagram insights are not configured."
		}
	}
	return result
}
