package sqlite

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/domain"
)

type scanner interface {
	Scan(dest ...any) error
}

func scanCapturedRequest(s scanner) (domain.CapturedRequest, error) {
	var idStr, binIDStr, method, path, headersStr, queryParamsStr, contentType, remoteAddr, capturedAtStr string
	var seqNum, bodySize int
	var rawPayload []byte

	if err := s.Scan(
		&idStr, &binIDStr, &seqNum, &method, &path, &headersStr,
		&queryParamsStr, &bodySize, &contentType, &remoteAddr,
		&capturedAtStr, &rawPayload,
	); err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to scan captured request: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to parse request id: %w", err)
	}

	binID, err := uuid.Parse(binIDStr)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to parse bin id: %w", err)
	}

	var headers map[string][]string
	if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	var queryParams map[string][]string
	if err := json.Unmarshal([]byte(queryParamsStr), &queryParams); err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to unmarshal query params: %w", err)
	}

	capturedAt, err := time.Parse(time.RFC3339Nano, capturedAtStr)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to parse captured_at: %w", err)
	}

	return domain.RehydrateCapturedRequest(
		id, binID, seqNum, method, path, headers, queryParams,
		bodySize, contentType, remoteAddr, capturedAt,
		domain.NewRawPayload(rawPayload),
	), nil
}
