package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/luongquochai/s3-performance-test/internal/config"
	"github.com/luongquochai/s3-performance-test/internal/s3"
)

// PresignedURLResponse represents the response format for presigned URL generation
type PresignedURLResponse struct {
	URLs []URLEntry `json:"urls"`
}

// URLEntry represents a single presigned URL entry
type URLEntry struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// Server manages the HTTP API server
type Server struct {
	router   *mux.Router
	server   *http.Server
	s3Client *s3.Client
	config   *config.Config
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, s3Client *s3.Client) *Server {
	router := mux.NewRouter()

	server := &Server{
		router:   router,
		s3Client: s3Client,
		config:   cfg,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.API.Port),
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}

	// Register routes
	server.registerRoutes()

	return server
}

// registerRoutes sets up all API routes
func (s *Server) registerRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.healthCheckHandler).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/presigned-urls", s.getPresignedURLsHandler).Methods("GET")
	api.HandleFunc("/presigned-url", s.getSinglePresignedURLHandler).Methods("GET")
}

// Start begins the API server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the API server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

// healthCheckHandler handles health check requests
func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// getPresignedURLsHandler generates multiple presigned URLs
func (s *Server) getPresignedURLsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	countParam := r.URL.Query().Get("count")
	expiryParam := r.URL.Query().Get("expiry")

	// Default values
	count := 1
	expiry := int64(3600) // 1 hour in seconds

	// Parse count parameter
	if countParam != "" {
		fmt.Sscanf(countParam, "%d", &count)
		if count <= 0 {
			count = 1
		} else if count > 100 {
			count = 100 // Limit maximum count
		}
	}

	// Parse expiry parameter
	if expiryParam != "" {
		fmt.Sscanf(expiryParam, "%d", &expiry)
		if expiry <= 0 {
			expiry = 3600
		} else if expiry > 86400 {
			expiry = 86400 // Limit to 24 hours
		}
	}

	// Generate presigned URLs
	urls, err := s.s3Client.GenerateMultiplePresignedURLs(
		r.Context(),
		"test",
		count,
		time.Duration(expiry)*time.Second,
	)

	if err != nil {
		log.Printf("Error generating presigned URLs: %v", err)
		http.Error(w, "Failed to generate presigned URLs", http.StatusInternalServerError)
		return
	}

	// Format response
	urlEntries := make([]URLEntry, 0, len(urls))
	for key, url := range urls {
		urlEntries = append(urlEntries, URLEntry{
			URL: url,
			Key: key,
		})
	}

	response := PresignedURLResponse{
		URLs: urlEntries,
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// getSinglePresignedURLHandler generates a single presigned URL
func (s *Server) getSinglePresignedURLHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	expiryParam := r.URL.Query().Get("expiry")
	keyParam := r.URL.Query().Get("key")

	// Default values
	expiry := int64(3600) // 1 hour in seconds

	// Parse expiry parameter
	if expiryParam != "" {
		fmt.Sscanf(expiryParam, "%d", &expiry)
		if expiry <= 0 {
			expiry = 3600
		} else if expiry > 86400 {
			expiry = 86400 // Limit to 24 hours
		}
	}

	// Generate key if not provided
	if keyParam == "" {
		keyParam = fmt.Sprintf("test/%s-%d.dat", time.Now().UTC().Format("20060102-150405"), 0)
	}

	// Generate presigned URL
	url, err := s.s3Client.GeneratePresignedURL(
		r.Context(),
		keyParam,
		time.Duration(expiry)*time.Second,
	)

	if err != nil {
		log.Printf("Error generating presigned URL: %v", err)
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	// Format response
	urlEntry := URLEntry{
		URL: url,
		Key: keyParam,
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(urlEntry); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
