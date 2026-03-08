package service

import (
	"context"
	"errors"
	"fmt"

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

func NewService(r repo.Repo, p eventpublisher.EventPublisher) *Service {
	return &Service{repo: r, pub: p}
}
