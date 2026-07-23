package ipace

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func setVeoTestConfig(t *testing.T) veoConfig {
	t.Helper()
	t.Setenv("FIREBASE_PROJECT_ID", "ipace-owners-staging")
	t.Setenv("CAMPAIGN_MEDIA_BUCKET", "ipace-owners-staging-campaign-media")
	t.Setenv("VEO_LOCATION", "global")
	t.Setenv("VEO_MODEL_ID", "veo-3.1-generate-001")
	config, err := currentVeoConfig()
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestVeoStartBuildsAuthenticatedTemporalVideoRequests(t *testing.T) {
	config := setVeoTestConfig(t)
	originalClient := veoHTTPClient
	t.Cleanup(func() { veoHTTPClient = originalClient })

	var requests []*http.Request
	var bodies []map[string]any
	veoHTTPClient = func(context.Context) (*http.Client, error) {
		return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			requests = append(requests, req)
			bodies = append(bodies, body)
			operationName := "projects/ipace-owners-staging/locations/global/publishers/google/models/veo-3.1-generate-001/operations/test-operation"
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"name":"` + operationName + `"}`)), Header: make(http.Header)}, nil
		})}, nil
	}

	if _, err := startVeoOperation(context.Background(), config, "veo-aaaaaaaaaaaaaaaaaaaaaaaa", "master prompt", "", "initial"); err != nil {
		t.Fatal(err)
	}
	inputURI := "gs://ipace-owners-staging-campaign-media/work/veo-aaaaaaaaaaaaaaaaaaaaaaaa/initial/sample_0.mp4"
	if _, err := startVeoOperation(context.Background(), config, "veo-aaaaaaaaaaaaaaaaaaaaaaaa", "master prompt", inputURI, "extension"); err != nil {
		t.Fatal(err)
	}

	if len(requests) != 2 || requests[0].URL.Host != "aiplatform.googleapis.com" || !strings.HasSuffix(requests[0].URL.Path, ":predictLongRunning") {
		t.Fatalf("unexpected requests: %#v", requests)
	}
	initialParameters := bodies[0]["parameters"].(map[string]any)
	if initialParameters["aspectRatio"] != "9:16" || initialParameters["resolution"] != "1080p" || initialParameters["generateAudio"] != true || initialParameters["durationSeconds"] != float64(8) {
		t.Fatalf("unexpected initial parameters: %#v", initialParameters)
	}
	extensionParameters := bodies[1]["parameters"].(map[string]any)
	if _, exists := extensionParameters["durationSeconds"]; exists {
		t.Fatalf("extension must use Veo's fixed seven-second continuation: %#v", extensionParameters)
	}
	extensionInstances := bodies[1]["instances"].([]any)
	extension := extensionInstances[0].(map[string]any)
	video := extension["video"].(map[string]any)
	if video["gcsUri"] != inputURI || !strings.Contains(extension["prompt"].(string), "exactly two crisp normal vehicle-lock confirmation chirps") {
		t.Fatalf("unexpected extension request: %#v", extension)
	}
}

func TestInstagramGenerateRequiresExplicitBillableConfirmation(t *testing.T) {
	setVeoTestConfig(t)
	originalAuth := veoAuthorize
	originalReserve := veoReserveJob
	t.Cleanup(func() {
		veoAuthorize = originalAuth
		veoReserveJob = originalReserve
	})
	veoAuthorize = func(context.Context, *http.Request) error { return nil }
	called := false
	veoReserveJob = func(context.Context, string, string, veoConfig) (veoGenerationJob, bool, error) {
		called = true
		return veoGenerationJob{}, false, nil
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-generate", strings.NewReader(`{"requestId":"1234567890abcdef","prompt":"car","confirmation":"yes"}`))
	res := httptest.NewRecorder()
	AdminInstagramGenerate(res, req)
	if res.Code != http.StatusConflict || called {
		t.Fatalf("status=%d called=%v body=%s", res.Code, called, res.Body.String())
	}
}

func TestInstagramGenerateStartsOneIdempotentVertexOperation(t *testing.T) {
	config := setVeoTestConfig(t)
	originalAuth, originalReserve, originalStart, originalUpdate := veoAuthorize, veoReserveJob, veoStartOperation, veoUpdateJob
	t.Cleanup(func() {
		veoAuthorize, veoReserveJob, veoStartOperation, veoUpdateJob = originalAuth, originalReserve, originalStart, originalUpdate
	})
	veoAuthorize = func(context.Context, *http.Request) error { return nil }
	job := veoGenerationJob{ID: "veo-aaaaaaaaaaaaaaaaaaaaaaaa", Prompt: "car", Status: "starting", Phase: "initial", ModelID: config.ModelID, Location: config.Location}
	veoReserveJob = func(context.Context, string, string, veoConfig) (veoGenerationJob, bool, error) {
		return job, true, nil
	}
	starts := 0
	veoStartOperation = func(context.Context, veoConfig, string, string, string, string) (string, error) {
		starts++
		return "projects/ipace-owners-staging/locations/global/publishers/google/models/veo-3.1-generate-001/operations/initial", nil
	}
	veoUpdateJob = func(context.Context, veoGenerationJob) error { return nil }
	req := httptest.NewRequest(http.MethodPost, "/api/admin/instagram-generate", strings.NewReader(`{"requestId":"1234567890abcdef","prompt":"car","confirmation":"GENERATE VIDEO"}`))
	res := httptest.NewRecorder()
	AdminInstagramGenerate(res, req)
	if res.Code != http.StatusAccepted || starts != 1 || !strings.Contains(res.Body.String(), `"status":"processing"`) {
		t.Fatalf("status=%d starts=%d body=%s", res.Code, starts, res.Body.String())
	}
}

func TestProgressVeoGenerationStartsSevenSecondExtension(t *testing.T) {
	config := setVeoTestConfig(t)
	originalFetch, originalClaim, originalStart, originalUpdate := veoFetchOperation, veoClaimExtension, veoStartOperation, veoUpdateJob
	t.Cleanup(func() {
		veoFetchOperation, veoClaimExtension, veoStartOperation, veoUpdateJob = originalFetch, originalClaim, originalStart, originalUpdate
	})
	inputURI := "gs://" + config.Bucket + "/work/veo-aaaaaaaaaaaaaaaaaaaaaaaa/initial/sample_0.mp4"
	veoFetchOperation = func(context.Context, veoConfig, string) (veoOperation, error) {
		var operation veoOperation
		operation.Done = true
		operation.Response.Videos = append(operation.Response.Videos, struct {
			GCSURI   string `json:"gcsUri"`
			MimeType string `json:"mimeType"`
		}{GCSURI: inputURI, MimeType: "video/mp4"})
		return operation, nil
	}
	veoClaimExtension = func(_ context.Context, job veoGenerationJob, videoURI string) (veoGenerationJob, bool, error) {
		job.Status = "starting_extension"
		job.OperationName = ""
		job.InitialGCSURI = videoURI
		return job, true, nil
	}
	startedWith := ""
	veoStartOperation = func(_ context.Context, _ veoConfig, _ string, _ string, videoURI string, phase string) (string, error) {
		startedWith = videoURI + ":" + phase
		return "extension-operation", nil
	}
	veoUpdateJob = func(context.Context, veoGenerationJob) error { return nil }
	job := veoGenerationJob{ID: "veo-aaaaaaaaaaaaaaaaaaaaaaaa", Prompt: "master", Status: "processing", Phase: "initial", OperationName: "initial-operation", ModelID: config.ModelID}
	progressed, err := progressVeoGeneration(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if progressed.Phase != "extension" || progressed.InitialGCSURI != inputURI || startedWith != inputURI+":extension" {
		t.Fatalf("unexpected progression: %#v started=%q", progressed, startedWith)
	}
}

func TestProgressVeoGenerationDoesNotStartDuplicateExtension(t *testing.T) {
	config := setVeoTestConfig(t)
	originalFetch, originalClaim, originalStart := veoFetchOperation, veoClaimExtension, veoStartOperation
	t.Cleanup(func() {
		veoFetchOperation, veoClaimExtension, veoStartOperation = originalFetch, originalClaim, originalStart
	})
	inputURI := "gs://" + config.Bucket + "/work/veo-aaaaaaaaaaaaaaaaaaaaaaaa/initial/sample_0.mp4"
	veoFetchOperation = func(context.Context, veoConfig, string) (veoOperation, error) {
		var operation veoOperation
		operation.Done = true
		operation.Response.Videos = append(operation.Response.Videos, struct {
			GCSURI   string `json:"gcsUri"`
			MimeType string `json:"mimeType"`
		}{GCSURI: inputURI, MimeType: "video/mp4"})
		return operation, nil
	}
	veoClaimExtension = func(_ context.Context, job veoGenerationJob, _ string) (veoGenerationJob, bool, error) {
		job.Status = "starting_extension"
		job.OperationName = ""
		return job, false, nil
	}
	starts := 0
	veoStartOperation = func(context.Context, veoConfig, string, string, string, string) (string, error) {
		starts++
		return "unexpected", nil
	}
	job := veoGenerationJob{ID: "veo-aaaaaaaaaaaaaaaaaaaaaaaa", Prompt: "master", Status: "processing", Phase: "initial", OperationName: "initial-operation", ModelID: config.ModelID}
	progressed, err := progressVeoGeneration(context.Background(), job)
	if err != nil || starts != 0 || progressed.Status != "starting_extension" {
		t.Fatalf("progressed=%#v starts=%d err=%v", progressed, starts, err)
	}
}

func TestVeoDeliveryPathsAndRangesAreStrict(t *testing.T) {
	jobID := "veo-aaaaaaaaaaaaaaaaaaaaaaaa"
	token := strings.Repeat("A", 43)
	if gotJob, gotToken, ok := parseVeoMediaPath("/api/instagram-media/" + jobID + "/" + token + ".mp4"); !ok || gotJob != jobID || gotToken != token {
		t.Fatalf("valid delivery path rejected: %q %q %v", gotJob, gotToken, ok)
	}
	if _, _, ok := parseVeoMediaPath("/api/instagram-media/" + jobID + "/../secret.mp4"); ok {
		t.Fatal("path traversal accepted")
	}
	start, length, partial, err := parseByteRange("bytes=100-199", 1000)
	if err != nil || start != 100 || length != 100 || !partial {
		t.Fatalf("unexpected range: start=%d length=%d partial=%v err=%v", start, length, partial, err)
	}
	if _, _, _, err := parseByteRange("bytes=1000-", 1000); err == nil {
		t.Fatal("out-of-bounds range accepted")
	}
}

func TestVeoOutputMustStayInsideReservedWorkingPrefix(t *testing.T) {
	config := setVeoTestConfig(t)
	jobID := "veo-aaaaaaaaaaaaaaaaaaaaaaaa"
	if err := validateVeoOutputURI(config, jobID, "gs://"+config.Bucket+"/work/"+jobID+"/extension/sample_0.mp4"); err != nil {
		t.Fatal(err)
	}
	if err := validateVeoOutputURI(config, jobID, "gs://attacker-bucket/work/"+jobID+"/sample.mp4"); err == nil {
		t.Fatal("cross-bucket output accepted")
	}
}
