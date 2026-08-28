package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/waylontech/backup/pkg/crypto"
	"github.com/waylontech/backup/server/internal/storage"
)

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" {
		writeError(w, http.StatusForbidden, "vendor_admin required")
		return
	}
	tenants, err := s.db.ListTenants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tenants)
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "tenantID")
	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && claims.TenantID != id {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	t, err := s.db.GetTenantByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" {
		writeError(w, http.StatusForbidden, "vendor_admin required")
		return
	}

	var req struct {
		Name             string `json:"name"`
		Slug             string `json:"slug"`
		StorageEndpoint  string `json:"storage_endpoint"`
		StorageBucket    string `json:"storage_bucket"`
		StorageRegion    string `json:"storage_region"`
		StorageAccessKey string `json:"storage_access_key"`
		StorageSecretKey string `json:"storage_secret_key"`
		StoragePathStyle bool   `json:"storage_path_style"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	dek, err := crypto.GenerateDEK()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generate dek: "+err.Error())
		return
	}
	dekEnc, err := crypto.EncryptDEK(s.masterKey, dek)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt dek: "+err.Error())
		return
	}

	tenant, err := s.db.CreateTenant(r.Context(), req.Name, req.Slug, dekEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create tenant: "+err.Error())
		return
	}

	creds := storage.Credentials{
		AccessKeyID:     req.StorageAccessKey,
		SecretAccessKey: req.StorageSecretKey,
	}
	credsEnc, err := storage.EncryptCredentials(s.masterKey, creds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encrypt creds: "+err.Error())
		return
	}

	target, err := s.db.CreateStorageTarget(r.Context(),
		tenant.ID, "primary", "s3",
		req.StorageEndpoint, req.StorageBucket, req.StorageRegion,
		credsEnc, true, req.StoragePathStyle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create storage: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant":         tenant,
		"storage_target": target,
	})
}

func (s *Server) handleCreateStorageTarget(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && claims.Role != "customer_admin" {
		writeError(w, http.StatusForbidden, "insufficient role")
		return
	}

	var req struct {
		TenantID  string `json:"tenant_id"`
		Label     string `json:"label"`
		Endpoint  string `json:"endpoint"`
		Bucket    string `json:"bucket"`
		Region    string `json:"region"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		IsPrimary bool   `json:"is_primary"`
		PathStyle bool   `json:"path_style"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tenantID := req.TenantID
	if claims.Role != "vendor_admin" {
		tenantID = claims.TenantID
	}

	creds := storage.Credentials{AccessKeyID: req.AccessKey, SecretAccessKey: req.SecretKey}
	credsEnc, err := storage.EncryptCredentials(s.masterKey, creds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	target, err := s.db.CreateStorageTarget(r.Context(), tenantID, req.Label, "s3",
		req.Endpoint, req.Bucket, req.Region, credsEnc, req.IsPrimary, req.PathStyle)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, target)
}
