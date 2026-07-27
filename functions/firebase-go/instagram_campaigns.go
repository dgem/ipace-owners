package ipace

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const instagramCaptionLimit = 2200

type instagramDraftRequest struct {
	CampaignID       string `json:"campaignId"`
	SourceCampaignID string `json:"sourceCampaignId"`
	Name             string `json:"name"`
	MediaPath        string `json:"mediaPath"`
	Caption          string `json:"caption"`
	MediaReviewed    bool   `json:"mediaReviewed"`
}

type instagramPublishRequest struct {
	CampaignID    string `json:"campaignId"`
	Name          string `json:"name"`
	MediaPath     string `json:"mediaPath"`
	Caption       string `json:"caption"`
	MediaReviewed bool   `json:"mediaReviewed"`
	Confirmation  string `json:"confirmation"`
}

type instagramPreview struct {
	CampaignID       string `json:"campaignId"`
	SourceCampaignID string `json:"sourceCampaignId,omitempty"`
	Name             string `json:"name"`
	MediaPath        string `json:"mediaPath"`
	MediaURL         string `json:"mediaUrl"`
	Caption          string `json:"caption"`
	Confirmation     string `json:"confirmation"`
	Configured       bool   `json:"configured"`
}

type instagramPublishResult struct {
	CampaignID string `json:"campaignId"`
	MediaID    string `json:"mediaId"`
}

type instagramInsights struct {
	Available         bool      `json:"available" firestore:"available"`
	Views             int64     `json:"views" firestore:"views"`
	Reach             int64     `json:"reach" firestore:"reach"`
	Likes             int64     `json:"likes" firestore:"likes"`
	Comments          int64     `json:"comments" firestore:"comments"`
	Saved             int64     `json:"saved" firestore:"saved"`
	Shares            int64     `json:"shares" firestore:"shares"`
	TotalInteractions int64     `json:"totalInteractions" firestore:"totalInteractions"`
	UpdatedAt         time.Time `json:"updatedAt,omitempty" firestore:"updatedAt,omitempty"`
}

type instagramCampaignRecord struct {
	CampaignID       string            `json:"campaignId" firestore:"id"`
	SourceCampaignID string            `json:"sourceCampaignId,omitempty" firestore:"sourceCampaignId,omitempty"`
	Name             string            `json:"name" firestore:"name"`
	Status           string            `json:"status" firestore:"status"`
	MediaPath        string            `json:"mediaPath" firestore:"mediaPath"`
	Caption          string            `json:"caption" firestore:"caption"`
	MediaID          string            `json:"mediaId,omitempty" firestore:"mediaId,omitempty"`
	CreatedAt        time.Time         `json:"createdAt" firestore:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt" firestore:"updatedAt"`
	PublishedAt      time.Time         `json:"publishedAt,omitempty" firestore:"publishedAt,omitempty"`
	Insights         instagramInsights `json:"insights" firestore:"insights"`
}

type instagramCampaignHistory struct {
	Campaigns          []instagramCampaignRecord `json:"campaigns"`
	InsightsConfigured bool                      `json:"insightsConfigured"`
}

var instagramCampaignAuthorize = campaignAuthorize
var instagramCampaignPublish = publishInstagramReel
var instagramCampaignReserve = reserveInstagramCampaign
var instagramCampaignComplete = completeInstagramCampaign
var instagramCampaignSaveDraft = saveInstagramCampaignDraft
var instagramCampaignLoad = loadInstagramCampaign
var instagramCampaignHistoryLoad = loadInstagramCampaignHistory
var instagramCampaignInsights = fetchInstagramInsights
var instagramCampaignNewID = newInstagramCampaignID
var instagramHTTPClient = http.DefaultClient
var instagramPollWait = func() { time.Sleep(2 * time.Second) }

var instagramCampaignIDRegexp = regexp.MustCompile(`^instagram-[a-z0-9][a-z0-9-]{9,79}$`)

func AdminInstagramPreview(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := instagramCampaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	var input instagramDraftRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := previewInstagramCampaign(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	record, err := instagramCampaignSaveDraft(r.Context(), input, preview)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	preview.CampaignID = record.CampaignID
	preview.SourceCampaignID = record.SourceCampaignID
	preview.Name = record.Name
	writeJSON(w, http.StatusOK, preview)
}

func AdminInstagramCampaignHistory(w http.ResponseWriter, r *http.Request) {
	if !adminCampaignRequestAllowed(w, r) {
		return
	}
	history, err := instagramCampaignHistoryLoad(r.Context())
	if err != nil {
		logEvent("admin-instagram-campaign-history", "error", "history load failed", map[string]any{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load Instagram campaign history"})
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func AdminInstagramPublish(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := instagramCampaignAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	var input instagramPublishRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	preview, err := previewInstagramCampaign(instagramDraftRequest{Name: input.Name, MediaPath: input.MediaPath, Caption: input.Caption, MediaReviewed: input.MediaReviewed})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if !instagramCampaignIDRegexp.MatchString(input.CampaignID) || input.Confirmation != preview.Confirmation {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Campaign or confirmation changed; preview again. Nothing was published."})
		return
	}
	record, err := instagramCampaignLoad(r.Context(), input.CampaignID)
	if err != nil || record.Name != preview.Name || record.MediaPath != preview.MediaPath || record.Caption != preview.Caption {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Saved campaign content changed; preview again. Nothing was published."})
		return
	}
	preview.CampaignID = record.CampaignID
	preview.SourceCampaignID = record.SourceCampaignID
	if !preview.Configured {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Instagram publishing is not configured"})
		return
	}
	existingMediaID, err := instagramCampaignReserve(r.Context(), preview)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	if existingMediaID != "" {
		writeJSON(w, http.StatusOK, instagramPublishResult{CampaignID: preview.CampaignID, MediaID: existingMediaID})
		return
	}
	mediaID, err := instagramCampaignPublish(r.Context(), preview)
	if err != nil {
		_ = instagramCampaignComplete(r.Context(), preview.CampaignID, "", err)
		logEvent("admin-instagram-publish", "error", "publish failed", map[string]any{"campaignId": preview.CampaignID, "error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Instagram did not accept the campaign; nothing further was sent"})
		return
	}
	if err := instagramCampaignComplete(r.Context(), preview.CampaignID, mediaID, nil); err != nil {
		logEvent("admin-instagram-publish", "error", "publish ledger update failed", map[string]any{"campaignId": preview.CampaignID, "mediaId": mediaID, "error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Instagram published the Reel, but the local campaign record could not be completed; verify the Instagram account before retrying"})
		return
	}
	logEvent("admin-instagram-publish", "info", "campaign published", map[string]any{"campaignId": preview.CampaignID, "mediaId": mediaID})
	writeJSON(w, http.StatusOK, instagramPublishResult{CampaignID: preview.CampaignID, MediaID: mediaID})
}

func newInstagramCampaignID(base string) string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return base + "-" + strings.ToLower(strconv.FormatInt(time.Now().UnixNano(), 36))
	}
	return base + "-" + hex.EncodeToString(suffix[:])
}

func saveInstagramCampaignDraft(ctx context.Context, input instagramDraftRequest, preview instagramPreview) (instagramCampaignRecord, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return instagramCampaignRecord{}, fmt.Errorf("campaign ledger is unavailable")
	}
	campaignID := strings.TrimSpace(input.CampaignID)
	sourceID := strings.TrimSpace(input.SourceCampaignID)
	if sourceID != "" && !instagramCampaignIDRegexp.MatchString(sourceID) {
		return instagramCampaignRecord{}, fmt.Errorf("source campaign ID is invalid")
	}
	if campaignID == "" {
		campaignID = preview.CampaignID
		if sourceID != "" {
			campaignID = instagramCampaignNewID(campaignID)
		}
	}
	if !instagramCampaignIDRegexp.MatchString(campaignID) {
		return instagramCampaignRecord{}, fmt.Errorf("campaign ID is invalid")
	}

	ref := db.Collection("instagramCampaigns").Doc(campaignID)
	record := instagramCampaignRecord{}
	err = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		now := time.Now().UTC()
		snapshot, getErr := tx.Get(ref)
		if status.Code(getErr) == codes.NotFound {
			record = instagramCampaignRecord{
				CampaignID: campaignID, SourceCampaignID: sourceID, Name: preview.Name,
				Status: "draft", MediaPath: preview.MediaPath, Caption: preview.Caption,
				CreatedAt: now, UpdatedAt: now,
			}
			return tx.Create(ref, record)
		}
		if getErr != nil {
			return getErr
		}
		if err := snapshot.DataTo(&record); err != nil {
			return err
		}
		if record.Status != "draft" {
			return fmt.Errorf("published or in-progress campaigns must be cloned before editing")
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		}
		record.CampaignID = campaignID
		record.SourceCampaignID = sourceID
		record.Name = preview.Name
		record.MediaPath = preview.MediaPath
		record.Caption = preview.Caption
		record.UpdatedAt = now
		return tx.Set(ref, record)
	})
	if err != nil {
		return instagramCampaignRecord{}, err
	}
	return record, nil
}

func loadInstagramCampaign(ctx context.Context, campaignID string) (instagramCampaignRecord, error) {
	if !instagramCampaignIDRegexp.MatchString(campaignID) {
		return instagramCampaignRecord{}, fmt.Errorf("campaign ID is invalid")
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return instagramCampaignRecord{}, err
	}
	snapshot, err := db.Collection("instagramCampaigns").Doc(campaignID).Get(ctx)
	if err != nil {
		return instagramCampaignRecord{}, err
	}
	var record instagramCampaignRecord
	if err := snapshot.DataTo(&record); err != nil {
		return instagramCampaignRecord{}, err
	}
	if record.CampaignID == "" {
		record.CampaignID = snapshot.Ref.ID
	}
	if record.Name == "" {
		record.Name = instagramCampaignNameFromCaption(record.Caption)
	}
	return record, nil
}

func loadInstagramCampaignHistory(ctx context.Context) (instagramCampaignHistory, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return instagramCampaignHistory{}, err
	}
	records := []instagramCampaignRecord{}
	iter := db.Collection("instagramCampaigns").Documents(ctx)
	for {
		doc, nextErr := iter.Next()
		if nextErr == iterator.Done {
			break
		}
		if nextErr != nil {
			iter.Stop()
			return instagramCampaignHistory{}, nextErr
		}
		var record instagramCampaignRecord
		if doc.DataTo(&record) != nil {
			continue
		}
		if record.CampaignID == "" {
			record.CampaignID = doc.Ref.ID
		}
		if record.Name == "" {
			record.Name = instagramCampaignNameFromCaption(record.Caption)
		}
		records = append(records, record)
	}
	iter.Stop()
	sort.Slice(records, func(i, j int) bool { return records[i].UpdatedAt.After(records[j].UpdatedAt) })

	insightsConfigured := instagramInsightsConfigurationValid()
	if insightsConfigured {
		refreshed := 0
		for index := range records {
			record := &records[index]
			if record.Status != "published" || record.MediaID == "" || refreshed >= 10 {
				continue
			}
			if record.Insights.Available && time.Since(record.Insights.UpdatedAt) < 15*time.Minute {
				continue
			}
			insights, insightErr := instagramCampaignInsights(ctx, record.MediaID)
			if insightErr != nil {
				continue
			}
			record.Insights = insights
			refreshed++
			_, _ = db.Collection("instagramCampaigns").Doc(record.CampaignID).Set(ctx, map[string]any{
				"insights": insights,
			}, firestore.MergeAll)
		}
	}
	return instagramCampaignHistory{Campaigns: records, InsightsConfigured: insightsConfigured}, nil
}

func instagramInsightsConfigurationValid() bool {
	return instagramConfigurationValid()
}

func fetchInstagramInsights(ctx context.Context, mediaID string) (instagramInsights, error) {
	if !instagramInsightsConfigurationValid() || strings.TrimSpace(mediaID) == "" {
		return instagramInsights{}, fmt.Errorf("Instagram insights are not configured")
	}
	version := strings.TrimSpace(os.Getenv("INSTAGRAM_GRAPH_API_VERSION"))
	endpoint := "https://graph.instagram.com/" + url.PathEscape(version) + "/" + url.PathEscape(mediaID) + "/insights"
	query := url.Values{"metric": {"views,reach,likes,comments,saved,shares,total_interactions"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN")))
	res, err := instagramHTTPClient.Do(req)
	if err != nil {
		return instagramInsights{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return instagramInsights{}, fmt.Errorf("Instagram insights returned status %d", res.StatusCode)
	}
	var payload struct {
		Data []struct {
			Name   string `json:"name"`
			Values []struct {
				Value any `json:"value"`
			} `json:"values"`
			TotalValue struct {
				Value any `json:"value"`
			} `json:"total_value"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return instagramInsights{}, fmt.Errorf("invalid Instagram insights response")
	}
	result := instagramInsights{Available: true, UpdatedAt: time.Now().UTC()}
	for _, metric := range payload.Data {
		value := metricNumber(metric.TotalValue.Value)
		if len(metric.Values) > 0 {
			value = metricNumber(metric.Values[0].Value)
		}
		switch metric.Name {
		case "views":
			result.Views = value
		case "reach":
			result.Reach = value
		case "likes":
			result.Likes = value
		case "comments":
			result.Comments = value
		case "saved":
			result.Saved = value
		case "shares":
			result.Shares = value
		case "total_interactions":
			result.TotalInteractions = value
		}
	}
	return result, nil
}

func metricNumber(value any) int64 {
	switch number := value.(type) {
	case float64:
		return int64(number)
	case json.Number:
		result, _ := number.Int64()
		return result
	case int64:
		return number
	case int:
		return int64(number)
	default:
		return 0
	}
}

func reserveInstagramCampaign(ctx context.Context, preview instagramPreview) (string, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return "", fmt.Errorf("campaign ledger is unavailable")
	}
	ref := db.Collection("instagramCampaigns").Doc(preview.CampaignID)
	existingMediaID := ""
	err = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, getErr := tx.Get(ref)
		if status.Code(getErr) == codes.NotFound {
			return tx.Create(ref, map[string]any{
				"id": preview.CampaignID, "type": "instagramReel", "status": "publishing",
				"name": preview.Name, "sourceCampaignId": preview.SourceCampaignID,
				"mediaPath": preview.MediaPath, "caption": preview.Caption,
				"createdAt": firestore.ServerTimestamp, "updatedAt": firestore.ServerTimestamp,
			})
		}
		if getErr != nil {
			return getErr
		}
		var record struct {
			Status    string `firestore:"status"`
			MediaID   string `firestore:"mediaId"`
			Name      string `firestore:"name"`
			MediaPath string `firestore:"mediaPath"`
			Caption   string `firestore:"caption"`
		}
		if err := snapshot.DataTo(&record); err != nil {
			return err
		}
		if record.Status == "published" && record.MediaID != "" {
			existingMediaID = record.MediaID
			return nil
		}
		if record.Status == "draft" {
			if record.Name != preview.Name || record.MediaPath != preview.MediaPath || record.Caption != preview.Caption {
				return fmt.Errorf("saved campaign content changed; preview again")
			}
			return tx.Update(ref, []firestore.Update{
				{Path: "status", Value: "publishing"},
				{Path: "updatedAt", Value: firestore.ServerTimestamp},
			})
		}
		return fmt.Errorf("this exact campaign is already %s; verify Instagram before creating a different draft", record.Status)
	})
	if err != nil {
		return "", err
	}
	return existingMediaID, nil
}

func completeInstagramCampaign(ctx context.Context, campaignID, mediaID string, publishErr error) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	fields := []firestore.Update{{Path: "updatedAt", Value: firestore.ServerTimestamp}}
	if publishErr != nil {
		fields = append(fields, firestore.Update{Path: "status", Value: "failed"})
	} else {
		fields = append(fields, firestore.Update{Path: "status", Value: "published"}, firestore.Update{Path: "mediaId", Value: mediaID}, firestore.Update{Path: "publishedAt", Value: firestore.ServerTimestamp})
	}
	_, err = db.Collection("instagramCampaigns").Doc(campaignID).Update(ctx, fields)
	return err
}

func previewInstagramCampaign(input instagramDraftRequest) (instagramPreview, error) {
	if !input.MediaReviewed {
		return instagramPreview{}, fmt.Errorf("watch and approve the complete final video before previewing")
	}
	caption := strings.TrimSpace(input.Caption)
	if caption == "" || len([]rune(caption)) > instagramCaptionLimit {
		return instagramPreview{}, fmt.Errorf("caption must contain between 1 and %d characters", instagramCaptionLimit)
	}
	mediaPath := strings.TrimSpace(input.MediaPath)
	parsed, err := url.Parse(mediaPath)
	if err != nil || !strings.HasPrefix(mediaPath, "/") || strings.HasPrefix(mediaPath, "//") || parsed.IsAbs() || parsed.RawQuery != "" || parsed.Fragment != "" {
		return instagramPreview{}, fmt.Errorf("media path must be a site-relative URL without a query or fragment")
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	if ext != ".mp4" && ext != ".mov" {
		return instagramPreview{}, fmt.Errorf("media must be an MP4 or MOV file")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = instagramCampaignNameFromCaption(caption)
	}
	if len([]rune(name)) > 100 {
		return instagramPreview{}, fmt.Errorf("campaign name must contain at most 100 characters")
	}
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("INSTAGRAM_MEDIA_BASE_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("RESEND_ASSET_BASE_URL")), "/")
	}
	mediaURL := ""
	if base != "" {
		baseURL, parseErr := url.Parse(base)
		if parseErr != nil || baseURL.Scheme != "https" || baseURL.Host == "" {
			return instagramPreview{}, fmt.Errorf("Instagram media base URL must be an absolute HTTPS URL")
		}
		mediaURL = base + mediaPath
	}
	digest := sha256.Sum256([]byte(mediaPath + "\n" + caption))
	short := strings.ToUpper(hex.EncodeToString(digest[:])[:10])
	return instagramPreview{
		CampaignID: "instagram-" + strings.ToLower(short), Name: name, MediaPath: mediaPath, MediaURL: mediaURL,
		Caption: caption, Confirmation: "PUBLISH " + short,
		Configured: mediaURL != "" && instagramConfigurationValid(),
	}, nil
}

func instagramCampaignNameFromCaption(caption string) string {
	name := strings.TrimSpace(strings.SplitN(caption, "\n", 2)[0])
	runes := []rune(name)
	if len(runes) > 80 {
		name = strings.TrimSpace(string(runes[:77])) + "…"
	}
	if name == "" {
		return "Instagram Reel"
	}
	return name
}

func instagramConfigurationValid() bool {
	version := strings.TrimSpace(os.Getenv("INSTAGRAM_GRAPH_API_VERSION"))
	return regexp.MustCompile(`^v[0-9]+\.[0-9]+$`).MatchString(version) &&
		strings.TrimSpace(os.Getenv("INSTAGRAM_USER_ID")) != "" &&
		strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN")) != ""
}

func publishInstagramReel(ctx context.Context, preview instagramPreview) (string, error) {
	version := strings.TrimSpace(os.Getenv("INSTAGRAM_GRAPH_API_VERSION"))
	userID := strings.TrimSpace(os.Getenv("INSTAGRAM_USER_ID"))
	token := strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN"))
	base := "https://graph.instagram.com/" + url.PathEscape(version)
	create := url.Values{"media_type": {"REELS"}, "video_url": {preview.MediaURL}, "caption": {preview.Caption}, "share_to_feed": {"true"}}
	containerID, err := instagramPostID(ctx, base+"/"+url.PathEscape(userID)+"/media", create)
	if err != nil {
		return "", fmt.Errorf("create Reel container: %w", err)
	}

	finished := false
	for attempt := 0; attempt < 30; attempt++ {
		statusURL := base + "/" + url.PathEscape(containerID) + "?fields=status_code"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := instagramHTTPClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("check Reel processing: %w", err)
		}
		var status struct {
			StatusCode string `json:"status_code"`
		}
		decodeErr := json.NewDecoder(res.Body).Decode(&status)
		res.Body.Close()
		if res.StatusCode >= 300 || decodeErr != nil {
			return "", fmt.Errorf("check Reel processing returned status %d", res.StatusCode)
		}
		if status.StatusCode == "FINISHED" {
			finished = true
			break
		}
		if status.StatusCode == "ERROR" || status.StatusCode == "EXPIRED" {
			return "", fmt.Errorf("Reel processing ended with %s", status.StatusCode)
		}
		instagramPollWait()
	}
	if !finished {
		return "", fmt.Errorf("Reel processing timed out")
	}
	publish := url.Values{"creation_id": {containerID}}
	mediaID, err := instagramPostID(ctx, base+"/"+url.PathEscape(userID)+"/media_publish", publish)
	if err != nil {
		return "", fmt.Errorf("publish Reel: %w", err)
	}
	return mediaID, nil
}

func instagramPostID(ctx context.Context, endpoint string, form url.Values) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(os.Getenv("INSTAGRAM_ACCESS_TOKEN")))
	res, err := instagramHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("invalid provider response")
	}
	if res.StatusCode >= 300 || payload.ID == "" {
		return "", fmt.Errorf("provider returned status %d", res.StatusCode)
	}
	return payload.ID, nil
}
