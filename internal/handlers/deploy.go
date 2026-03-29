package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rcservers/rcserver/internal/config"
)

type DeployRequest struct {
	SiteName   string `json:"site_name"`
	ServerName string `json:"server_name"`
	RootPath   string `json:"root_path"`
}

func Deploy(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req DeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.SiteName == "" || req.ServerName == "" {
			http.Error(w, "site_name and server_name required", http.StatusBadRequest)
			return
		}
		safeName := filepath.Base(req.SiteName)
		if safeName != req.SiteName || strings.Contains(safeName, "..") {
			http.Error(w, "invalid site_name", http.StatusBadRequest)
			return
		}
		root := req.RootPath
		if root == "" {
			root = filepath.Join(cfg.DeployDir, safeName)
		}
		fullRoot, err := resolveAllowedPath(cfg, root)
		if err != nil {
			http.Error(w, "root path not allowed", http.StatusForbidden)
			return
		}
		if err := os.MkdirAll(fullRoot, 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		conf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.html index.htm;
    location / {
        try_files $uri $uri/ =404;
    }
}
`, req.ServerName, fullRoot)
		siteFile := filepath.Join(cfg.NginxSitesDir, safeName)
		if err := os.MkdirAll(filepath.Dir(siteFile), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(siteFile, []byte(conf), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := nginxTestAndReload(); err != nil {
			http.Error(w, "nginx: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"site_config": siteFile,
			"web_root":    fullRoot,
		})
	}
}
