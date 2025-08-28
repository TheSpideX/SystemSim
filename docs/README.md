# System Design Simulator - Documentation

## 📚 **Current Architecture Documentation**

### **🏗️ Core Architecture**
- **[Microservice Mesh Architecture](microservice-mesh-architecture.md)** - Current system architecture with gRPC mesh networking
- **[Implementation Roadmap](implementation-roadmap.md)** - Step-by-step implementation plan and timeline

### **🔧 Technical Specifications**
- **[Tech Stack](tech-stack.md)** - Technology choices and rationale
- **[Roadmap](roadmap.md)** - High-level project roadmap and milestones

### **🚀 Simulation Engine**
- **[Simulation Engine Architecture](simulation-engine-architecture.md)** - Core simulation engine design
- **[Simulation Engine Enhancements](simulation-engine-enhancements.md)** - Advanced features and optimizations
- **[Simulation Engine](simulation%20engine.md)** - Engine implementation details
- **[Backpressure Flow Control](backpressure-flow-control.md)** - Flow control mechanisms

## 🎯 **Quick Navigation**

### **For Developers**
1. Start with **[Microservice Mesh Architecture](microservice-mesh-architecture.md)** to understand the current system
2. Review **[Implementation Roadmap](implementation-roadmap.md)** for development tasks
3. Check **[Tech Stack](tech-stack.md)** for technology decisions

### **For System Design**
1. **[Microservice Mesh Architecture](microservice-mesh-architecture.md)** - Overall system design
2. **[Simulation Engine Architecture](simulation-engine-architecture.md)** - Simulation components

### **For Project Planning**
1. **[Implementation Roadmap](implementation-roadmap.md)** - Detailed implementation plan
2. **[Roadmap](roadmap.md)** - High-level project timeline

## 📋 **Current Status Summary**

### **✅ Completed**
- **Auth Service**: Production-ready microservice with JWT, PostgreSQL, Redis
- **Database Schema**: Users and sessions tables with proper indexing
- **Security Features**: Rate limiting, account lockout, password validation
- **Containerization**: Docker setup with health checks

### **🔄 In Progress**
- **gRPC Mesh Network**: Converting HTTP services to gRPC mesh
- **Health Checks**: Enhanced monitoring and health endpoints
- **Connection Pooling**: Dynamic gRPC connection management

### **📋 Planned**
- **Project Service**: CRUD operations for projects and templates
- **Simulation Service**: Real-time simulation processing
- **Server Service**: API Gateway with WebSocket support

## 🏗️ **Architecture Overview**

```
┌─────────────────────────────────────────────────────────────────┐
│                    MICROSERVICE MESH NETWORK                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Client ◄──► Server Service ◄──► Auth Service                   │
│  (HTTP/WS)   (8000-8020)         (9000-9020)                   │
│                    │                   │                       │
│                    ▼                   ▼                       │
│              Project Service ◄──► Simulation Service           │
│              (10000-10020)        (11000-11020)                │
│                                                                 │
│  Communication Channels:                                        │
│  • gRPC Mesh (20 connections per service)                      │
│  • HTTP + Async Queue (heavy processing)                       │
│  • Redis Pub/Sub (real-time simulation data)                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 🔗 **Key Features**

### **Microservice Architecture**
- **Service Isolation**: Each service has dedicated port pool (20 ports)
- **Multiple Communication Channels**: gRPC, HTTP+Queue, Redis Pub/Sub
- **Dynamic Connection Pooling**: 5-20 gRPC connections based on load
- **Fault Tolerance**: Circuit breakers and health checks

### **Real-time Capabilities**
- **WebSocket Connections**: Client real-time updates
- **Redis Pub/Sub**: Simulation data streaming
- **Low Latency**: <16ms for real-time operations

### **Security & Performance**
- **JWT Authentication**: Secure token-based auth
- **Rate Limiting**: Protection against abuse
- **Connection Pooling**: Optimized resource usage
- **Health Monitoring**: Comprehensive service health checks

---

*For detailed information, please refer to the specific documentation files linked above.*

*Last Updated: January 2025*
*Architecture Version: Microservice Mesh 1.0*
