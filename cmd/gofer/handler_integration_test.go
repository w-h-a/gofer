package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/w-h-a/gofer/internal/client/event_publisher/sse"
	"github.com/w-h-a/gofer/internal/client/repo"
	mockrepo "github.com/w-h-a/gofer/internal/client/repo/mock"
	"github.com/w-h-a/gofer/internal/client/repo/sqlite"
	"github.com/w-h-a/gofer/internal/service"
)

func TestHealthz(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/healthz")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

func TestCreateBin_Success(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Post(
		ts.URL+"/api/bins",
		"application/json",
		strings.NewReader(`{"ttl":"1h"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusCreated, rsp.StatusCode)
	require.Equal(t, "application/json", rsp.Header.Get("Content-Type"))

	var body createBinResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))

	require.NotEmpty(t, body.ID)
	require.Len(t, body.Slug, 8)
	require.NotEmpty(t, body.CreatedAt)
	require.NotEmpty(t, body.ExpiresAt)
}

func TestCreateBin_InternalError(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	r := mockrepo.NewRepo(
		mockrepo.WithSaveBinErr(errors.New("db down")),
	)
	pub, err := sse.NewEventPublisher()
	require.NoError(t, err)
	svc := service.NewService(r, pub)
	h := newHandler(svc)

	ts := httptest.NewServer(h.routes())
	t.Cleanup(ts.Close)

	// Act
	rsp, err := ts.Client().Post(
		ts.URL+"/api/bins",
		"application/json",
		strings.NewReader(`{"ttl":"1h"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusInternalServerError, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "internal error", body.Error)
}

func TestCaptureRequest_UnknownSlug(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Post(
		ts.URL+"/gofer/zzzzzzzz/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "bin not found", body.Error)
}

func TestCaptureRequest_ExpiredBin(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Create a bin with minimal TTL
	rsp, err := ts.Client().Post(
		ts.URL+"/api/bins",
		"application/json",
		strings.NewReader(`{"ttl":"1ms"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusCreated, rsp.StatusCode)

	var bin createBinResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&bin))

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	// Act
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusGone, rsp2.StatusCode)

	var errBody errorResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&errBody))
	require.Equal(t, "bin is expired", errBody.Error)
}

func TestCaptureRequest_OversizedBody(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	rsp, err := ts.Client().Post(
		ts.URL+"/api/bins",
		"application/json",
		strings.NewReader(`{"ttl":"1h"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusCreated, rsp.StatusCode)

	var bin createBinResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&bin))

	oversized := strings.NewReader(strings.Repeat("x", maxBodySize+1))

	// Act
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/octet-stream",
		oversized,
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusRequestEntityTooLarge, rsp2.StatusCode)

	var errBody errorResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&errBody))
	require.Equal(t, "request body too large", errBody.Error)
}

func TestCaptureRequest_Success(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	rsp, err := ts.Client().Post(
		ts.URL+"/api/bins",
		"application/json",
		strings.NewReader(`{"ttl":"1h"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusCreated, rsp.StatusCode)

	var bin createBinResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&bin))

	// Act
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, rsp2.StatusCode)
	require.Equal(t, "application/json", rsp2.Header.Get("Content-Type"))

	var captured captureRequestResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&captured))

	require.NotEmpty(t, captured.ID)
	require.Equal(t, bin.ID, captured.BinID)
	require.Equal(t, 1, captured.SequenceNum)
	require.Equal(t, "POST", captured.Method)
	require.Equal(t, "/webhook", captured.Path)
	require.Equal(t, "application/json", captured.ContentType)
	require.Equal(t, len(`{"key":"val"}`), captured.BodySize)
	require.NotEmpty(t, captured.CapturedAt)
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	r, err := sqlite.NewRepo(repo.WithLocation(":memory:"))
	require.NoError(t, err)

	p, err := sse.NewEventPublisher()
	require.NoError(t, err)

	svc := service.NewService(r, p)
	h := newHandler(svc)

	ts := httptest.NewServer(h.routes())
	t.Cleanup(ts.Close)

	return ts
}
