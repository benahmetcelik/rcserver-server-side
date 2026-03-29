package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rcservers/rcserver/internal/config"
)

func nginxAllowedPath(cfg *config.Config, name string) (string, error) {
	base := filepath.Clean(cfg.NginxSitesDir)
	if name == "" || strings.Contains(name, "..") {
		return "", os.ErrPermission
	}
	full := filepath.Join(base, filepath.Base(name))
	if !strings.HasPrefix(full, base+string(filepath.Separator)) && full != base {
		return "", os.ErrPermission
	}
	return full, nil
}

func NginxList(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := filepath.Clean(cfg.NginxSitesDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				names = append(names, e.Name())
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"sites": names})
	}
}

func NginxGet(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		full, err := nginxAllowedPath(cfg, name)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		data, err := os.ReadFile(full)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"content": string(data), "path": full})
	}
}

func NginxPut(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := chi.URLParam(r, "name")
		full, err := nginxAllowedPath(cfg, name)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var body struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(full, []byte(body.Content), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := nginxTestAndReload(); err != nil {
			http.Error(w, "nginx test/reload failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func nginxTestAndReload() error {
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	if err != nil {
		return err
	}
	_ = out
	_ = exec.Command("systemctl", "reload", "nginx").Run()
	return nil
}
