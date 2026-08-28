package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	var (
		devices interface{}
		err     error
	)
	if claims.Role == "vendor_admin" {
		devices, err = s.db.ListAllDevices(r.Context())
	} else {
		devices, err = s.db.ListDevices(r.Context(), claims.TenantID)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (s *Server) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	dev, err := s.db.GetDeviceByID(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && dev.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	writeJSON(w, http.StatusOK, dev)
}

func (s *Server) handleTriggerJob(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	dev, err := s.db.GetDeviceByID(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && dev.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	job, err := s.db.CreateBackupJob(r.Context(), deviceID, dev.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	dev, err := s.db.GetDeviceByID(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && dev.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	jobs, err := s.db.ListJobsForDevice(r.Context(), deviceID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	job, err := s.db.GetBackupJobByID(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && job.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")
	dev, err := s.db.GetDeviceByID(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	claims := claimsFromCtx(r)
	if claims.Role != "vendor_admin" && dev.TenantID != claims.TenantID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	snapshots, err := s.db.ListSnapshotsForDevice(r.Context(), deviceID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snapshots)
}
