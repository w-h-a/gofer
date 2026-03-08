package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
