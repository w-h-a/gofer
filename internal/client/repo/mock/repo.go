package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
)

type mockRepo struct {
	options             repo.Options
	recorder            *callRecorder
	saveBinErr          error
	findBinResult       domain.Bin
	findBinErr          error
	deleteExpiredResult int
	deleteExpiredErr    error
	saveCapturedResult  domain.CapturedRequest
	saveCapturedErr     error
	findByBinIDResult   []domain.CapturedRequest
	findByBinIDErr      error
	findByIDResult      domain.CapturedRequest
	findByIDErr         error
}

func (r *mockRepo) SaveBin(_ context.Context, _ domain.Bin) error {
	r.recorder.record("SaveBin")
	return r.saveBinErr
}

func (r *mockRepo) FindBinBySlug(_ context.Context, _ domain.Slug) (domain.Bin, error) {
	r.recorder.record("FindBinBySlug")
	return r.findBinResult, r.findBinErr
}

func (r *mockRepo) DeleteExpiredBin(_ context.Context, _ time.Time) (int, error) {
	r.recorder.record("DeleteExpiredBin")
	return r.deleteExpiredResult, r.deleteExpiredErr
}

func (r *mockRepo) SaveCapturedRequest(_ context.Context, _ domain.CapturedRequest) (domain.CapturedRequest, error) {
	r.recorder.record("SaveCapturedRequest")
	return r.saveCapturedResult, r.saveCapturedErr
}

func (r *mockRepo) FindCapturedRequestByBinID(_ context.Context, _ uuid.UUID) ([]domain.CapturedRequest, error) {
	r.recorder.record("FindCapturedRequestByBinID")
	return r.findByBinIDResult, r.findByBinIDErr
}

func (r *mockRepo) FindCapturedRequestByID(_ context.Context, _ uuid.UUID) (domain.CapturedRequest, error) {
	r.recorder.record("FindCapturedRequestByID")
	return r.findByIDResult, r.findByIDErr
}

func (r *mockRepo) Calls() []string {
	return r.recorder.calls
}

func NewRepo(opts ...repo.Option) *mockRepo {
	options := repo.NewOptions(opts...)

	r := &mockRepo{
		options:  options,
		recorder: &callRecorder{calls: []string{}},
	}

	if e, ok := SaveBinErrFrom(options.Context); ok {
		r.saveBinErr = e
	}

	if b, ok := FindBinResultFrom(options.Context); ok {
		r.findBinResult = b
	}

	if e, ok := FindBinErrFrom(options.Context); ok {
		r.findBinErr = e
	}

	if n, ok := DeleteExpiredResultFrom(options.Context); ok {
		r.deleteExpiredResult = n
	}

	if e, ok := DeleteExpiredErrFrom(options.Context); ok {
		r.deleteExpiredErr = e
	}

	if c, ok := SaveCapturedResultFrom(options.Context); ok {
		r.saveCapturedResult = c
	}

	if e, ok := SaveCapturedErrFrom(options.Context); ok {
		r.saveCapturedErr = e
	}

	if cs, ok := FindByBinIDResultFrom(options.Context); ok {
		r.findByBinIDResult = cs
	}

	if e, ok := FindByBinIDErrFrom(options.Context); ok {
		r.findByBinIDErr = e
	}

	if c, ok := FindByIDResultFrom(options.Context); ok {
		r.findByIDResult = c
	}

	if e, ok := FindByIDErrFrom(options.Context); ok {
		r.findByIDErr = e
	}

	return r
}
