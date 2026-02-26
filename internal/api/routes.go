package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChronoCoders/sentra/internal/auth"
	"github.com/ChronoCoders/sentra/internal/config"
	"github.com/ChronoCoders/sentra/internal/control"
	"github.com/ChronoCoders/sentra/internal/models"
	"github.com/ChronoCoders/sentra/internal/store"
	"github.com/ChronoCoders/sentra/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg    *config.Config
	store  *store.Store
	client control.AgentClient
	hub    *ws.Hub
	bus    *control.EventBus
	auth   *auth.JWTManager
	router *chi.Mux
}

func NewServer(cfg *config.Config, store *store.Store, client control.AgentClient, hub *ws.Hub, bus *control.EventBus) *Server {
	// Initialize router
	r := chi.NewRouter()

	s := &Server{
		cfg:    cfg,
		store:  store,
		client: client,
		hub:    hub,
		bus:    bus,
		auth:   auth.NewJWTManager(cfg.JWTSecret),
		router: r,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Public routes
	s.router.Post("/api/login", s.handleLogin)
	s.router.Post("/api/report", s.handleReport)    // Agent reporting
	s.router.Get("/api/cert", s.handleCertDownload) // Download CA cert

	// Authenticated routes
	s.router.Group(func(r chi.Router) {
		r.Use(s.jwtMiddleware)

		// Viewer access (includes Admin)
		r.Group(func(r chi.Router) {
			// Add middleware to check role if needed, e.g. s.RequireRole("viewer")
			// For now, assume any valid token is viewer

			r.Get("/api/health", s.handleHealth)
			r.Get("/api/status", s.handleStatus)
			r.Get("/ws", s.handleWs)
		})
	})

	// Static Files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web"))
	FileServer(s.router, "/", filesDir)
}

func (s *Server) handleCertDownload(w http.ResponseWriter, r *http.Request) {
	certPath := s.cfg.TLSCert
	if certPath == "" {
		certPath = "cert.pem"
	}

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		http.Error(w, "Certificate not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", "attachment; filename=sentra-ca.crt")
	http.ServeFile(w, r, certPath)
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
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
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		http.Error(w, "missing server_id", http.StatusBadRequest)
		return
	}
	status, err := s.client.GetStatus(r.Context(), serverID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get status")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if status == nil {
		http.Error(w, "server not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleWs(w http.ResponseWriter, r *http.Request) {
	// JWT validation handled by middleware
	client := ws.ServeWs(s.hub, w, r)
	if client != nil {
		events := s.client.GetAllStatuses()
		for _, event := range events {
			client.Send(event)
		}
	}
}

// jwtMiddleware moved to middleware.go

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
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
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := s.auth.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate token")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	// Simple token check
	if s.cfg.AuthToken != "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+s.cfg.AuthToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var event models.StatusEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	event.Time = time.Now()
	s.bus.Publish(event)

	w.WriteHeader(http.StatusOK)
}
