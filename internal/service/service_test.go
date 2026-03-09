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
