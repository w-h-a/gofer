package mock

import (
	"context"

	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/domain"
)

type publishErrKey struct{}

func WithPublishErr(err error) eventpublisher.Option {
	return func(o *eventpublisher.Options) {
		o.Context = context.WithValue(o.Context, publishErrKey{}, err)
	}
}

func PublishErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(publishErrKey{}).(error)
	return err, ok
}

type subscribeResultKey struct{}

func WithSubscribeResult(ch <-chan domain.CapturedRequest) eventpublisher.Option {
	return func(o *eventpublisher.Options) {
		o.Context = context.WithValue(o.Context, subscribeResultKey{}, ch)
	}
}

func SubscribeResultFrom(ctx context.Context) (<-chan domain.CapturedRequest, bool) {
	ch, ok := ctx.Value(subscribeResultKey{}).(<-chan domain.CapturedRequest)
	return ch, ok
}

type subscribeErrKey struct{}

func WithSubscribeErr(err error) eventpublisher.Option {
	return func(o *eventpublisher.Options) {
		o.Context = context.WithValue(o.Context, subscribeErrKey{}, err)
	}
}

func SubscribeErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(subscribeErrKey{}).(error)
	return err, ok
}

type unsubscribeErrKey struct{}

func WithUnsubscribeErr(err error) eventpublisher.Option {
	return func(o *eventpublisher.Options) {
		o.Context = context.WithValue(o.Context, unsubscribeErrKey{}, err)
	}
}

func UnsubscribeErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(unsubscribeErrKey{}).(error)
	return err, ok
}
