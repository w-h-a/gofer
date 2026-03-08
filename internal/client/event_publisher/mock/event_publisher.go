package mock

import (
	"context"

	"github.com/google/uuid"
	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/domain"
)

type mockEventPublisher struct {
	options    eventpublisher.Options
	recorder   *callRecorder
	publishErr error
}

func (p *mockEventPublisher) Publish(_ context.Context, _ domain.CapturedRequest) error {
	p.recorder.record("Publish")
	return p.publishErr
}

func (p *mockEventPublisher) Subscribe(_ context.Context, _ uuid.UUID) (<-chan domain.CapturedRequest, error) {
	return nil, nil
}

func (p *mockEventPublisher) Unsubscribe(_ context.Context, _ uuid.UUID, _ <-chan domain.CapturedRequest) error {
	return nil
}

func (p *mockEventPublisher) Calls() []string {
	return p.recorder.calls
}

func NewEventPublisher(opts ...eventpublisher.Option) *mockEventPublisher {
	options := eventpublisher.NewOptions(opts...)

	p := &mockEventPublisher{
		options:  options,
		recorder: &callRecorder{calls: []string{}},
	}

	if e, ok := PublishErrFrom(options.Context); ok {
		p.publishErr = e
	}

	return p
}
