package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
)

var (
	ErrBinExpired = errors.New("bin is expired")
)

type Service struct {
	repo repo.Repo
	pub  eventpublisher.EventPublisher
}

func (s *Service) CreateBin(ctx context.Context, in CreateBinInput) (CreateBinOutput, error) {
	slug, err := domain.NewSlug()
	if err != nil {
		return CreateBinOutput{}, fmt.Errorf("failed to generate slug: %w", err)
	}

	bin, err := domain.NewBin(slug, in.TTL)
	if err != nil {
		return CreateBinOutput{}, fmt.Errorf("failed to create bin: %w", err)
	}

	if err := s.repo.SaveBin(ctx, bin); err != nil {
		return CreateBinOutput{}, fmt.Errorf("failed to save bin: %w", err)
	}

	return CreateBinOutput{
		ID:        bin.ID(),
		Slug:      bin.Slug().String(),
		CreatedAt: bin.CreatedAt(),
		ExpiresAt: bin.ExpiresAt(),
	}, nil
}

func (s *Service) CaptureRequest(ctx context.Context, in CaptureRequestInput) (CaptureRequestOutput, error) {
	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		return CaptureRequestOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		return CaptureRequestOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		return CaptureRequestOutput{}, ErrBinExpired
	}

	payload := domain.NewRawPayload(in.Body)

	req, err := domain.NewCapturedRequest(
		bin.ID(),
		1, // placeholder (logic is in repo)
		in.Method, in.Path,
		in.Headers, in.QueryParams,
		in.ContentType, in.RemoteAddr,
		payload,
	)
	if err != nil {
		return CaptureRequestOutput{}, fmt.Errorf("failed to create captured request: %w", err)
	}

	saved, err := s.repo.SaveCapturedRequest(ctx, req)
	if err != nil {
		return CaptureRequestOutput{}, fmt.Errorf("failed to save captured request: %w", err)
	}

	if err := s.pub.Publish(ctx, saved); err != nil {
		return CaptureRequestOutput{}, fmt.Errorf("failed to publish captured request: %w", err)
	}

	return CaptureRequestOutput{
		ID:          saved.ID(),
		BinID:       saved.BinID(),
		SequenceNum: saved.SequenceNum(),
		Method:      saved.Method(),
		Path:        saved.Path(),
		ContentType: saved.ContentType(),
		BodySize:    saved.BodySize(),
		CapturedAt:  saved.CapturedAt(),
	}, nil
}

func NewService(r repo.Repo, p eventpublisher.EventPublisher) *Service {
	return &Service{repo: r, pub: p}
}
