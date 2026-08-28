package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	DEKEnc    []byte    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Device struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	SiteID       *string    `json:"site_id,omitempty"`
	Hostname     string     `json:"hostname"`
	OSType       string     `json:"os_type"`
	Arch         string     `json:"arch"`
	AgentVersion string     `json:"agent_version"`
	Status       string     `json:"status"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	EnrolledAt   time.Time  `json:"enrolled_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type BackupJob struct {
	ID             string     `json:"id"`
	DeviceID       string     `json:"device_id"`
	TenantID       string     `json:"tenant_id"`
	Status         string     `json:"status"`
	BytesScanned   int64      `json:"bytes_scanned"`
	BytesChanged   int64      `json:"bytes_changed"`
	ChunksTotal    int64      `json:"chunks_total"`
	ChunksUploaded int64      `json:"chunks_uploaded"`
	ChunksDeduped  int64      `json:"chunks_deduped"`
	ErrorMsg       *string    `json:"error_msg,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Snapshot struct {
	ID         string    `json:"id"`
	DeviceID   string    `json:"device_id"`
	TenantID   string    `json:"tenant_id"`
	JobID      *string   `json:"job_id,omitempty"`
	FileCount  int64     `json:"file_count"`
	TotalSize  int64     `json:"total_size"`
	DedupeSize int64     `json:"dedupe_size"`
	CreatedAt  time.Time `json:"created_at"`
}

type FileEntry struct {
	ID          string     `json:"id"`
	SnapshotID  string     `json:"snapshot_id"`
	Path        string     `json:"path"`
	ParentPath  string     `json:"parent_path"`
	Name        string     `json:"name"`
	Size        int64      `json:"size"`
	Mtime       *time.Time `json:"mtime,omitempty"`
	Mode        int        `json:"mode"`
	IsDir       bool       `json:"is_dir"`
	ChunkHashes []string   `json:"chunk_hashes,omitempty"`
}

type StorageTarget struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Label          string    `json:"label"`
	Provider       string    `json:"provider"`
	Endpoint       string    `json:"endpoint"`
	Bucket         string    `json:"bucket"`
	Region         string    `json:"region"`
	CredentialsEnc []byte    `json:"-"`
	IsPrimary      bool      `json:"is_primary"`
	PathStyle      bool      `json:"path_style"`
	CreatedAt      time.Time `json:"created_at"`
}

type Chunk struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Hash            string    `json:"hash"`
	PlaintextSize   int64     `json:"plaintext_size"`
	StoredSize      int64     `json:"stored_size"`
	StorageTargetID string    `json:"storage_target_id"`
	ObjectKey       string    `json:"object_key"`
	RefCount        int64     `json:"ref_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func pool(d *DB) *pgxpool.Pool { return d.Pool }

func (d *DB) CreateTenant(ctx context.Context, name, slug string, dekEnc []byte) (*Tenant, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO tenants (name, slug, dek_enc)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, status, dek_enc, created_at, updated_at`,
		name, slug, dekEnc)
	return scanTenant(row)
}

func (d *DB) GetTenantByID(ctx context.Context, id string) (*Tenant, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, name, slug, status, dek_enc, created_at, updated_at
		FROM tenants WHERE id = $1`, id)
	return scanTenant(row)
}

func (d *DB) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, name, slug, status, dek_enc, created_at, updated_at
		FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanTenant(row pgx.Row) (*Tenant, error) {
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.DEKEnc, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (d *DB) CreateUser(ctx context.Context, tenantID, email, passwordHash, role, displayName string) (*User, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, role, display_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, email, password_hash, role, display_name, created_at`,
		tenantID, email, passwordHash, role, displayName)
	return scanUser(row)
}

func (d *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, role, display_name, created_at
		FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (d *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, tenant_id, email, password_hash, role, display_name, created_at
		FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (d *DB) CreateEnrollmentToken(ctx context.Context, tenantID, tokenHash, label string, expiresAt time.Time) error {
	_, err := pool(d).Exec(ctx, `
		INSERT INTO agent_enrollments (tenant_id, token_hash, label, expires_at)
		VALUES ($1, $2, $3, $4)`,
		tenantID, tokenHash, label, expiresAt)
	return err
}

func (d *DB) UseEnrollmentToken(ctx context.Context, tokenHash string) (tenantID string, err error) {
	var id string
	err = pool(d).QueryRow(ctx, `
		UPDATE agent_enrollments
		SET used_at = NOW(), device_id = NULL
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING tenant_id`, tokenHash).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("token not found or expired: %w", err)
	}
	return id, nil
}

func (d *DB) CreateDevice(ctx context.Context, tenantID, hostname, osType, arch, agentVersion string) (*Device, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO devices (tenant_id, hostname, os_type, arch, agent_version)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, site_id, hostname, os_type, arch, agent_version, status, last_seen_at, enrolled_at, created_at`,
		tenantID, hostname, osType, arch, agentVersion)
	return scanDevice(row)
}

func (d *DB) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, tenant_id, site_id, hostname, os_type, arch, agent_version, status, last_seen_at, enrolled_at, created_at
		FROM devices WHERE id = $1`, id)
	return scanDevice(row)
}

func (d *DB) ListDevices(ctx context.Context, tenantID string) ([]Device, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, tenant_id, site_id, hostname, os_type, arch, agent_version, status, last_seen_at, enrolled_at, created_at
		FROM devices WHERE tenant_id = $1 ORDER BY hostname`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		dev, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *dev)
	}
	return out, rows.Err()
}

func (d *DB) ListAllDevices(ctx context.Context) ([]Device, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, tenant_id, site_id, hostname, os_type, arch, agent_version, status, last_seen_at, enrolled_at, created_at
		FROM devices ORDER BY tenant_id, hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		dev, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *dev)
	}
	return out, rows.Err()
}

func (d *DB) TouchDevice(ctx context.Context, deviceID string) error {
	_, err := pool(d).Exec(ctx, `UPDATE devices SET last_seen_at = NOW() WHERE id = $1`, deviceID)
	return err
}

func scanDevice(row pgx.Row) (*Device, error) {
	var dev Device
	err := row.Scan(&dev.ID, &dev.TenantID, &dev.SiteID, &dev.Hostname, &dev.OSType, &dev.Arch,
		&dev.AgentVersion, &dev.Status, &dev.LastSeenAt, &dev.EnrolledAt, &dev.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (d *DB) CreateDeviceToken(ctx context.Context, deviceID, tokenHash string) error {
	_, err := pool(d).Exec(ctx, `
		INSERT INTO device_tokens (device_id, token_hash) VALUES ($1, $2)`,
		deviceID, tokenHash)
	return err
}

func (d *DB) GetDeviceByTokenHash(ctx context.Context, tokenHash string) (*Device, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT d.id, d.tenant_id, d.site_id, d.hostname, d.os_type, d.arch, d.agent_version, d.status, d.last_seen_at, d.enrolled_at, d.created_at
		FROM device_tokens dt
		JOIN devices d ON d.id = dt.device_id
		WHERE dt.token_hash = $1`, tokenHash)
	return scanDevice(row)
}

func (d *DB) CreateStorageTarget(ctx context.Context, tenantID, label, provider, endpoint, bucket, region string, credEnc []byte, isPrimary, pathStyle bool) (*StorageTarget, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO storage_targets (tenant_id, label, provider, endpoint, bucket, region, credentials_enc, is_primary, path_style)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, label, provider, endpoint, bucket, region, credentials_enc, is_primary, path_style, created_at`,
		tenantID, label, provider, endpoint, bucket, region, credEnc, isPrimary, pathStyle)
	return scanStorageTarget(row)
}

func (d *DB) GetPrimaryStorageTarget(ctx context.Context, tenantID string) (*StorageTarget, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, tenant_id, label, provider, endpoint, bucket, region, credentials_enc, is_primary, path_style, created_at
		FROM storage_targets WHERE tenant_id = $1 AND is_primary = TRUE LIMIT 1`, tenantID)
	return scanStorageTarget(row)
}

func (d *DB) GetStorageTargetByID(ctx context.Context, id string) (*StorageTarget, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, tenant_id, label, provider, endpoint, bucket, region, credentials_enc, is_primary, path_style, created_at
		FROM storage_targets WHERE id = $1`, id)
	return scanStorageTarget(row)
}

func scanStorageTarget(row pgx.Row) (*StorageTarget, error) {
	var st StorageTarget
	err := row.Scan(&st.ID, &st.TenantID, &st.Label, &st.Provider, &st.Endpoint, &st.Bucket,
		&st.Region, &st.CredentialsEnc, &st.IsPrimary, &st.PathStyle, &st.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (d *DB) CreateBackupJob(ctx context.Context, deviceID, tenantID string) (*BackupJob, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO backup_jobs (device_id, tenant_id, status, started_at)
		VALUES ($1, $2, 'running', NOW())
		RETURNING id, device_id, tenant_id, status, bytes_scanned, bytes_changed, chunks_total, chunks_uploaded, chunks_deduped, error_msg, started_at, finished_at, created_at`,
		deviceID, tenantID)
	return scanJob(row)
}

func (d *DB) UpdateJobStatus(ctx context.Context, jobID, status string, errMsg *string) error {
	_, err := pool(d).Exec(ctx, `
		UPDATE backup_jobs SET status = $2, error_msg = $3, finished_at = CASE WHEN $2 IN ('completed','failed','cancelled') THEN NOW() ELSE finished_at END
		WHERE id = $1`, jobID, status, errMsg)
	return err
}

func (d *DB) UpdateJobStats(ctx context.Context, jobID string, bytesScanned, bytesChanged, total, uploaded, deduped int64) error {
	_, err := pool(d).Exec(ctx, `
		UPDATE backup_jobs SET bytes_scanned=$2, bytes_changed=$3, chunks_total=$4, chunks_uploaded=$5, chunks_deduped=$6
		WHERE id = $1`, jobID, bytesScanned, bytesChanged, total, uploaded, deduped)
	return err
}

func (d *DB) GetBackupJobByID(ctx context.Context, id string) (*BackupJob, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, device_id, tenant_id, status, bytes_scanned, bytes_changed, chunks_total, chunks_uploaded, chunks_deduped, error_msg, started_at, finished_at, created_at
		FROM backup_jobs WHERE id = $1`, id)
	return scanJob(row)
}

func (d *DB) ListJobsForDevice(ctx context.Context, deviceID string, limit int) ([]BackupJob, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, device_id, tenant_id, status, bytes_scanned, bytes_changed, chunks_total, chunks_uploaded, chunks_deduped, error_msg, started_at, finished_at, created_at
		FROM backup_jobs WHERE device_id = $1 ORDER BY created_at DESC LIMIT $2`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackupJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func scanJob(row pgx.Row) (*BackupJob, error) {
	var j BackupJob
	err := row.Scan(&j.ID, &j.DeviceID, &j.TenantID, &j.Status,
		&j.BytesScanned, &j.BytesChanged, &j.ChunksTotal, &j.ChunksUploaded, &j.ChunksDeduped,
		&j.ErrorMsg, &j.StartedAt, &j.FinishedAt, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (d *DB) CheckChunksExist(ctx context.Context, tenantID string, hashes []string) (map[string]bool, error) {
	if len(hashes) == 0 {
		return map[string]bool{}, nil
	}
	rows, err := pool(d).Query(ctx, `
		SELECT hash FROM chunks WHERE tenant_id = $1 AND hash = ANY($2)`,
		tenantID, hashes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]bool, len(hashes))
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		result[h] = true
	}
	return result, rows.Err()
}

func (d *DB) RecordChunk(ctx context.Context, tenantID, hash, storageTargetID, objectKey string, plaintextSize, storedSize int64) error {
	_, err := pool(d).Exec(ctx, `
		INSERT INTO chunks (tenant_id, hash, plaintext_size, stored_size, storage_target_id, object_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, hash) DO UPDATE SET ref_count = chunks.ref_count + 1`,
		tenantID, hash, plaintextSize, storedSize, storageTargetID, objectKey)
	return err
}

func (d *DB) GetChunksByHashes(ctx context.Context, tenantID string, hashes []string) ([]Chunk, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, tenant_id, hash, plaintext_size, stored_size, storage_target_id, object_key, ref_count, created_at
		FROM chunks WHERE tenant_id = $1 AND hash = ANY($2)`,
		tenantID, hashes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Hash, &c.PlaintextSize, &c.StoredSize,
			&c.StorageTargetID, &c.ObjectKey, &c.RefCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) CreateSnapshot(ctx context.Context, deviceID, tenantID, jobID string, fileCount, totalSize, dedupeSize int64) (*Snapshot, error) {
	row := pool(d).QueryRow(ctx, `
		INSERT INTO snapshots (device_id, tenant_id, job_id, file_count, total_size, dedupe_size)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, device_id, tenant_id, job_id, file_count, total_size, dedupe_size, created_at`,
		deviceID, tenantID, jobID, fileCount, totalSize, dedupeSize)
	return scanSnapshot(row)
}

func (d *DB) GetSnapshotByID(ctx context.Context, id string) (*Snapshot, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, device_id, tenant_id, job_id, file_count, total_size, dedupe_size, created_at
		FROM snapshots WHERE id = $1`, id)
	return scanSnapshot(row)
}

func (d *DB) ListSnapshotsForDevice(ctx context.Context, deviceID string, limit int) ([]Snapshot, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, device_id, tenant_id, job_id, file_count, total_size, dedupe_size, created_at
		FROM snapshots WHERE device_id = $1 ORDER BY created_at DESC LIMIT $2`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func scanSnapshot(row pgx.Row) (*Snapshot, error) {
	var s Snapshot
	err := row.Scan(&s.ID, &s.DeviceID, &s.TenantID, &s.JobID, &s.FileCount, &s.TotalSize, &s.DedupeSize, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) BulkInsertFileEntries(ctx context.Context, entries []FileEntry) error {
	if len(entries) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(`
			INSERT INTO file_tree_entries (snapshot_id, path, parent_path, name, size, mtime, mode, is_dir, chunk_hashes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (snapshot_id, path) DO NOTHING`,
			e.SnapshotID, e.Path, e.ParentPath, e.Name, e.Size, e.Mtime, e.Mode, e.IsDir, e.ChunkHashes)
	}
	br := pool(d).SendBatch(ctx, batch)
	defer br.Close()
	for range entries {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ListFileEntries(ctx context.Context, snapshotID, parentPath string) ([]FileEntry, error) {
	rows, err := pool(d).Query(ctx, `
		SELECT id, snapshot_id, path, parent_path, name, size, mtime, mode, is_dir, chunk_hashes
		FROM file_tree_entries
		WHERE snapshot_id = $1 AND parent_path = $2
		ORDER BY is_dir DESC, name ASC`,
		snapshotID, parentPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileEntry
	for rows.Next() {
		var e FileEntry
		if err := rows.Scan(&e.ID, &e.SnapshotID, &e.Path, &e.ParentPath, &e.Name,
			&e.Size, &e.Mtime, &e.Mode, &e.IsDir, &e.ChunkHashes); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) GetFileEntry(ctx context.Context, snapshotID, path string) (*FileEntry, error) {
	row := pool(d).QueryRow(ctx, `
		SELECT id, snapshot_id, path, parent_path, name, size, mtime, mode, is_dir, chunk_hashes
		FROM file_tree_entries WHERE snapshot_id = $1 AND path = $2`, snapshotID, path)
	var e FileEntry
	err := row.Scan(&e.ID, &e.SnapshotID, &e.Path, &e.ParentPath, &e.Name,
		&e.Size, &e.Mtime, &e.Mode, &e.IsDir, &e.ChunkHashes)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *DB) WriteAuditLog(ctx context.Context, tenantID, userID, action, resourceType string, resourceID *string, meta map[string]any, ip string) {
	pool(d).Exec(ctx, `
		INSERT INTO audit_log (tenant_id, user_id, action, resource_type, resource_id, meta, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, userID, action, resourceType, resourceID, meta, ip)
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
