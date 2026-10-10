package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"google.golang.org/api/googleapi"
	secretmanager "google.golang.org/api/secretmanager/v1"
)

type secret struct{ ID, Env string }

func main() {
	project := os.Getenv("GCP_PROJECT_ID")
	if project == "" {
		fail("GCP_PROJECT_ID is required")
	}
	rotations := map[string]string{}
	if err := json.Unmarshal([]byte(os.Getenv("SECRET_VERSION_ROTATIONS")), &rotations); err != nil {
		fail("SECRET_VERSION_ROTATIONS must be JSON")
	}
	api, err := secretmanager.NewService(context.Background())
	if err != nil {
		fail("could not create Secret Manager client")
	}
	secrets := map[string]secret{
		"vin_pepper":              {"vin-pepper", "VIN_PEPPER"},
		"resend_api_key":          {"resend-api-key", "RESEND_API_KEY"},
		"ideal_postcodes_api_key": {"ideal-postcodes-api-key", "IDEAL_POSTCODES_API_KEY"},
		"instagram_access_token":  {"instagram-access-token", "INSTAGRAM_ACCESS_TOKEN"},
	}
	for key, label := range rotations {
		if label == "" {
			continue
		}
		cfg, ok := secrets[key]
		if !ok {
			fail("unsupported secret rotation key: " + key)
		}
		value := []byte(os.Getenv(cfg.Env))
		if len(value) == 0 {
			fail(cfg.Env + " must be set when rotating " + cfg.ID)
		}
		name := fmt.Sprintf("projects/%s/secrets/%s", project, cfg.ID)
		current, err := api.Projects.Secrets.Versions.Access(name + "/versions/latest").Do()
		if err == nil {
			latest, _ := base64.StdEncoding.DecodeString(current.Payload.Data)
			if same(latest, value) {
				fmt.Println(cfg.ID + ": latest version already matches")
				continue
			}
		} else if !notFound(err) {
			fail("could not read the latest version of " + cfg.ID)
		}
		_, err = api.Projects.Secrets.AddVersion(name, &secretmanager.AddSecretVersionRequest{Payload: &secretmanager.SecretPayload{Data: base64.StdEncoding.EncodeToString(value)}}).Do()
		if err != nil {
			fail("could not add a version for " + cfg.ID)
		}
		fmt.Println(cfg.ID + ": added a new version")
	}
}
func same(a, b []byte) bool { return sha256.Sum256(a) == sha256.Sum256(b) }
func notFound(err error) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == 404
}
func fail(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
