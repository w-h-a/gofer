package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mockeventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher/mock"
	mockrepo "github.com/w-h-a/gofer/internal/client/repo/mock"
	"github.com/w-h-a/gofer/internal/domain"
)

func TestCreateBin_Success(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo()
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	out, err := svc.CreateBin(context.Background(), CreateBinInput{TTL: 48 * time.Hour})

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, out.ID)
	require.Len(t, out.Slug, 8)
	require.False(t, out.ExpiresAt.Before(out.CreatedAt))
	require.Equal(t, []string{"SaveBin"}, r.Calls())
}

func TestCreateBin_RepoError(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithSaveBinErr(errors.New("db down")))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.CreateBin(context.Background(), CreateBinInput{TTL: 48 * time.Hour})

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "save bin")
}

func activeBin() domain.Bin {
	slug, _ := domain.ParseSlug("abcd1234")
	return domain.RehydrateBin(
		uuid.New(),
		slug,
		time.Now(),
		time.Now().Add(1*time.Hour),
	)
}

func expiredBin() domain.Bin {
	slug, _ := domain.ParseSlug("abcd1234")
	return domain.RehydrateBin(
		uuid.New(),
		slug,
		time.Now().Add(-1*time.Hour),
		time.Now().Add(-1*time.Hour),
	)
}

func sampleCapturedRequest(binID uuid.UUID) domain.CapturedRequest {
	return domain.RehydrateCapturedRequest(
		uuid.New(), binID, 1,
		"POST", "/webhook",
		map[string][]string{"Content-Type": {"application/json"}},
		nil,
		13, "application/json", "127.0.0.1",
		time.Now(),
		domain.NewRawPayload([]byte(`{"key":"val"}`)),
	)
}
