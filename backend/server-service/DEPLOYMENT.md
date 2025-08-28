# API Gateway Deployment Guide

This guide covers deployment strategies and configurations for the API Gateway in different environments.

## 🏗️ Architecture Overview

The API Gateway serves as the central entry point for the System Design Website, handling:
- HTTP/2 client connections
- Authentication and authorization
- Request routing to microservices
- WebSocket connections for real-time features
- Circuit breaking and error handling

## 🚀 Deployment Options

### 1. Local Development

#### Prerequisites
- Go 1.21+
- Redis server
- PostgreSQL (for auth service)

#### Quick Start
```bash
# Clone and build
git clone <repository>
cd backend/server-service
go mod download
go build -o api-gateway ./cmd/main.go

# Start Redis (if not running)
redis-server

# Start the gateway
./api-gateway
```

The server will start on `https://localhost:8000` with self-signed certificates.

### 2. Docker Deployment

#### Single Container
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-gateway ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/api-gateway .
COPY --from=builder /app/certs ./certs

EXPOSE 8000
CMD ["./api-gateway"]
```

#### Build and Run
```bash
# Build image
docker build -t api-gateway:latest .

# Run container
docker run -d \
  --name api-gateway \
  -p 8000:8000 \
  -e SERVER_HOST=0.0.0.0 \
  -e REDIS_HOST=redis \
  api-gateway:latest
```

#### Docker Compose
```yaml
version: '3.8'

services:
  api-gateway:
    build: .
    ports:
      - "8000:8000"
    environment:
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8000
      - TLS_ENABLED=true
      - REDIS_HOST=redis
      - AUTH_SERVICE_HOST=auth-service
      - PROJECT_SERVICE_HOST=project-service
      - SIMULATION_SERVICE_HOST=simulation-service
    depends_on:
      - redis
      - auth-service
    networks:
      - microservices

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    networks:
      - microservices

  auth-service:
    image: auth-service:latest
    ports:
      - "9001:9001"
    networks:
      - microservices

networks:
  microservices:
    driver: bridge
```

### 3. Kubernetes Deployment

#### Namespace
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: system-design-website
```

#### ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-gateway-config
  namespace: system-design-website
data:
  config.yaml: |
    server:
      host: "0.0.0.0"
      port: "8000"
      tls_enabled: true
      read_timeout: "30s"
      write_timeout: "30s"
      idle_timeout: "60s"
      max_concurrent_streams: 1000
    
    services:
      auth:
        host: "auth-service"
        port: 9001
      project:
        host: "project-service"
        port: 9002
      simulation:
        host: "simulation-service"
        port: 9003
    
    redis:
      host: "redis-service"
      port: 6379
```

#### Secret for TLS
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-gateway-tls
  namespace: system-design-website
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
```

#### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: system-design-website
  labels:
    app: api-gateway
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: api-gateway:latest
        ports:
        - containerPort: 8000
          name: https
        env:
        - name: SERVER_HOST
          value: "0.0.0.0"
        - name: SERVER_PORT
          value: "8000"
        - name: TLS_ENABLED
          value: "true"
        - name: REDIS_HOST
          value: "redis-service"
        - name: AUTH_SERVICE_HOST
          value: "auth-service"
        - name: PROJECT_SERVICE_HOST
          value: "project-service"
        - name: SIMULATION_SERVICE_HOST
          value: "simulation-service"
        volumeMounts:
        - name: config
          mountPath: /app/config
        - name: tls-certs
          mountPath: /app/certs
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8000
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: api-gateway-config
      - name: tls-certs
        secret:
          secretName: api-gateway-tls
```

#### Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-gateway-service
  namespace: system-design-website
spec:
  selector:
    app: api-gateway
  ports:
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8000
  type: LoadBalancer
```

#### Ingress (Optional)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: system-design-website
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
spec:
  tls:
  - hosts:
    - api.systemdesign.example.com
    secretName: api-gateway-tls
  rules:
  - host: api.systemdesign.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-gateway-service
            port:
              number: 443
```

## 🔧 Configuration

### Environment Variables
```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8000
TLS_ENABLED=true
CERT_FILE=/app/certs/server.crt
KEY_FILE=/app/certs/server.key

# Service Discovery
AUTH_SERVICE_HOST=auth-service
AUTH_SERVICE_PORT=9001
PROJECT_SERVICE_HOST=project-service
PROJECT_SERVICE_PORT=9002
SIMULATION_SERVICE_HOST=simulation-service
SIMULATION_SERVICE_PORT=9003

# Redis Configuration
REDIS_HOST=redis-service
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Performance Tuning
MAX_CONCURRENT_STREAMS=1000
MAX_FRAME_SIZE=16384
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
IDLE_TIMEOUT=60s
HTTP2_IDLE_TIMEOUT=300s

# Circuit Breaker Settings
CB_AUTH_MAX_REQUESTS=3
CB_AUTH_TIMEOUT=30s
CB_AUTH_FAILURE_THRESHOLD=3

CB_PROJECT_MAX_REQUESTS=5
CB_PROJECT_TIMEOUT=45s
CB_PROJECT_FAILURE_THRESHOLD=5

CB_SIMULATION_MAX_REQUESTS=2
CB_SIMULATION_TIMEOUT=60s
CB_SIMULATION_FAILURE_THRESHOLD=2

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Production Configuration
```yaml
# config/production.yaml
server:
  host: "0.0.0.0"
  port: "8000"
  tls_enabled: true
  cert_file: "/app/certs/server.crt"
  key_file: "/app/certs/server.key"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "60s"
  max_request_body_size: 10485760
  max_concurrent_streams: 2000
  max_frame_size: 32768
  http2_idle_timeout: "300s"

services:
  auth:
    host: "auth-service.system-design-website.svc.cluster.local"
    port: 9001
  project:
    host: "project-service.system-design-website.svc.cluster.local"
    port: 9002
  simulation:
    host: "simulation-service.system-design-website.svc.cluster.local"
    port: 9003

redis:
  host: "redis-service.system-design-website.svc.cluster.local"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0
  pool_size: 20
  min_idle_conns: 5

circuit_breaker:
  auth:
    max_requests: 5
    timeout: "30s"
    failure_threshold: 3
  project:
    max_requests: 10
    timeout: "45s"
    failure_threshold: 5
  simulation:
    max_requests: 3
    timeout: "60s"
    failure_threshold: 2

logging:
  level: "info"
  format: "json"
  output: "stdout"

monitoring:
  metrics_enabled: true
  health_check_interval: "10s"
  request_logging: true
```

## 🔒 Security Configuration

### TLS Certificates

#### Development (Self-Signed)
```bash
# Generate self-signed certificates
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost"
```

#### Production (Let's Encrypt)
```bash
# Using certbot
certbot certonly --standalone -d api.yourdomain.com

# Copy certificates
cp /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem certs/server.crt
cp /etc/letsencrypt/live/api.yourdomain.com/privkey.pem certs/server.key
```

### Kubernetes TLS Secret
```bash
# Create TLS secret from certificates
kubectl create secret tls api-gateway-tls \
  --cert=certs/server.crt \
  --key=certs/server.key \
  -n system-design-website
```

## 📊 Monitoring and Observability

### Health Checks
```bash
# Basic health check
curl -k https://api-gateway:8000/health

# Detailed metrics
curl -k https://api-gateway:8000/metrics
```

### Prometheus Integration
```yaml
# prometheus-config.yaml
scrape_configs:
  - job_name: 'api-gateway'
    static_configs:
      - targets: ['api-gateway-service:8000']
    scheme: https
    tls_config:
      insecure_skip_verify: true
    metrics_path: /metrics
    scrape_interval: 15s
```

### Grafana Dashboard
Key metrics to monitor:
- Request rate and response times
- Circuit breaker states
- WebSocket connection count
- gRPC connection pool utilization
- Error rates by service

## 🚨 Troubleshooting

### Common Issues

#### 1. TLS Certificate Issues
```bash
# Check certificate validity
openssl x509 -in certs/server.crt -text -noout

# Verify certificate matches key
openssl x509 -noout -modulus -in certs/server.crt | openssl md5
openssl rsa -noout -modulus -in certs/server.key | openssl md5
```

#### 2. Service Connection Issues
```bash
# Check service connectivity
curl -k https://auth-service:9001/health
curl -k https://project-service:9002/health
curl -k https://simulation-service:9003/health

# Check Redis connectivity
redis-cli -h redis-service ping
```

#### 3. Performance Issues
```bash
# Check resource usage
kubectl top pods -n system-design-website

# Check logs
kubectl logs -f deployment/api-gateway -n system-design-website

# Monitor metrics
curl -k https://api-gateway:8000/metrics | grep -E "(requests|response_time|circuit_breaker)"
```

### Log Analysis
```bash
# Filter error logs
kubectl logs deployment/api-gateway -n system-design-website | grep ERROR

# Monitor circuit breaker events
kubectl logs deployment/api-gateway -n system-design-website | grep "Circuit breaker"

# Track authentication failures
kubectl logs deployment/api-gateway -n system-design-website | grep "Token validation failed"
```

## 🔄 Scaling and Performance

### Horizontal Scaling
```bash
# Scale deployment
kubectl scale deployment api-gateway --replicas=5 -n system-design-website

# Auto-scaling
kubectl autoscale deployment api-gateway --cpu-percent=70 --min=3 --max=10 -n system-design-website
```

### Performance Tuning
- Adjust `max_concurrent_streams` based on load
- Tune circuit breaker thresholds
- Optimize connection pool sizes
- Configure appropriate resource limits

### Load Testing
```bash
# Using wrk
wrk -t12 -c400 -d30s --latency https://api-gateway:8000/health

# Using hey
hey -n 10000 -c 100 https://api-gateway:8000/health
```

## 📝 Maintenance

### Updates and Rollbacks
```bash
# Rolling update
kubectl set image deployment/api-gateway api-gateway=api-gateway:v2.0.0 -n system-design-website

# Check rollout status
kubectl rollout status deployment/api-gateway -n system-design-website

# Rollback if needed
kubectl rollout undo deployment/api-gateway -n system-design-website
```

### Backup and Recovery
- Configuration backups
- Certificate management
- Service dependency coordination
- Data consistency checks

This deployment guide provides comprehensive coverage for deploying the API Gateway in various environments with proper security, monitoring, and maintenance procedures.
