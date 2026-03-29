package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-chi/chi/v5"
)

func dockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func DockerList(w http.ResponseWriter, r *http.Request) {
	c, err := dockerClient()
	if err != nil {
		http.Error(w, "docker unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	list, err := c.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

type dockerExecReq struct {
	Cmd []string `json:"cmd"`
}

func DockerExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	var req dockerExecReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Cmd) == 0 {
		http.Error(w, "cmd required", http.StatusBadRequest)
		return
	}
	c, err := dockerClient()
	if err != nil {
		http.Error(w, "docker unavailable", http.StatusServiceUnavailable)
		return
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	execCfg := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          req.Cmd,
	}
	eid, err := c.ContainerExecCreate(ctx, id, execCfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	att, err := c.ContainerExecAttach(ctx, eid.ID, container.ExecStartOptions{Detach: false, Tty: false})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer att.Close()
	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, att.Reader); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	combined := outBuf.String() + errBuf.String()
	inspect, err := c.ContainerExecInspect(ctx, eid.ID)
	exit := 0
	if err == nil {
		exit = inspect.ExitCode
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"exit_code": exit,
		"output":    combined,
	})
}

func DockerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	tail := r.URL.Query().Get("tail")
	tailN := 200
	if tail != "" {
		if n, err := strconv.Atoi(tail); err == nil {
			tailN = n
		}
	}
	c, err := dockerClient()
	if err != nil {
		http.Error(w, "docker unavailable", http.StatusServiceUnavailable)
		return
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	opts := container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: strconv.Itoa(tailN)}
	rc, err := c.ContainerLogs(ctx, id, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, rc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	combined := outBuf.String() + errBuf.String()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"logs": combined})
}

func DockerPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Image string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Image == "" {
		http.Error(w, "image required", http.StatusBadRequest)
		return
	}
	c, err := dockerClient()
	if err != nil {
		http.Error(w, "docker unavailable", http.StatusServiceUnavailable)
		return
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	rc, err := c.ImagePull(ctx, body.Image, image.PullOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	out, _ := io.ReadAll(rc)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "stream": string(out)})
}
