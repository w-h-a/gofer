package service

import (
	"time"

	"github.com/google/uuid"
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

type ViewRequestInput struct {
	ID string
}

type ViewRequestOutput struct {
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

type CleanupOutput struct {
	Deleted int
}
