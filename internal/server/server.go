package server

import (
	"context"
	"control-plane/internal/config"
	"control-plane/internal/domain"
	"control-plane/internal/handler"
	"fmt"
	"net/http"
	"time"

	customMiddleware "control-plane/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	http   *http.Server
	router *chi.Mux
}

type Dependencies struct {
	AppService  domain.ApplicationService
	DepService  domain.DeploymentService
	UserService domain.UserService
}

func New(cfg *config.Config, deps *Dependencies) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(customMiddleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.SetHeader("Content-Type", "application/json"))

		appHandler := handler.NewApplicationHandler(deps.AppService, deps.DepService)
		appHandler.Routes(r)

		userHandler := handler.NewUserHandler(deps.UserService)
		userHandler.Routes(r)
	})

	fmt.Println("--- Registered Routes ---")
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("%s %s\n", method, route)
		return nil
	}
	if err := chi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Walking err: %s\n", err.Error())
	}
	fmt.Println("=========================")

	return &Server{
		router: r,
		http: &http.Server{
			Addr:         cfg.ServerAddr,
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}
