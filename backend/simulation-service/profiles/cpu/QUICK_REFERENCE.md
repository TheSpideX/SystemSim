# CPU Profile Quick Reference

## Profile Structure Overview

```json
{
  "name": "CPU Name",
  "type": 0,
  "description": "CPU Description", 
  "version": "2.1",
  "baseline_performance": {
    "base_processing_time": 0.08,
    "cores": 24,
    "base_clock": 3.0,
    "boost_clock": 4.0
  },
  "technology_specs": {
    "cache_l1_kb": 32,
    "cache_l2_kb": 1024, 
    "cache_l3_mb": 35,
    "memory_channels": 6,
    "tdp": 205,
    "thermal_limit": 85,
    "manufacturing_process": "14nm",
    "socket": "LGA3647",
    "max_memory": 1024
  },
  "load_curves": {
    "default": {
      "optimal_threshold": 0.70,
      "warning_threshold": 0.85,
      "critical_threshold": 0.95,
      "optimal_factor": 1.0,
      "warning_factor": 1.5,
      "critical_factor": 3.0
    }
  },
  "engine_specific": {
    "vectorization": { /* SIMD settings */ },
    "cache_behavior": { /* Cache settings */ },
    "thermal_behavior": { /* Thermal settings */ },
    "numa_behavior": { /* NUMA settings */ },
    "boost_behavior": { /* Boost settings */ },
    "hyperthreading": { /* HT settings */ },
    "parallel_processing": { /* Parallel settings */ },
    "memory_bandwidth": { /* Memory settings */ },
    "branch_prediction": { /* Branch prediction */ },
    "advanced_prefetch": { /* Prefetch settings */ },
    "language_multipliers": { /* Language performance */ },
    "complexity_factors": { /* Algorithm complexity */ }
  }
}
```

## Key Configuration Sections

### SIMD/Vectorization
```json
"vectorization": {
  "supported_instructions": ["SSE4.2", "AVX2", "AVX512"],
  "vector_width": 512,
  "simd_efficiency": 0.85,
  "operation_vectorizability": {
    "matrix_multiply": 0.90,    // High vectorization
    "image_process": 0.85,
    "array_sum": 0.95,
    "string_process": 0.20,     // Low vectorization
    "database_query": 0.15
  }
}
```

### Cache Behavior
```json
"cache_behavior": {
  "l1_hit_ratio_target": 0.95,
  "l2_hit_ratio_target": 0.85,
  "l3_hit_ratio_target": 0.70,
  "cache_line_size": 64,
  "memory_access_multiplier": 8.0
}
```

### Language Performance
```json
"language_multipliers": {
  "c": 1.3,        // 30% faster than baseline
  "cpp": 1.3,
  "rust": 1.2,
  "go": 1.0,       // Baseline
  "java": 1.1,
  "python": 0.3,   // 3.3x slower than baseline
  "nodejs": 0.8
}
```

### Algorithm Complexity
```json
"complexity_factors": {
  "O(1)": 1.0,     // Constant time
  "O(log n)": 1.5,
  "O(n)": 2.0,     // Linear time
  "O(n²)": 5.0,    // Quadratic time
  "O(2^n)": 15.0   // Exponential time
}
```

## Common Profile Types

### Server CPU (High-End)
- **Cores**: 24-64
- **Cache L3**: 35-60 MB
- **Memory Bandwidth**: 131-204 GB/s
- **SIMD**: AVX-512 (512-bit)
- **Use Cases**: Enterprise servers, databases, compute

### Desktop CPU (Consumer)
- **Cores**: 6-16
- **Cache L3**: 16-32 MB
- **Memory Bandwidth**: 51-76 GB/s
- **SIMD**: AVX2 (256-bit)
- **Use Cases**: Gaming, development, general purpose

### ARM Server CPU
- **Cores**: 32-64
- **Cache L3**: 32-64 MB
- **Memory Bandwidth**: 100-200 GB/s
- **SIMD**: NEON/SVE (128-bit)
- **Use Cases**: Cloud servers, energy-efficient computing

## Performance Tuning Tips

### For High Performance
- Increase `base_clock` and `boost_clock`
- Set high `simd_efficiency` (0.85-0.95)
- Use large cache sizes
- Set low `base_processing_time`

### For Database Workloads
- Large L3 cache (40-60 MB)
- High memory bandwidth
- Lower vectorization ratios
- Multiple NUMA nodes

### For Compute Workloads
- High SIMD efficiency
- Large vector widths (512-bit)
- High vectorization ratios
- Optimized for parallel processing

### For Web Workloads
- Balanced cache hierarchy
- Moderate SIMD support
- Good thermal characteristics
- Efficient parallel processing

## Validation Commands

```bash
# Test all profiles
go run test_profile_loading.go

# Test specific profile
go run test_cpu_simd_with_profile.go
```

## Common Mistakes

❌ **Don't:**
- Put language_multipliers outside engine_specific
- Use unrealistic cache sizes
- Set thermal_limit above 100°C for consumer CPUs
- Use vector_width > 512 for current CPUs
- Set simd_efficiency > 0.95

✅ **Do:**
- Keep all multipliers within realistic ranges
- Match cache sizes to real hardware
- Use appropriate thermal limits
- Set realistic memory bandwidth
- Test profiles after creation

## Quick Profile Creation

1. **Copy existing profile** closest to your target
2. **Modify core specs** (cores, clocks, cache)
3. **Adjust SIMD settings** for your use case
4. **Update language multipliers** if needed
5. **Test with validation tool**
6. **Fine-tune based on results**
