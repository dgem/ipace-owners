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
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
)

var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var vinRE = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)

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
	return dec.Decode(v)
}

func cors(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" && originAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func originAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	defaults := []string{
		"https://ipace-owners.org",
		"https://www.ipace-owners.org",
		"http://localhost:8080",
		"http://localhost:5000",
		"http://localhost:8888",
	}
	for _, allowed := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if strings.TrimSpace(allowed) == origin {
			return true
		}
	}
	for _, allowed := range defaults {
		if allowed == origin {
			return true
		}
	}
	host := origin
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if strings.HasSuffix(host, ".web.app") || strings.HasSuffix(host, ".firebaseapp.com") {
		return true
	}
	return false
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
		return nil, err
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
		return nil, err
	}
	if user == nil || user.UID == "" {
		return nil, errors.New("sign in required")
	}
	return user, nil
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
