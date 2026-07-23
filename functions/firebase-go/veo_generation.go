package ipace

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	veoPromptLimit      = 12000
	veoDeliveryTokenTTL = 24 * time.Hour
	veoLocation         = "us-central1"
)

var veoRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,100}$`)

type veoGenerateRequest struct {
	RequestID    string `json:"requestId"`
	Prompt       string `json:"prompt"`
	Confirmation string `json:"confirmation"`
}

type veoStatusRequest struct {
	JobID string `json:"jobId"`
}

type veoGenerationJob struct {
	ID             string    `firestore:"id"`
	RequestIDHash  string    `firestore:"requestIdHash"`
	Prompt         string    `firestore:"prompt"`
	PromptHash     string    `firestore:"promptHash"`
	Status         string    `firestore:"status"`
	Phase          string    `firestore:"phase"`
	OperationName  string    `firestore:"operationName"`
	InitialGCSURI  string    `firestore:"initialGcsUri,omitempty"`
	MasterObject   string    `firestore:"masterObject,omitempty"`
	ModelID        string    `firestore:"modelId"`
	Location       string    `firestore:"location"`
	FailureCode    string    `firestore:"failureCode,omitempty"`
	FailureDetail  string    `firestore:"failureDetail,omitempty"`
	ProviderCode   int       `firestore:"providerCode,omitempty"`
	ProviderStatus string    `firestore:"providerStatus,omitempty"`
	DeliveryHash   string    `firestore:"deliveryTokenHash,omitempty"`
	DeliveryExpiry time.Time `firestore:"deliveryExpiresAt,omitempty"`
}

type veoGenerationResponse struct {
	JobID       string `json:"jobId"`
	Status      string `json:"status"`
	Phase       string `json:"phase"`
	ModelID     string `json:"modelId"`
	Location    string `json:"location"`
	MediaPath   string `json:"mediaPath,omitempty"`
	Message     string `json:"message"`
	FailureCode string `json:"failureCode,omitempty"`
	Configured  bool   `json:"configured"`
}

type veoOperationError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

type veoOperation struct {
	Name     string `json:"name"`
	Done     bool   `json:"done"`
	Response struct {
		FilteredCount   int      `json:"raiMediaFilteredCount"`
		FilteredReasons []string `json:"raiMediaFilteredReasons"`
		Videos          []struct {
			GCSURI   string `json:"gcsUri"`
			MimeType string `json:"mimeType"`
		} `json:"videos"`
	} `json:"response"`
	Error *veoOperationError `json:"error"`
}

type veoConfig struct {
	ProjectID string
	Location  string
	ModelID   string
	Bucket    string
}

var veoAuthorize = campaignAuthorize
var veoStartOperation = startVeoOperation
var veoFetchOperation = fetchVeoOperation
var veoLoadJob = loadVeoGenerationJob
var veoReserveJob = reserveVeoGenerationJob
var veoUpdateJob = updateVeoGenerationJob
var veoClaimExtension = claimVeoExtension
var veoPromoteMaster = promoteVeoMaster
var veoIssueDelivery = issueVeoDeliveryToken
var veoHTTPClient = func(ctx context.Context) (*http.Client, error) {
	return google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
}

func AdminInstagramGenerate(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := veoAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var input veoGenerateRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
		return
	}
	config, err := currentVeoConfig()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Veo generation is not configured"})
		return
	}
	prompt := strings.TrimSpace(input.Prompt)
	if !veoRequestIDPattern.MatchString(input.RequestID) || prompt == "" || len([]rune(prompt)) > veoPromptLimit {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "A valid request ID and prompt are required"})
		return
	}
	if input.Confirmation != "GENERATE VIDEO" {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "Type GENERATE VIDEO exactly before starting a billable job"})
		return
	}
	job, created, err := veoReserveJob(r.Context(), input.RequestID, prompt, config)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}
	if !created {
		response, responseErr := veoResponse(r.Context(), job)
		if responseErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not restore the existing generation job"})
			return
		}
		writeJSON(w, http.StatusOK, response)
		return
	}
	operationName, err := veoStartOperation(r.Context(), config, job.ID, prompt, "", "initial")
	if err != nil {
		job.Status = "failed"
		job.FailureCode = "vertex_start_failed"
		_ = veoUpdateJob(r.Context(), job)
		logEvent("admin-instagram-generate", "error", "Veo start failed", map[string]any{"jobId": job.ID, "error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Vertex AI did not start the generation job"})
		return
	}
	job.Status = "processing"
	job.OperationName = operationName
	if err := veoUpdateJob(r.Context(), job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Generation started but its operation record could not be saved; contact an administrator before retrying"})
		return
	}
	logEvent("admin-instagram-generate", "info", "Veo generation started", map[string]any{"jobId": job.ID, "phase": job.Phase, "modelId": job.ModelID})
	writeJSON(w, http.StatusAccepted, generationResponse(job, "Veo is generating the eight-second approach and controlled stop."))
}

func AdminInstagramGenerationStatus(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
		return
	}
	if err := veoAuthorize(r.Context(), r); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Admin role required"})
		return
	}
	var input veoStatusRequest
	if err := decodeJSON(r, &input); err != nil || !validVeoJobID(input.JobID) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Valid generation job ID required"})
		return
	}
	job, err := veoLoadJob(r.Context(), input.JobID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Generation job not found"})
		return
	}
	job, err = progressVeoGeneration(r.Context(), job)
	if err != nil {
		logEvent("admin-instagram-generation-status", "error", "Veo progress failed", map[string]any{"jobId": job.ID, "phase": job.Phase, "error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "Could not advance the Veo generation job; retry status shortly"})
		return
	}
	response, err := veoResponse(r.Context(), job)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not prepare the generated master for review"})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func progressVeoGeneration(ctx context.Context, job veoGenerationJob) (veoGenerationJob, error) {
	if job.Status == "completed" || job.Status == "failed" || job.OperationName == "" {
		return job, nil
	}
	config, err := currentVeoConfig()
	if err != nil {
		return job, err
	}
	operation, err := veoFetchOperation(ctx, config, job.OperationName)
	if err != nil || !operation.Done {
		return job, err
	}
	if operation.Error != nil {
		job.Status = "failed"
		job.FailureCode, job.FailureDetail = classifyVeoOperationError(operation.Error)
		job.ProviderCode = operation.Error.Code
		job.ProviderStatus = strings.TrimSpace(operation.Error.Status)
		if err := veoUpdateJob(ctx, job); err != nil {
			return job, err
		}
		logEvent("admin-instagram-generation-status", "error", "Veo operation failed", map[string]any{
			"jobId": job.ID, "phase": job.Phase, "failureCode": job.FailureCode,
			"providerCode": job.ProviderCode, "providerStatus": job.ProviderStatus,
		})
		return job, nil
	}
	if operation.Response.FilteredCount > 0 || len(operation.Response.Videos) != 1 {
		job.Status = "failed"
		job.FailureCode = "vertex_output_filtered"
		if err := veoUpdateJob(ctx, job); err != nil {
			return job, err
		}
		return job, nil
	}
	videoURI := operation.Response.Videos[0].GCSURI
	if err := validateVeoOutputURI(config, job.ID, videoURI); err != nil {
		return job, err
	}
	if job.Phase == "initial" {
		claimedJob, claimed, err := veoClaimExtension(ctx, job, videoURI)
		if err != nil || !claimed {
			return claimedJob, err
		}
		operationName, err := veoStartOperation(ctx, config, claimedJob.ID, claimedJob.Prompt, videoURI, "extension")
		if err != nil {
			claimedJob.Status = "failed"
			claimedJob.FailureCode = "vertex_extension_start_failed"
			if updateErr := veoUpdateJob(ctx, claimedJob); updateErr != nil {
				return claimedJob, updateErr
			}
			return claimedJob, nil
		}
		claimedJob.Phase = "extension"
		claimedJob.OperationName = operationName
		claimedJob.Status = "processing"
		if err := veoUpdateJob(ctx, claimedJob); err != nil {
			return claimedJob, err
		}
		logEvent("admin-instagram-generate", "info", "Veo extension started", map[string]any{"jobId": claimedJob.ID, "phase": claimedJob.Phase})
		return claimedJob, nil
	}
	masterObject, err := veoPromoteMaster(ctx, config, job.ID, videoURI)
	if err != nil {
		return job, err
	}
	job.MasterObject = masterObject
	job.Status = "completed"
	job.OperationName = ""
	job.FailureCode = ""
	job.FailureDetail = ""
	job.ProviderCode = 0
	job.ProviderStatus = ""
	if err := veoUpdateJob(ctx, job); err != nil {
		return job, err
	}
	logEvent("admin-instagram-generate", "info", "Veo master completed", map[string]any{"jobId": job.ID, "masterObject": masterObject})
	return job, nil
}

func veoResponse(ctx context.Context, job veoGenerationJob) (veoGenerationResponse, error) {
	message := "Veo is processing the generation job."
	if job.Phase == "extension" && job.Status == "processing" {
		message = "Veo is extending the shot by seven seconds for the hero hold and two synchronized lock chirps."
	}
	if job.Status == "starting_extension" {
		message = "The opening shot is complete and the continuation is being reserved."
	}
	if job.Status == "failed" {
		message = "Generation stopped safely. No media was published. Start a new job after checking the configuration or prompt."
		if job.FailureDetail != "" {
			message = job.FailureDetail
		}
	}
	response := generationResponse(job, message)
	if job.Status == "completed" {
		mediaPath, err := veoIssueDelivery(ctx, job)
		if err != nil {
			return response, err
		}
		response.MediaPath = mediaPath
		response.Message = "The private 15-second master is ready. Watch it completely before approving the Instagram preview."
	}
	return response, nil
}

func generationResponse(job veoGenerationJob, message string) veoGenerationResponse {
	return veoGenerationResponse{
		JobID: job.ID, Status: job.Status, Phase: job.Phase, ModelID: job.ModelID,
		Location: job.Location, Message: message, FailureCode: job.FailureCode,
		Configured: veoConfigurationValid(),
	}
}

func classifyVeoOperationError(providerErr *veoOperationError) (string, string) {
	if providerErr == nil {
		return "vertex_operation_failed", "Vertex AI stopped the generation operation. No media was published. Check Cloud Logging before starting a new job."
	}
	message := strings.ToLower(providerErr.Message)
	if providerErr.Code == 9 && strings.Contains(message, "service agents are being provisioned") {
		return "vertex_service_agent_provisioning", "Vertex AI could not access the campaign-media bucket while its service identity was being provisioned. Apply the infrastructure configuration, allow IAM propagation to finish, then start a new job."
	}
	if strings.Contains(message, "permission") || strings.Contains(message, "access denied") {
		return "vertex_storage_access_denied", "Vertex AI could not access the private campaign-media bucket. Check the managed Vertex service identity and bucket IAM grant before starting a new job."
	}
	return "vertex_operation_failed", "Vertex AI stopped the generation operation. No media was published. Check Cloud Logging before starting a new job."
}

func currentVeoConfig() (veoConfig, error) {
	config := veoConfig{
		ProjectID: strings.TrimSpace(projectID()),
		Location:  strings.TrimSpace(os.Getenv("VEO_LOCATION")),
		ModelID:   strings.TrimSpace(os.Getenv("VEO_MODEL_ID")),
		Bucket:    strings.TrimSpace(os.Getenv("CAMPAIGN_MEDIA_BUCKET")),
	}
	if config.ProjectID == "" || config.Location == "" || config.ModelID == "" || config.Bucket == "" {
		return veoConfig{}, fmt.Errorf("incomplete Veo configuration")
	}
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,221}[a-z0-9]$`).MatchString(config.Bucket) ||
		!regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(config.Location) ||
		!regexp.MustCompile(`^veo-[a-z0-9.-]+$`).MatchString(config.ModelID) {
		return veoConfig{}, fmt.Errorf("invalid Veo configuration")
	}
	if config.Location != veoLocation {
		return veoConfig{}, fmt.Errorf("unsupported Veo location")
	}
	return config, nil
}

func veoConfigurationValid() bool {
	_, err := currentVeoConfig()
	return err == nil
}

func reserveVeoGenerationJob(ctx context.Context, requestID, prompt string, config veoConfig) (veoGenerationJob, bool, error) {
	requestHash := sha256.Sum256([]byte(requestID))
	promptHash := sha256.Sum256([]byte(prompt))
	jobID := "veo-" + hex.EncodeToString(requestHash[:])[:24]
	job := veoGenerationJob{
		ID: jobID, RequestIDHash: hex.EncodeToString(requestHash[:]), Prompt: prompt,
		PromptHash: hex.EncodeToString(promptHash[:]), Status: "starting", Phase: "initial",
		ModelID: config.ModelID, Location: config.Location,
	}
	db, err := firestoreClient(ctx)
	if err != nil {
		return job, false, fmt.Errorf("generation ledger is unavailable")
	}
	ref := db.Collection("instagramGenerationJobs").Doc(jobID)
	created := false
	err = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, getErr := tx.Get(ref)
		if status.Code(getErr) == codes.NotFound {
			created = true
			return tx.Create(ref, map[string]any{
				"id": job.ID, "requestIdHash": job.RequestIDHash, "prompt": job.Prompt,
				"promptHash": job.PromptHash, "status": job.Status, "phase": job.Phase,
				"modelId": job.ModelID, "location": job.Location,
				"createdAt": firestore.ServerTimestamp, "updatedAt": firestore.ServerTimestamp,
			})
		}
		if getErr != nil {
			return getErr
		}
		if err := snapshot.DataTo(&job); err != nil {
			return err
		}
		if job.PromptHash != hex.EncodeToString(promptHash[:]) {
			return fmt.Errorf("request ID was already used for a different prompt")
		}
		return nil
	})
	return job, created, err
}

func loadVeoGenerationJob(ctx context.Context, jobID string) (veoGenerationJob, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return veoGenerationJob{}, err
	}
	snapshot, err := db.Collection("instagramGenerationJobs").Doc(jobID).Get(ctx)
	if err != nil {
		return veoGenerationJob{}, err
	}
	var job veoGenerationJob
	if err := snapshot.DataTo(&job); err != nil {
		return veoGenerationJob{}, err
	}
	return job, nil
}

func updateVeoGenerationJob(ctx context.Context, job veoGenerationJob) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	_, err = db.Collection("instagramGenerationJobs").Doc(job.ID).Set(ctx, map[string]any{
		"status": job.Status, "phase": job.Phase, "operationName": job.OperationName,
		"initialGcsUri": job.InitialGCSURI, "masterObject": job.MasterObject,
		"failureCode": job.FailureCode, "failureDetail": job.FailureDetail,
		"providerCode": job.ProviderCode, "providerStatus": job.ProviderStatus,
		"updatedAt": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	return err
}

func claimVeoExtension(ctx context.Context, observed veoGenerationJob, initialGCSURI string) (veoGenerationJob, bool, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return observed, false, err
	}
	ref := db.Collection("instagramGenerationJobs").Doc(observed.ID)
	current := observed
	claimed := false
	err = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snapshot, err := tx.Get(ref)
		if err != nil {
			return err
		}
		if err := snapshot.DataTo(&current); err != nil {
			return err
		}
		if current.Status != "processing" || current.Phase != "initial" || current.OperationName != observed.OperationName {
			return nil
		}
		claimed = true
		current.Status = "starting_extension"
		current.OperationName = ""
		current.InitialGCSURI = initialGCSURI
		return tx.Set(ref, map[string]any{
			"status": current.Status, "operationName": current.OperationName,
			"initialGcsUri": current.InitialGCSURI, "updatedAt": firestore.ServerTimestamp,
		}, firestore.MergeAll)
	})
	return current, claimed, err
}

func startVeoOperation(ctx context.Context, config veoConfig, jobID, masterPrompt, inputVideoURI, phase string) (string, error) {
	prompt := masterPrompt + "\n\nStage direction: Generate the opening eight seconds only. The car approaches with continuous physical motion, decelerates naturally, and completes its normal controlled stop by the end of this clip. Build the dynamic road and electric-drivetrain sound, but do not play the two lock chirps yet."
	instance := map[string]any{"prompt": prompt}
	parameters := map[string]any{
		"storageUri":  "gs://" + config.Bucket + "/work/" + jobID + "/initial/",
		"sampleCount": 1, "aspectRatio": "9:16", "durationSeconds": 8,
		"resolution": "1080p", "generateAudio": true, "personGeneration": "dont_allow",
		"negativePrompt": "breakdown, power loss, emergency braking, warning lights, hazard lights, smoke, skid marks, collision, swerving, reckless speed, damage, emergency services, people, distressed occupants, readable registration, logos, watermarks, generated text, distorted wheels, changing body panels, siren, horn, alarm, warning tone",
	}
	if phase == "extension" {
		prompt = masterPrompt + "\n\nStage direction: Continue this exact shot for seven seconds with the same car, road, camera, lighting, reflections and audio perspective. The car remains safely stopped for a composed hero hold. After a short quiet beat, play exactly two crisp normal vehicle-lock confirmation chirps—beep, beep—clearly separated and synchronized with two subtle amber indicator flashes. Let the ambience fade naturally after the second chirp."
		instance = map[string]any{"prompt": prompt, "video": map[string]any{"gcsUri": inputVideoURI, "mimeType": "video/mp4"}}
		parameters["storageUri"] = "gs://" + config.Bucket + "/work/" + jobID + "/extension/"
		delete(parameters, "durationSeconds")
	}
	payload := map[string]any{"instances": []any{instance}, "parameters": parameters}
	var response veoOperation
	if err := veoRequest(ctx, config, ":predictLongRunning", payload, &response); err != nil {
		return "", err
	}
	if !validVeoOperationName(config, response.Name) {
		return "", fmt.Errorf("Vertex AI returned an invalid operation name")
	}
	return response.Name, nil
}

func fetchVeoOperation(ctx context.Context, config veoConfig, operationName string) (veoOperation, error) {
	if !validVeoOperationName(config, operationName) {
		return veoOperation{}, fmt.Errorf("invalid stored operation name")
	}
	var response veoOperation
	err := veoRequest(ctx, config, ":fetchPredictOperation", map[string]any{"operationName": operationName}, &response)
	return response, err
}

func veoRequest(ctx context.Context, config veoConfig, method string, payload any, destination any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := veoEndpoint(config) + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client, err := veoHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("create authenticated Vertex AI client: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call Vertex AI: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		return fmt.Errorf("Vertex AI returned status %d", res.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(destination); err != nil {
		return fmt.Errorf("decode Vertex AI response: %w", err)
	}
	return nil
}

func veoEndpoint(config veoConfig) string {
	host := "https://" + config.Location + "-aiplatform.googleapis.com"
	if config.Location == "global" {
		host = "https://aiplatform.googleapis.com"
	}
	return host + "/v1/projects/" + url.PathEscape(config.ProjectID) + "/locations/" + url.PathEscape(config.Location) + "/publishers/google/models/" + url.PathEscape(config.ModelID)
}

func validVeoOperationName(config veoConfig, name string) bool {
	prefix := "projects/" + config.ProjectID + "/locations/" + config.Location + "/publishers/google/models/" + config.ModelID + "/operations/"
	return strings.HasPrefix(name, prefix) && len(name) > len(prefix)
}

func validateVeoOutputURI(config veoConfig, jobID, gcsURI string) error {
	bucket, object, err := parseGCSURI(gcsURI)
	if err != nil || bucket != config.Bucket || !strings.HasPrefix(object, "work/"+jobID+"/") || !strings.HasSuffix(strings.ToLower(object), ".mp4") {
		return fmt.Errorf("Vertex AI returned an unexpected output location")
	}
	return nil
}

func promoteVeoMaster(ctx context.Context, config veoConfig, jobID, gcsURI string) (string, error) {
	if err := validateVeoOutputURI(config, jobID, gcsURI); err != nil {
		return "", err
	}
	_, sourceObject, _ := parseGCSURI(gcsURI)
	destination := "masters/" + jobID + "/instagram-launch-reel.mp4"
	client, err := gcsClient(ctx)
	if err != nil {
		return "", err
	}
	copier := client.Bucket(config.Bucket).Object(destination).CopierFrom(client.Bucket(config.Bucket).Object(sourceObject))
	copier.ContentType = "video/mp4"
	if _, err := copier.Run(ctx); err != nil {
		return "", err
	}
	return destination, nil
}

func issueVeoDeliveryToken(ctx context.Context, job veoGenerationJob) (string, error) {
	if job.Status != "completed" || job.MasterObject == "" {
		return "", fmt.Errorf("master is not complete")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	digest := sha256.Sum256([]byte(token))
	db, err := firestoreClient(ctx)
	if err != nil {
		return "", err
	}
	expires := time.Now().UTC().Add(veoDeliveryTokenTTL)
	_, err = db.Collection("instagramGenerationJobs").Doc(job.ID).Set(ctx, map[string]any{
		"deliveryTokenHash": hex.EncodeToString(digest[:]), "deliveryExpiresAt": expires,
		"updatedAt": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	if err != nil {
		return "", err
	}
	return "/api/instagram-media/" + job.ID + "/" + token + ".mp4", nil
}

func InstagramGeneratedMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	jobID, token, ok := parseVeoMediaPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	job, err := loadVeoGenerationJob(r.Context(), jobID)
	if err != nil || job.Status != "completed" || job.MasterObject == "" || time.Now().UTC().After(job.DeliveryExpiry) {
		http.NotFound(w, r)
		return
	}
	digest := sha256.Sum256([]byte(token))
	expected, err := hex.DecodeString(job.DeliveryHash)
	if err != nil || len(expected) != len(digest) || subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
		http.NotFound(w, r)
		return
	}
	config, err := currentVeoConfig()
	if err != nil {
		http.Error(w, "Media unavailable", http.StatusServiceUnavailable)
		return
	}
	servePrivateVideo(w, r, config.Bucket, job.MasterObject)
}

func servePrivateVideo(w http.ResponseWriter, r *http.Request, bucket, object string) {
	client, err := gcsClient(r.Context())
	if err != nil {
		http.Error(w, "Media unavailable", http.StatusServiceUnavailable)
		return
	}
	handle := client.Bucket(bucket).Object(object)
	attrs, err := handle.Attrs(r.Context())
	if err != nil {
		http.NotFound(w, r)
		return
	}
	start, length, partial, err := parseByteRange(r.Header.Get("Range"), attrs.Size)
	if err != nil {
		w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(attrs.Size, 10))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	if partial {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+length-1, attrs.Size))
		w.WriteHeader(http.StatusPartialContent)
	}
	if r.Method == http.MethodHead {
		return
	}
	reader, err := handle.NewRangeReader(r.Context(), start, length)
	if err != nil {
		return
	}
	defer reader.Close()
	_, _ = io.Copy(w, reader)
}

func parseByteRange(value string, size int64) (start, length int64, partial bool, err error) {
	if value == "" {
		return 0, size, false, nil
	}
	if !strings.HasPrefix(value, "bytes=") || strings.Contains(value, ",") {
		return 0, 0, false, fmt.Errorf("unsupported range")
	}
	parts := strings.Split(strings.TrimPrefix(value, "bytes="), "-")
	if len(parts) != 2 || parts[0] == "" {
		return 0, 0, false, fmt.Errorf("invalid range")
	}
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, false, fmt.Errorf("invalid range")
	}
	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return 0, 0, false, fmt.Errorf("invalid range")
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end - start + 1, true, nil
}

func parseVeoMediaPath(mediaPath string) (jobID, token string, ok bool) {
	parts := strings.Split(strings.TrimPrefix(mediaPath, "/api/instagram-media/"), "/")
	if len(parts) != 2 || !validVeoJobID(parts[0]) || !strings.HasSuffix(parts[1], ".mp4") {
		return "", "", false
	}
	token = strings.TrimSuffix(parts[1], ".mp4")
	if len(token) < 40 || len(token) > 60 || !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(token) {
		return "", "", false
	}
	return parts[0], token, true
}

func validVeoJobID(jobID string) bool {
	return regexp.MustCompile(`^veo-[a-f0-9]{24}$`).MatchString(jobID)
}

func parseGCSURI(value string) (bucket, object string, err error) {
	if !strings.HasPrefix(value, "gs://") {
		return "", "", fmt.Errorf("not a GCS URI")
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "gs://"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "..") {
		return "", "", fmt.Errorf("invalid GCS URI")
	}
	return parts[0], parts[1], nil
}
