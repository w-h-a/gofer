package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/domain"
)

type Repo interface {
	SaveBin(ctx context.Context, bin domain.Bin) error
	FindBinBySlug(ctx context.Context, slug domain.Slug) (domain.Bin, error)
	DeleteExpiredBin(ctx context.Context, now time.Time) (int, error)
	SaveCapturedRequest(ctx context.Context, req domain.CapturedRequest) error
	FindCapturedRequestByBinID(ctx context.Context, binID uuid.UUID) ([]domain.CapturedRequest, error)
	FindCapturedRequestByID(ctx context.Context, id uuid.UUID) (domain.CapturedRequest, error)
}
