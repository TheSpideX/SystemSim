package engines

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ProfileLoader handles loading profiles from files and directories
type ProfileLoader struct {
	ProfilesDir string
	Cache       map[string]*EngineProfile
}

// NewProfileLoader creates a new profile loader
func NewProfileLoader(profilesDir string) *ProfileLoader {
	return &ProfileLoader{
		ProfilesDir: profilesDir,
		Cache:       make(map[string]*EngineProfile),
	}
}

// LoadProfilesFromDirectory loads all profiles from the profiles directory
func (pl *ProfileLoader) LoadProfilesFromDirectory() (*ProfileManager, error) {
	pm := NewProfileManager()
	
	// Check if profiles directory exists
	if _, err := os.Stat(pl.ProfilesDir); os.IsNotExist(err) {
		// Create profiles directory with default profiles
		if err := pl.CreateDefaultProfileFiles(); err != nil {
			return nil, fmt.Errorf("failed to create default profile files: %w", err)
		}
	}
	
	// Walk through profiles directory
	err := filepath.WalkDir(pl.ProfilesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and non-JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		
		// Load profile from file
		profile, err := pl.LoadProfileFromFile(path)
		if err != nil {
			fmt.Printf("Warning: Failed to load profile from %s: %v\n", path, err)
			return nil // Continue loading other profiles
		}
		
		// Add to profile manager
		switch profile.Type {
		case CPUEngineType:
			pm.CPUProfiles[profile.Name] = profile
		case MemoryEngineType:
			pm.MemoryProfiles[profile.Name] = profile
		case StorageEngineType:
			pm.StorageProfiles[profile.Name] = profile
		case NetworkEngineType:
			pm.NetworkProfiles[profile.Name] = profile
		default:
			fmt.Printf("Warning: Unknown profile type %v in file %s\n", profile.Type, path)
		}
		
		// Cache the profile
		pl.Cache[profile.Name] = profile
		
		fmt.Printf("Loaded profile: %s (%s) from %s\n", profile.Name, profile.Type.String(), path)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk profiles directory: %w", err)
	}
	
	return pm, nil
}

// LoadProfileFromFile loads a single profile from a JSON file
func (pl *ProfileLoader) LoadProfileFromFile(filePath string) (*EngineProfile, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file %s: %w", filePath, err)
	}
	
	// Parse JSON
	var profile EngineProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON from %s: %w", filePath, err)
	}
	
	// Validate profile
	if err := pl.ValidateProfile(&profile); err != nil {
		return nil, fmt.Errorf("invalid profile in %s: %w", filePath, err)
	}
	
	return &profile, nil
}

// ValidateProfile validates a profile structure
func (pl *ProfileLoader) ValidateProfile(profile *EngineProfile) error {
	if profile.Name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	
	if profile.Type < CPUEngineType || profile.Type > NetworkEngineType {
		return fmt.Errorf("invalid profile type: %v", profile.Type)
	}
	
	if profile.BaselinePerformance == nil {
		return fmt.Errorf("baseline_performance cannot be nil")
	}
	
	// Type-specific validation
	switch profile.Type {
	case CPUEngineType:
		return pl.validateCPUProfile(profile)
	case MemoryEngineType:
		return pl.validateMemoryProfile(profile)
	case StorageEngineType:
		return pl.validateStorageProfile(profile)
	case NetworkEngineType:
		return pl.validateNetworkProfile(profile)
	}
	
	return nil
}

// validateCPUProfile validates CPU-specific profile fields
func (pl *ProfileLoader) validateCPUProfile(profile *EngineProfile) error {
	required := []string{"cores", "base_clock", "base_processing_time"}
	for _, field := range required {
		if _, ok := profile.BaselinePerformance[field]; !ok {
			return fmt.Errorf("CPU profile missing required field: %s", field)
		}
	}
	return nil
}

// validateMemoryProfile validates Memory-specific profile fields
func (pl *ProfileLoader) validateMemoryProfile(profile *EngineProfile) error {
	required := []string{"capacity_gb", "access_time", "bandwidth_gbps"}
	for _, field := range required {
		if _, ok := profile.BaselinePerformance[field]; !ok {
			return fmt.Errorf("Memory profile missing required field: %s", field)
		}
	}
	return nil
}

// validateStorageProfile validates Storage-specific profile fields
func (pl *ProfileLoader) validateStorageProfile(profile *EngineProfile) error {
	required := []string{"capacity_gb", "max_iops", "avg_latency_ms"}
	for _, field := range required {
		if _, ok := profile.BaselinePerformance[field]; !ok {
			return fmt.Errorf("Storage profile missing required field: %s", field)
		}
	}
	return nil
}

// validateNetworkProfile validates Network-specific profile fields
func (pl *ProfileLoader) validateNetworkProfile(profile *EngineProfile) error {
	required := []string{"bandwidth_mbps", "base_latency_ms"}
	for _, field := range required {
		if _, ok := profile.BaselinePerformance[field]; !ok {
			return fmt.Errorf("Network profile missing required field: %s", field)
		}
	}
	return nil
}

// CreateDefaultProfileFiles creates default profile JSON files
func (pl *ProfileLoader) CreateDefaultProfileFiles() error {
	// Create profiles directory
	if err := os.MkdirAll(pl.ProfilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}
	
	// Create subdirectories for organization
	subdirs := []string{"cpu", "memory", "storage", "network"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(pl.ProfilesDir, subdir), 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}
	
	// Get default profiles from profile manager
	pm := NewProfileManager()
	
	// Save CPU profiles
	for name, profile := range pm.CPUProfiles {
		if err := pl.SaveProfileToFile(profile, filepath.Join(pl.ProfilesDir, "cpu", name+".json")); err != nil {
			return fmt.Errorf("failed to save CPU profile %s: %w", name, err)
		}
	}
	
	// Save Memory profiles
	for name, profile := range pm.MemoryProfiles {
		if err := pl.SaveProfileToFile(profile, filepath.Join(pl.ProfilesDir, "memory", name+".json")); err != nil {
			return fmt.Errorf("failed to save Memory profile %s: %w", name, err)
		}
	}
	
	// Save Storage profiles
	for name, profile := range pm.StorageProfiles {
		if err := pl.SaveProfileToFile(profile, filepath.Join(pl.ProfilesDir, "storage", name+".json")); err != nil {
			return fmt.Errorf("failed to save Storage profile %s: %w", name, err)
		}
	}
	
	// Save Network profiles
	for name, profile := range pm.NetworkProfiles {
		if err := pl.SaveProfileToFile(profile, filepath.Join(pl.ProfilesDir, "network", name+".json")); err != nil {
			return fmt.Errorf("failed to save Network profile %s: %w", name, err)
		}
	}
	
	fmt.Printf("Created default profile files in %s\n", pl.ProfilesDir)
	return nil
}

// SaveProfileToFile saves a profile to a JSON file
func (pl *ProfileLoader) SaveProfileToFile(profile *EngineProfile, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	
	// Marshal profile to JSON with indentation
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile to JSON: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile file %s: %w", filePath, err)
	}
	
	return nil
}

// ListAvailableProfiles lists all available profiles in the directory
func (pl *ProfileLoader) ListAvailableProfiles() (map[string][]string, error) {
	profiles := make(map[string][]string)
	profiles["CPU"] = []string{}
	profiles["Memory"] = []string{}
	profiles["Storage"] = []string{}
	profiles["Network"] = []string{}
	
	// Check if profiles directory exists
	if _, err := os.Stat(pl.ProfilesDir); os.IsNotExist(err) {
		return profiles, nil
	}
	
	// Walk through profiles directory
	err := filepath.WalkDir(pl.ProfilesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and non-JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		
		// Determine engine type from directory structure
		relPath, _ := filepath.Rel(pl.ProfilesDir, path)
		parts := strings.Split(relPath, string(filepath.Separator))
		
		if len(parts) >= 2 {
			engineDir := parts[0]
			fileName := strings.TrimSuffix(parts[1], ".json")
			
			switch engineDir {
			case "cpu":
				profiles["CPU"] = append(profiles["CPU"], fileName)
			case "memory":
				profiles["Memory"] = append(profiles["Memory"], fileName)
			case "storage":
				profiles["Storage"] = append(profiles["Storage"], fileName)
			case "network":
				profiles["Network"] = append(profiles["Network"], fileName)
			}
		}
		
		return nil
	})
	
	return profiles, err
}

// GetProfilePath returns the expected file path for a profile
func (pl *ProfileLoader) GetProfilePath(engineType EngineType, profileName string) string {
	var subdir string
	switch engineType {
	case CPUEngineType:
		subdir = "cpu"
	case MemoryEngineType:
		subdir = "memory"
	case StorageEngineType:
		subdir = "storage"
	case NetworkEngineType:
		subdir = "network"
	default:
		subdir = "unknown"
	}
	
	return filepath.Join(pl.ProfilesDir, subdir, profileName+".json")
}
