package ipace

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
)

var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var vinRE = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
var authTraceRE = regexp.MustCompile(`^IP-[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}$`)

type authTraceContextKey struct{}

type authorizationError struct {
	status  int
	message string
	cause   error
}

func (err *authorizationError) Error() string {
	return err.message
}

func (err *authorizationError) Unwrap() error {
	return err.cause
}

func authorizationFailure(status int, message string, cause error) error {
	return &authorizationError{status: status, message: message, cause: cause}
}

func writeAdminAuthorizationError(w http.ResponseWriter, err error) {
	status := http.StatusForbidden
	message := "Admin role required"
	var authorizationErr *authorizationError
	if errors.As(err, &authorizationErr) {
		status = authorizationErr.status
		message = authorizationErr.message
	}
	writeJSON(w, status, map[string]any{"error": message})
}

func writeMemberAuthorizationError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	message := "Sign in required"
	var authorizationErr *authorizationError
	if errors.As(err, &authorizationErr) {
		status = authorizationErr.status
		message = authorizationErr.message
	}
	writeJSON(w, status, map[string]any{"error": message})
}

type authServiceUnavailableError struct {
	cause error
}

func (err *authServiceUnavailableError) Error() string { return "sign-in verification unavailable" }
func (err *authServiceUnavailableError) Unwrap() error { return err.cause }

func authTraceCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !authTraceRE.MatchString(value) {
		return ""
	}
	return value
}

func authTraceFields(r *http.Request) map[string]any {
	traceCode := authTraceCode(r.Header.Get("X-Ipace-Auth-Trace"))
	if traceCode == "" {
		return map[string]any{}
	}
	return map[string]any{"authTrace": traceCode}
}

func addAuthTrace(fields map[string]any, r *http.Request) map[string]any {
	for key, value := range authTraceFields(r) {
		fields[key] = value
	}
	return fields
}

func contextWithAuthTrace(ctx context.Context, r *http.Request) context.Context {
	traceCode := authTraceCode(r.Header.Get("X-Ipace-Auth-Trace"))
	if traceCode == "" {
		return ctx
	}
	return context.WithValue(ctx, authTraceContextKey{}, traceCode)
}

func authTraceFromContext(ctx context.Context) string {
	traceCode, _ := ctx.Value(authTraceContextKey{}).(string)
	return authTraceCode(traceCode)
}

func appendAuthTraceToContinueURL(continueURL string, traceCode string) string {
	traceCode = authTraceCode(traceCode)
	if traceCode == "" {
		return continueURL
	}
	parsed, err := url.Parse(continueURL)
	if err != nil {
		return continueURL
	}
	query := parsed.Query()
	query.Set("authTrace", traceCode)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("json encode failed: %v", err)
	}
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("missing body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func cors(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" && originAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Ipace-Auth-Trace")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func rejectDisallowedOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || originAllowed(origin) {
		return false
	}
	logEvent("api", "warn", "request rejected: disallowed origin", map[string]any{
		"origin": origin,
		"path":   r.URL.Path,
		"method": r.Method,
	})
	writeJSON(w, http.StatusForbidden, map[string]any{"error": "Origin not allowed"})
	return true
}

func rejectMissingOrDisallowedOrigin(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Origin") == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "Origin required"})
		return true
	}
	return rejectDisallowedOrigin(w, r)
}

func originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	if strings.Contains(origin, ",") || strings.ContainsAny(origin, " \t\r\n") {
		return false
	}
	if _, err := url.ParseRequestURI(origin); err != nil {
		return false
	}
	for _, allowed := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	if originDefaultAllowed(origin) {
		return true
	}
	return firebasePreviewOriginAllowed(origin)
}

func originDefaultAllowed(origin string) bool {
	defaults := []string{
		"https://ipace-owners.org",
		"https://www.ipace-owners.org",
		"http://localhost:8080",
		"http://localhost:5000",
		"http://localhost:8888",
	}
	for _, allowed := range defaults {
		if allowed == origin {
			return true
		}
	}
	return false
}

func firebasePreviewOriginAllowed(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "https" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	host := parsed.Hostname()
	project := projectID()
	if project == "" {
		return false
	}
	return (strings.HasPrefix(host, project+"--pr-") && strings.HasSuffix(host, ".web.app")) ||
		(strings.HasPrefix(host, project+"--pr-") && strings.HasSuffix(host, ".firebaseapp.com"))
}

func emailContinueURLForOrigin(origin string) string {
	if originAllowed(origin) {
		parsed, err := url.Parse(origin)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			parsed.Path = "/member/account/"
			parsed.RawQuery = ""
			parsed.Fragment = ""
			return parsed.String()
		}
	}
	if value := os.Getenv("FIREBASE_EMAIL_CONTINUE_URL"); value != "" {
		return value
	}
	return "https://ipace-owners.org/member/account/"
}

func firebaseEmailLinkDomainForContinueURL(continueURL string) string {
	parsed, err := url.Parse(continueURL)
	if err != nil || parsed.Scheme != "https" {
		return ""
	}
	host := parsed.Hostname()
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".web.app") ||
		strings.HasSuffix(host, ".firebaseapp.com") {
		return ""
	}
	return host
}

func cleanString(value string, max int) string {
	value = strings.TrimSpace(value)
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}

func cleanEmail(value string) string {
	return strings.ToLower(cleanString(value, 254))
}

func isEmail(value string) bool {
	return emailRE.MatchString(value)
}

func cleanEnum(value string, allowed []string) string {
	value = cleanString(value, 100)
	for _, item := range allowed {
		if value == item {
			return value
		}
	}
	return ""
}

func cleanEnums(values []string, allowed []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if cleaned := cleanEnum(value, allowed); cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}

func cleanDate(value string) string {
	value = cleanString(value, 20)
	if value == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return ""
	}
	return value
}

func cleanInt(value string, min int, max int) *int {
	value = cleanString(value, 20)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		return nil
	}
	return &parsed
}

func cleanDecimal(value string, min float64, max float64) *float64 {
	value = cleanString(value, 20)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < min || parsed > max {
		return nil
	}
	rounded := float64(int(parsed*10+0.5)) / 10
	return &rounded
}

func emailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(email)))
	return hex.EncodeToString(sum[:])[:16]
}

func maskedEmail(email string) string {
	email = cleanEmail(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	local := parts[0]
	domain := parts[1]
	localPrefix := local[:1]
	domainParts := strings.Split(domain, ".")
	domainPrefix := domain[:1]
	tld := ""
	if len(domainParts) > 1 {
		tld = domainParts[len(domainParts)-1]
	}
	if tld != "" {
		return localPrefix + "***@" + domainPrefix + "***." + tld
	}
	return localPrefix + "***@" + domainPrefix + "***"
}

func emailLogFields(email string) map[string]any {
	return map[string]any{
		"emailHash":   emailFingerprint(email),
		"emailMasked": maskedEmail(email),
	}
}

func urlHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func hmacValue(value string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func submissionID(prefix string) string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

type firebaseUser struct {
	UID    string
	Email  string
	Claims map[string]any
}

func optionalUser(ctx context.Context, r *http.Request) (*firebaseUser, error) {
	token := bearerToken(r)
	if token == "" {
		return nil, nil
	}
	client, err := firebaseAuth(ctx)
	if err != nil {
		return nil, &authServiceUnavailableError{cause: err}
	}
	verified, err := client.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return userFromToken(ctx, client, verified), nil
}

func requireUser(ctx context.Context, r *http.Request) (*firebaseUser, error) {
	user, err := optionalUser(ctx, r)
	if err != nil {
		var unavailable *authServiceUnavailableError
		if errors.As(err, &unavailable) {
			logAuthorizationDecision(r, "member", "verification-unavailable", http.StatusServiceUnavailable)
			return nil, authorizationFailure(http.StatusServiceUnavailable, "Sign-in verification is temporarily unavailable", err)
		}
		logAuthorizationDecision(r, "member", "invalid-token", http.StatusUnauthorized)
		return nil, authorizationFailure(http.StatusUnauthorized, "Sign in required", err)
	}
	if user == nil || user.UID == "" {
		logAuthorizationDecision(r, "member", "missing-token", http.StatusUnauthorized)
		return nil, authorizationFailure(http.StatusUnauthorized, "Sign in required", nil)
	}
	logAuthorizationDecision(r, "member", "allowed", http.StatusOK)
	return user, nil
}

func requireAdmin(ctx context.Context, r *http.Request) (*firebaseUser, error) {
	user, err := requireUser(ctx, r)
	if err != nil {
		return nil, err
	}
	if !isAdmin(user) {
		logAuthorizationDecision(r, "admin", "admin-claim-missing", http.StatusForbidden)
		return nil, authorizationFailure(http.StatusForbidden, "Admin role required", nil)
	}
	logAuthorizationDecision(r, "admin", "allowed", http.StatusOK)
	return user, nil
}

func logAuthorizationDecision(r *http.Request, requiredRole string, decision string, status int) {
	if authTraceCode(r.Header.Get("X-Ipace-Auth-Trace")) == "" {
		return
	}
	fields := addAuthTrace(map[string]any{
		"route":        authorizationRoute(r.URL.Path),
		"requiredRole": requiredRole,
		"decision":     decision,
		"status":       status,
	}, r)
	logEvent("authorization", "info", "authorization decision", fields)
}

func authorizationRoute(path string) string {
	route := strings.TrimRight(path, "/")
	if route == "" {
		return "/"
	}
	return route
}

func userFromToken(ctx context.Context, client *auth.Client, token *auth.Token) *firebaseUser {
	user := &firebaseUser{UID: token.UID, Claims: token.Claims}
	if email, ok := token.Claims["email"].(string); ok {
		user.Email = email
		return user
	}
	if record, err := client.GetUser(ctx, token.UID); err == nil {
		user.Email = record.Email
	}
	return user
}

func isAdmin(user *firebaseUser) bool {
	if user == nil {
		return false
	}
	if admin, ok := user.Claims["admin"].(bool); ok && admin {
		return true
	}
	if roles, ok := user.Claims["roles"].([]any); ok {
		for _, role := range roles {
			if role == "admin" {
				return true
			}
		}
	}
	return false
}

func logEvent(functionName string, level string, message string, fields map[string]any) {
	payload := map[string]any{
		"function": functionName,
		"level":    level,
		"message":  message,
	}
	for key, value := range fields {
		payload[key] = value
	}
	encoded, _ := json.Marshal(payload)
	log.Print(string(encoded))
}
