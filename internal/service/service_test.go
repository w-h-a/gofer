package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mockeventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher/mock"
	"github.com/w-h-a/gofer/internal/client/repo"
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

func TestSubscribeToBin_Success(t *testing.T) {
	// Arrange
	bin := activeBin()
	ch := make(chan domain.CapturedRequest)
	r := mockrepo.NewRepo(mockrepo.WithFindBinResult(bin))
	p := mockeventpublisher.NewEventPublisher(mockeventpublisher.WithSubscribeResult(ch))
	svc := NewService(r, p)

	// Act
	out, err := svc.SubscribeToBin(context.Background(), SubscribeToBinInput{
		Slug: bin.Slug().String(),
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, bin.ID(), out.BinID)
	require.Equal(t, (<-chan domain.CapturedRequest)(ch), out.Channel)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{"Subscribe"}, p.Calls())
}

func TestSubscribeToBin_NotFound(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindBinErr(repo.ErrNotFound))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.SubscribeToBin(context.Background(), SubscribeToBinInput{
		Slug: "abcd1234",
	})

	// Assert
	require.ErrorIs(t, err, repo.ErrNotFound)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{}, p.Calls())
}

func TestSubscribeToBin_Expired(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindBinResult(expiredBin()))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.SubscribeToBin(context.Background(), SubscribeToBinInput{
		Slug: "abcd1234",
	})

	// Assert
	require.ErrorIs(t, err, ErrBinExpired)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{}, p.Calls())
}

func TestSubscribeToBin_SubscribeError(t *testing.T) {
	// Arrange
	bin := activeBin()
	r := mockrepo.NewRepo(mockrepo.WithFindBinResult(bin))
	p := mockeventpublisher.NewEventPublisher(
		mockeventpublisher.WithSubscribeErr(errors.New("hub down")),
	)
	svc := NewService(r, p)

	// Act
	_, err := svc.SubscribeToBin(context.Background(), SubscribeToBinInput{
		Slug: bin.Slug().String(),
	})

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to subscribe to bin")
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{"Subscribe"}, p.Calls())
}

func TestUnsubscribeFromBin_Success(t *testing.T) {
	// Arrange
	ch := make(chan domain.CapturedRequest)
	r := mockrepo.NewRepo()
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	err := svc.UnsubscribeFromBin(context.Background(), UnsubscribeFromBinInput{
		BinID:   uuid.New(),
		Channel: ch,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, []string{}, r.Calls())
	require.Equal(t, []string{"Unsubscribe"}, p.Calls())
}

func TestUnsubscribeFromBin_Error(t *testing.T) {
	// Arrange
	ch := make(chan domain.CapturedRequest)
	r := mockrepo.NewRepo()
	p := mockeventpublisher.NewEventPublisher(
		mockeventpublisher.WithUnsubscribeErr(errors.New("hub error")),
	)
	svc := NewService(r, p)

	// Act
	err := svc.UnsubscribeFromBin(context.Background(), UnsubscribeFromBinInput{
		BinID:   uuid.New(),
		Channel: ch,
	})

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unsubscribe from bin")
	require.Equal(t, []string{}, r.Calls())
	require.Equal(t, []string{"Unsubscribe"}, p.Calls())
}

func TestCaptureRequest_OrchestrationOrder(t *testing.T) {
	// Arrange
	bin := activeBin()
	saved := sampleCapturedRequest(bin.ID())
	r := mockrepo.NewRepo(
		mockrepo.WithFindBinResult(bin),
		mockrepo.WithSaveCapturedResult(saved),
	)
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	out, err := svc.CaptureRequest(context.Background(), CaptureRequestInput{
		Slug:   bin.Slug().String(),
		Method: "POST",
		Path:   "/webhook",
		Body:   []byte(`{"key":"val"}`),
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, saved.ID(), out.ID)
	require.Equal(t, []string{"FindBinBySlug", "SaveCapturedRequest"}, r.Calls())
	require.Equal(t, []string{"Publish"}, p.Calls())
}

func TestCaptureRequest_BinNotFound(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindBinErr(repo.ErrNotFound))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.CaptureRequest(context.Background(), CaptureRequestInput{
		Slug:   "abcd1234",
		Method: "POST",
		Path:   "/webhook",
	})

	// Assert
	require.ErrorIs(t, err, repo.ErrNotFound)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{}, p.Calls())
}

func TestCaptureRequest_BinExpired(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindBinResult(expiredBin()))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.CaptureRequest(context.Background(), CaptureRequestInput{
		Slug:   "abcd1234",
		Method: "POST",
		Path:   "/webhook",
	})

	// Assert
	require.ErrorIs(t, err, ErrBinExpired)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
	require.Equal(t, []string{}, p.Calls())
}

func TestCaptureRequest_SaveError_NoPublish(t *testing.T) {
	// Arrange
	bin := activeBin()
	r := mockrepo.NewRepo(
		mockrepo.WithFindBinResult(bin),
		mockrepo.WithSaveCapturedErr(errors.New("write failed")),
	)
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.CaptureRequest(context.Background(), CaptureRequestInput{
		Slug:   bin.Slug().String(),
		Method: "POST",
		Path:   "/webhook",
	})

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "save captured request")
	require.Equal(t, []string{"FindBinBySlug", "SaveCapturedRequest"}, r.Calls())
	require.Equal(t, []string{}, p.Calls())
}

func TestViewBin_Success(t *testing.T) {
	// Arrange
	bin := activeBin()
	reqs := []domain.CapturedRequest{sampleCapturedRequest(bin.ID())}
	r := mockrepo.NewRepo(
		mockrepo.WithFindBinResult(bin),
		mockrepo.WithFindByBinIDResult(reqs),
	)
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	out, err := svc.ViewBin(context.Background(), ViewBinInput{Slug: bin.Slug().String()})

	// Assert
	require.NoError(t, err)
	require.Equal(t, bin.ID(), out.ID)
	require.Equal(t, bin.Slug().String(), out.Slug)
	require.Len(t, out.Requests, 1)
	require.Equal(t, []string{"FindBinBySlug", "FindCapturedRequestByBinID"}, r.Calls())
}

func TestViewBin_Expired(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindBinResult(expiredBin()))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.ViewBin(context.Background(), ViewBinInput{Slug: "abcd1234"})

	// Assert
	require.ErrorIs(t, err, ErrBinExpired)
	require.Equal(t, []string{"FindBinBySlug"}, r.Calls())
}

func TestViewBin_FindRequestsError(t *testing.T) {
	// Arrange
	bin := activeBin()
	r := mockrepo.NewRepo(
		mockrepo.WithFindBinResult(bin),
		mockrepo.WithFindByBinIDErr(errors.New("db read failed")),
	)
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.ViewBin(context.Background(), ViewBinInput{Slug: bin.Slug().String()})

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "find captured requests")
	require.Equal(t, []string{"FindBinBySlug", "FindCapturedRequestByBinID"}, r.Calls())
}

func TestViewCapturedRequest_Success(t *testing.T) {
	// Arrange
	req := sampleCapturedRequest(uuid.New())
	r := mockrepo.NewRepo(mockrepo.WithFindByIDResult(req))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	out, err := svc.ViewCapturedRequest(context.Background(), ViewCapturedRequestInput{ID: req.ID().String()})

	// Assert
	require.NoError(t, err)
	require.Equal(t, req.ID(), out.ID)
	require.Equal(t, req.Method(), out.Method)
	require.Equal(t, []string{"FindCapturedRequestByID"}, r.Calls())
}

func TestViewCapturedRequest_NotFound(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithFindByIDErr(repo.ErrNotFound))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.ViewCapturedRequest(context.Background(), ViewCapturedRequestInput{ID: uuid.New().String()})

	// Assert
	require.ErrorIs(t, err, repo.ErrNotFound)
	require.Equal(t, []string{"FindCapturedRequestByID"}, r.Calls())
}

func TestViewCapturedRequest_InvalidID(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo()
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.ViewCapturedRequest(context.Background(), ViewCapturedRequestInput{ID: "not-a-uuid"})

	// Assert
	require.ErrorIs(t, err, domain.ErrInvalidID)
	require.Contains(t, err.Error(), "failed to parse request id")
	require.Equal(t, []string{}, r.Calls())
}

func TestCleanupExpiredBins_Success(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithDeleteExpiredResult(3))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	out, err := svc.CleanupExpiredBins(context.Background())

	// Assert
	require.NoError(t, err)
	require.Equal(t, 3, out.Deleted)
	require.Equal(t, []string{"DeleteExpiredBin"}, r.Calls())
}

func TestCleanupExpiredBins_RepoError(t *testing.T) {
	// Arrange
	r := mockrepo.NewRepo(mockrepo.WithDeleteExpiredErr(errors.New("db down")))
	p := mockeventpublisher.NewEventPublisher()
	svc := NewService(r, p)

	// Act
	_, err := svc.CleanupExpiredBins(context.Background())

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to cleanup expired bins")
	require.Equal(t, []string{"DeleteExpiredBin"}, r.Calls())
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
