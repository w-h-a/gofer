package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/w-h-a/gofer/internal/client/repo"
	"github.com/w-h-a/gofer/internal/domain"
	"go.nhat.io/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	_ "modernc.org/sqlite"
)

var DRIVER string

func init() {
	driver, err := otelsql.Register(
		"sqlite",
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
		otelsql.WithSystem(semconv.DBSystemSqlite),
	)
	if err != nil {
		detail := "failed to register sqlite driver with otel"
		slog.ErrorContext(context.Background(), detail, "error", err)
		panic(detail)
	}

	DRIVER = driver
}

type sqliteRepo struct {
	options repo.Options
	conn    *sql.DB
}

func (r *sqliteRepo) SaveBin(ctx context.Context, bin domain.Bin) error {
	_, err := r.conn.ExecContext(
		ctx,
		`INSERT INTO bins (id, slug, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		bin.ID().String(),
		bin.Slug().String(),
		bin.CreatedAt().UTC().Format(time.RFC3339Nano),
		bin.ExpiresAt().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("failed to save bin: %w", err)
	}

	return nil
}

func (r *sqliteRepo) FindBinBySlug(ctx context.Context, slug domain.Slug) (domain.Bin, error) {
	var idStr, slugStr, createdStr, expiresStr string

	err := r.conn.QueryRowContext(
		ctx,
		`SELECT id, slug, created_at, expires_at FROM bins WHERE slug = ?`,
		slug.String(),
	).Scan(&idStr, &slugStr, &createdStr, &expiresStr)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bin{}, repo.ErrNotFound
	}
	if err != nil {
		return domain.Bin{}, fmt.Errorf("failed to find bin by slug: %w", err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return domain.Bin{}, fmt.Errorf("failed to find bin by slug: %w", err)
	}

	parsedSlug, err := domain.ParseSlug(slugStr)
	if err != nil {
		return domain.Bin{}, fmt.Errorf("failed to parse slug: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdStr)
	if err != nil {
		return domain.Bin{}, fmt.Errorf("failed to parse created_at: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, expiresStr)
	if err != nil {
		return domain.Bin{}, fmt.Errorf("failed to parse expires_at: %w", err)
	}

	return domain.RehydrateBin(id, parsedSlug, createdAt, expiresAt), nil
}

func (r *sqliteRepo) DeleteExpiredBin(ctx context.Context, now time.Time) (int, error) {
	result, err := r.conn.ExecContext(
		ctx,
		`DELETE FROM bins WHERE expires_at <= ?`,
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired bin: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check rows affected: %w", err)
	}

	return int(n), nil
}

func (r *sqliteRepo) SaveCapturedRequest(ctx context.Context, req domain.CapturedRequest) (domain.CapturedRequest, error) {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	var seqNum int
	err = tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(sequence_num), 0) + 1 FROM captured_requests WHERE bin_id = ?`,
		req.BinID().String(),
	).Scan(&seqNum)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to scan next sequence number: %w", err)
	}

	headersJSON, err := json.Marshal(req.Headers())
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to marshal headers: %w", err)
	}

	queryParamsJSON, err := json.Marshal(req.QueryParams())
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to marshal query params: %w", err)
	}

	capturedAt := req.CapturedAt().UTC().Format(time.RFC3339Nano)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO captured_requests
		(id, bin_id, sequence_num, method, path, headers, query_params, body_size, content_type, remote_addr, captured_at, raw_payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID().String(),
		req.BinID().String(),
		seqNum,
		req.Method(),
		req.Path(),
		string(headersJSON),
		string(queryParamsJSON),
		req.BodySize(),
		req.ContentType(),
		req.RemoteAddr(),
		capturedAt,
		req.RawPayload().Bytes(),
	)
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to insert captured request: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to commit: %w", err)
	}

	return domain.RehydrateCapturedRequest(
		req.ID(), req.BinID(), seqNum,
		req.Method(), req.Path(),
		req.Headers(), req.QueryParams(),
		req.BodySize(), req.ContentType(), req.RemoteAddr(),
		req.CapturedAt(),
		req.RawPayload(),
	), nil
}

func (r *sqliteRepo) FindCapturedRequestByBinID(ctx context.Context, binID uuid.UUID) ([]domain.CapturedRequest, error) {
	rows, err := r.conn.QueryContext(
		ctx,
		`SELECT id, bin_id, sequence_num, method, path, headers, query_params,
		body_size, content_type, remote_addr, captured_at, raw_payload
		FROM captured_requests WHERE bin_id = ? ORDER BY sequence_num ASC`,
		binID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find captured requests by bin id: %w", err)
	}
	defer rows.Close()

	var results []domain.CapturedRequest

	for rows.Next() {
		req, err := scanCapturedRequest(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, req)
	}

	return results, rows.Err()
}

func (r *sqliteRepo) FindCapturedRequestByID(ctx context.Context, id uuid.UUID) (domain.CapturedRequest, error) {
	row := r.conn.QueryRowContext(
		ctx,
		`SELECT id, bin_id, sequence_num, method, path, headers, query_params,
		body_size, content_type, remote_addr, captured_at, raw_payload
		FROM captured_requests WHERE id = ?`,
		id.String(),
	)

	req, err := scanCapturedRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CapturedRequest{}, repo.ErrNotFound
	}
	if err != nil {
		return domain.CapturedRequest{}, fmt.Errorf("failed to find captured request by id: %w", err)
	}

	return req, nil
}

func (r *sqliteRepo) configure() error {
	var journalMode string
	if err := r.conn.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		return fmt.Errorf("failed to set journal mode: %w", err)
	}

	if _, err := r.conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	slog.Debug("pragmas configured", "journal_mode", journalMode, "foreign_keys", true)

	if _, err := r.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to run schema migration: %w", err)
	}

	return nil
}

func NewRepo(opts ...repo.Option) (repo.Repo, error) {
	options := repo.NewOptions(opts...)

	conn, err := sql.Open(DRIVER, options.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	r := &sqliteRepo{
		options: options,
		conn:    conn,
	}

	if err := r.configure(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	return r, nil
}
