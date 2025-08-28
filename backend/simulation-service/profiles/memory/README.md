# Memory Profile Documentation

This document explains the Memory profile format used by the simulation engine to model different memory architectures and behaviors.

## Overview

Memory profiles define the characteristics and behavior of different memory types, from DDR4 consumer memory to high-bandwidth server memory configurations like DDR5 and HBM2. Each profile contains detailed specifications that enable realistic performance modeling including:

- **Hardware Prefetching** - Configurable prefetcher count, accuracy, and distance
- **Cache Conflict Modeling** - Bank conflicts and memory controller contention
- **Memory Ordering** - TSO, weak, and relaxed memory ordering models
- **Virtual Memory** - TLB modeling and page management
- **ECC Modeling** - Error correction and detection simulation
- **Power States** - Memory power management and transitions
- **Thermal Throttling** - Temperature-based performance scaling

The engine supports **4 complexity levels** (0-3) with progressive feature enablement for optimal performance vs. accuracy trade-offs.

## Profile Structure

### Core Profile Fields

```json
{
  "name": "ddr5_6400_server",
  "type": 1,
  "description": "DDR5-6400 Server Memory Configuration",
  "version": "1.0",
  "created_by": "Augment Agent",
  "created_at": "2025-01-22T00:00:00Z",

  "baseline_performance": { ... },
  "technology_specs": { ... },
  "engine_specific": { ... }
}
```

**Important**: The `type` field must be `1` for Memory Engine profiles (EngineType enum value).

### 1. Baseline Performance
Core memory specifications that drive all performance calculations:

```json
"baseline_performance": {
  "capacity_gb": 128,                       // Total memory capacity in GB
  "frequency_mhz": 6400,                    // Memory frequency in MHz
  "cas_latency": 32,                        // CAS latency in cycles
  "channels": 4,                            // Number of memory channels
  "bandwidth_gbps": 204.8,                  // Peak theoretical bandwidth in GB/s
  "access_time": 6.25,                      // Base access time in nanoseconds
  "frequency_normalization_baseline": 3200.0 // Baseline frequency for normalization
}
```

**Key Points:**
- `access_time` is calculated as: `(1000 / frequency_mhz) * cas_latency / 2`
- `bandwidth_gbps` should match: `(frequency_mhz * channels * 8) / 1000`
- All performance calculations are derived from these baseline values

### 2. Technology Specs
Hardware specifications and processing parameters:

```json
"technology_specs": {
  "memory_type": "DDR5",       // Memory type (DDR4, DDR5, HBM2, etc.)
  "queue_processing": {
    "max_ops_per_tick": 3,     // Maximum operations processed per tick
    "base_complexity_factor": 1.15, // Base complexity factor for queue calculations
    "avg_operation_duration_ms": 0.8 // Average operation duration in milliseconds
  }
}
```

**Supported Memory Types:**
- `"DDR4"` - DDR4 SDRAM (consumer/server)
- `"DDR5"` - DDR5 SDRAM (next-generation)
- `"HBM2"` - High Bandwidth Memory 2 (HPC/AI)
- `"LPDDR5"` - Low Power DDR5 (mobile)
- `"GDDR6"` - Graphics DDR6 (GPU memory)

### 3. Engine Specific Configuration

**Critical**: All feature configurations must be nested under `"engine_specific"` in the JSON.

#### DDR Timings
Realistic DDR timing parameters that affect memory access latency:

```json
"engine_specific": {
  "ddr_timings": {
    "trcd": 32.0,                       // RAS to CAS delay (cycles)
    "trp": 32.0,                        // Row precharge time (cycles)
    "tras": 52.0,                       // Row active time (cycles)
    "trc": 84.0,                        // Row cycle time (cycles)
    "trfc": 560.0                       // Refresh cycle time (cycles)
  },
```

**DDR Timing Rules:**
- `tRAS ≥ tRCD + CAS_LATENCY` (row active constraint)
- `tRC = tRAS + tRP` (row cycle constraint)
- Higher frequencies typically require higher timing values

#### NUMA Configuration
Multi-socket memory topology and access patterns:

```json
  "numa_configuration": {
    "socket_count": 2,                  // Number of CPU sockets
    "cross_socket_penalty": 1.6,        // Performance penalty multiplier (>1.0)
    "inter_socket_latency_ns": 80.0,    // Additional latency for cross-socket access
    "local_access_ratio": 0.8           // Ratio of local vs remote memory access
  },
```

**NUMA Guidelines:**
- Single socket: `socket_count: 1, cross_socket_penalty: 1.0, local_access_ratio: 1.0`
- Dual socket: `socket_count: 2, cross_socket_penalty: 1.6-1.8, local_access_ratio: 0.7-0.8`
- Quad socket: `socket_count: 4, cross_socket_penalty: 2.0-2.5, local_access_ratio: 0.6-0.7`

#### Advanced NUMA Configuration
For advanced NUMA page migration and optimization:

```json
  "advanced_numa": {
    "node_affinity_policy": "preferred",    // "strict", "preferred", "interleave"
    "migration_threshold": 0.6,             // Threshold for page migration (0.0-1.0)
    "base_inter_node_latency": 80.0,        // Base inter-node latency in nanoseconds
    "base_bandwidth_gbps": 120.0,           // Base inter-node bandwidth in GB/s
    "migration_benefit": 60.0,              // Benefit from successful migration (ns)
    "migration_base_cost": 80.0,            // Base cost of page migration (ns)
    "strict_affinity_penalty": 40.0,        // Penalty for strict affinity violations (ns)
    "preferred_affinity_penalty": 15.0      // Penalty for preferred affinity violations (ns)
  },
```

**NUMA Affinity Policies:**
- **Strict**: Pages must stay on assigned nodes, high penalty for violations
- **Preferred**: Pages prefer assigned nodes but can migrate when beneficial
- **Interleave**: Pages are distributed across nodes for load balancing

#### Bandwidth Characteristics
Memory bandwidth modeling and saturation behavior:

```json
  "bandwidth_characteristics": {
    "peak_bandwidth_gbps": 204.8,      // Peak theoretical bandwidth
    "sustained_bandwidth_gbps": 180.0, // Sustained bandwidth under load
    "saturation_threshold": 0.75       // Utilization where saturation begins (0.0-1.0)
  },
```

**Bandwidth Guidelines:**
- `peak_bandwidth_gbps` should match theoretical maximum from specifications
- `sustained_bandwidth_gbps` is typically 85-90% of peak for real-world workloads
- `saturation_threshold` varies by memory type: DDR4 (0.8), DDR5 (0.75), HBM2 (0.7)

#### Priority 1 Features (Critical - Always Enabled at Complexity 3)

**Hardware Prefetching Configuration:**
```json
  "hardware_prefetch": {
    "prefetcher_count": 4,              // Number of hardware prefetchers
    "sequential_accuracy": 0.90,        // Accuracy for sequential patterns
    "stride_accuracy": 0.80,            // Accuracy for stride patterns
    "pattern_accuracy": 0.60,           // Accuracy for complex patterns
    "prefetch_distance": 8,             // Prefetch distance in cache lines
    "bandwidth_usage": 0.03,            // Bandwidth overhead (0.0-1.0)
    "prefetch_benefit": 0.35            // Performance benefit multiplier
  },
```

**Cache Conflict Modeling:**
```json
  "cache_conflicts": {
    "cache_line_size": 64,              // Cache line size in bytes
    "associativity": 8,                 // Cache associativity
    "conflict_probability": 0.15,       // Probability of cache conflicts
    "conflict_penalty": 2.5,            // Performance penalty multiplier
    "thrashing_threshold": 0.85         // Utilization where thrashing begins
  },
```

**Memory Ordering Models:**
```json
  "memory_ordering": {
    "ordering_model": "weak",           // "tso", "weak", "strong", "pso"
    "reordering_window": 16,            // Maximum operations in reordering window
    "memory_barrier_cost": 8.0,         // Memory barrier cost in nanoseconds
    "load_store_reordering": true,      // Allow load-store reordering
    "store_store_reordering": true,     // Allow store-store reordering
    "load_load_reordering": true,       // Allow load-load reordering
    "base_reordering_delay": 1,         // Base delay for reordering (ticks)
    "reordering_benefit": 0.20,         // Performance benefit from reordering
    "dependency_delay": 3.0             // Delay for memory dependencies (ns)
  },
```

**Memory Ordering Models:**
- **TSO (Total Store Ordering)**: Prevents store reordering, allows load reordering
- **Weak**: Allows all types of reordering for maximum performance
- **Strong**: Prevents all reordering, maintains strict ordering
- **PSO (Partial Store Ordering)**: Allows some reordering with constraints

#### Priority 2 Features (Advanced - Enabled at Complexity 2+)

**Virtual Memory Management:**
```json
  "virtual_memory": {
    "page_size": 4096,                  // Page size in bytes
    "tlb_size": 128,                    // TLB entries
    "tlb_miss_penalty": 50.0,           // TLB miss penalty in nanoseconds
    "page_fault_probability": 0.001,    // Probability of page faults
    "page_fault_penalty": 10000.0       // Page fault penalty in nanoseconds
  },
```

**ECC Error Modeling:**
```json
  "ecc_modeling": {
    "single_bit_error_rate": 1e-6,      // Single-bit error rate per operation
    "multi_bit_error_rate": 1e-9,       // Multi-bit error rate per operation
    "correction_latency": 5.0,          // ECC correction latency in nanoseconds
    "scrubbing_overhead": 0.01          // Memory scrubbing overhead
  },
```

#### Priority 3 Features (Expert - Enabled at Complexity 3 Only)

**Power State Management:**
```json
  "power_states": {
    "active_power": 15.0,               // Active power consumption in watts
    "idle_power": 2.0,                  // Idle power consumption in watts
    "transition_latency": 100.0,        // State transition latency in nanoseconds
    "power_gating_threshold": 0.1       // Utilization threshold for power gating
  },
```

**Thermal Throttling:**
```json
  "enhanced_thermal": {
    "heat_dissipation_rate": 0.85,      // Heat dissipation rate
    "thermal_capacity": 60.0,           // Thermal capacity
    "ambient_temperature": 22.0,        // Ambient temperature in Celsius
    "base_heat_per_operation": 0.05,    // Heat generated per operation
    "thermal_zones": [                  // Multiple thermal zones
      {
        "max_temperature": 80.0,
        "heat_generation": 8.0,
        "cooling_capacity": 25.0,
        "thermal_mass": 30.0
      }
    ],
    "throttling_thresholds": [70.0, 75.0, 80.0],  // Temperature thresholds
    "throttling_levels": [0.05, 0.2, 0.5]         // Performance reduction levels
  }
}  // End of engine_specific
```

### 5. Health Thresholds
System health monitoring:

```json
"health_thresholds": {
  "utilization_warning": 0.8,    // Warning utilization threshold
  "utilization_critical": 0.95,  // Critical utilization threshold
  "temperature_warning": 85.0,   // Warning temperature (°C)
  "temperature_critical": 95.0,  // Critical temperature (°C)
  "error_rate_warning": 0.001,   // Warning error rate
  "error_rate_critical": 0.01    // Critical error rate
}
```

### 6. Convergence Models
Statistical convergence for realistic behavior:

```json
"convergence_models": {
  "row_buffer_hits": {
    "convergence_point": 0.65,    // Target hit rate
    "base_variance": 0.05,        // Base variance
    "min_operations": 500         // Minimum operations for convergence
  },
  "bandwidth_utilization": {
    "convergence_point": 0.75,    // Target utilization
    "base_variance": 0.08,        // Base variance
    "min_operations": 1000        // Minimum operations for convergence
  },
  "numa_locality": {
    "convergence_point": 0.95,    // Target locality ratio
    "base_variance": 0.02,        // Base variance
    "min_operations": 800         // Minimum operations for convergence
  }
}
```

## Memory Engine Complexity Levels

The Memory Engine supports **4 complexity levels** (0-3) with progressive feature enablement:

### Complexity Level 0 (Minimal)
- **Features Enabled**: None (0%)
- **Performance**: Fastest simulation
- **Use Case**: Basic bandwidth and latency modeling only
- **Features**: Core memory access simulation

### Complexity Level 1 (Basic)
- **Features Enabled**: None (0%)
- **Performance**: Fast simulation
- **Use Case**: Simple memory modeling with realistic timings
- **Features**: DDR timing effects, basic bandwidth modeling

### Complexity Level 2 (Advanced)
- **Features Enabled**: 2 out of 7 (28.6%)
- **Performance**: Balanced simulation
- **Use Case**: Comprehensive memory modeling
- **Features**:
  - ✅ Cache Conflicts (bank conflicts, memory controller contention)
  - ✅ Virtual Memory (TLB modeling, page management)

### Complexity Level 3 (Maximum)
- **Features Enabled**: All 7 (100%)
- **Performance**: Most realistic simulation
- **Use Case**: Maximum accuracy for research and validation
- **Features**:
  - ✅ Hardware Prefetch (configurable prefetchers)
  - ✅ Cache Conflicts (bank conflicts, memory controller contention)
  - ✅ Memory Ordering (TSO, weak, relaxed models)
  - ✅ Virtual Memory (TLB modeling, page management)
  - ✅ ECC Modeling (error correction and detection)
  - ✅ Power States (power management and transitions)
  - ✅ Thermal Throttling (temperature-based performance scaling)

### Feature Flags

The Memory Engine uses feature flags for conditional processing:

**Core Features (Always Enabled):**
- `ddr_timing_effects` - DDR timing modeling (tRP, tRCD, tRAS)
- `bandwidth_saturation` - Memory bandwidth saturation curves
- `basic_numa` - NUMA topology effects
- `memory_pressure` - OS memory pressure modeling
- `access_patterns` - Access pattern optimization
- `channel_utilization` - Memory channel utilization

**Advanced Features (Complexity Level 2+):**
- `memory_controller` - Memory controller modeling with queue depth
- `garbage_collection` - Language-specific GC effects (Java, Go, C#, Python)
- `memory_fragmentation` - Heap fragmentation modeling
- `cache_line_conflicts` - False sharing and cache line conflict detection
- `virtual_memory` - TLB modeling and page management
- `advanced_numa` - NUMA page migration and optimization

**Expert Features (Complexity Level 3 Only):**
- `memory_ordering` - Memory ordering and reordering effects
- `ecc_modeling` - ECC error correction modeling
- `power_states` - Memory power state transitions
- `thermal_throttling` - Memory thermal effects

## Available Memory Profiles

### DDR4 Configurations
- **ddr4_2400_single_channel.json** - DDR4-2400 single channel (16GB, budget systems)
  - Bandwidth: 19.2 GB/s, CAS: 17, Channels: 1
- **ddr4_3200_dual_channel.json** - DDR4-3200 dual channel (32GB, consumer/gaming)
  - Bandwidth: 45.0 GB/s, CAS: 16, Channels: 2, Prefetchers: 2

### DDR5 Configurations
- **ddr5_6400_consumer_dual_channel.json** - DDR5-6400 dual channel (32GB, high-end consumer)
  - Bandwidth: 102.4 GB/s, CAS: 32, Channels: 2
- **ddr5_6400_quad_channel.json** - DDR5-6400 quad channel (128GB, workstation)
  - Bandwidth: 204.8 GB/s, CAS: 32, Channels: 4
- **ddr5_6400_server.json** - DDR5-6400 server (128GB, enterprise server)
  - Bandwidth: 180.0 GB/s, CAS: 32, Channels: 4, Prefetchers: 4

### HBM Configurations
- **hbm2_server.json** - HBM2 server memory (64GB, HPC/AI workloads)
  - Bandwidth: 750.0 GB/s, CAS: 14, Channels: 8, Prefetchers: 8
- **hbm2_server_memory.json** - HBM2 alternative configuration (64GB, enterprise)

## Profile Selection Guidelines

### For Budget/Entry-Level Workloads
Use `ddr4_2400_single_channel.json`:
- Single channel configuration
- Lower bandwidth (19.2 GB/s)
- Budget-grade timings
- Higher latency characteristics

### For Consumer Workloads
Use `ddr4_3200_dual_channel.json`:
- Dual channel configuration
- Moderate bandwidth (51.2 GB/s)
- Consumer-grade timings
- Single socket NUMA

### For High-Performance Workloads
Use `ddr5_6400_quad_channel.json`:
- Quad channel configuration
- High bandwidth (204.8 GB/s)
- Advanced DDR5 timings
- Multi-socket NUMA support

### For Server/Enterprise Workloads
Use `hbm2_server_memory.json`:
- Octa channel configuration
- Extreme bandwidth (819.2 GB/s)
- Minimal latency
- Specialized access patterns
- Advanced thermal modeling

## Best Practices

### 1. Profile Customization
- Modify existing profiles rather than creating from scratch
- Validate all timing values are realistic for the memory type
- Test profiles with the validation tool

### 2. Performance Tuning
- Adjust `access_time` for overall memory performance scaling
- Modify `bandwidth_characteristics` for bandwidth-sensitive workloads
- Tune `access_patterns` for workload-specific optimizations

### 3. NUMA Modeling
- Set realistic `cross_socket_penalty` values
- Configure `local_access_ratio` based on workload locality
- Adjust `inter_socket_latency_ns` for multi-socket systems

### 4. Bandwidth Characteristics
- Set `peak_bandwidth_gbps` based on actual hardware specifications
- Adjust `saturation_threshold` based on memory controller capabilities
- Configure `degradation_curve` for realistic bandwidth scaling

## Validation and Testing

### Comprehensive Profile Testing
```bash
# Test all memory profiles across complexity levels
go test -v -run TestMemoryEngineComprehensiveComparison -timeout 120s

# Test specific features across profiles
go test -v -run TestMemoryEngineFeatureComparison -timeout 60s

# Test individual profile loading
go test -v -run TestMemoryEngineProfileComparison -timeout 60s
```

### Profile-Specific Testing
```bash
# Test DDR4 profile
go test -v -run TestMemoryEngineComprehensiveComparison/ddr4_3200_dual_channel

# Test DDR5 profile
go test -v -run TestMemoryEngineComprehensiveComparison/ddr5_6400_server

# Test HBM2 profile
go test -v -run TestMemoryEngineComprehensiveComparison/hbm2_server
```

### Feature-Specific Testing
```bash
# Test hardware prefetching across profiles
go test -v -run TestMemoryEngineFeatureComparison/hardware_prefetching

# Test ECC modeling
go test -v -run TestMemoryEngineECCModeling

# Test thermal throttling
go test -v -run TestMemoryEngineThermalThrottling
```

## Version History

- **v1.0** - Initial Memory profile format with DDR timing, NUMA, and bandwidth modeling

## Common Mistakes

❌ **Don't:**
- Set unrealistic DDR timing values
- Use bandwidth values higher than theoretical maximums
- Set CAS latency values inconsistent with frequency
- Configure more channels than physically possible
- Set thermal limits above memory specifications

✅ **Do:**
- Use realistic DDR timing relationships (tRAS > tRCD + CAS)
- Match bandwidth to channel count and frequency
- Set appropriate NUMA penalties for multi-socket systems
- Configure convergence models for realistic behavior
- Test profiles with actual workloads
