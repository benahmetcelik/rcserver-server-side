package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rcservers/rcserver/internal/config"
)

type fileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModUnix int64  `json:"mod_unix"`
}

func resolveAllowedPath(cfg *config.Config, p string) (string, error) {
	clean := filepath.Clean(p)
	for _, root := range cfg.FileRoots {
		cr := filepath.Clean(root)
		if clean == cr {
			return clean, nil
		}
		sep := string(filepath.Separator)
		if strings.HasPrefix(clean, cr+sep) {
			return clean, nil
		}
	}
	return "", os.ErrPermission
}

func Files(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		if p == "" {
			p = cfg.FileRoots[0]
		}
		full, err := resolveAllowedPath(cfg, p)
		if err != nil {
			http.Error(w, "path not allowed", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet:
			fi, err := os.Stat(full)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			if !fi.IsDir() {
				if r.URL.Query().Get("download") == "1" {
					w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(full))
					http.ServeFile(w, r, full)
					return
				}
				data, err := os.ReadFile(full)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(data)
				return
			}
			entries, err := os.ReadDir(full)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			out := make([]fileEntry, 0, len(entries))
			for _, e := range entries {
				info, err := e.Info()
				if err != nil {
					continue
				}
				child := filepath.Join(full, e.Name())
				out = append(out, fileEntry{
					Name:    e.Name(),
					Path:    child,
					IsDir:   info.IsDir(),
					Size:    info.Size(),
					ModUnix: info.ModTime().Unix(),
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": out, "path": full})
		case http.MethodPost:
			var body struct {
				Path    string `json:"path"`
				Content string `json:"content"`
				Base64  bool   `json:"base64"`
				IsDir   bool   `json:"is_dir"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			dest, err := resolveAllowedPath(cfg, body.Path)
			if err != nil {
				http.Error(w, "path not allowed", http.StatusForbidden)
				return
			}
			if body.IsDir {
				if err := os.MkdirAll(dest, 0o755); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusCreated)
				return
			}
			var raw []byte
			if body.Base64 {
				raw, err = base64.StdEncoding.DecodeString(body.Content)
				if err != nil {
					http.Error(w, "invalid base64", http.StatusBadRequest)
					return
				}
			} else {
				raw = []byte(body.Content)
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := os.WriteFile(dest, raw, 0o644); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			target, err := resolveAllowedPath(cfg, r.URL.Query().Get("path"))
			if err != nil || target == "" {
				http.Error(w, "path required", http.StatusBadRequest)
				return
			}
			for _, root := range cfg.FileRoots {
				if filepath.Clean(target) == filepath.Clean(root) {
					http.Error(w, "cannot delete root", http.StatusForbidden)
					return
				}
			}
			if err := os.RemoveAll(target); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPut:
			_, _ = io.Copy(io.Discard, r.Body)
			http.Error(w, "use POST with json", http.StatusMethodNotAllowed)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func Upload(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		destDir := r.FormValue("path")
		if destDir == "" {
			destDir = cfg.WWWRoot
		}
		base, err := resolveAllowedPath(cfg, destDir)
		if err != nil {
			http.Error(w, "path not allowed", http.StatusForbidden)
			return
		}
		name := hdr.Filename
		if name == "" {
			name = "upload.bin"
		}
		dest := filepath.Join(base, filepath.Base(name))
		if !strings.HasPrefix(filepath.Clean(dest), filepath.Clean(base)) {
			http.Error(w, "invalid name", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f, err := os.Create(dest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		if _, err := io.Copy(f, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"path": dest})
	}
}
