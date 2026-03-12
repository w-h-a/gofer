package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	eventpublisher "github.com/w-h-a/gofer/internal/client/event_publisher"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrBinExpired = errors.New("bin is expired")
)

type Service struct {
	repo   repo.Repo
	pub    eventpublisher.EventPublisher
	tracer trace.Tracer
}

func (s *Service) CreateBin(ctx context.Context, in CreateBinInput) (CreateBinOutput, error) {
	ctx, span := s.tracer.Start(ctx, "bin.Create")
	defer span.End()

	slug, err := domain.NewSlug()
	if err != nil {
		span.RecordError(err)
		return CreateBinOutput{}, fmt.Errorf("failed to generate slug: %w", err)
	}

	bin, err := domain.NewBin(slug, in.TTL)
	if err != nil {
		span.RecordError(err)
		return CreateBinOutput{}, fmt.Errorf("failed to create bin: %w", err)
	}

	if err := s.repo.SaveBin(ctx, bin); err != nil {
		span.RecordError(err)
		return CreateBinOutput{}, fmt.Errorf("failed to save bin: %w", err)
	}

	span.SetAttributes(
		attribute.String("bin.slug", bin.Slug().String()),
		attribute.String("bin.id", bin.ID().String()),
	)

	return CreateBinOutput{
		ID:        bin.ID(),
		Slug:      bin.Slug().String(),
		CreatedAt: bin.CreatedAt(),
		ExpiresAt: bin.ExpiresAt(),
	}, nil
}

func (s *Service) SubscribeToBin(ctx context.Context, in SubscribeToBinInput) (SubscribeToBinOutput, error) {
	ctx, span := s.tracer.Start(ctx, "bin.Subscribe")
	defer span.End()

	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		span.RecordError(err)
		return SubscribeToBinOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	span.SetAttributes(
		attribute.String("bin.slug", slug.String()),
	)

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return SubscribeToBinOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		span.RecordError(ErrBinExpired)
		return SubscribeToBinOutput{}, ErrBinExpired
	}

	ch, err := s.pub.Subscribe(ctx, bin.ID())
	if err != nil {
		span.RecordError(err)
		return SubscribeToBinOutput{}, fmt.Errorf("failed to subscribe to bin: %w", err)
	}

	span.SetAttributes(
		attribute.String("bin.id", bin.ID().String()),
	)

	return SubscribeToBinOutput{
		BinID:   bin.ID(),
		Channel: ch,
	}, nil
}

func (s *Service) UnsubscribeFromBin(ctx context.Context, in UnsubscribeFromBinInput) error {
	ctx, span := s.tracer.Start(ctx, "bin.Unsubscribe")
	defer span.End()

	span.SetAttributes(
		attribute.String("bin.id", in.BinID.String()),
	)

	if err := s.pub.Unsubscribe(ctx, in.BinID, in.Channel); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to unsubscribe from bin: %w", err)
	}

	return nil
}

func (s *Service) CaptureRequest(ctx context.Context, in CaptureRequestInput) (CaptureRequestOutput, error) {
	ctx, span := s.tracer.Start(ctx, "capture.Request")
	defer span.End()

	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		span.RecordError(err)
		return CaptureRequestOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	span.SetAttributes(
		attribute.String("bin.slug", slug.String()),
	)

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return CaptureRequestOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		span.RecordError(ErrBinExpired)
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
		span.RecordError(err)
		return CaptureRequestOutput{}, fmt.Errorf("failed to create captured request: %w", err)
	}

	saved, err := s.repo.SaveCapturedRequest(ctx, req)
	if err != nil {
		span.RecordError(err)
		return CaptureRequestOutput{}, fmt.Errorf("failed to save captured request: %w", err)
	}

	if err := s.pub.Publish(ctx, saved); err != nil {
		span.RecordError(err)
		return CaptureRequestOutput{}, fmt.Errorf("failed to publish captured request: %w", err)
	}

	span.SetAttributes(
		attribute.Int("sequence_num", saved.SequenceNum()),
	)

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
	ctx, span := s.tracer.Start(ctx, "bin.View")
	defer span.End()

	slug, err := domain.ParseSlug(in.Slug)
	if err != nil {
		span.RecordError(err)
		return ViewBinOutput{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	span.SetAttributes(
		attribute.String("bin.slug", slug.String()),
	)

	bin, err := s.repo.FindBinBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return ViewBinOutput{}, fmt.Errorf("failed to find bin: %w", err)
	}

	if bin.IsExpired(time.Now()) {
		span.RecordError(ErrBinExpired)
		return ViewBinOutput{}, ErrBinExpired
	}

	reqs, err := s.repo.FindCapturedRequestByBinID(ctx, bin.ID())
	if err != nil {
		span.RecordError(err)
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

	span.SetAttributes(
		attribute.Int("requests.count", len(reqs)),
	)

	return ViewBinOutput{
		ID:        bin.ID(),
		Slug:      bin.Slug().String(),
		CreatedAt: bin.CreatedAt(),
		ExpiresAt: bin.ExpiresAt(),
		Requests:  summaries,
	}, nil
}

func (s *Service) ViewCapturedRequest(ctx context.Context, in ViewCapturedRequestInput) (ViewCapturedRequestOutput, error) {
	ctx, span := s.tracer.Start(ctx, "request.View")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.id", in.ID),
	)

	id, err := domain.ParseID(in.ID)
	if err != nil {
		span.RecordError(err)
		return ViewCapturedRequestOutput{}, fmt.Errorf("failed to parse request id: %w", err)
	}

	req, err := s.repo.FindCapturedRequestByID(ctx, id)
	if err != nil {
		span.RecordError(err)
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

func (s *Service) CleanupExpiredBins(ctx context.Context) (CleanupOutput, error) {
	ctx, span := s.tracer.Start(ctx, "bins.Cleanup")
	defer span.End()

	deleted, err := s.repo.DeleteExpiredBin(ctx, time.Now())
	if err != nil {
		span.RecordError(err)
		return CleanupOutput{}, fmt.Errorf("failed to cleanup expired bins: %w", err)
	}

	span.SetAttributes(
		attribute.Int("deleted.count", deleted),
	)

	return CleanupOutput{Deleted: deleted}, nil
}

func NewService(r repo.Repo, p eventpublisher.EventPublisher) *Service {
	return &Service{
		repo:   r,
		pub:    p,
		tracer: otel.Tracer("github.com/w-h-a/gofer/internal/service"),
	}
}
