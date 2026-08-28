package api

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/waylontech/backup/server/internal/auth"
	"github.com/waylontech/backup/server/internal/db"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.IssueToken(s.jwtSecret, user.ID, user.TenantID, user.Role, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token        string `json:"token"`
		Hostname     string `json:"hostname"`
		OSType       string `json:"os_type"`
		Arch         string `json:"arch"`
		AgentVersion string `json:"agent_version"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	tenantID, err := s.db.UseEnrollmentToken(r.Context(), db.HashToken(req.Token))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired enrollment token")
		return
	}

	dev, err := s.db.CreateDevice(r.Context(), tenantID, req.Hostname, req.OSType, req.Arch, req.AgentVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create device: "+err.Error())
		return
	}

	deviceToken := uuid.NewString()
	if err := s.db.CreateDeviceToken(r.Context(), dev.ID, hashToken(deviceToken)); err != nil {
		writeError(w, http.StatusInternalServerError, "create token: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"device":       dev,
		"device_token": deviceToken,
	})
}

func (s *Server) handleCreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && claims.Role != "customer_admin" {
		writeError(w, http.StatusForbidden, "insufficient role")
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	decode(r, &req)

	token := uuid.NewString()
	expiresAt := time.Now().Add(72 * time.Hour)

	if err := s.db.CreateEnrollmentToken(r.Context(), claims.TenantID, db.HashToken(token), req.Label, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":      token,
		"expires_at": expiresAt,
		"label":      req.Label,
	})
}
