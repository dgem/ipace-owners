package ipace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	resendFeedbackRefreshInterval = 5 * time.Minute
	resendFeedbackPageLimit       = 100
	resendFeedbackMaxPages        = 20
)

var resendEmailsEndpoint = "https://api.resend.com/emails"
var resendFeedbackHTTPClient = http.DefaultClient

type campaignDeliveryFeedback struct {
	CampaignID     string
	ResendID       string
	ProviderStatus string
	Ref            *firestore.DocumentRef
}

type resendEmailStatus struct {
	ID        string `json:"id"`
	LastEvent string `json:"last_event"`
}

type resendEmailStatusPage struct {
	HasMore bool                `json:"has_more"`
	Data    []resendEmailStatus `json:"data"`
}

type resendFeedbackCache struct {
	CheckedAt time.Time `firestore:"checkedAt"`
	Complete  bool      `firestore:"complete"`
}

func refreshCampaignDeliveryFeedback(ctx context.Context, db *firestore.Client, deliveries []campaignDeliveryFeedback) (time.Time, bool, string) {
	if len(deliveries) == 0 {
		return time.Time{}, false, ""
	}
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	if apiKey == "" {
		return time.Time{}, false, "Delivery feedback is unavailable because Resend is not configured."
	}
	cacheRef := db.Collection("campaignMetadata").Doc("resendDeliveryFeedback")
	var cache resendFeedbackCache
	if snapshot, err := cacheRef.Get(ctx); err == nil && snapshot.DataTo(&cache) == nil &&
		!cache.CheckedAt.IsZero() && time.Since(cache.CheckedAt) < resendFeedbackRefreshInterval {
		return cache.CheckedAt, true, feedbackCoverageMessage(cache.Complete)
	}

	wanted := make(map[string]struct{}, len(deliveries))
	allHaveProviderIDs := true
	for _, delivery := range deliveries {
		if delivery.ResendID != "" {
			wanted[delivery.ResendID] = struct{}{}
		} else {
			allHaveProviderIDs = false
		}
	}
	if len(wanted) == 0 {
		return time.Time{}, false, "Delivery feedback is unavailable for campaign records without provider IDs."
	}
	statuses, complete, err := fetchResendEmailStatuses(ctx, apiKey, wanted)
	if err != nil {
		if !cache.CheckedAt.IsZero() {
			return cache.CheckedAt, true, "Showing cached delivery feedback; Resend could not be refreshed."
		}
		return time.Time{}, false, "Delivery feedback is temporarily unavailable."
	}
	complete = complete && allHaveProviderIDs

	batch := db.Batch()
	pending := 0
	commit := func() error {
		if pending == 0 {
			return nil
		}
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
		batch = db.Batch()
		pending = 0
		return nil
	}
	for _, index := range mergeResendEmailStatuses(deliveries, statuses) {
		delivery := deliveries[index]
		if delivery.Ref == nil {
			continue
		}
		batch.Set(delivery.Ref, map[string]any{
			"providerStatus":    delivery.ProviderStatus,
			"providerUpdatedAt": firestore.ServerTimestamp,
		}, firestore.MergeAll)
		pending++
		if pending == 400 {
			if err := commit(); err != nil {
				return time.Now().UTC(), true, "Delivery feedback was refreshed, but updated statuses could not be cached."
			}
		}
	}
	if err := commit(); err != nil {
		return time.Now().UTC(), true, "Delivery feedback was refreshed, but updated statuses could not be cached."
	}
	now := time.Now().UTC()
	if _, err := cacheRef.Set(ctx, resendFeedbackCache{CheckedAt: now, Complete: complete}); err != nil {
		return now, true, feedbackCoverageMessage(complete)
	}
	return now, true, feedbackCoverageMessage(complete)
}

func mergeResendEmailStatuses(deliveries []campaignDeliveryFeedback, statuses map[string]string) []int {
	changed := make([]int, 0, len(statuses))
	for index := range deliveries {
		providerStatus, ok := statuses[deliveries[index].ResendID]
		if !ok || providerStatus == deliveries[index].ProviderStatus {
			continue
		}
		deliveries[index].ProviderStatus = providerStatus
		changed = append(changed, index)
	}
	return changed
}

func fetchResendEmailStatuses(ctx context.Context, apiKey string, wanted map[string]struct{}) (map[string]string, bool, error) {
	found := make(map[string]string, len(wanted))
	after := ""
	for page := 0; page < resendFeedbackMaxPages; page++ {
		endpoint, err := url.Parse(resendEmailsEndpoint)
		if err != nil {
			return nil, false, err
		}
		query := endpoint.Query()
		query.Set("limit", fmt.Sprint(resendFeedbackPageLimit))
		if after != "" {
			query.Set("after", after)
		}
		endpoint.RawQuery = query.Encode()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, false, err
		}
		request.Header.Set("Authorization", "Bearer "+apiKey)
		response, err := resendFeedbackHTTPClient.Do(request)
		if err != nil {
			return nil, false, err
		}
		var result resendEmailStatusPage
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result)
		response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode > 299 {
			return nil, false, fmt.Errorf("resend returned %d", response.StatusCode)
		}
		if decodeErr != nil {
			return nil, false, decodeErr
		}
		for _, email := range result.Data {
			if _, ok := wanted[email.ID]; !ok {
				continue
			}
			if providerStatus := normaliseResendProviderStatus(email.LastEvent); providerStatus != "" {
				found[email.ID] = providerStatus
			}
		}
		if len(found) == len(wanted) {
			return found, true, nil
		}
		if !result.HasMore || len(result.Data) == 0 {
			return found, len(found) == len(wanted), nil
		}
		after = result.Data[len(result.Data)-1].ID
	}
	return found, len(found) == len(wanted), nil
}

func normaliseResendProviderStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "queued", "scheduled", "sent", "delivered", "delivery_delayed", "bounced",
		"failed", "suppressed", "complained", "opened", "clicked", "canceled":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func feedbackCoverageMessage(complete bool) string {
	if complete {
		return ""
	}
	return "Delivery feedback is partial because some provider records are no longer available."
}

func addDeliveryFeedback(record *customCampaignRecord, providerStatus string) {
	switch normaliseResendProviderStatus(providerStatus) {
	case "delivered":
		record.Delivered++
	case "opened":
		record.Delivered++
		record.Opened++
	case "clicked":
		record.Delivered++
		record.Clicked++
	case "complained":
		record.Delivered++
		record.Complained++
	case "delivery_delayed":
		record.Delayed++
	case "bounced":
		record.Bounced++
		record.Undeliverable++
	case "failed", "canceled":
		record.ProviderFailed++
		record.Undeliverable++
	case "suppressed":
		record.Suppressed++
		record.Undeliverable++
	case "queued", "scheduled", "sent":
		record.AwaitingDelivery++
	}
}
