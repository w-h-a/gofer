package eventpublisher

import "context"

type Option func(*Options)

type Options struct {
	Context context.Context
}

func NewOptions(opts ...Option) Options {
	options := Options{
		Context: context.Background(),
	}

	for _, fn := range opts {
		fn(&options)
	}

	return options
}
