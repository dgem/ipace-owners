package ipace

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

func merchTestConfig(t *testing.T) merchConfig {
	t.Helper()
	c := merchConfig{Enabled: true, ShopID: "123", BaseURL: "https://shop.example", CostToGBP: 800000, ShippingToGBP: 800000, FeeBPS: 300, FeeFixed: 20, ReviewedUntil: time.Now().Add(24 * time.Hour), Countries: map[string]merchCountry{"GB": {Name: "United Kingdom", SupplierTaxBPS: 2000}, "AU": {Name: "Australia", SupplierTaxBPS: 1000}}, TermsReviewed: true, SellerName: "Example seller", SellerAddress: "Example postal address"}
	data, _ := json.Marshal(c)
	t.Setenv("MERCH_CONFIG", string(data))
	t.Setenv("PRINTIFY_API_TOKEN", "secret-printify")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_secret-stripe")
	t.Setenv("STRIPE_WEBHOOK_SECRET", "stripe-signing")
	t.Setenv("PRINTIFY_WEBHOOK_SECRET", "printify-signing")
	t.Setenv("ALLOWED_ORIGINS", "https://shop.example")
	return c
}
func merchTestRequest() merchQuoteRequest {
	return merchQuoteRequest{Lines: []merchLine{{ProductID: strings.Repeat("a", 24), VariantID: 10, Quantity: 2}}, Address: merchAddress{FirstName: "Test", LastName: "Shopper", Email: "shopper@example.com", Country: "GB", Address1: "1 Example Road", City: "London", Zip: "SW1A 1AA"}, Terms: true}
}
func merchTestSign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
func TestMerchRetailPrice(t *testing.T) {
	for input, want := range map[int64]int64{1: 49, 49: 49, 50: 99, 99: 99, 100: 149, 149: 149, 150: 199, 1996: 1999, 2000: 2049} {
		if got := merchRetailPrice(input); got != want {
			t.Errorf("%d -> %d want %d", input, got, want)
		}
	}
	for cost := int64(1); cost < 100000; cost++ {
		got := merchRetailPrice(cost)
		if got < cost || got-cost >= 50 || (got%100 != 49 && got%100 != 99) {
			t.Fatalf("invalid rounding %d -> %d", cost, got)
		}
	}
}
func TestMerchPricingCoversCostsAndNoMargin(t *testing.T) {
	c := merchTestConfig(t)
	got, err := merchCalculate(1600, 500, c, c.Countries["GB"])
	if err != nil {
		t.Fatal(err)
	}
	if got.Items != 1600 || got.Shipping != 400 || got.SupplierTax != 400 || got.Total != 2499 {
		t.Fatalf("totals=%+v", got)
	}
	costs := got.Items + got.Shipping + got.SupplierTax
	if got.Total-merchCeil(got.Total*c.FeeBPS, 10000)-c.FeeFixed < costs {
		t.Fatal("under cost")
	}
	for _, input := range [][2]int64{{0, 100}, {-1, 1}, {100, -1}, {1000001, 0}, {1, 1000001}} {
		if _, err := merchCalculate(input[0], input[1], c, c.Countries["GB"]); err == nil {
			t.Fatal("accepted invalid amount")
		}
	}
}
func TestMerchConfigFailsClosed(t *testing.T) {
	c := merchTestConfig(t)
	for _, edit := range []func(*merchConfig){func(c *merchConfig) { c.Enabled = false }, func(c *merchConfig) { c.ReviewedUntil = time.Now().Add(-time.Hour) }, func(c *merchConfig) { c.CostToGBP = 0 }, func(c *merchConfig) { c.ShippingToGBP = 0 }, func(c *merchConfig) { c.BaseURL = "https://shop.example@evil.example/path" }, func(c *merchConfig) { c.FeeBPS = 10000 }, func(c *merchConfig) { c.ShopID = "../123" }} {
		copy := c
		edit(&copy)
		raw, _ := json.Marshal(copy)
		t.Setenv("MERCH_CONFIG", string(raw))
		if _, err := loadMerchConfig(); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
}
func TestMerchInternationalRequestValidation(t *testing.T) {
	c := merchTestConfig(t)
	q := merchTestRequest()
	q.Address.Country = "AU"
	if err := validateMerchRequest(q, c); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*merchQuoteRequest){func(q *merchQuoteRequest) { q.Terms = false }, func(q *merchQuoteRequest) { q.Website = "bot" }, func(q *merchQuoteRequest) { q.Address.Country = "XX" }, func(q *merchQuoteRequest) { q.Address.Email = "Bad <other@example.com>" }, func(q *merchQuoteRequest) { q.Address.Address1 = "" }, func(q *merchQuoteRequest) { q.Lines[0].Quantity = -1 }, func(q *merchQuoteRequest) { q.Lines[0].Quantity = 11 }, func(q *merchQuoteRequest) { q.Lines[0].ProductID = "../../orders" }, func(q *merchQuoteRequest) { q.Lines = append(q.Lines, q.Lines[0]) }} {
		q := merchTestRequest()
		edit(&q)
		if validateMerchRequest(q, c) == nil {
			t.Fatal("invalid request accepted")
		}
	}
}
func TestMerchQuoteGuestHappyPathAndInvalidBodies(t *testing.T) {
	merchTestConfig(t)
	oldCall, oldSave := merchCallPrintify, merchSaveQuote
	t.Cleanup(func() { merchCallPrintify = oldCall; merchSaveQuote = oldSave })
	calls, saves := 0, 0
	merchCallPrintify = func(_ context.Context, _ merchConfig, method, path string, input, out any) error {
		calls++
		var raw string
		if method == "GET" {
			raw = `{"id":"aaaaaaaaaaaaaaaaaaaaaaaa","visible":true,"title":"Mug","variants":[{"id":10,"title":"White","cost":1000,"is_enabled":true,"is_available":true}]}`
		} else {
			if path != "orders/shipping.json" {
				t.Fatal("quote placed an order")
			}
			raw = `{"standard":500}`
		}
		return json.Unmarshal([]byte(raw), out)
	}
	merchSaveQuote = func(_ context.Context, o merchOrder) error {
		saves++
		if o.Address.Email != "shopper@example.com" || o.ShopID != "123" || o.Status != "quoted" || o.Totals.Total != 2499 || o.TermsVersion == "" {
			t.Fatalf("order=%+v", o)
		}
		return nil
	}
	q := merchTestRequest()
	body, _ := json.Marshal(q)
	req := httptest.NewRequest("POST", "/api/shop/quote", strings.NewReader(string(body)))
	req.Header.Set("Origin", "https://shop.example")
	res := httptest.NewRecorder()
	MerchandiseQuote(res, req)
	if res.Code != 200 || calls != 2 || saves != 1 {
		t.Fatalf("status=%d calls=%d saves=%d body=%s", res.Code, calls, saves, res.Body)
	}
	var result struct {
		Receipt string `json:"receipt"`
		OrderID string `json:"orderId"`
	}
	_ = json.Unmarshal(res.Body.Bytes(), &result)
	if !merchIDPattern.MatchString(result.Receipt) || merchReceiptID(result.Receipt) != result.OrderID {
		t.Fatal("invalid receipt")
	}
	if strings.Contains(res.Body.String(), "secret-") || strings.Contains(res.Body.String(), "shopper@example.com") {
		t.Fatal("private data exposed")
	}
	for _, body := range []string{"", `{`, `{} {}`, `{"price":1}`, strings.Repeat(" ", 17000)} {
		req := httptest.NewRequest("POST", "/api/shop/quote", strings.NewReader(body))
		req.Header.Set("Origin", "https://shop.example")
		res := httptest.NewRecorder()
		MerchandiseQuote(res, req)
		if res.Code != 400 {
			t.Fatalf("status=%d", res.Code)
		}
	}
	if saves != 1 {
		t.Fatal("invalid request persisted")
	}
}
func TestMerchPublicEndpointsAndReceiptProtection(t *testing.T) {
	merchTestConfig(t)
	for _, handler := range []http.HandlerFunc{MerchandiseQuote, MerchandiseCheckout} {
		for _, origin := range []string{"", "https://evil.example"} {
			req := httptest.NewRequest("POST", "/api/shop", strings.NewReader("{}"))
			req.Header.Set("Origin", origin)
			res := httptest.NewRecorder()
			handler(res, req)
			if res.Code != 403 {
				t.Fatalf("origin %q status %d", origin, res.Code)
			}
		}
	}
	req := httptest.NewRequest("GET", "/api/shop/order", nil)
	res := httptest.NewRecorder()
	MerchandiseOrder(res, req)
	if res.Code != 404 {
		t.Fatalf("unauthorized receipt status %d", res.Code)
	}
}
func TestMerchCheckoutUsesSavedTotalAndIdempotency(t *testing.T) {
	merchTestConfig(t)
	t.Setenv("ALLOWED_ORIGINS", "https://preview.example,https://shop.example/path")
	oldRead, oldStripe, oldSave := merchReceiptOrder, merchCallStripe, merchSaveSession
	t.Cleanup(func() { merchReceiptOrder = oldRead; merchCallStripe = oldStripe; merchSaveSession = oldSave })
	order := merchOrder{ID: strings.Repeat("a", 64), Status: "quoted", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(20 * time.Minute), Totals: merchTotals{Total: 2499}, Address: merchAddress{Email: "shopper@example.com"}}
	merchReceiptOrder = func(*http.Request) (*firestore.DocumentRef, merchOrder, error) { return nil, order, nil }
	calls := 0
	merchCallStripe = func(_ context.Context, method, path string, form url.Values, key string, out any) error {
		calls++
		if form.Get("line_items[0][price_data][unit_amount]") != "2499" || form.Get("line_items[0][price_data][currency]") != "gbp" || key != "merch-"+order.ID || form.Get("metadata[orderId]") != order.ID || form.Get("success_url") != "https://preview.example/shop/?order="+order.ID || form.Get("cancel_url") != "https://preview.example/shop/?cancelled=1" {
			t.Fatalf("unexpected checkout %v", form)
		}
		return json.Unmarshal([]byte(`{"id":"cs_test_123","url":"https://checkout.stripe.com/c/pay/test"}`), out)
	}
	merchSaveSession = func(_ context.Context, _ *firestore.DocumentRef, id, u string) error {
		order.SessionID = id
		order.CheckoutURL = u
		return nil
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/shop/checkout", strings.NewReader(`{"total":1}`))
		req.Header.Set("Origin", "https://preview.example")
		res := httptest.NewRecorder()
		MerchandiseCheckout(res, req)
		if res.Code != 200 {
			t.Fatalf("status=%d body=%s", res.Code, res.Body)
		}
	}
	if calls != 1 {
		t.Fatal("checkout session duplicated")
	}
	badOrigin := httptest.NewRequest("POST", "/api/shop/checkout", nil)
	badOrigin.Header.Set("Origin", "https://shop.example/path")
	badOriginResult := httptest.NewRecorder()
	MerchandiseCheckout(badOriginResult, badOrigin)
	if badOriginResult.Code != http.StatusForbidden {
		t.Fatalf("path-bearing origin accepted: status=%d body=%s", badOriginResult.Code, badOriginResult.Body)
	}
	order.ExpiresAt = time.Now().Add(-time.Minute)
	req := httptest.NewRequest("POST", "/api/shop/checkout", nil)
	req.Header.Set("Origin", "https://preview.example")
	res := httptest.NewRecorder()
	MerchandiseCheckout(res, req)
	if res.Code != 409 {
		t.Fatal("expired quote accepted")
	}
}
func TestMerchPaymentClaims(t *testing.T) {
	o := merchOrder{ID: strings.Repeat("a", 64), ShopID: "123", SessionID: "cs_123", Status: "quoted", Totals: merchTotals{Total: 2499}}
	s := merchPaidSession{ID: "cs_123", PaymentStatus: "paid", Currency: "gbp", AmountTotal: 2499}
	s.Metadata.OrderID = o.ID
	if claim, err := merchShouldClaim(o, s, "123"); !claim || err != nil {
		t.Fatal("valid payment rejected")
	}
	for _, state := range []string{"submitting", "needs-review", "canceled"} {
		copy := o
		copy.Status = state
		if claim, err := merchShouldClaim(copy, s, "123"); claim || err == nil {
			t.Fatal("unsafe retry allowed")
		}
	}
	for _, state := range []string{"submitted", "on-hold", "in-production", "fulfilled", "has-issues"} {
		copy := o
		copy.Status = state
		copy.PrintifyID = "existing"
		if claim, err := merchShouldClaim(copy, s, "123"); claim || err != nil {
			t.Fatal("duplicate order allowed")
		}
	}
	for _, edit := range []func(*merchPaidSession){func(s *merchPaidSession) { s.AmountTotal = 1 }, func(s *merchPaidSession) { s.Currency = "usd" }, func(s *merchPaidSession) { s.PaymentStatus = "unpaid" }, func(s *merchPaidSession) { s.ID = "cs_other" }, func(s *merchPaidSession) { s.Metadata.OrderID = "other" }} {
		copy := s
		edit(&copy)
		if merchPaymentMatches(o, copy, "123") == nil {
			t.Fatal("mismatched payment accepted")
		}
	}
	if merchPaymentMatches(o, s, "999") == nil {
		t.Fatal("cross-shop order accepted")
	}
}
func TestMerchStripeSignaturesAndWebhook(t *testing.T) {
	merchTestConfig(t)
	now := time.Now()
	body := []byte(`{"type":"checkout.session.completed","data":{"object":{"id":"cs_123","payment_status":"paid","metadata":{"orderId":"` + strings.Repeat("a", 64) + `"}}}}`)
	stamp := strconv.FormatInt(now.Unix(), 10)
	header := "t=" + stamp + ",v1=" + merchTestSign(append([]byte(stamp+"."), body...), "stripe-signing")
	if !merchStripeSignature(body, header, "stripe-signing", now) || merchStripeSignature(append(body, 'x'), header, "stripe-signing", now) || merchStripeSignature(body, header, "stripe-signing", now.Add(6*time.Minute)) || merchStripeSignature(body, header, "", now) {
		t.Fatal("signature validation failed")
	}
	old := merchFulfil
	t.Cleanup(func() { merchFulfil = old })
	calls := 0
	merchFulfil = func(context.Context, merchPaidSession) error { calls++; return nil }
	for _, valid := range []bool{false, true} {
		req := httptest.NewRequest("POST", "/api/shop/stripe-webhook", bytesReader(body))
		if valid {
			req.Header.Set("Stripe-Signature", header)
		}
		res := httptest.NewRecorder()
		MerchandiseStripeWebhook(res, req)
		want := 400
		if valid {
			want = 200
		}
		if res.Code != want {
			t.Fatalf("status=%d", res.Code)
		}
	}
	if calls != 1 {
		t.Fatal("unverified webhook triggered fulfilment")
	}
	merchFulfil = func(context.Context, merchPaidSession) error { return errors.New("retry") }
	req := httptest.NewRequest("POST", "/api/shop/stripe-webhook", bytesReader(body))
	req.Header.Set("Stripe-Signature", header)
	res := httptest.NewRecorder()
	MerchandiseStripeWebhook(res, req)
	if res.Code != 503 {
		t.Fatal("failure did not request retry")
	}
}
func bytesReader(body []byte) *strings.Reader { return strings.NewReader(string(body)) }
func TestMerchPrintifySignature(t *testing.T) {
	merchTestConfig(t)
	body := []byte(`{"type":"product:created"}`)
	for _, valid := range []bool{false, true} {
		req := httptest.NewRequest("POST", "/api/shop/printify-webhook", bytesReader(body))
		if valid {
			req.Header.Set("X-Pfy-Signature", "sha256="+merchTestSign(body, "printify-signing"))
		}
		res := httptest.NewRecorder()
		MerchandisePrintifyWebhook(res, req)
		want := 400
		if valid {
			want = 200
		}
		if res.Code != want {
			t.Fatalf("status=%d", res.Code)
		}
	}
}

func TestMerchCatalogueGuestFilteringAndCaching(t *testing.T) {
	merchTestConfig(t)
	old := merchCallPrintify
	t.Cleanup(func() {
		merchCallPrintify = old
		merchCache.Lock()
		merchCache.Until = time.Time{}
		merchCache.Unlock()
	})
	merchCache.Lock()
	merchCache.Until = time.Time{}
	merchCache.Unlock()
	calls := 0
	merchCallPrintify = func(_ context.Context, _ merchConfig, method, path string, input, out any) error {
		calls++
		return json.Unmarshal([]byte(`{"last_page":1,"data":[{"id":"aaaaaaaaaaaaaaaaaaaaaaaa","title":"Mug","description":"<p>Ceramic &amp; glossy</p>","visible":true,"images":[{"src":"javascript:alert(1)"},{"src":"https://images.example/mug.jpg","variant_ids":[1]}],"variants":[{"id":1,"title":"White","cost":503,"is_enabled":true,"is_available":true},{"id":2,"title":"Unavailable","cost":503,"is_enabled":true,"is_available":false}]},{"id":"bbbbbbbbbbbbbbbbbbbbbbbb","title":"Hidden","visible":false}]}`), out)
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/shop/catalogue", nil)
		res := httptest.NewRecorder()
		MerchandiseCatalogue(res, req)
		if res.Code != 200 {
			t.Fatalf("status=%d body=%s", res.Code, res.Body)
		}
		body := res.Body.String()
		if strings.Contains(body, "Unavailable") || strings.Contains(body, "Hidden") || strings.Contains(body, "javascript:") || strings.Contains(body, "secret-") || strings.Contains(body, "<p>") {
			t.Fatalf("unsafe catalogue %s", body)
		}
		if !strings.Contains(body, `"productionPrice":449`) {
			t.Fatalf("unrounded catalogue %s", body)
		}
	}
	if calls != 1 {
		t.Fatal("catalogue cache unused")
	}
}
func TestMerchQuoteRejectsStaleProductsAndProviderFailures(t *testing.T) {
	merchTestConfig(t)
	oldCall, oldSave := merchCallPrintify, merchSaveQuote
	t.Cleanup(func() { merchCallPrintify = oldCall; merchSaveQuote = oldSave })
	merchSaveQuote = func(context.Context, merchOrder) error { t.Fatal("invalid quote persisted"); return nil }
	for _, scenario := range []string{"disabled", "unavailable", "provider-error", "shipping-error", "shipping-missing"} {
		merchCallPrintify = func(_ context.Context, _ merchConfig, method, path string, input, out any) error {
			if scenario == "provider-error" || method == "POST" && scenario == "shipping-error" {
				return errors.New("provider failure")
			}
			if method == "POST" {
				return json.Unmarshal([]byte(`{}`), out)
			}
			p := merchProduct{ID: strings.Repeat("a", 24), Visible: true, Variants: []merchVariant{{ID: 10, Cost: 1000, Enabled: scenario != "disabled", Available: scenario != "unavailable"}}}
			b, _ := json.Marshal(p)
			return json.Unmarshal(b, out)
		}
		body, _ := json.Marshal(merchTestRequest())
		req := httptest.NewRequest("POST", "/api/shop/quote", bytesReader(body))
		req.Header.Set("Origin", "https://shop.example")
		res := httptest.NewRecorder()
		MerchandiseQuote(res, req)
		if res.Code == 200 {
			t.Fatalf("accepted %s", scenario)
		}
	}
}
func TestMerchProductPublishAcknowledgement(t *testing.T) {
	c := merchTestConfig(t)
	old := merchCallPrintify
	t.Cleanup(func() { merchCallPrintify = old })
	id := strings.Repeat("a", 24)
	calls := 0
	merchCallPrintify = func(_ context.Context, _ merchConfig, method, path string, input, out any) error {
		calls++
		if method == "GET" {
			return json.Unmarshal([]byte(`{"id":"`+id+`","visible":true}`), out)
		}
		if path != "products/"+id+"/publishing_succeeded.json" {
			t.Fatal(path)
		}
		external := input.(map[string]any)["external"].(map[string]string)
		if external["handle"] != "https://shop.example/shop/#product-"+id {
			t.Fatal("invalid external handle")
		}
		return nil
	}
	if err := merchPublishProduct(context.Background(), c, id, "create"); err != nil || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	c.Enabled = false
	if merchPublishProduct(context.Background(), c, id, "update") == nil {
		t.Fatal("acknowledged an unavailable storefront")
	}
}
func TestMerchUnpaidWebhookNeverFulfils(t *testing.T) {
	merchTestConfig(t)
	old := merchFulfil
	t.Cleanup(func() { merchFulfil = old })
	merchFulfil = func(context.Context, merchPaidSession) error { t.Fatal("unpaid order fulfilled"); return nil }
	body := []byte(`{"type":"checkout.session.completed","data":{"object":{"payment_status":"unpaid","metadata":{"orderId":"` + strings.Repeat("a", 64) + `"}}}}`)
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	req := httptest.NewRequest("POST", "/api/shop/stripe-webhook", bytesReader(body))
	req.Header.Set("Stripe-Signature", "t="+stamp+",v1="+merchTestSign(append([]byte(stamp+"."), body...), "stripe-signing"))
	res := httptest.NewRecorder()
	MerchandiseStripeWebhook(res, req)
	if res.Code != 200 {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestMerchTestPaymentsCannotCreatePrintifyOrders(t *testing.T) {
	c := merchTestConfig(t)
	o := merchOrder{ID: strings.Repeat("a", 64), ShopID: c.ShopID, SessionID: "cs_test", Status: "quoted", Totals: merchTotals{Total: 2499}}
	s := merchPaidSession{ID: o.SessionID, PaymentStatus: "paid", Currency: "gbp", AmountTotal: 2499}
	s.Metadata.OrderID = o.ID
	if next, err := merchPaymentDecision(o, s, c); err != nil || next != "test-paid" {
		t.Fatalf("test payment decision %q %v", next, err)
	}
	c.StripeLive = true
	if _, err := merchPaymentDecision(o, s, c); err == nil {
		t.Fatal("test event accepted with live config")
	}
	s.LiveMode = true
	if next, err := merchPaymentDecision(o, s, c); err != nil || next != "submitting" {
		t.Fatalf("live decision %q %v", next, err)
	}
	c.StripeLive = false
	if _, err := merchPaymentDecision(o, s, c); err == nil {
		t.Fatal("live event accepted with test config")
	}
}
