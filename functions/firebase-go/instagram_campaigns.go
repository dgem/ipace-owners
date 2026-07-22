package ipace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const instagramCaptionLimit = 2200

type instagramDraftRequest struct {
	MediaPath     string `json:"mediaPath"`
	Caption       string `json:"caption"`
	MediaReviewed bool   `json:"mediaReviewed"`
}

type instagramPublishRequest struct {
	CampaignID    string `json:"campaignId"`
	MediaPath     string `json:"mediaPath"`
	Caption       string `json:"caption"`
	MediaReviewed bool   `json:"mediaReviewed"`
	Confirmation  string `json:"confirmation"`
}

type instagramPreview struct {
	CampaignID   string `json:"campaignId"`
	MediaPath    string `json:"mediaPath"`
	MediaURL     string `json:"mediaUrl"`
	Caption      string `json:"caption"`
	Confirmation string `json:"confirmation"`
	Configured   bool   `json:"configured"`
}

type instagramPublishResult struct {
	CampaignID string `json:"campaignId"`
	MediaID    string `json:"mediaId"`
}

var instagramCampaignAuthorize = campaignAuthorize
var instagramCampaignPublish = publishInstagramReel
var instagramCampaignReserve = reserveInstagramCampaign
var instagramCampaignComplete = completeInstagramCampaign
var instagramHTTPClient = http.DefaultClient
var instagramPollWait = func() { time.Sleep(2 * time.Second) }

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
	writeJSON(w, http.StatusOK, preview)
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
	preview, err := previewInstagramCampaign(instagramDraftRequest{MediaPath: input.MediaPath, Caption: input.Caption, MediaReviewed: input.MediaReviewed})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if input.CampaignID != preview.CampaignID || input.Confirmation != preview.Confirmation {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Campaign or confirmation changed; preview again. Nothing was published."})
		return
	}
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
				"mediaPath": preview.MediaPath, "caption": preview.Caption,
				"createdAt": firestore.ServerTimestamp, "updatedAt": firestore.ServerTimestamp,
			})
		}
		if getErr != nil {
			return getErr
		}
		var record struct {
			Status  string `firestore:"status"`
			MediaID string `firestore:"mediaId"`
		}
		if err := snapshot.DataTo(&record); err != nil {
			return err
		}
		if record.Status == "published" && record.MediaID != "" {
			existingMediaID = record.MediaID
			return nil
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
		CampaignID: "instagram-" + strings.ToLower(short), MediaPath: mediaPath, MediaURL: mediaURL,
		Caption: caption, Confirmation: "PUBLISH " + short,
		Configured: mediaURL != "" && instagramConfigurationValid(),
	}, nil
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
