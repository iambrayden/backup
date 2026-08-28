package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/waylontech/backup/server/internal/auth"
	"github.com/waylontech/backup/server/internal/db"
	"github.com/waylontech/backup/server/internal/storage"
)

type Server struct {
	db        *db.DB
	jwtSecret []byte
	masterKey []byte
	log       *zap.Logger
}

func NewServer(database *db.DB, jwtSecret, masterKey []byte, log *zap.Logger) *Server {
	return &Server{
		db:        database,
		jwtSecret: jwtSecret,
		masterKey: masterKey,
		log:       log,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(zapLogger(s.log))
	r.Use(middleware.Recoverer)
	r.Use(corsHeaders)

	r.Get("/healthz", s.handleHealthz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/enroll", s.handleEnroll)

		r.Group(func(r chi.Router) {
			r.Use(s.requireUser)

			r.Get("/tenants", s.handleListTenants)
			r.Post("/tenants", s.handleCreateTenant)
			r.Get("/tenants/{tenantID}", s.handleGetTenant)

			r.Get("/devices", s.handleListDevices)
			r.Get("/devices/{deviceID}", s.handleGetDevice)
			r.Post("/devices/{deviceID}/jobs", s.handleTriggerJob)
			r.Get("/devices/{deviceID}/jobs", s.handleListJobs)
			r.Get("/devices/{deviceID}/snapshots", s.handleListSnapshots)

			r.Get("/jobs/{jobID}", s.handleGetJob)

			r.Get("/snapshots/{snapshotID}/tree", s.handleFileTree)
			r.Get("/snapshots/{snapshotID}/download", s.handleDownloadFile)

			r.Post("/enrollment-tokens", s.handleCreateEnrollmentToken)
			r.Post("/storage-targets", s.handleCreateStorageTarget)
		})

		r.Group(func(r chi.Router) {
			r.Use(s.requireDevice)

			r.Post("/agent/heartbeat", s.handleHeartbeat)
			r.Get("/agent/dek", s.handleGetDEK)
			r.Post("/agent/chunks/check", s.handleChunkCheck)
			r.Post("/agent/chunks/presign", s.handleChunkPresign)
			r.Post("/agent/chunks/confirm", s.handleChunkConfirm)
			r.Post("/agent/snapshots", s.handleCreateSnapshot)
			r.Put("/agent/jobs/{jobID}", s.handleUpdateJob)
		})
	})

	return r
}

func (s *Server) newStorageClient(ctx interface{ Value(any) any }, tenantID string) (*storage.Client, error) {
	return nil, nil
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func corsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func zapLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
			)
		})
	}
}

type contextKey string

const (
	ctxClaims    contextKey = "claims"
	ctxDeviceKey contextKey = "device"
)

func claimsFromCtx(r *http.Request) *auth.Claims {
	v, _ := r.Context().Value(ctxClaims).(*auth.Claims)
	return v
}

func deviceFromCtx(r *http.Request) *db.Device {
	v, _ := r.Context().Value(ctxDeviceKey).(*db.Device)
	return v
}
