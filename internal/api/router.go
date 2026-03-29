package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rcservers/rcserver/internal/auth"
	"github.com/rcservers/rcserver/internal/config"
	"github.com/rcservers/rcserver/internal/handlers"
	"github.com/rcservers/rcserver/internal/ratelimit"
)

func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handlers.Health)

	store := ratelimit.New(cfg.RatePerSecond, cfg.RateBurst)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.Middleware(cfg))
		r.Use(ratelimit.Middleware(store))

		r.Get("/system", handlers.System)
		r.Post("/exec", handlers.Exec(cfg))
		r.Get("/files", handlers.Files(cfg))
		r.Post("/files/upload", handlers.Upload(cfg))
		r.Get("/ws/terminal", handlers.Terminal(cfg))

		r.Get("/docker/containers", handlers.DockerList)
		r.Post("/docker/containers/{id}/exec", handlers.DockerExec)
		r.Get("/docker/containers/{id}/logs", handlers.DockerLogs)
		r.Post("/docker/pull", handlers.DockerPull)

		r.Get("/nginx/sites", handlers.NginxList(cfg))
		r.Get("/nginx/sites/{name}", handlers.NginxGet(cfg))
		r.Put("/nginx/sites/{name}", handlers.NginxPut(cfg))

		r.Post("/deploy/static", handlers.Deploy(cfg))
	})

	return r
}
