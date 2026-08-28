package api

import (
	"context"
	"net/http"
	"strings"
)

func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization")
			return
		}
		claims, err := s.parseJWT(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requireDevice(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization")
			return
		}
		dev, err := s.db.GetDeviceByTokenHash(r.Context(), hashToken(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid device token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxDeviceKey, dev)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return h[7:]
	}
	return ""
}
