package api

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/waylontech/backup/pkg/crypto"
	"github.com/waylontech/backup/server/internal/db"
)

func (s *Server) handleFileTree(w http.ResponseWriter, r *http.Request) {
	snapshotID := chi.URLParam(r, "snapshotID")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	snap, err := s.db.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		writeError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && snap.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	entries, err := s.db.ListFileEntries(r.Context(), snapshotID, path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = make([]db.FileEntry, 0)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot": snap,
		"path":     path,
		"entries":  entries,
	})
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	snapshotID := chi.URLParam(r, "snapshotID")
	filePath, err := url.QueryUnescape(r.URL.Query().Get("path"))
	if err != nil || filePath == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}

	snap, err := s.db.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		writeError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && snap.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	entry, err := s.db.GetFileEntry(r.Context(), snapshotID, filePath)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found in snapshot")
		return
	}

	if entry.IsDir {
		writeError(w, http.StatusBadRequest, "cannot download a directory; use a zip endpoint")
		return
	}

	tenant, err := s.db.GetTenantByID(r.Context(), snap.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant error")
		return
	}

	dek, err := crypto.DecryptDEK(s.masterKey, tenant.DEKEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "dek error")
		return
	}

	sc, err := s.storageClientForTenant(r, snap.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage error")
		return
	}

	chunks, err := s.db.GetChunksByHashes(r.Context(), snap.TenantID, entry.ChunkHashes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "chunk lookup error")
		return
	}

	chunkByHash := make(map[string]string, len(chunks))
	for _, ch := range chunks {
		chunkByHash[ch.Hash] = ch.ObjectKey
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, entry.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	for _, hash := range entry.ChunkHashes {
		objectKey, ok := chunkByHash[hash]
		if !ok {
			s.log.Error("missing chunk " + hash)
			return
		}

		blob, err := sc.GetObject(r.Context(), objectKey)
		if err != nil {
			s.log.Error("fetch chunk " + hash + ": " + err.Error())
			return
		}

		var chunkHash [32]byte
		copy(chunkHash[:], mustDecodeHex(hash))

		plain, err := crypto.DecryptChunk(dek, chunkHash, blob)
		if err != nil {
			s.log.Error("decrypt chunk " + hash + ": " + err.Error())
			return
		}

		if _, err := w.Write(plain); err != nil {
			return
		}
	}
}

func mustDecodeHex(h string) []byte {
	b := make([]byte, len(h)/2)
	for i := 0; i < len(h); i += 2 {
		hi := hexVal(h[i])
		lo := hexVal(h[i+1])
		b[i/2] = hi<<4 | lo
	}
	return b
}

func hexVal(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
