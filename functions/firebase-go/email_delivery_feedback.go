package ipace

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/resend/resend-go/v3"
)

const (
	resendFeedbackRefreshInterval = 5 * time.Minute
	resendFeedbackPageLimit       = 100
	resendFeedbackMaxPages        = 20
)

var resendEmailsEndpoint = "https://api.resend.com/emails"
var resendFeedbackHTTPClient = http.DefaultClient

func resendFeedbackClient(apiKey string) *resend.Client {
	client := resend.NewCustomClient(resendFeedbackHTTPClient, apiKey)
	baseURL, _ := url.Parse(strings.TrimSuffix(resendEmailsEndpoint, "/emails") + "/")
	client.BaseURL = baseURL
	return client
}

type campaignDeliveryFeedback struct {
	CampaignID     string
	ResendID       string
	ProviderStatus string
	Ref            *firestore.DocumentRef
}

// These mirror only the provider fields retained by the delivery-feedback tests.
// Production requests use Resend's official SDK types directly.
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
		limit := resendFeedbackPageLimit
		options := &resend.ListOptions{Limit: &limit}
		if after != "" {
			options.After = &after
		}
		result, err := resendFeedbackClient(apiKey).Emails.ListWithOptions(ctx, options)
		if err != nil {
			return nil, false, err
		}
		for _, email := range result.Data {
			if _, ok := wanted[email.Id]; !ok {
				continue
			}
			if providerStatus := normaliseResendProviderStatus(email.LastEvent); providerStatus != "" {
				found[email.Id] = providerStatus
			}
		}
		if len(found) == len(wanted) {
			return found, true, nil
		}
		if !result.HasMore || len(result.Data) == 0 {
			return found, len(found) == len(wanted), nil
		}
		after = result.Data[len(result.Data)-1].Id
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
