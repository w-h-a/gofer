package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Bin struct {
	id        uuid.UUID
	slug      Slug
	createdAt time.Time
	expiresAt time.Time
}

func (b Bin) ID() uuid.UUID {
	return b.id
}

func (b Bin) Slug() Slug {
	return b.slug
}

func (b Bin) CreatedAt() time.Time {
	return b.createdAt
}

func (b Bin) ExpiresAt() time.Time {
	return b.expiresAt
}

func (b Bin) IsExpired(now time.Time) bool {
	return !now.Before(b.expiresAt)
}

func NewBin(slug Slug, ttl time.Duration) (Bin, error) {
	if slug == "" {
		return Bin{}, errors.New("slug is required")
	}

	if ttl <= 0 {
		return Bin{}, errors.New("ttl must be positive")
	}

	now := time.Now()

	return Bin{
		id:        uuid.New(),
		slug:      slug,
		createdAt: now,
		expiresAt: now.Add(ttl),
	}, nil
}

func RehydrateBin(id uuid.UUID, slug Slug, createdAt, expiresAt time.Time) Bin {
	return Bin{
		id:        id,
		slug:      slug,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
}

const (
	slugAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	slugLength   = 8
)

var (
	ErrInvalidSlug = errors.New("invalid slug")
	ErrInvalidID   = errors.New("invalid id")
)

type Slug string

func (s Slug) String() string {
	return string(s)
}

func NewSlug() (Slug, error) {
	b := make([]byte, slugLength)

	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(slugAlphabet))))
		if err != nil {
			return "", fmt.Errorf("generate slug: %w", err)
		}
		b[i] = slugAlphabet[idx.Int64()]
	}

	return Slug(b), nil
}

func ParseSlug(s string) (Slug, error) {
	if len(s) != slugLength {
		return "", fmt.Errorf("%w: must be %d characters, got %d", ErrInvalidSlug, slugLength, len(s))
	}

	for _, c := range s {
		if !strings.ContainsRune(slugAlphabet, c) {
			return "", fmt.Errorf("%w: contains invalid characters: %c", ErrInvalidSlug, c)
		}
	}

	return Slug(s), nil
}

func ParseID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrInvalidID, err)
	}

	return id, nil
}
