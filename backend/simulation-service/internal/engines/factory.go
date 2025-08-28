package engines

import (
	"fmt"
)

// EngineFactory creates engines with profiles
type EngineFactory struct {
	ProfileManager *ProfileManager
	ProfileLoader  *ProfileLoader
}

// NewEngineFactory creates a new engine factory
func NewEngineFactory() *EngineFactory {
	return &EngineFactory{
		ProfileManager: NewProfileManager(),
		ProfileLoader:  NewProfileLoader("profiles"),
	}
}

// NewEngineFactoryWithPaths creates a new engine factory with custom paths
func NewEngineFactoryWithPaths(profilesDir string) *EngineFactory {
	return &EngineFactory{
		ProfileManager: NewProfileManager(),
		ProfileLoader:  NewProfileLoader(profilesDir),
	}
}

// LoadProfilesFromFiles loads all profiles from the profiles directory
func (ef *EngineFactory) LoadProfilesFromFiles() error {
	pm, err := ef.ProfileLoader.LoadProfilesFromDirectory()
	if err != nil {
		return fmt.Errorf("failed to load profiles from files: %w", err)
	}

	// Replace the profile manager with the loaded one
	ef.ProfileManager = pm
	return nil
}

// CreateEngine creates an engine with the specified profile
func (ef *EngineFactory) CreateEngine(engineType EngineType, profileName string, queueCapacity int) (BaseEngine, error) {
	// Get the profile
	profile, err := ef.ProfileManager.GetProfile(engineType, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	
	// Create the engine based on type
	var engine BaseEngine
	
	switch engineType {
	case CPUEngineType:
		cpuEngine := NewCPUEngine(queueCapacity)
		ef.configureCPUEngine(cpuEngine, profile)
		engine = cpuEngine
		
	case MemoryEngineType:
		memoryEngine := NewMemoryEngine(queueCapacity)
		ef.configureMemoryEngine(memoryEngine, profile)
		engine = memoryEngine
		
	case StorageEngineType:
		storageEngine := NewStorageEngine(queueCapacity)
		ef.configureStorageEngine(storageEngine, profile)
		engine = storageEngine
		
	case NetworkEngineType:
		networkEngine := NewNetworkEngine(queueCapacity)
		ef.configureNetworkEngine(networkEngine, profile)
		engine = networkEngine
		
	default:
		return nil, fmt.Errorf("unknown engine type: %v", engineType)
	}
	
	// Load the profile into the engine
	if err := engine.LoadProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to load profile into engine: %w", err)
	}
	
	return engine, nil
}

// CreateEngineWithDefaultProfile creates an engine with the default profile for its type
func (ef *EngineFactory) CreateEngineWithDefaultProfile(engineType EngineType, queueCapacity int) (BaseEngine, error) {
	profile, err := ef.ProfileManager.GetDefaultProfile(engineType)
	if err != nil {
		return nil, fmt.Errorf("failed to get default profile: %w", err)
	}
	
	return ef.CreateEngine(engineType, profile.Name, queueCapacity)
}

// configureCPUEngine configures a CPU engine with profile-specific settings
func (ef *EngineFactory) configureCPUEngine(cpu *CPUEngine, profile *EngineProfile) {
	// Configure from baseline performance
	if cores, ok := profile.BaselinePerformance["cores"]; ok {
		cpu.CoreCount = int(cores)
		cpu.CoreUtilization = make([]float64, cpu.CoreCount)
	}
	
	if baseClock, ok := profile.BaselinePerformance["base_clock"]; ok {
		cpu.BaseClockGHz = baseClock
	}
	
	if boostClock, ok := profile.BaselinePerformance["boost_clock"]; ok {
		cpu.BoostClockGHz = boostClock
	}
	
	// Configure from technology specs
	if specs, ok := profile.TechnologySpecs["tdp"]; ok {
		if tdp, ok := specs.(float64); ok {
			cpu.TDP = tdp
		}
	}
	
	if specs, ok := profile.TechnologySpecs["thermal_limit"]; ok {
		if limit, ok := specs.(float64); ok {
			cpu.ThermalLimitC = limit
		}
	}
	
	if specs, ok := profile.TechnologySpecs["cache_l1_kb"]; ok {
		if cache, ok := specs.(float64); ok {
			cpu.CacheL1KB = int(cache)
		}
	}
	
	if specs, ok := profile.TechnologySpecs["cache_l2_kb"]; ok {
		if cache, ok := specs.(float64); ok {
			cpu.CacheL2KB = int(cache)
		}
	}
	
	if specs, ok := profile.TechnologySpecs["cache_l3_mb"]; ok {
		if cache, ok := specs.(float64); ok {
			cpu.CacheL3MB = int(cache)
		}
	}
	
	// Configure engine-specific settings
	if engineSpecific, ok := profile.EngineSpecific["cache_behavior"]; ok {
		if cacheConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if hitRatio, ok := cacheConfig["l3_hit_ratio"].(float64); ok {
				cpu.CacheState.ConvergedHitRatio = hitRatio
			}
		}
	}
	
	if engineSpecific, ok := profile.EngineSpecific["thermal_behavior"]; ok {
		if thermalConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if coolingCapacity, ok := thermalConfig["cooling_capacity"].(float64); ok {
				cpu.ThermalState.CoolingCapacity = coolingCapacity
			}
			if ambientTemp, ok := thermalConfig["ambient_temp"].(float64); ok {
				cpu.ThermalState.AmbientTemperatureC = ambientTemp
			}
		}
	}
}

// configureMemoryEngine configures a Memory engine with profile-specific settings
func (ef *EngineFactory) configureMemoryEngine(memory *MemoryEngine, profile *EngineProfile) {
	// Configure from baseline performance
	if capacity, ok := profile.BaselinePerformance["capacity_gb"]; ok {
		memory.CapacityGB = int64(capacity)
	}
	
	if accessTime, ok := profile.BaselinePerformance["access_time"]; ok {
		memory.AccessTimeNs = accessTime
	}
	
	if bandwidth, ok := profile.BaselinePerformance["bandwidth_gbps"]; ok {
		memory.BandwidthGBps = bandwidth
	}
	
	// Configure memory type from technology specs
	if specs, ok := profile.TechnologySpecs["memory_type"]; ok {
		if memType, ok := specs.(string); ok {
			memory.MemoryType = memType
		}
	}

	// Configure from baseline performance
	if freq, ok := profile.BaselinePerformance["frequency_mhz"]; ok {
		memory.FrequencyMHz = int(freq)
	}

	if cas, ok := profile.BaselinePerformance["cas_latency"]; ok {
		memory.CASLatency = int(cas)
	}

	if channels, ok := profile.BaselinePerformance["channels"]; ok {
		memory.Channels = int(channels)
	}
	
	// Configure DDR timings from engine-specific settings
	if engineSpecific, ok := profile.EngineSpecific["ddr_timings"]; ok {
		if timingConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if trcd, ok := timingConfig["trcd"].(float64); ok {
				memory.TimingState.tRCD = int(trcd)
			}
			if trp, ok := timingConfig["trp"].(float64); ok {
				memory.TimingState.tRP = int(trp)
			}
			if tras, ok := timingConfig["tras"].(float64); ok {
				memory.TimingState.tRAS = int(tras)
			}
			if trefi, ok := timingConfig["trefi"].(float64); ok {
				memory.TimingState.tREFI = int(trefi)
			}
			if bankGroups, ok := timingConfig["bank_groups"].(float64); ok {
				memory.TimingState.BankGroups = int(bankGroups)
			}
			if banksPerGroup, ok := timingConfig["banks_per_group"].(float64); ok {
				memory.TimingState.BanksPerGroup = int(banksPerGroup)
			}
		}
	}

	// Configure NUMA settings
	if engineSpecific, ok := profile.EngineSpecific["numa_configuration"]; ok {
		if numaConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if socketCount, ok := numaConfig["socket_count"].(float64); ok {
				memory.NUMAState.SocketCount = int(socketCount)
			}
			if penalty, ok := numaConfig["cross_socket_penalty"].(float64); ok {
				memory.NUMAState.CrossSocketPenalty = penalty
			}
			if latency, ok := numaConfig["inter_socket_latency_ns"].(float64); ok {
				memory.NUMAState.InterSocketLatencyNs = latency
			}
			if ratio, ok := numaConfig["local_access_ratio"].(float64); ok {
				memory.NUMAState.LocalAccessRatio = ratio
			}
		}
	}
}

// configureStorageEngine configures a Storage engine with profile-specific settings
func (ef *EngineFactory) configureStorageEngine(storage *StorageEngine, profile *EngineProfile) {
	// Configure from baseline performance
	// Use the LoadProfile method instead of direct field assignment
	err := storage.LoadProfile(profile)
	if err != nil {
		// Log error but continue with defaults
		fmt.Printf("Warning: failed to load storage profile: %v\n", err)
	}

	// All configuration is now handled by LoadProfile method
}

// configureNetworkEngine configures a Network engine with profile-specific settings
func (ef *EngineFactory) configureNetworkEngine(network *NetworkEngine, profile *EngineProfile) {
	// Configure from baseline performance
	if bandwidth, ok := profile.BaselinePerformance["bandwidth_mbps"]; ok {
		network.BandwidthMbps = int(bandwidth)
	}
	
	if latency, ok := profile.BaselinePerformance["base_latency_ms"]; ok {
		network.BaseLatencyMs = latency
	}
	
	if connections, ok := profile.BaselinePerformance["max_connections"]; ok {
		network.MaxConnections = int(connections)
	}
	
	// Configure from technology specs
	if specs, ok := profile.TechnologySpecs["protocol"]; ok {
		if protocol, ok := specs.(string); ok {
			network.Protocol = protocol
			network.configureProtocol() // Reconfigure protocol-specific settings
		}
	}
	
	if specs, ok := profile.TechnologySpecs["network_type"]; ok {
		if netType, ok := specs.(string); ok {
			network.NetworkType = netType
		}
	}
	
	// Configure engine-specific settings
	if engineSpecific, ok := profile.EngineSpecific["geographic"]; ok {
		if geoConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if distance, ok := geoConfig["distance_km"].(float64); ok {
				network.GeographicDistance = distance
				network.calculatePhysicsLatency() // Recalculate physics-based latency
			}
			if overhead, ok := geoConfig["routing_overhead"].(float64); ok {
				network.GeographicState.RoutingOverhead = overhead
				network.calculatePhysicsLatency() // Recalculate with new overhead
			}
			if fiberFactor, ok := geoConfig["fiber_optic_factor"].(float64); ok {
				network.GeographicState.FiberOpticFactor = fiberFactor
				network.calculatePhysicsLatency() // Recalculate with new fiber factor
			}
		}
	}
	
	if engineSpecific, ok := profile.EngineSpecific["protocol_overhead"]; ok {
		if protocolConfig, ok := engineSpecific.(map[string]interface{}); ok {
			if efficiency, ok := protocolConfig["efficiency"].(float64); ok {
				network.ProtocolState.ProtocolEfficiency = efficiency
			}
		}
	}
}

// GetAvailableProfiles returns all available profiles grouped by engine type
func (ef *EngineFactory) GetAvailableProfiles() map[string][]string {
	return map[string][]string{
		"CPU":     ef.ProfileManager.ListProfiles(CPUEngineType),
		"Memory":  ef.ProfileManager.ListProfiles(MemoryEngineType),
		"Storage": ef.ProfileManager.ListProfiles(StorageEngineType),
		"Network": ef.ProfileManager.ListProfiles(NetworkEngineType),
	}
}

// GetAvailableProfilesFromFiles returns all available profiles from files
func (ef *EngineFactory) GetAvailableProfilesFromFiles() (map[string][]string, error) {
	return ef.ProfileLoader.ListAvailableProfiles()
}

// CreateEngineFromFile creates an engine with a profile loaded from file
func (ef *EngineFactory) CreateEngineFromFile(engineType EngineType, profileName string, queueCapacity int) (BaseEngine, error) {
	// Load profile from file
	profilePath := ef.ProfileLoader.GetProfilePath(engineType, profileName)
	profile, err := ef.ProfileLoader.LoadProfileFromFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile from file: %w", err)
	}

	// Create engine with loaded profile
	return ef.createEngineWithProfile(engineType, profile, queueCapacity)
}

// Note: State persistence is now handled directly by EngineWrapper
// Use wrapper.SaveState() and wrapper.LoadState() instead

// createEngineWithProfile creates an engine with a specific profile (internal helper)
func (ef *EngineFactory) createEngineWithProfile(engineType EngineType, profile *EngineProfile, queueCapacity int) (BaseEngine, error) {
	// Create the engine based on type
	var engine BaseEngine

	switch engineType {
	case CPUEngineType:
		cpuEngine := NewCPUEngine(queueCapacity)
		ef.configureCPUEngine(cpuEngine, profile)
		engine = cpuEngine

	case MemoryEngineType:
		memoryEngine := NewMemoryEngine(queueCapacity)
		ef.configureMemoryEngine(memoryEngine, profile)
		engine = memoryEngine

	case StorageEngineType:
		storageEngine := NewStorageEngine(queueCapacity)
		ef.configureStorageEngine(storageEngine, profile)
		engine = storageEngine

	case NetworkEngineType:
		networkEngine := NewNetworkEngine(queueCapacity)
		ef.configureNetworkEngine(networkEngine, profile)
		engine = networkEngine

	default:
		return nil, fmt.Errorf("unknown engine type: %v", engineType)
	}

	// Load the profile into the engine
	if err := engine.LoadProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to load profile into engine: %w", err)
	}

	return engine, nil
}

// CreateDefaultProfileFiles creates default profile files in the profiles directory
func (ef *EngineFactory) CreateDefaultProfileFiles() error {
	return ef.ProfileLoader.CreateDefaultProfileFiles()
}
