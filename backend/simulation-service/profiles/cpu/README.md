# CPU Profile Documentation

This document explains the CPU profile format used by the simulation engine to model different CPU architectures and behaviors.

## Overview

CPU profiles define the characteristics and behavior of different CPU types, from desktop processors to enterprise server CPUs. Each profile contains detailed specifications that enable realistic performance modeling including SIMD/Vectorization, cache behavior, thermal characteristics, and more.

## Profile Structure

### Core Profile Fields

```json
{
  "name": "Human-readable CPU name",
  "type": 0,
  "description": "Detailed description of the CPU",
  "version": "2.1",
  "baseline_performance": { ... },
  "technology_specs": { ... },
  "load_curves": { ... },
  "engine_specific": { ... }
}
```

## Detailed Field Descriptions

### 1. Baseline Performance
Defines core CPU specifications:

```json
"baseline_performance": {
  "base_processing_time": 0.08,    // Base processing time in seconds
  "cores": 24,                     // Number of physical cores
  "base_clock": 3.0,              // Base clock speed in GHz
  "boost_clock": 4.0              // Maximum boost clock in GHz
}
```

### 2. Technology Specs
Hardware specifications:

```json
"technology_specs": {
  "cache_l1_kb": 32,              // L1 cache size per core in KB
  "cache_l2_kb": 1024,            // L2 cache size per core in KB
  "cache_l3_mb": 35,              // L3 cache size total in MB
  "memory_channels": 6,           // Number of memory channels
  "tdp": 205,                     // Thermal Design Power in Watts
  "thermal_limit": 85,            // Thermal throttling temperature in °C
  "manufacturing_process": "14nm", // Manufacturing process node
  "socket": "LGA3647",            // CPU socket type
  "max_memory": 1024              // Maximum memory in GB
}
```

### 3. Load Curves
Performance scaling under different load conditions:

```json
"load_curves": {
  "default": {
    "optimal_threshold": 0.70,      // Optimal utilization threshold
    "warning_threshold": 0.85,      // Warning utilization threshold
    "critical_threshold": 0.95,     // Critical utilization threshold
    "optimal_factor": 1.0,          // Performance factor at optimal load
    "warning_factor": 1.5,          // Performance penalty at warning load
    "critical_factor": 3.0          // Performance penalty at critical load
  }
}
```

## Engine-Specific Configuration

The `engine_specific` section contains detailed CPU behavior modeling:

### 4. SIMD/Vectorization
Models SIMD instruction support and vectorization efficiency:

```json
"vectorization": {
  "supported_instructions": ["SSE4.2", "AVX2", "AVX512"],
  "vector_width": 512,             // Vector width in bits
  "simd_efficiency": 0.85,         // SIMD efficiency factor (0.0-1.0)
  "operation_vectorizability": {   // Vectorization ratios by operation type
    "matrix_multiply": 0.90,       // 90% of operation can be vectorized
    "image_process": 0.85,
    "ml_inference": 0.80,
    "array_sum": 0.95,
    "vector_add": 0.95,
    "fft": 0.85,
    "crypto_hash": 0.70,
    "string_process": 0.20,        // Low vectorization for string ops
    "database_query": 0.15,        // Minimal vectorization for DB ops
    "network_process": 0.10
  }
}
```

### 5. Cache Behavior
Models cache hierarchy performance:

```json
"cache_behavior": {
  "l1_hit_ratio_target": 0.95,     // Target L1 cache hit ratio
  "l2_hit_ratio_target": 0.85,     // Target L2 cache hit ratio
  "l3_hit_ratio_target": 0.70,     // Target L3 cache hit ratio
  "cache_line_size": 64,           // Cache line size in bytes
  "l1_hit_multiplier": 1.0,        // Performance multiplier for L1 hits
  "l2_hit_multiplier": 1.2,        // Performance multiplier for L2 hits
  "l3_hit_multiplier": 2.0,        // Performance multiplier for L3 hits
  "memory_access_multiplier": 8.0, // Performance penalty for memory access
  "warmup_operations": 100,        // Operations needed for cache warmup
  "prefetch_efficiency": 0.85      // Hardware prefetch efficiency
}
```

### 6. Thermal Behavior
Models thermal characteristics and throttling:

```json
"thermal_behavior": {
  "heat_generation_rate": 1.2,     // Heat generation rate factor
  "cooling_capacity": 250,         // Cooling capacity in Watts
  "cooling_efficiency": 0.95,      // Cooling system efficiency
  "ambient_temp": 22,              // Ambient temperature in °C
  "thermal_throttle_temp": 85,     // Throttling temperature in °C
  "thermal_mass": 45               // Thermal mass factor
}
```

### 7. NUMA Behavior
Models Non-Uniform Memory Access characteristics:

```json
"numa_behavior": {
  "cross_socket_penalty": 1.8,     // Performance penalty for cross-socket access
  "memory_bandwidth": 131072,      // Memory bandwidth in MB/s
  "numa_nodes": 2,                 // Number of NUMA nodes
  "local_memory_ratio": 0.80       // Ratio of local memory accesses
}
```

### 8. Boost Behavior
Models CPU boost/turbo behavior:

```json
"boost_behavior": {
  "single_core_boost": 4.0,        // Single core boost frequency in GHz
  "all_core_boost": 3.3,           // All core boost frequency in GHz
  "boost_duration": 10.0,          // Boost duration in seconds
  "thermal_dependent": true,       // Whether boost depends on thermal state
  "load_dependent": true           // Whether boost depends on load
}
```

### 9. Hyperthreading
Models simultaneous multithreading:

```json
"hyperthreading": {
  "enabled": true,                 // Whether hyperthreading is enabled
  "threads_per_core": 2,           // Number of threads per core
  "efficiency_factor": 0.75        // Efficiency factor for additional threads
}
```

### 10. Parallel Processing
Models parallel execution capabilities:

```json
"parallel_processing": {
  "enabled": true,                 // Whether parallel processing is enabled
  "max_parallelizable_ratio": 0.95, // Maximum parallelizable portion (Amdahl's Law)
  "overhead_per_core": 0.02,       // Overhead per additional core
  "synchronization_overhead": 0.1, // Synchronization overhead factor
  "parallelizability_by_complexity": {
    "O(1)": 0.20,                  // Parallelizability for O(1) algorithms
    "O(log n)": 0.60,              // Parallelizability for O(log n) algorithms
    "O(n)": 0.80,                  // Parallelizability for O(n) algorithms
    "O(n log n)": 0.85,            // Parallelizability for O(n log n) algorithms
    "O(n²)": 0.90                  // Parallelizability for O(n²) algorithms
  },
  "max_cores_for_complexity": {    // Maximum effective cores by complexity
    "O(1)": 1,
    "O(log n)": 4,
    "O(n)": 12,
    "O(n log n)": 20,
    "O(n²)": 24
  },
  "efficiency_curve": {            // Parallel efficiency by core count
    "1": 1.0,
    "2": 0.95,
    "4": 0.90,
    "8": 0.85,
    "12": 0.80,
    "16": 0.75,
    "20": 0.70,
    "24": 0.65
  }
}
```

### 11. Memory Bandwidth
Models memory bandwidth contention:

```json
"memory_bandwidth": {
  "total_bandwidth_gbps": 131.0,   // Total memory bandwidth in GB/s
  "per_core_degradation": 0.03,    // Bandwidth degradation per active core
  "contention_threshold": 8,       // Core count where contention starts
  "severe_contention_probability": 0.15, // Probability of severe contention
  "severe_contention_penalty": 0.15      // Performance penalty for severe contention
}
```

### 12. Branch Prediction
Models branch prediction performance:

```json
"branch_prediction": {
  "base_accuracy": 0.96,           // Base branch prediction accuracy
  "random_pattern_accuracy": 0.85, // Accuracy for random patterns
  "loop_pattern_accuracy": 0.98,   // Accuracy for loop patterns
  "call_return_accuracy": 0.99,    // Accuracy for call/return patterns
  "misprediction_penalty": 0.15,   // Performance penalty for misprediction
  "pipeline_depth": 14             // CPU pipeline depth
}
```

### 13. Advanced Prefetch
Models hardware prefetching:

```json
"advanced_prefetch": {
  "hardware_prefetchers": 4,       // Number of hardware prefetchers
  "sequential_accuracy": 0.90,     // Accuracy for sequential access patterns
  "stride_accuracy": 0.85,         // Accuracy for stride access patterns
  "pattern_accuracy": 0.75,        // Accuracy for complex patterns
  "prefetch_distance": 8,          // Prefetch distance in cache lines
  "bandwidth_usage": 0.15          // Memory bandwidth usage for prefetching
}
```

### 14. Language Multipliers
Performance multipliers for different programming languages:

```json
"language_multipliers": {
  "c": 1.3,                       // C performance factor
  "cpp": 1.3,                     // C++ performance factor
  "rust": 1.2,                    // Rust performance factor
  "go": 1.0,                      // Go performance factor (baseline)
  "java": 1.1,                    // Java performance factor
  "csharp": 1.05,                 // C# performance factor
  "nodejs": 0.8,                  // Node.js performance factor
  "python": 0.3,                  // Python performance factor
  "ruby": 0.25,                   // Ruby performance factor
  "php": 0.4                      // PHP performance factor
}
```

### 15. Complexity Factors
Performance scaling factors for algorithm complexities:

```json
"complexity_factors": {
  "O(1)": 1.0,                    // Constant time complexity
  "O(log n)": 1.5,                // Logarithmic complexity
  "O(n)": 2.0,                    // Linear complexity
  "O(n log n)": 3.0,              // Linearithmic complexity
  "O(n²)": 5.0,                   // Quadratic complexity
  "O(n³)": 8.0,                   // Cubic complexity
  "O(2^n)": 15.0                  // Exponential complexity
}
```

## Available CPU Profiles

The following CPU profiles are available:

### Server CPUs
- **intel_xeon_server.json** - Intel Xeon Gold 6248R (24 cores, AVX-512)
- **database_server_cpu.json** - Intel Xeon Platinum 8380 (40 cores, optimized for databases)
- **web_server_cpu.json** - Intel Xeon Gold 5318Y (24 cores, optimized for web workloads)
- **compute_server_cpu.json** - Intel Xeon Platinum 8380H (28 cores, optimized for HPC)
- **arm_server_cpu.json** - AWS Graviton3 (64 cores, ARM-based server CPU)

### Desktop CPUs
- **desktop_cpu.json** - Intel Core i7-12700K (12 cores, high-end desktop)
- **amd_ryzen_desktop.json** - AMD Ryzen 9 5950X (16 cores, high-end desktop)

## Profile Selection Guidelines

### For Database Workloads
Use `database_server_cpu.json`:
- Large L3 cache (60MB)
- High memory bandwidth (204.8 GB/s)
- Optimized for low-vectorization workloads
- Multiple NUMA nodes for scalability

### For Web Server Workloads
Use `web_server_cpu.json`:
- Balanced performance characteristics
- Moderate cache sizes
- Good parallel processing efficiency
- Optimized for mixed workload patterns

### For Compute-Intensive Workloads
Use `compute_server_cpu.json`:
- High SIMD efficiency (90%)
- Excellent vectorization support
- Optimized for scientific computing
- High parallel processing ratios

### For General Purpose
Use `intel_xeon_server.json` or `desktop_cpu.json`:
- Well-balanced specifications
- Good for mixed workloads
- Realistic performance characteristics

## Best Practices

### 1. Profile Customization
- Modify existing profiles rather than creating from scratch
- Validate all numerical values are realistic
- Test profiles with the validation tool

### 2. Performance Tuning
- Adjust `base_processing_time` for overall performance scaling
- Modify `language_multipliers` for language-specific optimizations
- Tune `vectorization.operation_vectorizability` for workload-specific SIMD benefits

### 3. Thermal Modeling
- Set realistic `thermal_limit` values based on CPU specifications
- Adjust `cooling_capacity` based on cooling solution
- Consider `ambient_temp` for deployment environment

### 4. Memory Characteristics
- Set `memory_bandwidth` based on actual hardware specifications
- Adjust cache hit ratios based on workload characteristics
- Configure NUMA settings for multi-socket systems

## Validation

Use the profile validation tool to verify your profiles:

```bash
go run test_profile_loading.go
```

This will verify that all profile fields are correctly loaded and that the CPU engine can process operations using the profile data.

## Version History

- **v2.1** - Added SIMD/Vectorization support, improved thermal modeling
- **v2.0** - Added advanced features (NUMA, branch prediction, prefetching)
- **v1.0** - Initial CPU profile format
