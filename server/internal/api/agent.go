package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/waylontech/backup/pkg/crypto"
	"github.com/waylontech/backup/server/internal/db"
	"github.com/waylontech/backup/server/internal/storage"
)

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)
	s.db.TouchDevice(r.Context(), dev.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetDEK(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)

	tenant, err := s.db.GetTenantByID(r.Context(), dev.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant error")
		return
	}

	dek, err := crypto.DecryptDEK(s.masterKey, tenant.DEKEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "dek error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"dek": base64.StdEncoding.EncodeToString(dek),
	})
}

func (s *Server) handleChunkCheck(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)

	var req struct {
		Hashes []string `json:"hashes"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	existing, err := s.db.CheckChunksExist(r.Context(), dev.TenantID, req.Hashes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	missing := make([]string, 0)
	for _, h := range req.Hashes {
		if !existing[h] {
			missing = append(missing, h)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"missing": missing,
	})
}

func (s *Server) handleChunkConfirm(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)

	var req struct {
		Hash          string `json:"hash"`
		PlaintextSize int64  `json:"plaintext_size"`
		StoredSize    int64  `json:"stored_size"`
		ObjectKey     string `json:"object_key"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	target, err := s.db.GetPrimaryStorageTarget(r.Context(), dev.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage target: "+err.Error())
		return
	}

	if err := s.db.RecordChunk(r.Context(), dev.TenantID, req.Hash, target.ID, req.ObjectKey, req.PlaintextSize, req.StoredSize); err != nil {
		writeError(w, http.StatusInternalServerError, "record chunk: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type FileManifestEntry struct {
	Path        string   `json:"path"`
	ParentPath  string   `json:"parent_path"`
	Name        string   `json:"name"`
	Size        int64    `json:"size"`
	Mtime       *string  `json:"mtime,omitempty"`
	Mode        int      `json:"mode"`
	IsDir       bool     `json:"is_dir"`
	ChunkHashes []string `json:"chunk_hashes"`
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)

	var req struct {
		JobID   string              `json:"job_id"`
		Entries []FileManifestEntry `json:"entries"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var totalSize, fileCount int64
	for _, e := range req.Entries {
		if !e.IsDir {
			totalSize += e.Size
			fileCount++
		}
	}

	snap, err := s.db.CreateSnapshot(r.Context(), dev.ID, dev.TenantID, req.JobID, fileCount, totalSize, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create snapshot: "+err.Error())
		return
	}

	fileEntries := make([]db.FileEntry, 0, len(req.Entries))
	for _, e := range req.Entries {
		fe := db.FileEntry{
			SnapshotID:  snap.ID,
			Path:        e.Path,
			ParentPath:  e.ParentPath,
			Name:        e.Name,
			Size:        e.Size,
			Mode:        e.Mode,
			IsDir:       e.IsDir,
			ChunkHashes: e.ChunkHashes,
		}
		if e.Mtime != nil {
			t, err := time.Parse(time.RFC3339, *e.Mtime)
			if err == nil {
				fe.Mtime = &t
			}
		}
		fileEntries = append(fileEntries, fe)
	}

	if err := s.db.BulkInsertFileEntries(r.Context(), fileEntries); err != nil {
		writeError(w, http.StatusInternalServerError, "insert file entries: "+err.Error())
		return
	}

	if req.JobID != "" {
		s.db.UpdateJobStatus(r.Context(), req.JobID, "completed", nil)
	}

	writeJSON(w, http.StatusCreated, snap)
}

func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)
	jobID := chi.URLParam(r, "jobID")

	job, err := s.db.GetBackupJobByID(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if job.DeviceID != dev.ID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req struct {
		Status         string  `json:"status"`
		ErrorMsg       *string `json:"error_msg,omitempty"`
		BytesScanned   int64   `json:"bytes_scanned"`
		BytesChanged   int64   `json:"bytes_changed"`
		ChunksTotal    int64   `json:"chunks_total"`
		ChunksUploaded int64   `json:"chunks_uploaded"`
		ChunksDeduped  int64   `json:"chunks_deduped"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if req.Status != "" {
		s.db.UpdateJobStatus(r.Context(), jobID, req.Status, req.ErrorMsg)
	}
	s.db.UpdateJobStats(r.Context(), jobID,
		req.BytesScanned, req.BytesChanged,
		req.ChunksTotal, req.ChunksUploaded, req.ChunksDeduped)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) storageClientForTenant(r *http.Request, tenantID string) (*storage.Client, error) {
	target, err := s.db.GetPrimaryStorageTarget(r.Context(), tenantID)
	if err != nil {
		return nil, fmt.Errorf("storage target: %w", err)
	}

	creds, err := storage.DecryptCredentials(s.masterKey, target.CredentialsEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt creds: %w", err)
	}

	return storage.New(target.Endpoint, target.Region, target.Bucket,
		creds.AccessKeyID, creds.SecretAccessKey, target.PathStyle), nil
}

func chunkObjectKey(tenantID, hash string) string {
	return fmt.Sprintf("chunks/%s/%s/%s", tenantID, hash[:2], hash)
}

func (s *Server) handleChunkPresign(w http.ResponseWriter, r *http.Request) {
	dev := deviceFromCtx(r)

	var req struct {
		Hash string `json:"hash"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	target, err := s.db.GetPrimaryStorageTarget(r.Context(), dev.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage target: "+err.Error())
		return
	}

	creds, err := storage.DecryptCredentials(s.masterKey, target.CredentialsEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decrypt creds: "+err.Error())
		return
	}

	sc := storage.New(target.Endpoint, target.Region, target.Bucket,
		creds.AccessKeyID, creds.SecretAccessKey, target.PathStyle)

	objectKey := chunkObjectKey(dev.TenantID, req.Hash)
	presignURL, err := sc.PresignPut(r.Context(), objectKey, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "presign: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"presign_url": presignURL,
		"object_key":  objectKey,
	})
}

func splitPath(path string) (parent, name string) {
	path = strings.TrimRight(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "/", path
	}
	return path[:idx+1], path[idx+1:]
}
