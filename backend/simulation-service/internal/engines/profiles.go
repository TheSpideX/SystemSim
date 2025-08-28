package engines

import (
	"encoding/json"
	"fmt"
)

// ProfileManager manages hardware and workload profiles
type ProfileManager struct {
	CPUProfiles     map[string]*EngineProfile `json:"cpu_profiles"`
	MemoryProfiles  map[string]*EngineProfile `json:"memory_profiles"`
	StorageProfiles map[string]*EngineProfile `json:"storage_profiles"`
	NetworkProfiles map[string]*EngineProfile `json:"network_profiles"`
}

// NewProfileManager creates a new profile manager with default profiles
func NewProfileManager() *ProfileManager {
	pm := &ProfileManager{
		CPUProfiles:     make(map[string]*EngineProfile),
		MemoryProfiles:  make(map[string]*EngineProfile),
		StorageProfiles: make(map[string]*EngineProfile),
		NetworkProfiles: make(map[string]*EngineProfile),
	}
	
	// Load default profiles
	pm.loadDefaultProfiles()
	
	return pm
}

// GetProfile returns a profile by type and name
func (pm *ProfileManager) GetProfile(engineType EngineType, name string) (*EngineProfile, error) {
	var profiles map[string]*EngineProfile
	
	switch engineType {
	case CPUEngineType:
		profiles = pm.CPUProfiles
	case MemoryEngineType:
		profiles = pm.MemoryProfiles
	case StorageEngineType:
		profiles = pm.StorageProfiles
	case NetworkEngineType:
		profiles = pm.NetworkProfiles
	default:
		return nil, fmt.Errorf("unknown engine type: %v", engineType)
	}
	
	profile, exists := profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s for engine type %v", name, engineType)
	}
	
	return profile, nil
}

// LoadProfile loads a profile from JSON
func (pm *ProfileManager) LoadProfile(profileJSON []byte) error {
	var profile EngineProfile
	if err := json.Unmarshal(profileJSON, &profile); err != nil {
		return fmt.Errorf("failed to unmarshal profile: %w", err)
	}
	
	switch profile.Type {
	case CPUEngineType:
		pm.CPUProfiles[profile.Name] = &profile
	case MemoryEngineType:
		pm.MemoryProfiles[profile.Name] = &profile
	case StorageEngineType:
		pm.StorageProfiles[profile.Name] = &profile
	case NetworkEngineType:
		pm.NetworkProfiles[profile.Name] = &profile
	default:
		return fmt.Errorf("unknown profile type: %v", profile.Type)
	}
	
	return nil
}

// loadDefaultProfiles loads default hardware profiles
func (pm *ProfileManager) loadDefaultProfiles() {
	// Intel Xeon Gold 6248R CPU Profile
	intelXeonProfile := &EngineProfile{
		Name:        "intel_xeon_6248r",
		Type:        CPUEngineType,
		Description: "Intel Xeon Gold 6248R - 24 cores, 3.0GHz base, 4.0GHz boost",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"base_processing_time": 0.08, // milliseconds
			"cores":               24,
			"base_clock":          3.0, // GHz
			"boost_clock":         4.0, // GHz
		},
		TechnologySpecs: map[string]interface{}{
			"cache_l1_kb":      32,
			"cache_l2_kb":      1024,
			"cache_l3_mb":      35,
			"memory_channels":  6,
			"tdp":              205, // Watts
			"thermal_limit":    85,  // Celsius
		},
		EngineSpecific: map[string]interface{}{
			"cache_behavior": map[string]interface{}{
				"l1_hit_ratio":  0.95,
				"l2_hit_ratio":  0.85,
				"l3_hit_ratio":  0.70,
				"miss_penalty":  100.0, // 100x slowdown on L3 miss
			},
			"thermal_behavior": map[string]interface{}{
				"heat_generation_rate": 1.2,  // Watts per % CPU load
				"cooling_capacity":     250.0, // Watts cooling capacity
				"cooling_efficiency":   0.95,  // 95% cooling efficiency
				"ambient_temp":         22.0,  // Celsius datacenter temp
				"thermal_mass":         45.0,  // Seconds to heat up
			},
			"numa_behavior": map[string]interface{}{
				"cross_socket_penalty": 1.8,   // 1.8x penalty for cross-socket access
				"memory_bandwidth":     131072, // 128GB/s per socket
				"numa_nodes":           2,      // Dual socket
				"local_memory_ratio":   0.8,    // 80% local access
			},
			"boost_behavior": map[string]interface{}{
				"single_core_boost":  4.0,   // 4.0 GHz single core boost (Intel Xeon 6248R spec)
				"all_core_boost":     3.3,   // 3.3 GHz all core boost (Intel Xeon 6248R spec)
				"boost_duration":     10.0,  // 10 seconds boost duration
				"thermal_dependent":  true,  // Boost depends on thermal state
				"load_dependent":     true,  // Boost depends on load
			},
		},
	}
	pm.CPUProfiles["intel_xeon_6248r"] = intelXeonProfile
	
	// AMD EPYC 7742 CPU Profile
	amdEpycProfile := &EngineProfile{
		Name:        "amd_epyc_7742",
		Type:        CPUEngineType,
		Description: "AMD EPYC 7742 - 64 cores, 2.25GHz base, 3.4GHz boost",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"base_processing_time": 0.09, // milliseconds (slightly slower base)
			"cores":               64,
			"base_clock":          2.25, // GHz
			"boost_clock":         3.4,  // GHz
		},
		TechnologySpecs: map[string]interface{}{
			"cache_l1_kb":      32,
			"cache_l2_kb":      512,
			"cache_l3_mb":      256, // Much larger L3
			"memory_channels":  8,
			"tdp":              225, // Watts
			"thermal_limit":    90,  // Celsius (higher limit)
		},
		EngineSpecific: map[string]interface{}{
			"cache_behavior": map[string]interface{}{
				"l1_hit_ratio":  0.94,
				"l2_hit_ratio":  0.88,
				"l3_hit_ratio":  0.85, // Better L3 due to size
				"miss_penalty":  80.0,  // Slightly better than Intel
			},
			"thermal_behavior": map[string]interface{}{
				"heat_generation_rate": 1.4,  // Higher heat generation
				"cooling_capacity":     280.0, // Higher cooling capacity
				"cooling_efficiency":   0.92,  // Slightly lower efficiency
				"ambient_temp":         22.0,
				"thermal_mass":         50.0, // Larger thermal mass
			},
			"numa_behavior": map[string]interface{}{
				"cross_socket_penalty": 2.1,   // Higher NUMA penalty
				"memory_bandwidth":     204800, // 200GB/s per socket
			},
		},
	}
	pm.CPUProfiles["amd_epyc_7742"] = amdEpycProfile
	
	// DDR4-3200 Memory Profile
	ddr4Profile := &EngineProfile{
		Name:        "ddr4_3200_server",
		Type:        MemoryEngineType,
		Description: "DDR4-3200 Server Memory - 64GB capacity",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"capacity_gb":   64,
			"access_time":   10.0, // nanoseconds (CL16 @ 3200MHz)
			"bandwidth_gbps": 51.2, // GB/s theoretical
		},
		TechnologySpecs: map[string]interface{}{
			"memory_type":    "DDR4",
			"frequency":      3200, // MHz
			"cas_latency":    16,
			"channels":       4,
		},
		EngineSpecific: map[string]interface{}{
			"ddr_timings": map[string]interface{}{
				"trcd": 16.0,
				"trp":  16.0,
				"tras": 36.0,
			},
			"numa_configuration": map[string]interface{}{
				"socket_count":           2,
				"cross_socket_penalty":   1.8,
				"inter_socket_latency_ns": 100.0,
				"local_access_ratio":     0.7,
			},
			"bandwidth_characteristics": map[string]interface{}{
				"peak_bandwidth_gbps":     51.2,
				"sustained_bandwidth_gbps": 45.0,
				"saturation_threshold":    0.8,
			},
			"pressure_curves": map[string]interface{}{
				"optimal_threshold":  0.80,
				"warning_threshold":  0.90,
				"critical_threshold": 0.95,
				"swap_threshold":     0.90,
			},
			"gc_behavior": map[string]interface{}{
				"java": map[string]interface{}{
					"has_gc":             true,
					"trigger_threshold":  0.75,
					"pause_time_per_gb":  8.0,
					"efficiency_factor":  1.0,
				},
				"go": map[string]interface{}{
					"has_gc":             true,
					"trigger_threshold":  1.0,
					"pause_time_per_gb":  0.5,
					"efficiency_factor":  0.8,
				},
			},
			// PRIORITY 1 CRITICAL FEATURES
			"hardware_prefetch": map[string]interface{}{
				"prefetcher_count":     2,
				"sequential_accuracy":  0.85,
				"stride_accuracy":      0.70,
				"pattern_accuracy":     0.45,
				"prefetch_distance":    4,
				"bandwidth_usage":      0.05,
				"prefetch_benefit":     0.25,
			},
			"cache_line_conflicts": map[string]interface{}{
				"cache_line_size":         64,
				"false_sharing_detection": true,
				"conflict_threshold":      0.3,
				"conflict_penalty":        0.4,
				"conflict_window_ticks":   10,
			},
			"memory_ordering": map[string]interface{}{
				"ordering_model":        "tso",
				"reordering_window":     8,
				"memory_barrier_cost":   20.0,
				"load_store_reordering": false,
				"store_store_reordering": true,
				"load_load_reordering":  true,
				"reordering_benefit":    0.1,
				"base_reordering_delay": 2,
			},
			// PRIORITY 2 IMPORTANT FEATURES
			"memory_controller": map[string]interface{}{
				"controller_count":        2,
				"queue_depth":            16,
				"arbitration_policy":     "round_robin",
				"bandwidth_per_controller": 25.6,
				"controller_latency":     15.0,
				"backpressure_delay":     50.0,
				"base_queue_delay":       10.0,
			},
			"advanced_numa": map[string]interface{}{
				"node_affinity_policy":   "preferred",
				"migration_threshold":    0.7,
				"base_inter_node_latency": 100.0,
				"base_bandwidth_gbps":    100.0,
				"migration_benefit":      20.0,
				"strict_affinity_penalty": 50.0,
				"preferred_affinity_penalty": 20.0,
			},
			"virtual_memory": map[string]interface{}{
				"page_size":            4096,
				"tlb_size":             64,
				"tlb_hit_ratio":        0.95,
				"page_table_levels":    4,
				"page_walk_latency":    25.0,
				"swap_enabled":         true,
				"swap_latency":         10000000.0,
				"tlb_hit_latency":      1.0,
				"page_fault_probability": 0.01,
			},
			// PRIORITY 3 ENHANCEMENT FEATURES
			"ecc_modeling": map[string]interface{}{
				"ecc_enabled":           true,
				"single_bit_error_rate": 0.1,
				"multi_bit_error_rate":  0.001,
				"correction_latency":    50.0,
				"detection_latency":     5.0,
				"multi_bit_penalty":     1000.0,
			},
			"power_states": map[string]interface{}{
				"state_transition_cost": 100.0,
				"active_power_draw":     15.0,
				"standby_power_draw":    8.0,
				"sleep_power_draw":      2.0,
				"wakeup_latency":        500.0,
				"idle_threshold":        1000.0,
			},
			"enhanced_thermal": map[string]interface{}{
				"heat_dissipation_rate": 0.95,
				"thermal_capacity":      50.0,
				"ambient_temperature":   22.0,
				"base_heat_per_operation": 0.1,
				"thermal_zones": []map[string]interface{}{
					{
						"max_temperature":  85.0,
						"heat_generation":  5.0,
						"cooling_capacity": 20.0,
						"thermal_mass":     25.0,
					},
					{
						"max_temperature":  85.0,
						"heat_generation":  5.0,
						"cooling_capacity": 20.0,
						"thermal_mass":     25.0,
					},
				},
				"throttling_thresholds": []float64{75.0, 80.0, 85.0},
				"throttling_levels":     []float64{0.1, 0.3, 0.6},
			},
		},
	}
	pm.MemoryProfiles["ddr4_3200_server"] = ddr4Profile

	// DDR5-6400 Server Memory Profile
	ddr5Profile := &EngineProfile{
		Name:        "ddr5_6400_server",
		Type:        MemoryEngineType,
		Description: "DDR5-6400 Server Memory - 128GB capacity",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"capacity_gb":   128,
			"access_time":   5.0,   // nanoseconds (CL32 @ 6400MHz)
			"bandwidth_gbps": 204.8, // GB/s theoretical
		},
		TechnologySpecs: map[string]interface{}{
			"memory_type":    "DDR5",
			"frequency":      6400, // MHz
			"cas_latency":    32,
			"channels":       4,
		},
		EngineSpecific: map[string]interface{}{
			"ddr_timings": map[string]interface{}{
				"trcd": 32.0,
				"trp":  32.0,
				"tras": 52.0,
			},
			"numa_configuration": map[string]interface{}{
				"socket_count":           2,
				"cross_socket_penalty":   1.6,
				"inter_socket_latency_ns": 80.0,
				"local_access_ratio":     0.8,
			},
			"bandwidth_characteristics": map[string]interface{}{
				"peak_bandwidth_gbps":     204.8,
				"sustained_bandwidth_gbps": 180.0,
				"saturation_threshold":    0.75,
			},
			// PRIORITY 1 CRITICAL FEATURES - Enhanced for DDR5
			"hardware_prefetch": map[string]interface{}{
				"prefetcher_count":     4,
				"sequential_accuracy":  0.90,
				"stride_accuracy":      0.80,
				"pattern_accuracy":     0.60,
				"prefetch_distance":    8,
				"bandwidth_usage":      0.03,
				"prefetch_benefit":     0.35,
			},
			"cache_line_conflicts": map[string]interface{}{
				"cache_line_size":         64,
				"false_sharing_detection": true,
				"conflict_threshold":      0.25,
				"conflict_penalty":        0.3,
				"conflict_window_ticks":   8,
			},
			"memory_ordering": map[string]interface{}{
				"ordering_model":        "weak",
				"reordering_window":     16,
				"memory_barrier_cost":   15.0,
				"load_store_reordering": true,
				"store_store_reordering": true,
				"load_load_reordering":  true,
				"reordering_benefit":    0.15,
				"base_reordering_delay": 1,
			},
			// PRIORITY 2 IMPORTANT FEATURES - Enhanced for DDR5
			"memory_controller": map[string]interface{}{
				"controller_count":        4,
				"queue_depth":            32,
				"arbitration_policy":     "priority",
				"bandwidth_per_controller": 51.2,
				"controller_latency":     10.0,
				"backpressure_delay":     30.0,
				"base_queue_delay":       5.0,
			},
			"virtual_memory": map[string]interface{}{
				"page_size":            4096,
				"tlb_size":             128,
				"tlb_hit_ratio":        0.98,
				"page_table_levels":    5,
				"page_walk_latency":    20.0,
				"swap_enabled":         true,
				"swap_latency":         8000000.0,
				"tlb_hit_latency":      0.5,
				"page_fault_probability": 0.005,
			},
			// PRIORITY 3 ENHANCEMENT FEATURES - Enhanced for DDR5
			"ecc_modeling": map[string]interface{}{
				"ecc_enabled":           true,
				"single_bit_error_rate": 0.05,
				"multi_bit_error_rate":  0.0005,
				"correction_latency":    30.0,
				"detection_latency":     3.0,
				"multi_bit_penalty":     800.0,
			},
			"power_states": map[string]interface{}{
				"state_transition_cost": 50.0,
				"active_power_draw":     12.0,
				"standby_power_draw":    5.0,
				"sleep_power_draw":      1.0,
				"wakeup_latency":        300.0,
				"idle_threshold":        800.0,
			},
			"enhanced_thermal": map[string]interface{}{
				"heat_dissipation_rate": 0.98,
				"thermal_capacity":      60.0,
				"ambient_temperature":   22.0,
				"base_heat_per_operation": 0.08,
				"thermal_zones": []map[string]interface{}{
					{
						"max_temperature":  80.0,
						"heat_generation":  4.0,
						"cooling_capacity": 25.0,
						"thermal_mass":     30.0,
					},
				},
				"throttling_thresholds": []float64{70.0, 75.0, 80.0},
				"throttling_levels":     []float64{0.05, 0.2, 0.5},
			},
		},
	}
	pm.MemoryProfiles["ddr5_6400_server"] = ddr5Profile

	// HBM2 Server Memory Profile
	hbm2Profile := &EngineProfile{
		Name:        "hbm2_server",
		Type:        MemoryEngineType,
		Description: "HBM2 Server Memory - 64GB capacity, ultra-high bandwidth",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"capacity_gb":   64,
			"access_time":   2.5,   // nanoseconds (very low latency)
			"bandwidth_gbps": 819.2, // GB/s theoretical (4 stacks)
		},
		TechnologySpecs: map[string]interface{}{
			"memory_type":    "HBM2",
			"frequency":      1600, // MHz (effective 3200)
			"cas_latency":    14,
			"channels":       16, // 4 stacks × 4 channels each
		},
		EngineSpecific: map[string]interface{}{
			"ddr_timings": map[string]interface{}{
				"trcd": 14.0,
				"trp":  14.0,
				"tras": 28.0,
			},
			"numa_configuration": map[string]interface{}{
				"socket_count":           4,
				"cross_socket_penalty":   2.2,
				"inter_socket_latency_ns": 150.0,
				"local_access_ratio":     0.6,
			},
			"bandwidth_characteristics": map[string]interface{}{
				"peak_bandwidth_gbps":     819.2,
				"sustained_bandwidth_gbps": 750.0,
				"saturation_threshold":    0.70,
			},
			// PRIORITY 1 CRITICAL FEATURES - Optimized for HBM2
			"hardware_prefetch": map[string]interface{}{
				"prefetcher_count":     8,
				"sequential_accuracy":  0.95,
				"stride_accuracy":      0.90,
				"pattern_accuracy":     0.80,
				"prefetch_distance":    16,
				"bandwidth_usage":      0.02,
				"prefetch_benefit":     0.45,
			},
			"cache_line_conflicts": map[string]interface{}{
				"cache_line_size":         32, // Smaller cache lines for HBM2
				"false_sharing_detection": true,
				"conflict_threshold":      0.15,
				"conflict_penalty":        0.2,
				"conflict_window_ticks":   4,
			},
			"memory_ordering": map[string]interface{}{
				"ordering_model":        "weak",
				"reordering_window":     32,
				"memory_barrier_cost":   8.0,
				"load_store_reordering": true,
				"store_store_reordering": true,
				"load_load_reordering":  true,
				"reordering_benefit":    0.25,
				"base_reordering_delay": 1,
			},
			// PRIORITY 2 IMPORTANT FEATURES - Optimized for HBM2
			"memory_controller": map[string]interface{}{
				"controller_count":        8,
				"queue_depth":            64,
				"arbitration_policy":     "fair",
				"bandwidth_per_controller": 102.4,
				"controller_latency":     5.0,
				"backpressure_delay":     15.0,
				"base_queue_delay":       2.0,
			},
			"virtual_memory": map[string]interface{}{
				"page_size":            4096,
				"tlb_size":             256,
				"tlb_hit_ratio":        0.99,
				"page_table_levels":    4,
				"page_walk_latency":    10.0,
				"swap_enabled":         false, // HBM2 typically doesn't use swap
				"swap_latency":         0.0,
				"tlb_hit_latency":      0.2,
				"page_fault_probability": 0.001,
			},
			// PRIORITY 3 ENHANCEMENT FEATURES - Optimized for HBM2
			"ecc_modeling": map[string]interface{}{
				"ecc_enabled":           true,
				"single_bit_error_rate": 0.02,
				"multi_bit_error_rate":  0.0001,
				"correction_latency":    15.0,
				"detection_latency":     1.0,
				"multi_bit_penalty":     500.0,
			},
			"power_states": map[string]interface{}{
				"state_transition_cost": 25.0,
				"active_power_draw":     25.0, // Higher power for HBM2
				"standby_power_draw":    15.0,
				"sleep_power_draw":      5.0,
				"wakeup_latency":        100.0,
				"idle_threshold":        500.0,
			},
			"enhanced_thermal": map[string]interface{}{
				"heat_dissipation_rate": 0.92,
				"thermal_capacity":      40.0,
				"ambient_temperature":   22.0,
				"base_heat_per_operation": 0.15,
				"thermal_zones": []map[string]interface{}{
					{
						"max_temperature":  95.0, // Higher thermal limit
						"heat_generation":  8.0,
						"cooling_capacity": 35.0,
						"thermal_mass":     20.0,
					},
					{
						"max_temperature":  95.0,
						"heat_generation":  8.0,
						"cooling_capacity": 35.0,
						"thermal_mass":     20.0,
					},
					{
						"max_temperature":  95.0,
						"heat_generation":  8.0,
						"cooling_capacity": 35.0,
						"thermal_mass":     20.0,
					},
					{
						"max_temperature":  95.0,
						"heat_generation":  8.0,
						"cooling_capacity": 35.0,
						"thermal_mass":     20.0,
					},
				},
				"throttling_thresholds": []float64{85.0, 90.0, 95.0},
				"throttling_levels":     []float64{0.1, 0.4, 0.8},
			},
		},
	}
	pm.MemoryProfiles["hbm2_server"] = hbm2Profile

	// Samsung 980 PRO NVMe Profile
	samsung980Profile := &EngineProfile{
		Name:        "samsung_980_pro",
		Type:        StorageEngineType,
		Description: "Samsung 980 PRO NVMe SSD - 1TB capacity",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"capacity_gb":        1024,
			"max_iops":          1000000, // 1M IOPS
			"max_bandwidth_mbps": 7000,    // 7GB/s sequential
			"avg_latency_ms":     0.02,    // 20 microseconds
		},
		TechnologySpecs: map[string]interface{}{
			"storage_type":        "NVME_SSD",
			"queue_depth":         128,
			"controller_cache_mb": 1024,
			"thermal_limit":       70, // Celsius
		},
		EngineSpecific: map[string]interface{}{
			"iops_curves": map[string]interface{}{
				"optimal_threshold":  0.85,
				"warning_threshold":  0.95,
				"critical_threshold": 1.0,
			},
			"controller_cache": map[string]interface{}{
				"write_hit_ratio": 0.80,
				"read_hit_ratio":  0.60,
			},
			"wear_behavior": map[string]interface{}{
				"write_amplification": 1.1,
				"wear_leveling":       true,
				"over_provisioning":   0.07, // 7%
			},
		},
	}
	pm.StorageProfiles["samsung_980_pro"] = samsung980Profile
	
	// Gigabit Ethernet Network Profile
	gigabitProfile := &EngineProfile{
		Name:        "gigabit_ethernet",
		Type:        NetworkEngineType,
		Description: "Gigabit Ethernet - 1000 Mbps",
		Version:     "1.0",
		BaselinePerformance: map[string]float64{
			"bandwidth_mbps":   1000,
			"base_latency_ms":  1.0,
			"max_connections":  10000,
		},
		TechnologySpecs: map[string]interface{}{
			"network_type": "LAN",
			"protocol":     "TCP",
			"mtu":          1500,
		},
		EngineSpecific: map[string]interface{}{
			"congestion_curves": map[string]interface{}{
				"optimal_threshold":  0.70,
				"warning_threshold":  0.85,
				"critical_threshold": 0.95,
			},
			"protocol_overhead": map[string]interface{}{
				"tcp_header":  40,
				"http_header": 200,
				"efficiency":  1.0,
			},
			"geographic": map[string]interface{}{
				"distance_km":       0.1,   // 100m LAN
				"routing_overhead":  1.1,   // 10% routing overhead
				"fiber_optic_factor": 0.67, // Refractive index
			},
		},
	}
	pm.NetworkProfiles["gigabit_ethernet"] = gigabitProfile
}

// GetDefaultProfile returns the default profile for an engine type
func (pm *ProfileManager) GetDefaultProfile(engineType EngineType) (*EngineProfile, error) {
	switch engineType {
	case CPUEngineType:
		return pm.GetProfile(CPUEngineType, "intel_xeon_6248r")
	case MemoryEngineType:
		return pm.GetProfile(MemoryEngineType, "ddr4_3200_server")
	case StorageEngineType:
		return pm.GetProfile(StorageEngineType, "samsung_980_pro")
	case NetworkEngineType:
		return pm.GetProfile(NetworkEngineType, "gigabit_ethernet")
	default:
		return nil, fmt.Errorf("unknown engine type: %v", engineType)
	}
}

// ListProfiles returns all available profiles for an engine type
func (pm *ProfileManager) ListProfiles(engineType EngineType) []string {
	var profiles map[string]*EngineProfile
	
	switch engineType {
	case CPUEngineType:
		profiles = pm.CPUProfiles
	case MemoryEngineType:
		profiles = pm.MemoryProfiles
	case StorageEngineType:
		profiles = pm.StorageProfiles
	case NetworkEngineType:
		profiles = pm.NetworkProfiles
	default:
		return nil
	}
	
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	
	return names
}
