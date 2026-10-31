package ipace

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Provider costs are deliberately converted using explicit, reviewed configuration.
// Never assume that an unlabelled Printify amount is GBP.
type merchCountry struct {
	Name           string `json:"name"`
	SupplierTaxBPS int64  `json:"supplierTaxBps"`
}
type merchConfig struct {
	StripeLive    bool                    `json:"stripeLive"`
	SellerName    string                  `json:"sellerName"`
	SellerAddress string                  `json:"sellerAddress"`
	TermsReviewed bool                    `json:"termsReviewed"`
	Enabled       bool                    `json:"enabled"`
	ShopID        string                  `json:"shopId"`
	BaseURL       string                  `json:"baseUrl"`
	CostToGBP     int64                   `json:"costToGbpMicros"`
	ShippingToGBP int64                   `json:"shippingToGbpMicros"`
	FeeBPS        int64                   `json:"paymentFeeBps"`
	FeeFixed      int64                   `json:"paymentFeePence"`
	ReviewedUntil time.Time               `json:"reviewedUntil"`
	Countries     map[string]merchCountry `json:"countries"`
}
type merchVariant struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Cost      int64  `json:"cost"`
	Enabled   bool   `json:"is_enabled"`
	Available bool   `json:"is_available"`
}
type merchImage struct {
	Src        string `json:"src"`
	VariantIDs []int  `json:"variant_ids"`
	Default    bool   `json:"is_default"`
}
type merchProduct struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Visible     bool           `json:"visible"`
	Variants    []merchVariant `json:"variants"`
	Images      []merchImage   `json:"images"`
}
type merchLine struct {
	ProductID string `json:"product_id" firestore:"productId"`
	VariantID int    `json:"variant_id" firestore:"variantId"`
	Quantity  int    `json:"quantity" firestore:"quantity"`
}
type merchAddress struct {
	FirstName string `json:"first_name" firestore:"firstName"`
	LastName  string `json:"last_name" firestore:"lastName"`
	Email     string `json:"email" firestore:"email"`
	Phone     string `json:"phone" firestore:"phone"`
	Country   string `json:"country" firestore:"country"`
	Region    string `json:"region" firestore:"region"`
	Address1  string `json:"address1" firestore:"address1"`
	Address2  string `json:"address2" firestore:"address2"`
	City      string `json:"city" firestore:"city"`
	Zip       string `json:"zip" firestore:"zip"`
}
type merchQuoteRequest struct {
	Lines   []merchLine  `json:"line_items"`
	Address merchAddress `json:"address_to"`
	Terms   bool         `json:"terms"`
	Website string       `json:"website"`
}
type merchTotals struct {
	Items       int64 `json:"items" firestore:"items"`
	Shipping    int64 `json:"shipping" firestore:"shipping"`
	SupplierTax int64 `json:"supplierTax" firestore:"supplierTax"`
	Processing  int64 `json:"processing" firestore:"processing"`
	Total       int64 `json:"total" firestore:"total"`
}
type merchOrder struct {
	ShopID      string       `firestore:"shopId"`
	ID          string       `firestore:"id"`
	Lines       []merchLine  `firestore:"lines"`
	Address     merchAddress `firestore:"address"`
	Totals      merchTotals  `firestore:"totals"`
	Summary     string       `firestore:"summary"`
	CreatedAt   time.Time    `firestore:"createdAt"`
	ExpiresAt   time.Time    `firestore:"expiresAt"`
	Status      string       `firestore:"status"`
	SessionID   string       `firestore:"sessionId"`
	CheckoutURL string       `firestore:"checkoutUrl"`
	PrintifyID  string       `firestore:"printifyId"`

	TermsVersion string `firestore:"termsVersion"`
}

var merchIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var merchProductIDPattern = regexp.MustCompile(`^[a-f0-9]{24}$`)
var merchHTTP = &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
var merchDB = firestoreClient
var merchSaveQuote = func(ctx context.Context, order merchOrder) error {
	db, err := merchDB(ctx)
	if err != nil {
		return err
	}
	_, err = db.Collection("merchandiseOrders").Doc(order.ID).Create(ctx, order)
	return err
}
var merchReceiptOrder = merchLoadReceipt
var merchSaveSession = func(ctx context.Context, ref *firestore.DocumentRef, id, checkoutURL string) error {
	_, err := ref.Update(ctx, []firestore.Update{{Path: "sessionId", Value: id}, {Path: "checkoutUrl", Value: checkoutURL}})
	return err
}
var merchCallPrintify = merchPrintify
var merchCallStripe = merchStripe
var merchCache struct {
	sync.Mutex
	Key      string
	Until    time.Time
	Products []merchProduct
}

func loadMerchConfig() (merchConfig, error) {
	var c merchConfig
	if json.Unmarshal([]byte(os.Getenv("MERCH_CONFIG")), &c) != nil || !c.Enabled {
		return c, errors.New("shop unavailable")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || (u.Path != "" && u.Path != "/") {
		return c, errors.New("invalid shop URL")
	}
	if _, err := strconv.ParseUint(c.ShopID, 10, 64); err != nil {
		return c, errors.New("invalid shop")
	}
	if c.CostToGBP <= 0 || c.CostToGBP > 10000000 || c.ShippingToGBP <= 0 || c.ShippingToGBP > 10000000 || c.FeeBPS < 0 || c.FeeBPS > 1000 || c.FeeFixed < 0 || c.FeeFixed > 1000 || !c.ReviewedUntil.After(time.Now()) || len(c.Countries) == 0 {
		return c, errors.New("pricing needs review")
	}
	for code, country := range c.Countries {
		if len(code) != 2 || code != strings.ToUpper(code) || country.Name == "" || country.SupplierTaxBPS < 0 || country.SupplierTaxBPS > 10000 {
			return c, errors.New("invalid destination configuration")
		}
	}
	if os.Getenv("PRINTIFY_API_TOKEN") == "" {
		return c, errors.New("Printify unavailable")
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	return c, nil
}
func merchProvider(ctx context.Context, method, endpoint, token, contentType string, body []byte, key string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "IPACE-Owners-Merchandise/1.0")
	req.Header.Set("Content-Type", contentType)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, err := merchHTTP.Do(req)
	if err != nil {
		return errors.New("provider connection failed")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("provider HTTP %d", res.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out)
}
func merchPrintify(ctx context.Context, c merchConfig, method, path string, input, out any) error {
	var body []byte
	var err error
	if input != nil {
		body, err = json.Marshal(input)
		if err != nil {
			return err
		}
	}
	return merchProvider(ctx, method, "https://api.printify.com/v1/shops/"+c.ShopID+"/"+path, os.Getenv("PRINTIFY_API_TOKEN"), "application/json", body, "", out)
}
func merchStripe(ctx context.Context, method, path string, form url.Values, key string, out any) error {
	return merchProvider(ctx, method, "https://api.stripe.com/v1/"+path, os.Getenv("STRIPE_SECRET_KEY"), "application/x-www-form-urlencoded", []byte(form.Encode()), key, out)
}
func merchProducts(ctx context.Context, c merchConfig) ([]merchProduct, error) {
	merchCache.Lock()
	defer merchCache.Unlock()
	if merchCache.Key == c.ShopID && time.Now().Before(merchCache.Until) {
		return merchCache.Products, nil
	}
	products := []merchProduct{}
	for page := 1; page <= 20; page++ {
		var result struct {
			Data     []merchProduct `json:"data"`
			LastPage int            `json:"last_page"`
		}
		if err := merchCallPrintify(ctx, c, "GET", fmt.Sprintf("products.json?limit=50&page=%d", page), nil, &result); err != nil {
			return nil, err
		}
		products = append(products, result.Data...)
		if page >= result.LastPage {
			merchCache.Key = c.ShopID
			merchCache.Products = products
			merchCache.Until = time.Now().Add(time.Minute)
			return products, nil
		}
	}
	return nil, errors.New("catalogue too large")
}
func merchReady(c merchConfig) bool {
	prefix := "sk_test_"
	if c.StripeLive {
		prefix = "sk_live_"
	}
	return strings.HasPrefix(os.Getenv("STRIPE_SECRET_KEY"), prefix) && os.Getenv("STRIPE_SECRET_KEY") != "" && os.Getenv("STRIPE_WEBHOOK_SECRET") != "" && os.Getenv("PRINTIFY_WEBHOOK_SECRET") != ""
}
func merchPublicRequest(w http.ResponseWriter, r *http.Request, method string) bool {
	w.Header().Set("Cache-Control", "private, no-store")
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return false
	}
	if r.Method != method {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return false
	}
	if method == http.MethodPost && rejectMissingOrDisallowedOrigin(w, r) {
		return false
	}
	return true
}
func MerchandiseCatalogue(w http.ResponseWriter, r *http.Request) {
	if !merchPublicRequest(w, r, http.MethodGet) {
		return
	}
	c, err := loadMerchConfig()
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "The merchandise catalogue is being prepared. Please check back soon."})
		return
	}
	products, err := merchProducts(r.Context(), c)
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Products are temporarily unavailable. Please try again."})
		return
	}
	public := []map[string]any{}
	for _, p := range products {
		if !p.Visible {
			continue
		}
		variants := []map[string]any{}
		for _, v := range p.Variants {
			if v.Enabled && v.Available && v.Cost > 0 && v.Cost <= 1000000 {
				variants = append(variants, map[string]any{"id": v.ID, "title": v.Title, "productionPrice": merchRetailPrice(merchCeil(v.Cost*c.CostToGBP, 1000000))})
			}
		}
		if len(variants) == 0 {
			continue
		}
		images := []merchImage{}
		for _, im := range p.Images {
			if merchHTTPS(im.Src) {
				images = append(images, im)
			}
		}
		// Description is provider HTML. Return plain text; never inject provider markup.
		public = append(public, map[string]any{"id": p.ID, "title": p.Title, "description": merchDescription(p.Description), "variants": variants, "images": images})
	}
	countries := []map[string]string{}
	for code, v := range c.Countries {
		countries = append(countries, map[string]string{"code": code, "name": v.Name})
	}
	sort.Slice(countries, func(i, j int) bool { return countries[i]["name"] < countries[j]["name"] })
	writeJSON(w, 200, map[string]any{"products": public, "countries": countries, "currency": "GBP", "testMode": !c.StripeLive, "checkoutEnabled": merchReady(c) && c.TermsReviewed && c.SellerName != "" && c.SellerAddress != "", "sellerName": c.SellerName, "sellerAddress": c.SellerAddress})
}
func merchHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}
func merchCeil(n, d int64) int64 { return (n + d - 1) / d }

// The smallest .49/.99 price at or above cost; integer pence only.
func merchRetailPrice(pence int64) int64 {
	if pence <= 49 {
		return 49
	}
	return merchCeil(pence+1, 50)*50 - 1
}
func merchCalculate(items, shipping int64, c merchConfig, country merchCountry) (merchTotals, error) {
	if items <= 0 || items > 1000000 || shipping < 0 || shipping > 1000000 {
		return merchTotals{}, errors.New("invalid amount")
	}
	t := merchTotals{Items: items, Shipping: merchCeil(shipping*c.ShippingToGBP, 1000000)}
	t.SupplierTax = merchCeil((t.Items+t.Shipping)*country.SupplierTaxBPS, 10000)
	base := t.Items + t.Shipping + t.SupplierTax
	t.Total = merchRetailPrice(merchCeil((base+c.FeeFixed)*10000, 10000-c.FeeBPS))
	t.Processing = t.Total - base
	if t.Total < 30 || t.Total > 1000000 {
		return t, errors.New("order total outside supported range")
	}
	return t, nil
}
func validateMerchRequest(q merchQuoteRequest, c merchConfig) error {
	if !q.Terms || q.Website != "" || len(q.Lines) == 0 || len(q.Lines) > 10 {
		return errors.New("Choose merchandise and accept the merchandise terms.")
	}
	if _, ok := c.Countries[q.Address.Country]; !ok {
		return errors.New("Choose a supported delivery country.")
	}
	seen := map[string]bool{}
	count := 0
	for _, l := range q.Lines {
		k := fmt.Sprintf("%s:%d", l.ProductID, l.VariantID)
		if !merchProductIDPattern.MatchString(l.ProductID) || l.VariantID <= 0 || l.Quantity < 1 || l.Quantity > 10 || seen[k] {
			return errors.New("Invalid basket.")
		}
		seen[k] = true
		count += l.Quantity
	}
	if count > 20 {
		return errors.New("Please order no more than 20 items at a time.")
	}
	a := q.Address
	for _, s := range []string{a.FirstName, a.LastName, a.Email, a.Address1, a.City} {
		if strings.TrimSpace(s) == "" {
			return errors.New("Complete your name, email and delivery address.")
		}
	}
	for _, s := range []string{a.FirstName, a.LastName, a.Email, a.Phone, a.Address1, a.Address2, a.City, a.Zip, a.Region} {
		if len(s) > 254 || strings.ContainsAny(s, "\r\n\x00") {
			return errors.New("Check your delivery details.")
		}
	}
	parsed, err := mail.ParseAddress(a.Email)
	if err != nil || parsed.Address != a.Email {
		return errors.New("Enter a valid email address.")
	}
	return nil
}
func MerchandiseQuote(w http.ResponseWriter, r *http.Request) {
	if !merchPublicRequest(w, r, http.MethodPost) {
		return
	}
	c, err := loadMerchConfig()
	if err != nil || !merchReady(c) || !c.TermsReviewed || c.SellerName == "" || c.SellerAddress == "" {
		writeJSON(w, 503, map[string]any{"error": "Checkout is not available yet."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var q merchQuoteRequest
	if decodeJSON(r, &q) != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid request."})
		return
	}
	if err := validateMerchRequest(q, c); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	// Fetch each selected product afresh so stale catalogue entries cannot be bought.
	var items int64
	summary := []string{}
	for _, line := range q.Lines {
		var p merchProduct
		if merchCallPrintify(r.Context(), c, "GET", "products/"+line.ProductID+".json", nil, &p) != nil {
			writeJSON(w, 503, map[string]any{"error": "Could not check product availability."})
			return
		}
		found := false
		if p.ID == line.ProductID && p.Visible {
			for _, v := range p.Variants {
				if v.ID == line.VariantID && v.Enabled && v.Available && v.Cost > 0 && v.Cost <= 1000000 {
					found = true
					items += merchCeil(v.Cost*c.CostToGBP, 1000000) * int64(line.Quantity)
					summary = append(summary, fmt.Sprintf("%d × %s — %s", line.Quantity, p.Title, v.Title))
					break
				}
			}
		}
		if !found {
			writeJSON(w, 409, map[string]any{"error": "An item is no longer available. Refresh the Merchandise page and choose again."})
			return
		}
	}
	var shipping struct {
		Standard *int64 `json:"standard"`
	}
	if merchCallPrintify(r.Context(), c, "POST", "orders/shipping.json", map[string]any{"line_items": q.Lines, "address_to": q.Address}, &shipping) != nil || shipping.Standard == nil {
		writeJSON(w, 422, map[string]any{"error": "Delivery is unavailable for this basket or address. Check the address or try a different item."})
		return
	}
	totals, err := merchCalculate(items, *shipping.Standard, c, c.Countries[q.Address.Country])
	if err != nil {
		writeJSON(w, 422, map[string]any{"error": "Could not calculate a supported order total."})
		return
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		writeJSON(w, 503, map[string]any{"error": "Please try again."})
		return
	}
	receipt := hex.EncodeToString(secret)
	id := merchReceiptID(receipt)
	order := merchOrder{ShopID: c.ShopID, ID: id, Lines: q.Lines, Address: q.Address, Totals: totals, Summary: strings.Join(summary, "; "), CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(30 * time.Minute), Status: "quoted", TermsVersion: "2026-10-29"}
	err = merchSaveQuote(r.Context(), order)
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Could not save your quote. Please try again."})
		return
	}
	writeJSON(w, 200, map[string]any{"receipt": receipt, "orderId": id, "totals": map[string]int64{"total": totals.Total}, "expiresAt": order.ExpiresAt, "summary": order.Summary})
}
func merchReceiptID(receipt string) string {
	sum := sha256.Sum256([]byte(receipt))
	return hex.EncodeToString(sum[:])
}
func merchLoadReceipt(r *http.Request) (*firestore.DocumentRef, merchOrder, error) {
	var order merchOrder
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		return nil, order, errors.New("invalid receipt")
	}
	receipt := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !merchIDPattern.MatchString(receipt) {
		return nil, order, errors.New("invalid receipt")
	}
	db, err := merchDB(r.Context())
	if err != nil {
		return nil, order, err
	}
	ref := db.Collection("merchandiseOrders").Doc(merchReceiptID(receipt))
	snap, err := ref.Get(r.Context())
	if err == nil {
		err = snap.DataTo(&order)
	}
	return ref, order, err
}
func MerchandiseCheckout(w http.ResponseWriter, r *http.Request) {
	if !merchPublicRequest(w, r, http.MethodPost) {
		return
	}
	returnOrigin, err := merchCheckoutReturnOrigin(r)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Checkout origin is invalid."})
		return
	}
	c, err := loadMerchConfig()
	if err != nil || !merchReady(c) || !c.TermsReviewed || c.SellerName == "" || c.SellerAddress == "" {
		writeJSON(w, 503, map[string]any{"error": "Checkout is temporarily unavailable."})
		return
	}
	ref, o, err := merchReceiptOrder(r)
	if err != nil {
		writeJSON(w, 404, map[string]any{"error": "Quote not found. Please request a new quote."})
		return
	}
	if o.Status != "quoted" || !time.Now().Before(o.ExpiresAt) {
		writeJSON(w, 409, map[string]any{"error": "This quote has expired or has already been paid. Request a new quote if needed."})
		return
	}
	if o.SessionID != "" {
		writeJSON(w, 200, map[string]any{"url": o.CheckoutURL})
		return
	}
	form := url.Values{"mode": {"payment"}, "payment_method_types[0]": {"card"}, "customer_email": {o.Address.Email}, "client_reference_id": {o.ID}, "metadata[orderId]": {o.ID}, "success_url": {returnOrigin + "/shop/?order=" + o.ID}, "cancel_url": {returnOrigin + "/shop/?cancelled=1"}, "expires_at": {strconv.FormatInt(o.CreatedAt.Add(time.Hour).Unix(), 10)}, "line_items[0][price_data][currency]": {"gbp"}, "line_items[0][price_data][unit_amount]": {strconv.FormatInt(o.Totals.Total, 10)}, "line_items[0][price_data][product_data][name]": {"I-PACE Owners merchandise including delivery"}, "line_items[0][price_data][product_data][description]": {cleanString(o.Summary, 500)}, "line_items[0][quantity]": {"1"}}
	// The fixed expiry keeps repeated idempotent requests identical and leaves
	// at least 30 minutes for Checkout when the quote is still valid.
	var session struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if merchCallStripe(r.Context(), "POST", "checkout/sessions", form, "merch-"+o.ID, &session) != nil || session.ID == "" || !strings.HasPrefix(session.URL, "https://checkout.stripe.com/") {
		writeJSON(w, 503, map[string]any{"error": "Could not open secure checkout. Please retry this quote."})
		return
	}
	// The idempotency key makes concurrent calls return the same Stripe session.
	err = merchSaveSession(r.Context(), ref, session.ID, session.URL)
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Could not save checkout. Please retry this quote."})
		return
	}
	writeJSON(w, 200, map[string]any{"url": session.URL})
}

// Checkout returns to the same allowed site origin that started the payment.
// The receipt is kept in sessionStorage, which is isolated between preview and
// stable staging hosts. Provider webhooks continue to use the stable BaseURL.
func merchCheckoutReturnOrigin(r *http.Request) (string, error) {
	origin := r.Header.Get("Origin")
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid checkout origin")
	}
	return strings.TrimRight(origin, "/"), nil
}

func MerchandiseOrder(w http.ResponseWriter, r *http.Request) {
	if !merchPublicRequest(w, r, http.MethodGet) {
		return
	}
	_, o, err := merchReceiptOrder(r)
	if err != nil {
		writeJSON(w, 404, map[string]any{"error": "Order not found on this browser. Your payment receipt is available by email."})
		return
	}
	writeJSON(w, 200, map[string]any{"status": o.Status, "summary": o.Summary, "totals": map[string]int64{"total": o.Totals.Total}, "reference": o.ID[:12]})
}

// Webhook signatures are checked against the unmodified body before decoding.
func merchSignature(body []byte, secret, signature string) bool {
	if secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	decoded, err := hex.DecodeString(signature)
	return err == nil && hmac.Equal(mac.Sum(nil), decoded)
}
func merchStripeSignature(body []byte, header, secret string, now time.Time) bool {
	var timestamp string
	signatures := []string{}
	for _, part := range strings.Split(header, ",") {
		p := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(p) == 2 {
			if p[0] == "t" {
				timestamp = p[1]
			}
			if p[0] == "v1" {
				signatures = append(signatures, p[1])
			}
		}
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || unix < now.Add(-5*time.Minute).Unix() || unix > now.Add(5*time.Minute).Unix() {
		return false
	}
	signed := append([]byte(timestamp+"."), body...)
	for _, sig := range signatures {
		if merchSignature(signed, secret, sig) {
			return true
		}
	}
	return false
}
func merchWebhookBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"error": "Method Not Allowed"})
		return nil, errors.New("method")
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid body"})
	}
	return body, err
}

type merchPaidSession struct {
	LiveMode      bool   `json:"livemode"`
	ID            string `json:"id"`
	PaymentStatus string `json:"payment_status"`
	Currency      string `json:"currency"`
	AmountTotal   int64  `json:"amount_total"`
	Metadata      struct {
		OrderID string `json:"orderId"`
	} `json:"metadata"`
}

var merchFulfil = fulfilMerchOrder

func MerchandiseStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := merchWebhookBody(w, r)
	if err != nil {
		return
	}
	if !merchStripeSignature(body, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"), time.Now()) {
		writeJSON(w, 400, map[string]any{"error": "Invalid signature"})
		return
	}
	var event struct {
		Type string `json:"type"`
		Data struct {
			Object merchPaidSession `json:"object"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &event) != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid event"})
		return
	}
	if event.Type != "checkout.session.completed" && event.Type != "checkout.session.async_payment_succeeded" {
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if event.Data.Object.PaymentStatus != "paid" {
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if !merchIDPattern.MatchString(event.Data.Object.Metadata.OrderID) {
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if err := merchFulfil(r.Context(), event.Data.Object); err != nil {
		logEvent("merchandise", "error", "fulfilment needs retry or operator review", map[string]any{"orderId": event.Data.Object.Metadata.OrderID})
		writeJSON(w, 503, map[string]any{"error": "Fulfilment pending"})
		return
	}
	writeJSON(w, 200, map[string]any{"received": true})
}
func fulfilMerchOrder(ctx context.Context, session merchPaidSession) error {
	c, err := merchFulfilConfig()
	if err != nil {
		return err
	}
	db, err := merchDB(ctx)
	if err != nil {
		return err
	}
	ref := db.Collection("merchandiseOrders").Doc(session.Metadata.OrderID)
	var order merchOrder
	claimed := false
	err = db.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		claimed = false
		snap, err := tx.Get(ref)
		if status.Code(err) == codes.NotFound {
			return nil
		}
		if err != nil {
			return err
		}
		if err = snap.DataTo(&order); err != nil {
			return err
		}
		next, err := merchPaymentDecision(order, session, c)
		if err != nil || next == "" {
			return err
		}
		claimed = next == "submitting"
		return tx.Update(ref, []firestore.Update{{Path: "status", Value: next}})
	})
	if err != nil {
		return err
	}
	if !claimed {
		if order.PrintifyID != "" {
			return merchProduction(ctx, c, ref, order.PrintifyID)
		}
		return nil
	}
	// Persist the claim BEFORE the external side effect. Never blindly repeat a
	// timed-out Printify order creation: external_id is not a documented idempotency key.
	var result struct {
		ID string `json:"id"`
	}
	err = merchCallPrintify(ctx, c, "POST", "orders.json", map[string]any{"external_id": order.ID, "label": "IPACE-" + order.ID[:12], "line_items": order.Lines, "shipping_method": 1, "send_shipping_notification": true, "address_to": order.Address}, &result)
	if err != nil || result.ID == "" {
		_, _ = ref.Update(ctx, []firestore.Update{{Path: "status", Value: "needs-review"}})
		return errors.New("Printify submission requires review")
	}
	_, err = ref.Update(ctx, []firestore.Update{{Path: "printifyId", Value: result.ID}, {Path: "status", Value: "submitted"}})
	if err != nil {
		return err
	}
	return merchProduction(ctx, c, ref, result.ID)
}
func MerchandisePrintifyWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := merchWebhookBody(w, r)
	if err != nil {
		return
	}
	if !merchSignature(body, os.Getenv("PRINTIFY_WEBHOOK_SECRET"), strings.TrimPrefix(r.Header.Get("X-Pfy-Signature"), "sha256=")) {
		writeJSON(w, 400, map[string]any{"error": "Invalid signature"})
		return
	}
	var event struct {
		Type     string `json:"type"`
		Resource struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				ShopID int64  `json:"shop_id"`
				Action string `json:"action"`
			} `json:"data"`
		} `json:"resource"`
	}
	if json.Unmarshal(body, &event) != nil {
		writeJSON(w, 400, map[string]any{"error": "Invalid event"})
		return
	}
	if strings.HasPrefix(event.Type, "product:") && event.Resource.Type == "product" {
		c, err := merchFulfilConfig()
		if err != nil {
			writeJSON(w, 503, map[string]any{"error": "Unavailable"})
			return
		}
		if !merchProductIDPattern.MatchString(event.Resource.ID) || (event.Resource.Data.ShopID != 0 && strconv.FormatInt(event.Resource.Data.ShopID, 10) != c.ShopID) {
			writeJSON(w, 400, map[string]any{"error": "Invalid product event"})
			return
		}
		merchCache.Lock()
		merchCache.Until = time.Time{}
		merchCache.Unlock()
		if event.Type == "product:publish:started" {
			if err := merchPublishProduct(r.Context(), c, event.Resource.ID, event.Resource.Data.Action); err != nil {
				writeJSON(w, 503, map[string]any{"error": "Product publication pending"})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if !strings.HasPrefix(event.Type, "order:") || event.Resource.Type != "order" || !merchProductIDPattern.MatchString(event.Resource.ID) {
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	c, err := merchFulfilConfig()
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Unavailable"})
		return
	}
	if strconv.FormatInt(event.Resource.Data.ShopID, 10) != c.ShopID {
		writeJSON(w, 400, map[string]any{"error": "Wrong shop"})
		return
	}
	// Fetch current state instead of applying potentially out-of-order event data.
	db, err := merchDB(r.Context())
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Unavailable"})
		return
	}
	iter := db.Collection("merchandiseOrders").Where("printifyId", "==", event.Resource.ID).Limit(1).Documents(r.Context())
	defer iter.Stop()
	snap, err := iter.Next()
	if err == iterator.Done {
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]any{"error": "Unavailable"})
		return
	}
	var state struct {
		Status string `json:"status"`
	}
	if merchCallPrintify(r.Context(), c, "GET", "orders/"+event.Resource.ID+".json", nil, &state) != nil {
		writeJSON(w, 503, map[string]any{"error": "Unavailable"})
		return
	}
	switch state.Status {
	case "fulfilled", "in-production", "canceled", "has-issues", "on-hold", "sending-to-production":
	default:
		writeJSON(w, 200, map[string]any{"received": true})
		return
	}
	if _, err = snap.Ref.Update(r.Context(), []firestore.Update{{Path: "status", Value: state.Status}}); err != nil {
		writeJSON(w, 503, map[string]any{"error": "Unavailable"})
		return
	}
	writeJSON(w, 200, map[string]any{"received": true})
}

var merchMarkup = regexp.MustCompile(`<[^>]*>`)

func merchDescription(raw string) string {
	raw = strings.NewReplacer("</p>", "\n", "<br>", "\n", "<br />", "\n", "</li>", "\n", "</tr>", "\n", "</td>", " | ").Replace(raw)
	return html.UnescapeString(merchMarkup.ReplaceAllString(raw, ""))
}
func merchFulfilConfig() (merchConfig, error) {
	var c merchConfig
	if json.Unmarshal([]byte(os.Getenv("MERCH_CONFIG")), &c) != nil || os.Getenv("PRINTIFY_API_TOKEN") == "" {
		return c, errors.New("fulfilment not configured")
	}
	if _, err := strconv.ParseUint(c.ShopID, 10, 64); err != nil {
		return c, err
	}
	return c, nil
}
func merchPaymentMatches(order merchOrder, session merchPaidSession, shopID string) error {
	if order.ShopID != shopID || session.PaymentStatus != "paid" || session.Currency != "gbp" || session.AmountTotal != order.Totals.Total || session.ID == "" || session.ID != order.SessionID || session.Metadata.OrderID != order.ID {
		return errors.New("payment does not match order")
	}
	return nil
}
func merchProduction(ctx context.Context, c merchConfig, ref *firestore.DocumentRef, id string) error {
	var state struct {
		Status string `json:"status"`
	}
	if err := merchCallPrintify(ctx, c, "GET", "orders/"+id+".json", nil, &state); err != nil {
		return err
	}
	if state.Status == "on-hold" {
		if err := merchCallPrintify(ctx, c, "POST", "orders/"+id+"/send_to_production.json", nil, nil); err != nil {
			return err
		}
		state.Status = "sending-to-production"
	}
	switch state.Status {
	case "sending-to-production", "in-production", "fulfilled", "canceled", "has-issues":
	default:
		return errors.New("unexpected production state")
	}
	_, err := ref.Update(ctx, []firestore.Update{{Path: "status", Value: state.Status}})
	return err
}

func merchShouldClaim(order merchOrder, session merchPaidSession, shopID string) (bool, error) {
	if err := merchPaymentMatches(order, session, shopID); err != nil {
		return false, err
	}
	if order.PrintifyID != "" || order.Status == "test-paid" {
		return false, nil
	}
	if order.Status != "quoted" {
		return false, errors.New("ambiguous submission requires reconciliation")
	}
	return true, nil
}

func merchPublishProduct(ctx context.Context, c merchConfig, id, action string) error {
	if action == "delete" {
		return merchCallPrintify(ctx, c, "POST", "products/"+id+"/unpublish.json", nil, nil)
	}
	if action != "create" && action != "update" {
		return errors.New("unsupported publishing action")
	}
	if !c.Enabled || !merchHTTPS(c.BaseURL) {
		return errors.New("shop not published")
	}
	var p merchProduct
	if err := merchCallPrintify(ctx, c, "GET", "products/"+id+".json", nil, &p); err != nil {
		return err
	}
	if p.ID != id || !p.Visible {
		return errors.New("product is not visible")
	}
	return merchCallPrintify(ctx, c, "POST", "products/"+id+"/publishing_succeeded.json", map[string]any{"external": map[string]string{"id": id, "handle": strings.TrimRight(c.BaseURL, "/") + "/shop/#product-" + id}}, nil)
}

func merchPaymentDecision(order merchOrder, session merchPaidSession, c merchConfig) (string, error) {
	if session.LiveMode != c.StripeLive {
		return "", errors.New("payment mode mismatch")
	}
	claim, err := merchShouldClaim(order, session, c.ShopID)
	if err != nil || !claim {
		return "", err
	}
	if !session.LiveMode {
		return "test-paid", nil
	}
	return "submitting", nil
}
