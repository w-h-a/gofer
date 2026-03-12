package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
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

func TestHome_OK(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, rsp.StatusCode)
	require.Equal(t, "text/html; charset=utf-8", rsp.Header.Get("Content-Type"))

	body, err := io.ReadAll(rsp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "Gofer")
	require.Contains(t, string(body), "Create a Bin")
}

func TestInspectBin_OK(t *testing.T) {
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
	rsp2, err := ts.Client().Get(ts.URL + "/bins/" + bin.Slug)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, rsp2.StatusCode)
	require.Equal(t, "text/html; charset=utf-8", rsp2.Header.Get("Content-Type"))

	body, err := io.ReadAll(rsp2.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), bin.Slug)
	require.Contains(t, string(body), "/gofer/"+bin.Slug)
}

func TestInspectBin_NotFound(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/bins/zzzzzzzz")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)
}

func TestInspectBin_InvalidSlug(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/bins/x")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusBadRequest, rsp.StatusCode)
}

func TestInspectBin_Expired(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

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

	time.Sleep(5 * time.Millisecond)

	// Act
	rsp2, err := ts.Client().Get(ts.URL + "/bins/" + bin.Slug)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusGone, rsp2.StatusCode)
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
	h := newHandler(svc, 48*time.Hour, "test")

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

func TestViewBin_NotFound(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/api/bins/zzzzzzzz")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "bin not found", body.Error)
}

func TestViewBin_Expired(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

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
	rsp2, err := ts.Client().Get(ts.URL + "/api/bins/" + bin.Slug)
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusGone, rsp2.StatusCode)

	var errBody errorResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&errBody))
	require.Equal(t, "bin is expired", errBody.Error)
}

func TestViewBin_OK(t *testing.T) {
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

	// Capture a request so the list isn't empty
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()
	require.Equal(t, http.StatusCreated, rsp2.StatusCode)

	// Act
	rsp3, err := ts.Client().Get(ts.URL + "/api/bins/" + bin.Slug)
	require.NoError(t, err)
	defer rsp3.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, rsp3.StatusCode)
	require.Equal(t, "application/json", rsp3.Header.Get("Content-Type"))

	var viewed viewBinResponse
	require.NoError(t, json.NewDecoder(rsp3.Body).Decode(&viewed))
	require.Equal(t, bin.ID, viewed.ID)
	require.Equal(t, bin.Slug, viewed.Slug)
	require.Equal(t, bin.CreatedAt, viewed.CreatedAt)
	require.Equal(t, bin.ExpiresAt, viewed.ExpiresAt)
	require.Len(t, viewed.Requests, 1)
	require.Equal(t, 1, viewed.Requests[0].SequenceNum)
	require.Equal(t, "POST", viewed.Requests[0].Method)
	require.Equal(t, "/webhook", viewed.Requests[0].Path)
}

func TestViewBin_InvalidSlug(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/api/bins/x")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusBadRequest, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "invalid slug", body.Error)
}

func TestViewCapturedRequest_Success(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION to run")
	}

	// Arrange
	ts := newTestServer(t)

	resp, err := http.Post(ts.URL+"/api/bins", "application/json", strings.NewReader(`{"ttl":"1h"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var bin createBinResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&bin))

	body := `{"hello":"world"}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/gofer/"+bin.Slug+"/webhook", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom", "test-value")

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var captured captureRequestResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&captured))

	// Act
	resp, err = http.Get(ts.URL + "/api/requests/" + captured.ID)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result viewCapturedRequestResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, captured.ID, result.ID)
	require.Equal(t, captured.BinID, result.BinID)
	require.Equal(t, 1, result.SequenceNum)
	require.Equal(t, "POST", result.Method)
	require.Equal(t, "/webhook", result.Path)
	require.Equal(t, "application/json", result.ContentType)
	require.Equal(t, len(body), result.BodySize)
	require.Equal(t, body, result.Body)
	require.Contains(t, result.Headers["X-Custom"], "test-value")
}

func TestViewCapturedRequest_InvalidID(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION to run")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	resp, err := http.Get(ts.URL + "/api/requests/not-a-uuid")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	require.Equal(t, "invalid request id", errResp.Error)
}

func TestViewCapturedRequest_NotFound(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("set INTEGRATION to run")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	resp, err := http.Get(ts.URL + "/api/requests/00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var errResp errorResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errResp))
	require.Equal(t, "request not found", errResp.Error)
}

func TestSubscribeToBin_UnknownSlug(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Get(ts.URL + "/api/bins/zzzzzzzz/sse")
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusNotFound, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "bin not found", body.Error)
}

func TestSubscribeToBin_ExpiredBin(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

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
	rsp2, err := ts.Client().Get(ts.URL + "/api/bins/" + bin.Slug + "/sse")
	require.NoError(t, err)
	defer rsp2.Body.Close()

	// Assert
	require.Equal(t, http.StatusGone, rsp2.StatusCode)

	var errBody errorResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&errBody))
	require.Equal(t, "bin is expired", errBody.Error)
}

func TestSubscribeToBin_FanOut(t *testing.T) {
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

	// Connect SSE client
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/bins/"+bin.Slug+"/sse", nil)
	require.NoError(t, err)

	sseRsp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer sseRsp.Body.Close()
	require.Equal(t, http.StatusOK, sseRsp.StatusCode)
	require.Equal(t, "text/event-stream", sseRsp.Header.Get("Content-Type"))

	// Read SSE events in background
	eventCh := make(chan captureRequestResponse, 1)
	go func() {
		scanner := bufio.NewScanner(sseRsp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var event captureRequestResponse
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
				continue
			}
			eventCh <- event
			return
		}
	}()

	// Give SSE connection time to establish
	time.Sleep(50 * time.Millisecond)

	// Act — capture a request to trigger fan-out
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()
	require.Equal(t, http.StatusCreated, rsp2.StatusCode)

	var captured captureRequestResponse
	require.NoError(t, json.NewDecoder(rsp2.Body).Decode(&captured))

	// Assert — SSE client receives the event
	select {
	case event := <-eventCh:
		require.Equal(t, captured.ID, event.ID)
		require.Equal(t, captured.BinID, event.BinID)
		require.Equal(t, captured.SequenceNum, event.SequenceNum)
		require.Equal(t, captured.Method, event.Method)
		require.Equal(t, captured.Path, event.Path)
		require.Equal(t, captured.ContentType, event.ContentType)
		require.Equal(t, captured.BodySize, event.BodySize)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
	}
}

func TestSubscribeToBin_CleanupOnDisconnect(t *testing.T) {
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

	// Connect SSE client
	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/bins/"+bin.Slug+"/sse", nil)
	require.NoError(t, err)

	sseRsp, err := ts.Client().Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, sseRsp.StatusCode)

	// Act — disconnect the SSE client
	cancel()
	sseRsp.Body.Close()

	// Give the handler time to run the deferred unsubscribe
	time.Sleep(50 * time.Millisecond)

	// Assert — capturing a request still works (no panic from publishing to a closed channel)
	rsp2, err := ts.Client().Post(
		ts.URL+"/gofer/"+bin.Slug+"/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp2.Body.Close()
	require.Equal(t, http.StatusCreated, rsp2.StatusCode)
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
	require.Equal(t, http.StatusCreated, rsp2.StatusCode)
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

func TestCaptureRequest_InvalidSlug(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("skipping integration test")
	}

	// Arrange
	ts := newTestServer(t)

	// Act
	rsp, err := ts.Client().Post(
		ts.URL+"/gofer/x/webhook",
		"application/json",
		strings.NewReader(`{"key":"val"}`),
	)
	require.NoError(t, err)
	defer rsp.Body.Close()

	// Assert
	require.Equal(t, http.StatusBadRequest, rsp.StatusCode)

	var body errorResponse
	require.NoError(t, json.NewDecoder(rsp.Body).Decode(&body))
	require.Equal(t, "invalid slug", body.Error)
}

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

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	r, err := sqlite.NewRepo(repo.WithLocation(":memory:"))
	require.NoError(t, err)

	p, err := sse.NewEventPublisher()
	require.NoError(t, err)

	svc := service.NewService(r, p)
	h := newHandler(svc, 48*time.Hour, "test")

	ts := httptest.NewServer(h.routes())
	t.Cleanup(ts.Close)

	return ts
}
