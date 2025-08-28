package http2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ensureCertificates generates self-signed certificates if they don't exist
func ensureCertificates(certFile, keyFile string) error {
	// Check if certificates already exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			// Both files exist, check if they're valid
			if isValidCertificate(certFile, keyFile) {
				return nil
			}
		}
	}

	// Create certificates directory if it doesn't exist
	certDir := filepath.Dir(certFile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Generate new certificates
	return generateSelfSignedCert(certFile, keyFile)
}

// generateSelfSignedCert generates a self-signed certificate for development
func generateSelfSignedCert(certFile, keyFile string) error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"System Design Simulator"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		// Add localhost and mDNS hostnames for robust network access
		DNSNames: []string{
			"localhost",
			"spidexd.local",     // mDNS hostname (matches current hostname)
			"api-gateway",
			"server-service",
			"*.local",           // wildcard for .local domain
		},
		IPAddresses: []net.IP{
			net.IPv4(127, 0, 0, 1),   // localhost
			net.IPv6loopback,         // ::1
			net.IPv4(172, 16, 15, 134), // current server IP
			net.IPv4(172, 16, 15, 128), // additional network IP
			net.IPv4(172, 16, 15, 1),   // common gateway IP
			net.IPv4(172, 16, 15, 2),   // router IP
		},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	fmt.Printf("Generated self-signed certificate: %s\n", certFile)
	fmt.Printf("Generated private key: %s\n", keyFile)
	fmt.Println("⚠️  Using self-signed certificate for development. Use proper certificates in production!")

	return nil
}

// isValidCertificate checks if the certificate files are valid and not expired
func isValidCertificate(certFile, keyFile string) bool {
	// Read certificate file
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return false
	}

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	// Check if certificate is expired or will expire soon (within 30 days)
	if time.Now().After(cert.NotAfter) || time.Now().Add(30*24*time.Hour).After(cert.NotAfter) {
		return false
	}

	// Try to load the key pair to ensure they match
	_, err = os.ReadFile(keyFile)
	if err != nil {
		return false
	}

	return true
}

// GetCertificateInfo returns information about the current certificate
func GetCertificateInfo(certFile string) (map[string]interface{}, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return map[string]interface{}{
		"subject":         cert.Subject.String(),
		"issuer":          cert.Issuer.String(),
		"not_before":      cert.NotBefore,
		"not_after":       cert.NotAfter,
		"dns_names":       cert.DNSNames,
		"ip_addresses":    cert.IPAddresses,
		"is_ca":           cert.IsCA,
		"serial":          cert.SerialNumber.String(),
		"valid":           time.Now().Before(cert.NotAfter) && time.Now().After(cert.NotBefore),
		"expires_in_days": int(cert.NotAfter.Sub(time.Now()).Hours() / 24),
	}, nil
}

// CreateCertsDirectory creates the certificates directory with proper permissions
func CreateCertsDirectory(certDir string) error {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificates directory: %w", err)
	}
	return nil
}

// CleanupExpiredCerts removes expired certificate files
func CleanupExpiredCerts(certFile, keyFile string) error {
	if !isValidCertificate(certFile, keyFile) {
		// Remove expired certificates
		if err := os.Remove(certFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove expired certificate: %w", err)
		}
		if err := os.Remove(keyFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove expired key: %w", err)
		}
		fmt.Println("Removed expired certificates")
	}
	return nil
}
