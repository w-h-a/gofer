package mock

import (
	"context"

	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
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
