package mock

import (
	"context"

	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
)

type saveBinErrKey struct{}

func WithSaveBinErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, saveBinErrKey{}, err)
	}
}

func SaveBinErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(saveBinErrKey{}).(error)
	return err, ok
}

type findBinResultKey struct{}

func WithFindBinResult(b domain.Bin) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findBinResultKey{}, b)
	}
}

func FindBinResultFrom(ctx context.Context) (domain.Bin, bool) {
	b, ok := ctx.Value(findBinResultKey{}).(domain.Bin)
	return b, ok
}

type findBinErrKey struct{}

func WithFindBinErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findBinErrKey{}, err)
	}
}

func FindBinErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(findBinErrKey{}).(error)
	return err, ok
}

type deleteExpiredResultKey struct{}

func WithDeleteExpiredResult(n int) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, deleteExpiredResultKey{}, n)
	}
}

func DeleteExpiredResultFrom(ctx context.Context) (int, bool) {
	n, ok := ctx.Value(deleteExpiredResultKey{}).(int)
	return n, ok
}

type deleteExpiredErrKey struct{}

func WithDeleteExpiredErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, deleteExpiredErrKey{}, err)
	}
}

func DeleteExpiredErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(deleteExpiredErrKey{}).(error)
	return err, ok
}

type saveCapturedResultKey struct{}

func WithSaveCapturedResult(c domain.CapturedRequest) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, saveCapturedResultKey{}, c)
	}
}

func SaveCapturedResultFrom(ctx context.Context) (domain.CapturedRequest, bool) {
	c, ok := ctx.Value(saveCapturedResultKey{}).(domain.CapturedRequest)
	return c, ok
}

type saveCapturedErrKey struct{}

func WithSaveCapturedErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, saveCapturedErrKey{}, err)
	}
}

func SaveCapturedErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(saveCapturedErrKey{}).(error)
	return err, ok
}

type findByBinIDResultKey struct{}

func WithFindByBinIDResult(cs []domain.CapturedRequest) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findByBinIDResultKey{}, cs)
	}
}

func FindByBinIDResultFrom(ctx context.Context) ([]domain.CapturedRequest, bool) {
	cs, ok := ctx.Value(findByBinIDResultKey{}).([]domain.CapturedRequest)
	return cs, ok
}

type findByBinIDErrKey struct{}

func WithFindByBinIDErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findByBinIDErrKey{}, err)
	}
}

func FindByBinIDErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(findByBinIDErrKey{}).(error)
	return err, ok
}

type findByIDResultKey struct{}

func WithFindByIDResult(c domain.CapturedRequest) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findByIDResultKey{}, c)
	}
}

func FindByIDResultFrom(ctx context.Context) (domain.CapturedRequest, bool) {
	c, ok := ctx.Value(findByIDResultKey{}).(domain.CapturedRequest)
	return c, ok
}

type findByIDErrKey struct{}

func WithFindByIDErr(err error) repo.Option {
	return func(o *repo.Options) {
		o.Context = context.WithValue(o.Context, findByIDErrKey{}, err)
	}
}

func FindByIDErrFrom(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(findByIDErrKey{}).(error)
	return err, ok
}
