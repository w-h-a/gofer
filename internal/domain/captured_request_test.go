package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/w-h-a/gofer/internal/domain"
)

func TestRawPayload_ImmutableFromSource(t *testing.T) {
	// Arrange
	source := []byte("original payload")

	payload := domain.NewRawPayload(source)

	// Act: Mutate source after construction
	source[0] = 'X'

	// Assert
	require.Equal(t, []byte("original payload"), payload.Bytes())
}

func TestRawPayload_ImmutableFromBytes(t *testing.T) {
	// Arrange
	payload := domain.NewRawPayload([]byte("original payload"))

	// Act
	out := payload.Bytes()
	out[0] = 'X'

	// Assert
	require.Equal(t, []byte("original payload"), payload.Bytes())
}

func TestRawPayload_NilInput(t *testing.T) {
	// Act
	payload := domain.NewRawPayload(nil)

	// Assert
	require.Nil(t, payload.Bytes())
	require.Equal(t, 0, payload.Size())
}

func TestRawPayload_Size(t *testing.T) {
	// Act
	payload := domain.NewRawPayload([]byte("hello"))

	// Assert
	require.Equal(t, 5, payload.Size())
}

func TestNewCapturedRequest_Success(t *testing.T) {
	// Arrange
	binID := uuid.New()
	headers := map[string][]string{"Content-Type": {"application/json"}}
	query := map[string][]string{"foo": {"bar"}}
	body := domain.NewRawPayload([]byte(`{"event":"charge.created"}`))

	// Act
	req, err := domain.NewCapturedRequest(
		binID, 1, "POST", "/webhook",
		headers, query,
		"application/json", "192.168.1.1",
		body,
	)

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, req.ID())
	require.Equal(t, binID, req.BinID())
	require.Equal(t, 1, req.SequenceNum())
	require.Equal(t, "POST", req.Method())
	require.Equal(t, "/webhook", req.Path())
	require.Equal(t, "application/json", req.ContentType())
	require.Equal(t, "192.168.1.1", req.RemoteAddr())
	require.Equal(t, 26, req.BodySize())
	require.False(t, req.CapturedAt().IsZero())
}

func TestNewCapturedRequest_RequiresBinID(t *testing.T) {
	// Act
	_, err := domain.NewCapturedRequest(
		uuid.Nil, 1, "POST", "/webhook",
		nil, nil,
		"", "",
		domain.NewRawPayload(nil),
	)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "bin ID is required")
}

func TestNewCapturedRequest_RequiresPositiveSequenceNum(t *testing.T) {
	// Act
	_, err := domain.NewCapturedRequest(
		uuid.New(), 0, "POST", "/webhook",
		nil, nil,
		"", "",
		domain.NewRawPayload(nil),
	)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequence number must be positive")
}

func TestNewCapturedRequest_RequiresMethod(t *testing.T) {
	// Act
	_, err := domain.NewCapturedRequest(
		uuid.New(), 1, "", "/webhook",
		nil, nil,
		"", "",
		domain.NewRawPayload(nil),
	)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "method is required")
}

func TestCapturedRequest_HeadersDefensivelyCopied(t *testing.T) {
	// Arrange
	headers := map[string][]string{"X-Test": {"value"}}

	req, err := domain.NewCapturedRequest(
		uuid.New(), 1, "GET", "/",
		headers, nil,
		"", "",
		domain.NewRawPayload(nil),
	)
	require.NoError(t, err)

	// Act: Mutate original map after construction
	headers["X-Test"][0] = "mutated"

	// Assert
	require.Equal(t, []string{"value"}, req.Headers()["X-Test"])
}

func TestCapturedRequest_HeadersGetterReturnsCopy(t *testing.T) {
	// Arrange
	req, err := domain.NewCapturedRequest(
		uuid.New(), 1, "GET", "/",
		map[string][]string{"X-Test": {"value"}}, nil,
		"", "",
		domain.NewRawPayload(nil),
	)
	require.NoError(t, err)

	// Act: Mutate the returned map
	out := req.Headers()
	out["X-Test"][0] = "mutated"

	// Assert
	require.Equal(t, []string{"value"}, req.Headers()["X-Test"])
}
