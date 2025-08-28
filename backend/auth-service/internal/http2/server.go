package http2

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/systemsim/auth-service/internal/config"
	"golang.org/x/net/http2"
)

// Server represents an HTTP/2 server
type Server struct {
	server   *http.Server
	config   *config.Config
	router   *gin.Engine
	certFile string
	keyFile  string
}

// NewServer creates a new HTTP/2 server
func NewServer(cfg *config.Config, router *gin.Engine) (*Server, error) {
	// Validate TLS configuration
	certFile := cfg.Server.CertFile
	keyFile := cfg.Server.KeyFile

	// Check if certificate files exist, create self-signed if not
	if err := ensureCertificates(certFile, keyFile); err != nil {
		return nil, fmt.Errorf("failed to ensure certificates: %w", err)
	}

	// Create TLS configuration
	tlsConfig, err := createTLSConfig(cfg, certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create HTTP server with HTTP/2 support
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		TLSConfig:    tlsConfig,
	}

	// Configure HTTP/2 (required - no fallback)
	http2Server := &http2.Server{
		MaxConcurrentStreams: cfg.Server.MaxConcurrentStreams,
		MaxReadFrameSize:     cfg.Server.MaxFrameSize,
		IdleTimeout:          cfg.Server.HTTP2IdleTimeout,
	}

	if err := http2.ConfigureServer(httpServer, http2Server); err != nil {
		return nil, fmt.Errorf("failed to configure HTTP/2: %w", err)
	}

	return &Server{
		server:   httpServer,
		config:   cfg,
		router:   router,
		certFile: certFile,
		keyFile:  keyFile,
	}, nil
}

// Start starts the HTTP/2-only server with strict protocol enforcement
func (s *Server) Start() error {
	log.Printf("Starting strict HTTP/2-only server with TLS on port %s (API Gateway communication)", s.config.Server.Port)
	return s.server.ListenAndServeTLS(s.certFile, s.keyFile)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// createTLSConfig creates a TLS configuration for HTTP/2
func createTLSConfig(cfg *config.Config, certFile, keyFile string) (*tls.Config, error) {
	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Determine minimum TLS version
	var minVersion uint16
	switch cfg.Server.MinTLSVersion {
	case "1.3":
		minVersion = tls.VersionTLS13
	case "1.2":
		minVersion = tls.VersionTLS12
	default:
		minVersion = tls.VersionTLS12
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   minVersion,
		MaxVersion:   tls.VersionTLS13,
		
		// Cipher suites for TLS 1.2 (TLS 1.3 uses its own cipher suites)
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		
		// Prefer server cipher suites
		PreferServerCipherSuites: true,

		// Strict HTTP/2 only - absolutely no HTTP/1.1 fallback
		NextProtos: []string{"h2"},

		// Custom verification to ensure HTTP/2
		VerifyConnection: func(cs tls.ConnectionState) error {
			if cs.NegotiatedProtocol != "h2" {
				return fmt.Errorf("HTTP/2-only server: protocol %s not allowed, only h2 accepted", cs.NegotiatedProtocol)
			}
			return nil
		},
	}

	return tlsConfig, nil
}

// ensureCertificates checks if certificates exist, creates self-signed ones if not
func ensureCertificates(certFile, keyFile string) error {
	// Check if both files exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			log.Printf("Using existing certificates: %s, %s", certFile, keyFile)
			return nil
		}
	}

	// Create directory if it doesn't exist
	certDir := filepath.Dir(certFile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	log.Printf("Generating self-signed certificates for development: %s, %s", certFile, keyFile)
	return generateSelfSignedCert(certFile, keyFile)
}
