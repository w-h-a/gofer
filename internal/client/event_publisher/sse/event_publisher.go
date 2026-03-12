package sse

import (
	"context"
	"sync"

	"github.com/google/uuid"
	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type sseEventPublisher struct {
	options     eventpublisher.Options
	mtx         sync.RWMutex
	subscribers map[uuid.UUID][]chan domain.CapturedRequest
	tracer      trace.Tracer
}

func (p *sseEventPublisher) Publish(ctx context.Context, req domain.CapturedRequest) error {
	ctx, span := p.tracer.Start(ctx, "sseEventPublisher.Publish")
	defer span.End()

	p.mtx.RLock()
	defer p.mtx.RUnlock()

	subs := p.subscribers[req.BinID()]
	notified := 0

	for _, ch := range subs {
		select {
		case ch <- req:
			notified++
		default:
		}
	}

	span.SetAttributes(
		attribute.String("bin.id", req.BinID().String()),
		attribute.Int("subscribers.count", len(subs)),
		attribute.Int("subscribers.notified", notified),
	)

	return nil
}

func (p *sseEventPublisher) Subscribe(ctx context.Context, binID uuid.UUID) (<-chan domain.CapturedRequest, error) {
	_, span := p.tracer.Start(ctx, "sseEventPublisher.Subscribe")
	defer span.End()

	span.SetAttributes(
		attribute.String("bin.id", binID.String()),
	)

	ch := make(chan domain.CapturedRequest, 16)

	p.mtx.Lock()
	p.subscribers[binID] = append(p.subscribers[binID], ch)
	p.mtx.Unlock()

	return ch, nil
}

func (p *sseEventPublisher) Unsubscribe(ctx context.Context, binID uuid.UUID, ch <-chan domain.CapturedRequest) error {
	_, span := p.tracer.Start(ctx, "sseEventPublisher.Unsubscribe")
	defer span.End()

	p.mtx.Lock()
	defer p.mtx.Unlock()

	subs := p.subscribers[binID]

	for i, s := range subs {
		if s == ch {
			p.subscribers[binID] = append(subs[:i], subs[i+1:]...)
			close(s)
			break
		}
	}

	if len(p.subscribers[binID]) == 0 {
		delete(p.subscribers, binID)
	}

	span.SetAttributes(
		attribute.String("bin.id", binID.String()),
		attribute.Int("subscribers.remaining", len(p.subscribers[binID])),
	)

	return nil
}

func NewEventPublisher(opts ...eventpublisher.Option) (eventpublisher.EventPublisher, error) {
	options := eventpublisher.NewOptions(opts...)

	p := &sseEventPublisher{
		options:     options,
		mtx:         sync.RWMutex{},
		subscribers: map[uuid.UUID][]chan domain.CapturedRequest{},
		tracer:      otel.Tracer("github.com/w-h-a/gofer/internal/client/event_publisher/sse"),
	}

	return p, nil
}
