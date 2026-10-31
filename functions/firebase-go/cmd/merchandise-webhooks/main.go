package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	secretmanager "google.golang.org/api/secretmanager/v1"
)

const (
	stripeAPIBase   = "https://api.stripe.com/v1"
	printifyAPIBase = "https://api.printify.com/v1"
)

var stripeEvents = []string{"checkout.session.completed", "checkout.session.async_payment_succeeded"}

var printifyEvents = []string{
	"product:created", "product:updated", "product:deleted", "product:publish:started",
	"order:created", "order:updated", "order:line-item:canceled", "order:sent-to-production",
	"order:shipment:created", "order:shipment:delivered",
}

type shopConfig struct {
	ShopID     string `json:"shopId"`
	BaseURL    string `json:"baseUrl"`
	StripeLive bool   `json:"stripeLive"`
}

type secretStore interface {
	read(context.Context, string) ([]byte, error)
	write(context.Context, string, []byte) error
}

type gcpSecretStore struct {
	api     *secretmanager.Service
	project string
}

func main() {
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	if err := run(context.Background(), os.Getenv, client, newSecretStore); err != nil {
		fmt.Fprintln(os.Stderr, "merchandise webhook setup failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string, client *http.Client, newStore func(context.Context, string) (secretStore, error)) error {
	var cfg shopConfig
	if err := json.Unmarshal([]byte(getenv("MERCH_CONFIG")), &cfg); err != nil {
		return errors.New("MERCH_CONFIG is not valid JSON")
	}
	if cfg.ShopID == "" || cfg.BaseURL == "" {
		if getenv("MERCH_SECRETS_ENABLED") == "true" {
			return errors.New("MERCH_CONFIG must include shopId and baseUrl when merchandise secrets are enabled")
		}
		fmt.Println("merchandise webhooks: shopId/baseUrl not configured; skipping")
		return nil
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || (base.Path != "" && base.Path != "/") || base.RawQuery != "" || base.Fragment != "" {
		return errors.New("MERCH_CONFIG.baseUrl must be a stable HTTPS origin without a path, query, or fragment")
	}
	if _, err := strconv.ParseInt(cfg.ShopID, 10, 64); err != nil {
		return errors.New("MERCH_CONFIG.shopId must be a numeric Printify shop ID")
	}
	project := getenv("GCP_PROJECT_ID")
	if project == "" {
		return errors.New("GCP_PROJECT_ID is required")
	}
	store, err := newStore(ctx, project)
	if err != nil {
		return errors.New("could not initialize Secret Manager")
	}
	stripeKey := getenv("STRIPE_SECRET_KEY")
	printifyKey := getenv("PRINTIFY_API_TOKEN")
	if getenv("MERCH_SECRETS_ENABLED") == "true" && (stripeKey == "" || printifyKey == "") {
		return errors.New("STRIPE_SECRET_KEY and PRINTIFY_API_TOKEN are required when merchandise secrets are enabled")
	}
	stripeURL := strings.TrimRight(cfg.BaseURL, "/") + "/api/shop/stripe-webhook"
	printifyURL := strings.TrimRight(cfg.BaseURL, "/") + "/api/shop/printify-webhook"

	if stripeKey == "" {
		fmt.Println("Stripe webhook: STRIPE_SECRET_KEY not set locally; skipping")
	} else {
		if getenv("STRIPE_WEBHOOK_SECRET") != "" {
			return errors.New("do not provide STRIPE_WEBHOOK_SECRET; OpenTofu retrieves and stores Stripe's endpoint secret directly")
		}
		if cfg.StripeLive && !strings.HasPrefix(stripeKey, "sk_live_") || !cfg.StripeLive && !strings.HasPrefix(stripeKey, "sk_test_") {
			return errors.New("Stripe secret key mode does not match MERCH_CONFIG.stripeLive")
		}
		if err := reconcileStripe(ctx, client, stripeAPIBase, stripeKey, stripeURL, store); err != nil {
			return fmt.Errorf("Stripe webhook: %w", err)
		}
		fmt.Println("Stripe webhook: endpoint reconciled")
	}

	if printifyKey == "" {
		fmt.Println("Printify webhook: PRINTIFY_API_TOKEN not set locally; skipping")
	} else {
		if getenv("PRINTIFY_WEBHOOK_SECRET") != "" {
			return errors.New("do not provide PRINTIFY_WEBHOOK_SECRET; OpenTofu generates and stores the Printify signing secret directly")
		}
		if err := reconcilePrintify(ctx, client, printifyAPIBase, printifyKey, cfg.ShopID, printifyURL, store); err != nil {
			return fmt.Errorf("Printify webhook: %w", err)
		}
		fmt.Println("Printify webhooks: subscriptions reconciled")
	}
	return nil
}

func newSecretStore(ctx context.Context, project string) (secretStore, error) {
	api, err := secretmanager.NewService(ctx, option.WithQuotaProject(project))
	if err != nil {
		return nil, err
	}
	return &gcpSecretStore{api: api, project: project}, nil
}

func (s *gcpSecretStore) read(ctx context.Context, id string) ([]byte, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s", s.project, id)
	response, err := s.api.Projects.Secrets.Versions.Access(name + "/versions/latest").Context(ctx).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if errors.As(err, &apiErr) {
			if apiErr.Code == http.StatusNotFound || apiErr.Code == http.StatusBadRequest && strings.Contains(strings.ToUpper(apiErr.Message), "DESTROYED STATE") {
				return nil, nil
			}
		}
		return nil, err
	}
	value, err := base64.StdEncoding.DecodeString(response.Payload.Data)
	if err != nil {
		return nil, errors.New("could not decode existing secret")
	}
	return value, nil
}

func (s *gcpSecretStore) write(ctx context.Context, id string, value []byte) error {
	name := fmt.Sprintf("projects/%s/secrets/%s", s.project, id)
	current, err := s.read(ctx, id)
	if err != nil {
		return err
	}
	if bytes.Equal(current, value) {
		return nil
	}
	request := &secretmanager.AddSecretVersionRequest{Payload: &secretmanager.SecretPayload{Data: base64.StdEncoding.EncodeToString(value)}}
	_, err = s.api.Projects.Secrets.AddVersion(name, request).Context(ctx).Do()
	return err
}

func signingSecret(ctx context.Context, store secretStore, id string) ([]byte, error) {
	value, err := store.read(ctx, id)
	if err != nil {
		return nil, errors.New("could not read signing secret from Secret Manager")
	}
	if len(value) != 0 {
		return value, nil
	}
	value = make([]byte, 20)
	if _, err := rand.Read(value); err != nil {
		return nil, errors.New("could not generate webhook signing secret")
	}
	value = []byte(hex.EncodeToString(value))
	if err := store.write(ctx, id, value); err != nil {
		return nil, errors.New("could not store webhook signing secret in Secret Manager")
	}
	return value, nil
}

type stripeEndpoint struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Secret        string   `json:"secret"`
	EnabledEvents []string `json:"enabled_events"`
}

type stripeEndpointList struct {
	Data    []stripeEndpoint `json:"data"`
	HasMore bool             `json:"has_more"`
}

func reconcileStripe(ctx context.Context, client *http.Client, apiBase, key, endpointURL string, store secretStore) error {
	endpoints, err := listStripeEndpoints(ctx, client, apiBase, key, endpointURL)
	if err != nil {
		return errors.New("could not list Stripe webhook endpoints")
	}
	secret, err := store.read(ctx, "stripe-webhook-secret")
	if err != nil {
		return errors.New("could not read Stripe signing secret from Secret Manager")
	}
	if len(endpoints) > 0 && len(secret) > 0 {
		if err := updateStripeEndpoint(ctx, client, apiBase, key, endpoints[0].ID, endpointURL); err != nil {
			return errors.New("could not update Stripe webhook endpoint")
		}
		for _, duplicate := range endpoints[1:] {
			if err := deleteStripeEndpoint(ctx, client, apiBase, key, duplicate.ID); err != nil {
				return errors.New("could not remove a duplicate Stripe webhook endpoint")
			}
		}
		return nil
	}
	created, err := createStripeEndpoint(ctx, client, apiBase, key, endpointURL)
	if err != nil || created.ID == "" || created.Secret == "" {
		return errors.New("could not create Stripe webhook endpoint")
	}
	if err := store.write(ctx, "stripe-webhook-secret", []byte(created.Secret)); err != nil {
		// The endpoint remains created but is not usable until the secret is saved.
		return errors.New("Stripe created an endpoint, but its signing secret could not be stored; re-apply to reconcile")
	}
	for _, prior := range endpoints {
		if err := deleteStripeEndpoint(ctx, client, apiBase, key, prior.ID); err != nil {
			return errors.New("new Stripe endpoint is ready, but an old matching endpoint could not be removed")
		}
	}
	return nil
}

func listStripeEndpoints(ctx context.Context, client *http.Client, apiBase, key, targetURL string) ([]stripeEndpoint, error) {
	var matches []stripeEndpoint
	startingAfter := ""
	for page := 0; page < 10; page++ {
		query := url.Values{"limit": {"100"}}
		if startingAfter != "" {
			query.Set("starting_after", startingAfter)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/webhook_endpoints?"+query.Encode(), nil)
		req.SetBasicAuth(key, "")
		var response stripeEndpointList
		if err := doJSON(client, req, &response); err != nil {
			return nil, err
		}
		for _, endpoint := range response.Data {
			if endpoint.URL == targetURL {
				matches = append(matches, endpoint)
			}
		}
		if !response.HasMore || len(response.Data) == 0 {
			return matches, nil
		}
		startingAfter = response.Data[len(response.Data)-1].ID
	}
	return nil, errors.New("Stripe webhook list pagination exceeded safety limit")
}

func createStripeEndpoint(ctx context.Context, client *http.Client, apiBase, key, endpointURL string) (stripeEndpoint, error) {
	form := url.Values{"url": {endpointURL}}
	for _, event := range stripeEvents {
		form.Add("enabled_events[]", event)
	}
	return stripeWrite(ctx, client, apiBase+"/webhook_endpoints", key, form)
}

func updateStripeEndpoint(ctx context.Context, client *http.Client, apiBase, key, id, endpointURL string) error {
	form := url.Values{"url": {endpointURL}}
	for _, event := range stripeEvents {
		form.Add("enabled_events[]", event)
	}
	_, err := stripeWrite(ctx, client, apiBase+"/webhook_endpoints/"+url.PathEscape(id), key, form)
	return err
}

func deleteStripeEndpoint(ctx context.Context, client *http.Client, apiBase, key, id string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, apiBase+"/webhook_endpoints/"+url.PathEscape(id), nil)
	req.SetBasicAuth(key, "")
	return doJSON(client, req, &map[string]any{})
}

func stripeWrite(ctx context.Context, client *http.Client, endpoint, key string, form url.Values) (stripeEndpoint, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	req.SetBasicAuth(key, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var result stripeEndpoint
	err := doJSON(client, req, &result)
	return result, err
}

type printifyWebhook struct {
	ID     string `json:"id"`
	Topic  string `json:"topic"`
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

func reconcilePrintify(ctx context.Context, client *http.Client, apiBase, token, shopID, endpointURL string, store secretStore) error {
	secret, err := signingSecret(ctx, store, "printify-webhook-secret")
	if err != nil {
		return err
	}
	var installed []printifyWebhook
	endpoint := apiBase + "/shops/" + url.PathEscape(shopID) + "/webhooks.json"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err := doJSON(client, req, &installed); err != nil {
		return errors.New("could not list Printify webhooks")
	}
	for _, topic := range printifyEvents {
		found := false
		for _, webhook := range installed {
			if webhook.Topic == topic && webhook.URL == endpointURL {
				found = true
				if webhook.ID == "" {
					return errors.New("Printify returned a webhook without an ID")
				}
				if err := writePrintifyWebhook(ctx, client, http.MethodPut, endpoint+"/"+url.PathEscape(webhook.ID)+".json", token, printifyWebhook{URL: endpointURL, Secret: string(secret)}); err != nil {
					return errors.New("could not update a Printify webhook")
				}
				break
			}
		}
		if !found {
			if err := writePrintifyWebhook(ctx, client, http.MethodPost, endpoint, token, printifyWebhook{Topic: topic, URL: endpointURL, Secret: string(secret)}); err != nil {
				return errors.New("could not create a Printify webhook")
			}
		}
	}
	return nil
}

func writePrintifyWebhook(ctx context.Context, client *http.Client, method, endpoint, token string, body printifyWebhook) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return doJSON(client, req, &map[string]any{})
}

func doJSON(client *http.Client, req *http.Request, target any) error {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("provider API returned HTTP %d", response.StatusCode)
	}
	if target == nil {
		_, err = io.Copy(io.Discard, response.Body)
		return err
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target)
}
