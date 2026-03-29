package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/rcservers/rcserver/internal/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type wsMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func Terminal(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = cfg
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		cmd := exec.Command("/bin/sh", "-l")
		cmd.Env = append(os.Environ(), "TERM=xterm-256color")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to start shell: "+err.Error()))
			return
		}
		defer func() { _ = ptmx.Close(); _ = cmd.Process.Kill() }()

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					_ = conn.WriteJSON(wsMsg{Type: "output", Data: string(buf[:n])})
				}
				if err != nil {
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var m wsMsg
				if err := json.Unmarshal(data, &m); err != nil {
					continue
				}
				switch m.Type {
				case "input":
					_, _ = ptmx.Write([]byte(m.Data))
				case "resize":
					if m.Cols > 0 && m.Rows > 0 {
						_ = pty.Setsize(ptmx, &pty.Winsize{Cols: m.Cols, Rows: m.Rows})
					}
				case "ping":
					_ = conn.WriteJSON(wsMsg{Type: "pong", Data: ""})
				}
			}
		}()
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(24 * time.Hour):
		}
	}
}
