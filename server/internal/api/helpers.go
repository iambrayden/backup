package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/waylontech/backup/server/internal/auth"
	"github.com/waylontech/backup/server/internal/db"
)

const maxRequestBytes = 10 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, maxRequestBytes)).Decode(v)
}

func hashToken(token string) string {
	return db.HashToken(token)
}

func (s *Server) parseJWT(token string) (*auth.Claims, error) {
	return auth.ParseToken(s.jwtSecret, token)
}
