package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"time"

	"github.com/rcservers/rcserver/internal/config"
	"github.com/rcservers/rcserver/internal/security"
)

type ExecRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

type ExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func Exec(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Command == "" {
			http.Error(w, "command required", http.StatusBadRequest)
			return
		}
		if !security.CommandAllowed(req.Command) || !security.ArgAllowed(req.Args) {
			http.Error(w, "command blocked by policy", http.StatusForbidden)
			return
		}
		timeout := time.Duration(cfg.ExecTimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 120 * time.Second
		}
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, req.Command, req.Args...)
		if req.Cwd != "" {
			cmd.Dir = req.Cwd
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		exit := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				exit = ee.ExitCode()
			} else {
				exit = -1
			}
		}
		maxOut := cfg.MaxOutputBytes
		if maxOut <= 0 {
			maxOut = 2 * 1024 * 1024
		}
		outStr := stdout.String()
		errStr := stderr.String()
		if len(outStr) > maxOut {
			outStr = outStr[:maxOut] + "\n...truncated"
		}
		if len(errStr) > maxOut {
			errStr = errStr[:maxOut] + "\n...truncated"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ExecResponse{
			ExitCode: exit,
			Stdout:   outStr,
			Stderr:   errStr,
		})
	}
}
