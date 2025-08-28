#!/bin/bash

# Generate Certificate for Cross-Device Access
# This script creates a self-signed certificate that can be easily installed on other devices

set -e

CERT_DIR="certs"
HOSTNAME=$(hostname)
CURRENT_IP=$(ip route get 1 | awk '{print $7; exit}')

echo "🔐 Generating certificate for cross-device access..."
echo "📱 Hostname: $HOSTNAME"
echo "🌐 Current IP: $CURRENT_IP"

# Create certs directory
mkdir -p $CERT_DIR

# Create OpenSSL config file
cat > $CERT_DIR/openssl.conf << EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = v3_req

[dn]
C=US
ST=Development
L=Local
O=System Design Simulator
OU=Development
CN=siked.local

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = $HOSTNAME.local
DNS.3 = api-gateway
DNS.4 = server-service
DNS.5 = *.local
IP.1 = 127.0.0.1
IP.2 = ::1
IP.3 = $CURRENT_IP
IP.4 = 172.16.15.128
IP.5 = 172.16.15.134
IP.6 = 172.16.15.1
IP.7 = 172.16.15.2
IP.8 = 192.168.1.1
IP.9 = 192.168.0.1
IP.10 = 10.0.0.1
EOF

# Generate private key
echo "🔑 Generating private key..."
openssl genrsa -out $CERT_DIR/server.key 2048

# Generate certificate
echo "📜 Generating certificate..."
openssl req -new -x509 -key $CERT_DIR/server.key -out $CERT_DIR/server.crt -days 365 -config $CERT_DIR/openssl.conf -extensions v3_req

# Generate certificate in different formats for easy installation
echo "📱 Generating mobile-friendly formats..."

# Convert to PEM format (for easy copy-paste)
cp $CERT_DIR/server.crt $CERT_DIR/server.pem

# Generate DER format (for some mobile devices)
openssl x509 -outform der -in $CERT_DIR/server.crt -out $CERT_DIR/server.der

# Generate PKCS12 format (for Windows/iOS)
openssl pkcs12 -export -out $CERT_DIR/server.p12 -inkey $CERT_DIR/server.key -in $CERT_DIR/server.crt -passout pass:

echo "✅ Certificate generated successfully!"
echo ""
echo "📋 Certificate files created:"
echo "   • $CERT_DIR/server.crt (Main certificate)"
echo "   • $CERT_DIR/server.key (Private key)"
echo "   • $CERT_DIR/server.pem (PEM format)"
echo "   • $CERT_DIR/server.der (DER format)"
echo "   • $CERT_DIR/server.p12 (PKCS12 format - no password)"
echo ""
echo "📱 To install on other devices:"
echo "   • Android: Copy server.crt to device, install via Settings > Security"
echo "   • iOS: Email server.crt to device, tap to install"
echo "   • Windows: Double-click server.crt, install to 'Trusted Root'"
echo "   • macOS: Double-click server.crt, add to Keychain"
echo ""
echo "🌐 Access URLs:"
echo "   • https://$HOSTNAME.local:8000"
echo "   • https://$CURRENT_IP:8000"
echo ""
echo "⚠️  Remember to restart the server after generating new certificates!"
