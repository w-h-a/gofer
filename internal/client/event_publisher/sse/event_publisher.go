package sse

import (
	"context"
	"sync"

	"github.com/google/uuid"
	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/domain"
)

type sseEventPublisher struct {
	options     eventpublisher.Options
	mtx         sync.RWMutex
	subscribers map[uuid.UUID][]chan domain.CapturedRequest
}

func (p *sseEventPublisher) Publish(ctx context.Context, req domain.CapturedRequest) error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	for _, ch := range p.subscribers[req.BinID()] {
		select {
		case ch <- req:
		default:
		}
	}

	return nil
}

func (p *sseEventPublisher) Subscribe(ctx context.Context, binID uuid.UUID) (<-chan domain.CapturedRequest, error) {
	ch := make(chan domain.CapturedRequest, 16)

	p.mtx.Lock()
	p.subscribers[binID] = append(p.subscribers[binID], ch)
	p.mtx.Unlock()

	return ch, nil
}

func (p *sseEventPublisher) Unsubscribe(ctx context.Context, binID uuid.UUID, ch <-chan domain.CapturedRequest) error {
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

	return nil
}

func NewEventPublisher(opts ...eventpublisher.Option) eventpublisher.EventPublisher {
	options := eventpublisher.NewOptions(opts...)

	p := &sseEventPublisher{
		options:     options,
		mtx:         sync.RWMutex{},
		subscribers: map[uuid.UUID][]chan domain.CapturedRequest{},
	}

	return p
}
