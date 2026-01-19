package gcs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"promo-collection-worker/internal/pkg/models"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

const testBucketName = "test-bucket"

func newFakeGCS(t *testing.T, handler http.Handler) (*storage.Client, func()) {
	server := httptest.NewServer(handler)

	client, err := storage.NewClient(
		context.Background(),
		option.WithoutAuthentication(),
		option.WithEndpoint(server.URL),
		option.WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("Failed to create fake GCS client: %v", err)
	}

	return client, server.Close
}

func TestNewGCSClientBucketName(t *testing.T) {
	client, err := NewGCSClient(context.Background(), testBucketName)
	assert.NoError(t, err)
	assert.Equal(t, testBucketName, client.(*GCSClient).BucketName)
}

func TestGCSClientCloseNilSafe(t *testing.T) {
	gcsClient := &GCSClient{Client: nil, BucketName: testBucketName}

	assert.NotPanics(t, func() {
		gcsClient.Close(context.Background())
	})
}

func TestUploadSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			w.Header().Set("Location", "/upload-session")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		case "PUT":

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		default:
			t.Fatalf("Unexpected call: %s %s", r.Method, r.URL.Path)
		}
	})

	client, closeServer := newFakeGCS(t, handler)
	defer closeServer()

	gcsClient := &GCSClient{
		Client:     client,
		BucketName: "test-bucket",
	}

	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:      "9999999999",
		PublishTime: time.Now(),
	}

	err := gcsClient.Upload(context.Background(), msg)
	assert.NoError(t, err)
}

func TestUploadPreconditionFailed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusPreconditionFailed)
	})

	client, closeServer := newFakeGCS(t, handler)
	defer closeServer()

	gcsClient := &GCSClient{
		Client:     client,
		BucketName: testBucketName,
	}

	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:      "2222222222",
		PublishTime: time.Now(),
	}

	err := gcsClient.Upload(context.Background(), msg)
	assert.Error(t, err)
}

func TestUploadWriteError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Location", "/upload-session")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})

	client, closeServer := newFakeGCS(t, handler)
	defer closeServer()

	gcsClient := &GCSClient{
		Client:     client,
		BucketName: testBucketName,
	}

	msg := &models.PromoCollectionPublishedMessage{
		Msisdn:      "3333333333",
		PublishTime: time.Now(),
	}

	err := gcsClient.Upload(context.Background(), msg)
	assert.Error(t, err)
}
