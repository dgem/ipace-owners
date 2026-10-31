package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memorySecrets map[string][]byte

func (m memorySecrets) read(_ context.Context, id string) ([]byte, error) { return m[id], nil }
func (m memorySecrets) write(_ context.Context, id string, value []byte) error {
	m[id] = append([]byte(nil), value...)
	return nil
}

func TestPrintifyWebhookReconcileCreatesAndUpdatesSubscriptions(t *testing.T) {
	secrets := memorySecrets{}
	installed := map[string]printifyWebhook{}
	methods := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer printify-test-token" {
			t.Errorf("authorization header was not set correctly")
		}
		methods[r.Method]++
		switch r.Method {
		case http.MethodGet:
			list := make([]printifyWebhook, 0, len(installed))
			for _, webhook := range installed {
				list = append(list, webhook)
			}
			_ = json.NewEncoder(w).Encode(list)
		case http.MethodPost:
			var webhook printifyWebhook
			if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
				t.Error(err)
			}
			if webhook.Secret == "" || webhook.URL != "https://shop.example/api/shop/printify-webhook" {
				t.Errorf("webhook secret or URL missing: %+v", webhook)
			}
			webhook.ID = fmt.Sprintf("id-%d", len(installed))
			installed[webhook.Topic] = webhook
			_ = json.NewEncoder(w).Encode(webhook)
		case http.MethodPut:
			var webhook printifyWebhook
			if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
				t.Error(err)
			}
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/shops/42/webhooks/"), ".json")
			for topic, prior := range installed {
				if prior.ID == id {
					prior.URL = webhook.URL
					prior.Secret = webhook.Secret
					installed[topic] = prior
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := server.Client()
	if err := reconcilePrintify(context.Background(), client, server.URL+"/v1", "printify-test-token", "42", "https://shop.example/api/shop/printify-webhook", secrets); err != nil {
		t.Fatal(err)
	}
	if len(installed) != len(printifyEvents) || methods[http.MethodPost] != len(printifyEvents) {
		t.Fatalf("created %d subscriptions; want %d", len(installed), len(printifyEvents))
	}
	stored := string(secrets["printify-webhook-secret"])
	if stored == "" {
		t.Fatal("Printify secret was not stored")
	}
	for _, topic := range printifyEvents {
		if installed[topic].Secret != stored {
			t.Fatalf("topic %s did not use stored signing secret", topic)
		}
	}
	if err := reconcilePrintify(context.Background(), client, server.URL+"/v1", "printify-test-token", "42", "https://shop.example/api/shop/printify-webhook", secrets); err != nil {
		t.Fatal(err)
	}
	if methods[http.MethodPost] != len(printifyEvents) || methods[http.MethodPut] != len(printifyEvents) {
		t.Fatalf("second reconcile was not idempotent: POST=%d PUT=%d", methods[http.MethodPost], methods[http.MethodPut])
	}
}

func TestStripeWebhookReconcileCreatesSecretWithoutPrintingItAndThenUpdatesEndpoint(t *testing.T) {
	secrets := memorySecrets{}
	endpoint := stripeEndpoint{}
	created := 0
	updated := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "sk_test_example" || password != "" {
			t.Errorf("Stripe API key was not sent with basic auth")
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(stripeEndpointList{Data: []stripeEndpoint{endpoint}})
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.URL.Path == "/v1/webhook_endpoints" {
				created++
				endpoint = stripeEndpoint{ID: "we_test", URL: r.Form.Get("url"), Secret: "whsec_generated_only_once"}
				if got := r.Form["enabled_events[]"]; len(got) != len(stripeEvents) {
					t.Errorf("enabled events = %v", got)
				}
				_ = json.NewEncoder(w).Encode(endpoint)
				return
			}
			updated++
			if r.URL.Path != "/v1/webhook_endpoints/we_test" || r.Form.Get("url") != endpoint.URL {
				t.Errorf("unexpected update path or URL: %s %s", r.URL.Path, r.Form.Get("url"))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected method %s", r.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := server.Client()
	webhookURL := "https://shop.example/api/shop/stripe-webhook"
	if err := reconcileStripe(context.Background(), client, server.URL+"/v1", "sk_test_example", webhookURL, secrets); err != nil {
		t.Fatal(err)
	}
	if got := string(secrets["stripe-webhook-secret"]); got != "whsec_generated_only_once" {
		t.Fatalf("Stripe signing secret = %q", got)
	}
	if created != 1 {
		t.Fatalf("created endpoints = %d", created)
	}
	if err := reconcileStripe(context.Background(), client, server.URL+"/v1", "sk_test_example", webhookURL, secrets); err != nil {
		t.Fatal(err)
	}
	if created != 1 || updated != 1 {
		t.Fatalf("not idempotent: created=%d updated=%d", created, updated)
	}
}

func TestRunRejectsStripeModeMismatchAndNonOriginBaseURL(t *testing.T) {
	newStore := func(context.Context, string) (secretStore, error) { return memorySecrets{}, nil }
	values := map[string]string{
		"GCP_PROJECT_ID":    "test-project",
		"MERCH_CONFIG":      `{"shopId":"42","baseUrl":"https://shop.example","stripeLive":false}`,
		"STRIPE_SECRET_KEY": "sk_live_example",
	}
	err := run(context.Background(), func(key string) string { return values[key] }, &http.Client{}, newStore)
	if err == nil || !strings.Contains(err.Error(), "mode does not match") {
		t.Fatalf("expected Stripe mode mismatch, got %v", err)
	}

	for _, invalid := range []string{"http://shop.example", "https://shop.example/path", "https://shop.example?x=1"} {
		values["MERCH_CONFIG"] = fmt.Sprintf(`{"shopId":"42","baseUrl":%q,"stripeLive":false}`, invalid)
		err = run(context.Background(), func(key string) string { return values[key] }, &http.Client{}, newStore)
		if err == nil || !strings.Contains(err.Error(), "stable HTTPS origin") {
			t.Errorf("base URL %q: expected origin validation failure, got %v", invalid, err)
		}
	}
}
