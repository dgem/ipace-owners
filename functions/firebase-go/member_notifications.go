package ipace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/iterator"
)

var memberNotificationIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{0,119}$`)
var memberNotificationsRequireUser = requireUser
var memberNotificationsLoad = loadMemberNotificationDismissals
var memberNotificationsDismiss = dismissMemberNotification

type memberNotificationDismissal struct {
	ID          string    `json:"id" firestore:"id"`
	DismissedAt time.Time `json:"dismissedAt" firestore:"dismissedAt"`
}

func MemberNotifications(w http.ResponseWriter, r *http.Request) {
	if cors(w, r) || rejectDisallowedOrigin(w, r) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	user, err := memberNotificationsRequireUser(r.Context(), r)
	if err != nil {
		writeMemberAuthorizationError(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		ids, err := memberNotificationsLoad(r.Context(), user.UID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not load notification preferences"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"dismissed": ids})
	case http.MethodPost:
		if rejectMissingOrDisallowedOrigin(w, r) {
			return
		}
		var request struct {
			ID string `json:"id"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := decodeJSON(r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request"})
			return
		}
		request.ID = strings.TrimSpace(request.ID)
		if !memberNotificationIDRE.MatchString(request.ID) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid notification ID"})
			return
		}
		if err := memberNotificationsDismiss(r.Context(), user.UID, request.ID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Could not save notification preference"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"dismissed": request.ID})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "Method Not Allowed"})
	}
}

func loadMemberNotificationDismissals(ctx context.Context, uid string) ([]string, error) {
	db, err := firestoreClient(ctx)
	if err != nil {
		return nil, err
	}
	iter := db.Collection("memberNotificationDismissals").Doc(uid).Collection("items").Limit(200).Documents(ctx)
	defer iter.Stop()
	ids := []string{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var dismissal memberNotificationDismissal
		if doc.DataTo(&dismissal) == nil && dismissal.ID != "" {
			ids = append(ids, dismissal.ID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func dismissMemberNotification(ctx context.Context, uid, id string) error {
	db, err := firestoreClient(ctx)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(id))
	docID := hex.EncodeToString(digest[:])
	_, err = db.Collection("memberNotificationDismissals").Doc(uid).Collection("items").Doc(docID).Set(ctx, memberNotificationDismissal{ID: id, DismissedAt: time.Now().UTC()})
	return err
}
