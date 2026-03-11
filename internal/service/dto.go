package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/domain"
)

type CreateBinInput struct {
	TTL time.Duration
}

type CreateBinOutput struct {
	ID        uuid.UUID
	Slug      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type CaptureRequestInput struct {
	Slug        string
	Method      string
	Path        string
	Headers     map[string][]string
	QueryParams map[string][]string
	Body        []byte
	ContentType string
	RemoteAddr  string
}

type CaptureRequestOutput struct {
	ID          uuid.UUID
	BinID       uuid.UUID
	SequenceNum int
	Method      string
	Path        string
	ContentType string
	BodySize    int
	CapturedAt  time.Time
}

type ViewBinInput struct {
	Slug string
}

type ViewBinOutput struct {
	ID        uuid.UUID
	Slug      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Requests  []CapturedRequestSummary
}

type CapturedRequestSummary struct {
	ID          uuid.UUID
	SequenceNum int
	Method      string
	Path        string
	ContentType string
	BodySize    int
	CapturedAt  time.Time
}

type ViewCapturedRequestInput struct {
	ID string
}

type ViewCapturedRequestOutput struct {
	ID          uuid.UUID
	BinID       uuid.UUID
	SequenceNum int
	Method      string
	Path        string
	Headers     map[string][]string
	QueryParams map[string][]string
	ContentType string
	RemoteAddr  string
	BodySize    int
	CapturedAt  time.Time
	Body        []byte
}

type SubscribeToBinInput struct {
	Slug string
}

type SubscribeToBinOutput struct {
	BinID   uuid.UUID
	Channel <-chan domain.CapturedRequest
}

type UnsubscribeFromBinInput struct {
	BinID   uuid.UUID
	Channel <-chan domain.CapturedRequest
}

type CleanupOutput struct {
	Deleted int
}
