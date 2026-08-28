package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/waylontech/backup/server/internal/auth"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *Server) parseJWT(token string) (*auth.Claims, error) {
	return auth.ParseToken(s.jwtSecret, token)
}
