package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/rcservers/rcserver/internal/config"
)

const HeaderKey = "X-RC-Key"

func Middleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg == nil || cfg.Hash == "" {
				http.Error(w, "server misconfigured", http.StatusInternalServerError)
				return
			}
			token := extractToken(r)
			if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Hash)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if k := r.Header.Get(HeaderKey); k != "" {
		return strings.TrimSpace(k)
	}
	return ""
}
