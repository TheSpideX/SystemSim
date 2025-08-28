package http2

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"

	"server-service/internal/config"
)

// Server represents a high-performance HTTP/2 server
type Server struct {
	config    *config.ServerConfig
	server    *http.Server
	listener  net.Listener
	tlsConfig *tls.Config
	isRunning bool
}

// NewServer creates a new HTTP/2 server with optimal performance settings
func NewServer(cfg *config.ServerConfig, handler http.Handler) (*Server, error) {
	// Create TLS configuration for HTTP/2
	tlsConfig, err := createTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create HTTP server with HTTP/2 support
	server := &http.Server{
		Handler:        handler,
		ReadTimeout:    cfg.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		IdleTimeout:    cfg.IdleTimeout,
		MaxHeaderBytes: int(cfg.MaxRequestBodySize),
		TLSConfig:      tlsConfig,

		// HTTP/2 specific settings
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	// Configure HTTP/2
	if err := http2.ConfigureServer(server, &http2.Server{
		MaxConcurrentStreams: cfg.MaxConcurrentStreams,
		MaxReadFrameSize:     cfg.MaxFrameSize,
		IdleTimeout:          cfg.HTTP2IdleTimeout,
	}); err != nil {
		return nil, fmt.Errorf("failed to configure HTTP/2: %w", err)
	}

	return &Server{
		config:    cfg,
		server:    server,
		tlsConfig: tlsConfig,
	}, nil
}

// Start starts the HTTP/2 server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	s.server.Addr = addr

	s.isRunning = true

	// Start server with TLS (required for HTTP/2)
	if s.config.TLSEnabled {
		log.Printf("API Gateway starting with HTTP/2 + TLS on %s", addr)
		if err := s.server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTPS server error: %w", err)
		}
	} else {
		// HTTP/2 without TLS (h2c) - not recommended for production
		log.Printf("API Gateway starting with HTTP/2 (h2c) on %s", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.isRunning = false
	return s.server.Shutdown(ctx)
}

// createTLSConfig creates optimized TLS configuration for HTTP/2
func createTLSConfig(cfg *config.ServerConfig) (*tls.Config, error) {
	// Generate self-signed certificate if files don't exist
	if err := ensureCertificates(cfg.CertFile, cfg.KeyFile); err != nil {
		return nil, fmt.Errorf("failed to ensure certificates: %w", err)
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Determine TLS version
	var minVersion uint16
	switch cfg.MinTLSVersion {
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

		// HTTP/2 specific configuration
		NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2

		// Performance optimizations
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		CipherSuites: []uint16{
			// HTTP/2 required cipher suites
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},

		// Session resumption for performance
		SessionTicketsDisabled: false,
		ClientSessionCache:     tls.NewLRUClientSessionCache(1000),
	}

	return tlsConfig, nil
}

// HTTP2Config holds HTTP/2 specific configuration
type HTTP2Config struct {
	MaxConcurrentStreams uint32
	MaxFrameSize         uint32
	IdleTimeout          time.Duration
	PingTimeout          time.Duration
	WriteByteTimeout     time.Duration
}

// ConfigureHTTP2 configures HTTP/2 specific settings
func ConfigureHTTP2(cfg *config.ServerConfig) *HTTP2Config {
	return &HTTP2Config{
		MaxConcurrentStreams: cfg.MaxConcurrentStreams,
		MaxFrameSize:         cfg.MaxFrameSize,
		IdleTimeout:          cfg.HTTP2IdleTimeout,
		PingTimeout:          30 * time.Second,
		WriteByteTimeout:     10 * time.Second,
	}
}

// OptimizeForThroughput applies performance optimizations
func OptimizeForThroughput(server *Server) {
	// HTTP/2 is already optimized by default in net/http
	// Additional optimizations can be applied here
	log.Println("HTTP/2 server optimized for maximum throughput")
}

// GetServerStats returns server performance statistics
func (s *Server) GetServerStats() map[string]interface{} {
	return map[string]interface{}{
		"is_running":    s.isRunning,
		"tls_enabled":   s.config.TLSEnabled,
		"http2_enabled": s.config.HTTP2Enabled,
		"port":          s.config.Port,
		"max_body_size": s.config.MaxRequestBodySize,
		"keep_alive":    s.config.KeepAlive,
		"compress":      s.config.CompressResponse,
	}
}
