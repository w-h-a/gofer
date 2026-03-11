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

func (s *Service) ViewBin(ctx context.Context, in ViewBinInput) (ViewBinOutput, error) {
	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		return ViewBinOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		return ViewBinOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		return ViewBinOutput{}, ErrBinExpired
	}

	reqs, err := s.repo.FindCapturedRequestByBinID(ctx, bin.ID())
	if err != nil {
		return ViewBinOutput{}, fmt.Errorf("failed to find captured requests: %w", err)
	}

	summaries := make([]CapturedRequestSummary, len(reqs))
	for i, r := range reqs {
		summaries[i] = CapturedRequestSummary{
			ID:          r.ID(),
			SequenceNum: r.SequenceNum(),
			Method:      r.Method(),
			Path:        r.Path(),
			ContentType: r.ContentType(),
			BodySize:    r.BodySize(),
			CapturedAt:  r.CapturedAt(),
		}
	}

	return ViewBinOutput{
		ID:        bin.ID(),
		Slug:      bin.Slug().String(),
		CreatedAt: bin.CreatedAt(),
		ExpiresAt: bin.ExpiresAt(),
		Requests:  summaries,
	}, nil
}

func (s *Service) ViewCapturedRequest(ctx context.Context, in ViewCapturedRequestInput) (ViewCapturedRequestOutput, error) {
	id, err := domain.ParseID(in.ID)
	if err != nil {
		return ViewCapturedRequestOutput{}, fmt.Errorf("failed to parse request id: %w", err)
	}

	req, err := s.repo.FindCapturedRequestByID(ctx, id)
	if err != nil {
		return ViewCapturedRequestOutput{}, fmt.Errorf("failed to find captured request: %w", err)
	}

	return ViewCapturedRequestOutput{
		ID:          req.ID(),
		BinID:       req.BinID(),
		SequenceNum: req.SequenceNum(),
		Method:      req.Method(),
		Path:        req.Path(),
		Headers:     req.Headers(),
		QueryParams: req.QueryParams(),
		ContentType: req.ContentType(),
		RemoteAddr:  req.RemoteAddr(),
		BodySize:    req.BodySize(),
		CapturedAt:  req.CapturedAt(),
		Body:        req.RawPayload().Bytes(),
	}, nil
}

func (s *Service) SubscribeToBin(ctx context.Context, in SubscribeToBinInput) (SubscribeToBinOutput, error) {
	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		return SubscribeToBinOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		return SubscribeToBinOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		return SubscribeToBinOutput{}, ErrBinExpired
	}

	ch, err := s.pub.Subscribe(ctx, bin.ID())
	if err != nil {
		return SubscribeToBinOutput{}, fmt.Errorf("failed to subscribe to bin: %w", err)
	}

	return SubscribeToBinOutput{
		BinID:   bin.ID(),
		Channel: ch,
	}, nil
}

func (s *Service) UnsubscribeFromBin(ctx context.Context, in UnsubscribeFromBinInput) error {
	if err := s.pub.Unsubscribe(ctx, in.BinID, in.Channel); err != nil {
		return fmt.Errorf("failed to unsubscribe from bin: %w", err)
	}

	return nil
}

func (s *Service) CleanupExpiredBins(ctx context.Context) (CleanupOutput, error) {
	deleted, err := s.repo.DeleteExpiredBin(ctx, time.Now())
	if err != nil {
		return CleanupOutput{}, fmt.Errorf("failed to cleanup expired bins: %w", err)
	}

	return CleanupOutput{Deleted: deleted}, nil
}

func NewService(r repo.Repo, p eventpublisher.EventPublisher) *Service {
	return &Service{repo: r, pub: p}
}
