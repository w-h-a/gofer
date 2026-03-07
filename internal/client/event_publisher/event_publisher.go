package eventpublisher

import (
	"context"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/domain"
)

type EventPublisher interface {
	Publish(ctx context.Context, req domain.CapturedRequest) error
	Subscribe(ctx context.Context, binID uuid.UUID) (<-chan domain.CapturedRequest, error)
	Unsubscribe(ctx context.Context, binID uuid.UUID, ch <-chan domain.CapturedRequest) error
}
