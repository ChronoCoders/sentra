package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ChronoCoders/sentra/internal/auth"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg    *config.Config
	store  *store.Store
	client control.AgentClient
	auth   *auth.JWTManager
	router *chi.Mux
}

func NewServer(cfg *config.Config, store *store.Store, client control.AgentClient) *Server {
	s := &Server{
		cfg:    cfg,
		store:  store,
		client: client,
		auth:   auth.NewJWTManager(cfg.JWTSecret),
		router: chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Sentra Control Plane v2 is running"))
	})

	s.router.Get("/health", s.handleHealth)

	s.router.Post("/api/login", s.handleLogin)

	s.router.Group(func(r chi.Router) {
		r.Use(s.jwtMiddleware)
		r.Get("/api/status", s.handleStatus)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.client.GetStatus(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get status")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenStr := parts[1]
		token, err := s.auth.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	token, err := s.auth.GenerateAccessToken(user.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate token")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
