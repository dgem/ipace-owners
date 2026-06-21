package ipace

import (
	"context"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

var (
	appOnce       sync.Once
	appClient     *firebase.App
	appErr        error
	authOnce      sync.Once
	authClient    *auth.Client
	authErr       error
	firestoreOnce sync.Once
	firestoreDB   *firestore.Client
	firestoreErr  error
	storageOnce   sync.Once
	storageClient *storage.Client
	storageErr    error
)

func projectID() string {
	for _, key := range []string{"GOOGLE_CLOUD_PROJECT", "GCP_PROJECT", "FIREBASE_PROJECT_ID", "PROJECT_ID"} {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func firestoreDatabaseID() string {
	if value := os.Getenv("FIRESTORE_DATABASE_ID"); value != "" {
		return value
	}
	return projectID()
}

func firebaseApp(ctx context.Context) (*firebase.App, error) {
	appOnce.Do(func() {
		appClient, appErr = firebase.NewApp(ctx, nil)
	})
	return appClient, appErr
}

func firebaseAuth(ctx context.Context) (*auth.Client, error) {
	authOnce.Do(func() {
		app, err := firebaseApp(ctx)
		if err != nil {
			authErr = err
			return
		}
		authClient, authErr = app.Auth(ctx)
	})
	return authClient, authErr
}

func firestoreClient(ctx context.Context) (*firestore.Client, error) {
	firestoreOnce.Do(func() {
		firestoreDB, firestoreErr = firestore.NewClientWithDatabase(ctx, projectID(), firestoreDatabaseID())
	})
	return firestoreDB, firestoreErr
}

func gcsClient(ctx context.Context) (*storage.Client, error) {
	storageOnce.Do(func() {
		storageClient, storageErr = storage.NewClient(ctx)
	})
	return storageClient, storageErr
}
