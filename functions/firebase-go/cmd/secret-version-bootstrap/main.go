package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	secretmanager "google.golang.org/api/secretmanager/v1"
)

type secret struct{ ID, Env string }

func main() {
	project := os.Getenv("GCP_PROJECT_ID")
	if project == "" {
		fail("GCP_PROJECT_ID is required")
	}
	api, err := secretmanager.NewService(context.Background(), option.WithQuotaProject(project))
	if err != nil {
		fail("could not create Secret Manager client")
	}
	secrets := []secret{
		{"vin-pepper", "VIN_PEPPER"},
		{"resend-api-key", "RESEND_API_KEY"},
		{"ideal-postcodes-api-key", "IDEAL_POSTCODES_API_KEY"},
		{"instagram-access-token", "INSTAGRAM_ACCESS_TOKEN"},
		{"printify-api-token", "PRINTIFY_API_TOKEN"},
		{"stripe-secret-key", "STRIPE_SECRET_KEY"},
	}
	for _, cfg := range secrets {
		value := []byte(os.Getenv(cfg.Env))
		if len(value) == 0 {
			fmt.Println(cfg.ID + ": not set locally; skipping")
			continue
		}
		name := fmt.Sprintf("projects/%s/secrets/%s", project, cfg.ID)
		current, err := api.Projects.Secrets.Versions.Access(name + "/versions/latest").Do()
		if err == nil {
			latest, _ := base64.StdEncoding.DecodeString(current.Payload.Data)
			if same(latest, value) {
				fmt.Println(cfg.ID + ": latest version already matches")
				continue
			}
		} else if !missingOrDestroyed(err) {
			fail(secretManagerError("read the latest version of "+cfg.ID, err))
		}
		_, err = api.Projects.Secrets.AddVersion(name, &secretmanager.AddSecretVersionRequest{Payload: &secretmanager.SecretPayload{Data: base64.StdEncoding.EncodeToString(value)}}).Do()
		if err != nil {
			fail(secretManagerError("add a version for "+cfg.ID, err))
		}
		fmt.Println(cfg.ID + ": added a new version")
	}
}
func same(a, b []byte) bool { return sha256.Sum256(a) == sha256.Sum256(b) }
func missingOrDestroyed(err error) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == 404 || (apiErr.Code == 400 && strings.Contains(strings.ToUpper(apiErr.Message), "DESTROYED STATE"))
}

func secretManagerError(action string, err error) string {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		message := strings.NewReplacer("\r", " ", "\n", " ").Replace(apiErr.Message)
		return fmt.Sprintf("could not %s (Secret Manager HTTP %d: %s)", action, apiErr.Code, message)
	}
	return fmt.Sprintf("could not %s (%T)", action, err)
}

func fail(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
