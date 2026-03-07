package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CapturedRequest struct {
	id          uuid.UUID
	binID       uuid.UUID
	sequenceNum int
	method      string
	path        string
	headers     map[string][]string
	queryParams map[string][]string
	bodySize    int
	contentType string
	remoteAddr  string
	capturedAt  time.Time
	rawPayload  RawPayload
}

func (c CapturedRequest) ID() uuid.UUID {
	return c.id
}

func (c CapturedRequest) BinID() uuid.UUID {
	return c.binID
}

func (c CapturedRequest) SequenceNum() int {
	return c.sequenceNum
}

func (c CapturedRequest) Method() string {
	return c.method
}

func (c CapturedRequest) Path() string {
	return c.path
}

func (c CapturedRequest) Headers() map[string][]string {
	return makeCopy(c.headers)
}

func (c CapturedRequest) QueryParams() map[string][]string {
	return makeCopy(c.queryParams)
}

func (c CapturedRequest) BodySize() int {
	return c.bodySize
}

func (c CapturedRequest) ContentType() string {
	return c.contentType
}

func (c CapturedRequest) RemoteAddr() string {
	return c.remoteAddr
}

func (c CapturedRequest) CapturedAt() time.Time {
	return c.capturedAt
}

func (c CapturedRequest) RawPayload() RawPayload {
	return c.rawPayload
}

func NewCapturedRequest(
	binID uuid.UUID,
	sequenceNum int,
	method string,
	path string,
	headers map[string][]string,
	queryParams map[string][]string,
	contentType string,
	remoteAddr string,
	rawPayload RawPayload,
) (CapturedRequest, error) {
	if binID == uuid.Nil {
		return CapturedRequest{}, errors.New("bin ID is required")
	}

	if sequenceNum < 1 {
		return CapturedRequest{}, errors.New("sequence number must be positive")
	}

	if method == "" {
		return CapturedRequest{}, errors.New("method is required")
	}

	return CapturedRequest{
		id:          uuid.New(),
		binID:       binID,
		sequenceNum: sequenceNum,
		method:      method,
		path:        path,
		headers:     makeCopy(headers),
		queryParams: makeCopy(queryParams),
		bodySize:    rawPayload.Size(),
		contentType: contentType,
		remoteAddr:  remoteAddr,
		capturedAt:  time.Now(),
		rawPayload:  rawPayload,
	}, nil
}

func RehydrateCapturedRequest(
	id uuid.UUID,
	binID uuid.UUID,
	sequenceNum int,
	method string,
	path string,
	headers map[string][]string,
	queryParams map[string][]string,
	bodySize int,
	contentType string,
	remoteAddr string,
	capturedAt time.Time,
	rawPayload RawPayload,
) CapturedRequest {
	return CapturedRequest{
		id:          id,
		binID:       binID,
		sequenceNum: sequenceNum,
		method:      method,
		path:        path,
		headers:     makeCopy(headers),
		queryParams: makeCopy(queryParams),
		bodySize:    bodySize,
		contentType: contentType,
		remoteAddr:  remoteAddr,
		capturedAt:  capturedAt,
		rawPayload:  rawPayload,
	}
}

type RawPayload struct {
	data []byte
}

func (r RawPayload) Bytes() []byte {
	if r.data == nil {
		return nil
	}

	cp := make([]byte, len(r.data))
	copy(cp, r.data)

	return cp
}

func (r RawPayload) Size() int {
	return len(r.data)
}

func NewRawPayload(data []byte) RawPayload {
	if data == nil {
		return RawPayload{}
	}

	cp := make([]byte, len(data))
	copy(cp, data)

	return RawPayload{data: cp}
}

func makeCopy(h map[string][]string) map[string][]string {
	if h == nil {
		return nil
	}

	cp := make(map[string][]string, len(h))

	for k, v := range h {
		vals := make([]string, len(v))
		copy(vals, v)
		cp[k] = vals
	}

	return cp
}
