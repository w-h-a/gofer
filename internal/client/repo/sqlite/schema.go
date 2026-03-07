package sqlite

const schema = `
CREATE TABLE IF NOT EXISTS bins (
	id TEXT PRIMARY KEY,
	slug TEXT UNIQUE NOT NULL,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS captured_requests (
	id TEXT PRIMARY KEY,
	bin_id TEXT NOT NULL,
	sequence_num INTEGER NOT NULL,
	method TEXT NOT NULL,
	path TEXT NOT NULL,
	headers TEXT NOT NULL,
	query_params TEXT NOT NULL,
	body_size INTEGER NOT NULL,
	content_type TEXT NOT NULL,
	remote_addr TEXT NOT NULL,
	captured_at TEXT NOT NULL,
	raw_payload BLOB,
	FOREIGN KEY (bin_id) REFERENCES bins(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_captured_requests_bin_id ON captured_requests(bin_id);
`
