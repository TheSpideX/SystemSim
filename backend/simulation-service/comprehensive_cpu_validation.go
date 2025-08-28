package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/systemsim/simulation-service/internal/engines"
)

func main() {
	fmt.Println("🧪 COMPREHENSIVE CPU ENGINE TEST SUITE")
	fmt.Println(strings.Repeat("=", 80))

	// Initialize test suite
	suite := NewCPUTestSuite()
	
	// Run all tests
	if err := suite.RunAllTests(); err != nil {
		fmt.Printf("❌ Test suite failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🎉 ALL CPU ENGINE TESTS PASSED!")
	fmt.Printf("📊 Total Tests: %d | Passed: %d | Failed: %d\n", 
		suite.TotalTests, suite.PassedTests, suite.FailedTests)
}

type CPUTestSuite struct {
	CPU         *engines.CPUEngine
	Profile     *engines.EngineProfile
	TotalTests  int
	PassedTests int
	FailedTests int
}

func NewCPUTestSuite() *CPUTestSuite {
	return &CPUTestSuite{
		CPU: engines.NewCPUEngine(1000),
	}
}

func (suite *CPUTestSuite) RunAllTests() error {
	fmt.Println("\n🔧 PHASE 1: INITIALIZATION AND PROFILE LOADING")
	if err := suite.TestInitialization(); err != nil {
		return err
	}
	if err := suite.TestProfileLoading(); err != nil {
		return err
	}

	fmt.Println("\n⚡ PHASE 2: CORE CPU FUNCTIONALITY")
	if err := suite.TestBasicProcessing(); err != nil {
		return err
	}
	if err := suite.TestLanguageMultipliers(); err != nil {
		return err
	}
	if err := suite.TestComplexityFactors(); err != nil {
		return err
	}

	fmt.Println("\n🚀 PHASE 3: SIMD/VECTORIZATION")
	if err := suite.TestSIMDVectorization(); err != nil {
		return err
	}

	fmt.Println("\n💾 PHASE 4: CACHE BEHAVIOR")
	if err := suite.TestCacheBehavior(); err != nil {
		return err
	}

	fmt.Println("\n🌡️ PHASE 5: THERMAL BEHAVIOR")
	if err := suite.TestThermalBehavior(); err != nil {
		return err
	}

	fmt.Println("\n🔗 PHASE 6: NUMA TOPOLOGY")
	if err := suite.TestNUMABehavior(); err != nil {
		return err
	}

	fmt.Println("\n⚡ PHASE 7: BOOST/TURBO BEHAVIOR")
	if err := suite.TestBoostBehavior(); err != nil {
		return err
	}

	fmt.Println("\n🧵 PHASE 8: HYPERTHREADING")
	if err := suite.TestHyperthreading(); err != nil {
		return err
	}

	fmt.Println("\n🔄 PHASE 9: PARALLEL PROCESSING")
	if err := suite.TestParallelProcessing(); err != nil {
		return err
	}

	fmt.Println("\n🚌 PHASE 10: MEMORY BANDWIDTH")
	if err := suite.TestMemoryBandwidth(); err != nil {
		return err
	}

	fmt.Println("\n🌿 PHASE 11: BRANCH PREDICTION")
	if err := suite.TestBranchPrediction(); err != nil {
		return err
	}

	fmt.Println("\n📡 PHASE 12: ADVANCED PREFETCHING")
	if err := suite.TestAdvancedPrefetching(); err != nil {
		return err
	}

	fmt.Println("\n🎯 PHASE 13: INTEGRATION TESTS")
	if err := suite.TestIntegration(); err != nil {
		return err
	}

	fmt.Println("\n📈 PHASE 14: PERFORMANCE VALIDATION")
	if err := suite.TestPerformanceValidation(); err != nil {
		return err
	}

	fmt.Println("\n🔍 PHASE 15: EDGE CASES AND STRESS TESTS")
	if err := suite.TestEdgeCases(); err != nil {
		return err
	}

	return nil
}

// PHASE 1: INITIALIZATION AND PROFILE LOADING

func (suite *CPUTestSuite) TestInitialization() error {
	fmt.Println("  🔧 Testing CPU Engine Initialization...")
	
	// Test 1: Basic initialization
	if err := suite.runTest("CPU Engine Creation", func() error {
		if suite.CPU == nil {
			return fmt.Errorf("CPU engine is nil")
		}
		if suite.CPU.CoreCount <= 0 {
			return fmt.Errorf("invalid core count: %d", suite.CPU.CoreCount)
		}
		if suite.CPU.BaseClockGHz <= 0 {
			return fmt.Errorf("invalid base clock: %.2f", suite.CPU.BaseClockGHz)
		}
		return nil
	}); err != nil {
		return err
	}

	// Test 2: Default state validation
	if err := suite.runTest("Default State Validation", func() error {
		if suite.CPU.ActiveCores < 0 {
			return fmt.Errorf("invalid active cores: %d", suite.CPU.ActiveCores)
		}
		if suite.CPU.ThermalState.CurrentTemperatureC <= 0 {
			return fmt.Errorf("invalid temperature: %.2f", suite.CPU.ThermalState.CurrentTemperatureC)
		}
		if len(suite.CPU.CoreUtilization) != suite.CPU.CoreCount {
			return fmt.Errorf("core utilization array size mismatch")
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (suite *CPUTestSuite) TestProfileLoading() error {
	fmt.Println("  📋 Testing Profile Loading...")

	// Load a comprehensive test profile
	profile, err := suite.loadTestProfile("profiles/cpu/intel_xeon_server.json")
	if err != nil {
		return fmt.Errorf("failed to load test profile: %v", err)
	}
	suite.Profile = profile

	// Test 1: Profile loading
	if err := suite.runTest("Profile Loading", func() error {
		return suite.CPU.LoadProfile(profile)
	}); err != nil {
		return err
	}

	// Test 2: Profile data validation
	if err := suite.runTest("Profile Data Validation", func() error {
		if suite.CPU.CoreCount != 24 {
			return fmt.Errorf("expected 24 cores, got %d", suite.CPU.CoreCount)
		}
		if suite.CPU.BaseClockGHz != 3.0 {
			return fmt.Errorf("expected 3.0 GHz base clock, got %.2f", suite.CPU.BaseClockGHz)
		}
		if len(suite.CPU.LanguageMultipliers) == 0 {
			return fmt.Errorf("language multipliers not loaded")
		}
		if len(suite.CPU.ComplexityFactors) == 0 {
			return fmt.Errorf("complexity factors not loaded")
		}
		return nil
	}); err != nil {
		return err
	}

	// Test 3: SIMD configuration validation
	if err := suite.runTest("SIMD Configuration Validation", func() error {
		if suite.CPU.VectorizationState.VectorWidth != 512 {
			return fmt.Errorf("expected 512-bit vector width, got %d", suite.CPU.VectorizationState.VectorWidth)
		}
		if len(suite.CPU.VectorizationState.SupportedInstructions) == 0 {
			return fmt.Errorf("no SIMD instructions loaded")
		}
		if len(suite.CPU.VectorizationState.OperationVectorizability) == 0 {
			return fmt.Errorf("no operation vectorizability data loaded")
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 2: CORE CPU FUNCTIONALITY

func (suite *CPUTestSuite) TestBasicProcessing() error {
	fmt.Println("  ⚡ Testing Basic Processing...")

	// Test 1: Simple operation processing
	if err := suite.runTest("Simple Operation Processing", func() error {
		op := &engines.Operation{
			ID:         "test_basic_1",
			Type:       "compute",
			Complexity: "O(1)",
			Language:   "cpp",
			DataSize:   1024,
		}

		result := suite.CPU.ProcessOperation(op, 1000)
		if result == nil {
			return fmt.Errorf("processing returned nil result")
		}
		if result.ProcessingTime <= 0 {
			return fmt.Errorf("invalid processing time: %v", result.ProcessingTime)
		}
		if result.OperationID != op.ID {
			return fmt.Errorf("operation ID mismatch")
		}
		return nil
	}); err != nil {
		return err
	}

	// Test 2: Multiple operations with different characteristics
	if err := suite.runTest("Multiple Operations Processing", func() error {
		operations := []*engines.Operation{
			{ID: "test_basic_2a", Type: "compute", Complexity: "O(1)", Language: "cpp", DataSize: 10240},
			{ID: "test_basic_2b", Type: "compute", Complexity: "O(n)", Language: "python", DataSize: 10240},
			{ID: "test_basic_2c", Type: "compute", Complexity: "O(n²)", Language: "go", DataSize: 10240},
		}

		var results []*engines.OperationResult
		for i, op := range operations {
			result := suite.CPU.ProcessOperation(op, int64(1000+i))
			if result == nil {
				return fmt.Errorf("operation %s returned nil result", op.ID)
			}
			results = append(results, result)
		}

		// Verify all operations completed successfully with valid metrics
		for i, result := range results {
			if result.ProcessingTime <= 0 {
				return fmt.Errorf("operation %d has invalid processing time: %v", i, result.ProcessingTime)
			}
			if len(result.Metrics) == 0 {
				return fmt.Errorf("operation %d has no metrics", i)
			}
			// Check for required metrics
			if _, exists := result.Metrics["complexity_factor"]; !exists {
				return fmt.Errorf("operation %d missing complexity_factor metric", i)
			}
			if _, exists := result.Metrics["language_factor"]; !exists {
				return fmt.Errorf("operation %d missing language_factor metric", i)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (suite *CPUTestSuite) TestLanguageMultipliers() error {
	fmt.Println("  🗣️ Testing Language Multipliers...")

	// Test 1: Language performance differences
	if err := suite.runTest("Language Performance Differences", func() error {
		baseOp := engines.Operation{
			ID:         "test_lang",
			Type:       "compute",
			Complexity: "O(n)",
			DataSize:   10240,
		}

		languages := []string{"cpp", "python", "go", "java"}
		var results []*engines.OperationResult

		for i, lang := range languages {
			op := baseOp
			op.ID = fmt.Sprintf("test_lang_%s", lang)
			op.Language = lang
			result := suite.CPU.ProcessOperation(&op, int64(2000+i))
			results = append(results, result)
		}

		// C++ should be fastest, Python should be slowest
		cppTime := results[0].ProcessingTime
		pythonTime := results[1].ProcessingTime
		
		if pythonTime <= cppTime {
			return fmt.Errorf("Python should be slower than C++: cpp=%v, python=%v", cppTime, pythonTime)
		}

		// Verify language multipliers are applied
		for i, result := range results {
			if langFactor, ok := result.Metrics["language_factor"]; ok {
				if langFactor.(float64) <= 0 {
					return fmt.Errorf("invalid language factor for %s: %v", languages[i], langFactor)
				}
			} else {
				return fmt.Errorf("language factor missing for %s", languages[i])
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (suite *CPUTestSuite) TestComplexityFactors() error {
	fmt.Println("  🔢 Testing Complexity Factors...")

	// Test 1: Complexity scaling
	if err := suite.runTest("Complexity Scaling", func() error {
		baseOp := engines.Operation{
			ID:       "test_complexity",
			Type:     "compute",
			Language: "cpp",
			DataSize: 10240,
		}

		complexities := []string{"O(1)", "O(log n)", "O(n)", "O(n²)"}
		var results []*engines.OperationResult

		for i, complexity := range complexities {
			op := baseOp
			op.ID = fmt.Sprintf("test_complexity_%s", strings.ReplaceAll(complexity, " ", "_"))
			op.Complexity = complexity
			result := suite.CPU.ProcessOperation(&op, int64(3000+i))
			results = append(results, result)
		}

		// Verify processing times increase with complexity
		for i := 1; i < len(results); i++ {
			if results[i].ProcessingTime <= results[i-1].ProcessingTime {
				return fmt.Errorf("complexity %s should be slower than %s: %s=%v, %s=%v",
					complexities[i], complexities[i-1],
					complexities[i-1], results[i-1].ProcessingTime,
					complexities[i], results[i].ProcessingTime)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 3: SIMD/VECTORIZATION

func (suite *CPUTestSuite) TestSIMDVectorization() error {
	fmt.Println("  🚀 Testing SIMD/Vectorization...")

	// Test 1: Vectorizable vs non-vectorizable operations
	if err := suite.runTest("Vectorizable vs Non-Vectorizable Operations", func() error {
		vectorizableOp := &engines.Operation{
			ID:         "test_simd_vectorizable",
			Type:       "matrix_multiply",
			Complexity: "O(n³)",
			Language:   "cpp",
			DataSize:   102400,
		}

		nonVectorizableOp := &engines.Operation{
			ID:         "test_simd_nonvectorizable",
			Type:       "database_query",
			Complexity: "O(n log n)",
			Language:   "cpp",
			DataSize:   102400,
		}

		vectorResult := suite.CPU.ProcessOperation(vectorizableOp, 4000)
		nonVectorResult := suite.CPU.ProcessOperation(nonVectorizableOp, 4001)

		// Check vectorization ratios
		vectorRatio := vectorResult.Metrics["vectorization_ratio"].(float64)
		nonVectorRatio := nonVectorResult.Metrics["vectorization_ratio"].(float64)

		if vectorRatio <= nonVectorRatio {
			return fmt.Errorf("matrix operations should have higher vectorization than database queries")
		}

		if vectorRatio < 0.8 {
			return fmt.Errorf("matrix operations should have high vectorization ratio: got %.2f", vectorRatio)
		}

		if nonVectorRatio > 0.3 {
			return fmt.Errorf("database operations should have low vectorization ratio: got %.2f", nonVectorRatio)
		}

		return nil
	}); err != nil {
		return err
	}

	// Test 2: SIMD efficiency modeling
	if err := suite.runTest("SIMD Efficiency Modeling", func() error {
		op := &engines.Operation{
			ID:         "test_simd_efficiency",
			Type:       "array_sum",
			Complexity: "O(n)",
			Language:   "cpp",
			DataSize:   1024000,
		}

		result := suite.CPU.ProcessOperation(op, 4100)
		vectorSpeedup := result.Metrics["vector_speedup"].(float64)

		if vectorSpeedup < 1.0 {
			return fmt.Errorf("vector speedup should be >= 1.0, got %.2f", vectorSpeedup)
		}

		if vectorSpeedup > 16.0 {
			return fmt.Errorf("vector speedup should be <= 16.0, got %.2f", vectorSpeedup)
		}

		// Check that vector operations are being counted
		if suite.CPU.VectorizationState.VectorOperationsCount == 0 {
			return fmt.Errorf("vector operations should be counted")
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 4: CACHE BEHAVIOR

func (suite *CPUTestSuite) TestCacheBehavior() error {
	fmt.Println("  💾 Testing Cache Behavior...")

	// Test 1: Cache hit ratio validation
	if err := suite.runTest("Cache Hit Ratio Validation", func() error {
		// Process operations to exercise cache
		for i := 0; i < 5; i++ {
			op := &engines.Operation{
				ID:         fmt.Sprintf("test_cache_%d", i),
				Type:       "compute",
				Complexity: "O(n)",
				Language:   "cpp",
				DataSize:   10240,
			}
			suite.CPU.ProcessOperation(op, int64(5000+i))
		}

		// Check that cache hit ratios are within expected ranges
		if suite.CPU.CacheState.L1HitRatio < 0.0 || suite.CPU.CacheState.L1HitRatio > 1.0 {
			return fmt.Errorf("invalid L1 hit ratio: %.3f", suite.CPU.CacheState.L1HitRatio)
		}

		if suite.CPU.CacheState.L2HitRatio < 0.0 || suite.CPU.CacheState.L2HitRatio > 1.0 {
			return fmt.Errorf("invalid L2 hit ratio: %.3f", suite.CPU.CacheState.L2HitRatio)
		}

		if suite.CPU.CacheState.L3HitRatio < 0.0 || suite.CPU.CacheState.L3HitRatio > 1.0 {
			return fmt.Errorf("invalid L3 hit ratio: %.3f", suite.CPU.CacheState.L3HitRatio)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 5: THERMAL BEHAVIOR

func (suite *CPUTestSuite) TestThermalBehavior() error {
	fmt.Println("  🌡️ Testing Thermal Behavior...")

	// Test 1: Temperature tracking
	if err := suite.runTest("Temperature Tracking", func() error {
		initialTemp := suite.CPU.ThermalState.CurrentTemperatureC

		// Process intensive operations to generate heat
		for i := 0; i < 10; i++ {
			op := &engines.Operation{
				ID:         fmt.Sprintf("test_thermal_%d", i),
				Type:       "compute",
				Complexity: "O(n²)",
				Language:   "cpp",
				DataSize:   1024000,
			}
			suite.CPU.ProcessOperation(op, int64(6000+i))
		}

		finalTemp := suite.CPU.ThermalState.CurrentTemperatureC

		if finalTemp < initialTemp {
			return fmt.Errorf("temperature should increase under load: initial=%.2f, final=%.2f",
				initialTemp, finalTemp)
		}

		if finalTemp > suite.CPU.ThermalLimitC {
			return fmt.Errorf("temperature exceeded thermal limit: %.2f > %.2f",
				finalTemp, suite.CPU.ThermalLimitC)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 6: NUMA TOPOLOGY

func (suite *CPUTestSuite) TestNUMABehavior() error {
	fmt.Println("  🔗 Testing NUMA Topology...")

	// Test 1: NUMA configuration validation
	if err := suite.runTest("NUMA Configuration Validation", func() error {
		if suite.CPU.NUMAState.NumaNodes <= 0 {
			return fmt.Errorf("invalid NUMA node count: %d", suite.CPU.NUMAState.NumaNodes)
		}

		if suite.CPU.NUMAState.CrossSocketPenalty < 1.0 {
			return fmt.Errorf("cross-socket penalty should be >= 1.0: %.2f",
				suite.CPU.NUMAState.CrossSocketPenalty)
		}

		if suite.CPU.NUMAState.LocalMemoryRatio < 0.0 || suite.CPU.NUMAState.LocalMemoryRatio > 1.0 {
			return fmt.Errorf("invalid local memory ratio: %.2f",
				suite.CPU.NUMAState.LocalMemoryRatio)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 7: BOOST/TURBO BEHAVIOR

func (suite *CPUTestSuite) TestBoostBehavior() error {
	fmt.Println("  ⚡ Testing Boost/Turbo Behavior...")

	// Test 1: Boost state validation
	if err := suite.runTest("Boost State Validation", func() error {
		if suite.CPU.BoostState.SingleCoreBoostGHz <= suite.CPU.BaseClockGHz {
			return fmt.Errorf("single core boost should be higher than base clock")
		}

		if suite.CPU.BoostState.AllCoreBoostGHz <= suite.CPU.BaseClockGHz {
			return fmt.Errorf("all core boost should be higher than base clock")
		}

		if suite.CPU.BoostState.CurrentClockGHz <= 0 {
			return fmt.Errorf("current clock should be positive: %.2f",
				suite.CPU.BoostState.CurrentClockGHz)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 8: HYPERTHREADING

func (suite *CPUTestSuite) TestHyperthreading() error {
	fmt.Println("  🧵 Testing Hyperthreading...")

	// Test 1: Hyperthreading configuration
	if err := suite.runTest("Hyperthreading Configuration", func() error {
		if suite.CPU.HyperthreadingState.ThreadsPerCore <= 0 {
			return fmt.Errorf("invalid threads per core: %d",
				suite.CPU.HyperthreadingState.ThreadsPerCore)
		}

		if suite.CPU.HyperthreadingState.EfficiencyFactor < 0.0 ||
		   suite.CPU.HyperthreadingState.EfficiencyFactor > 1.0 {
			return fmt.Errorf("invalid efficiency factor: %.2f",
				suite.CPU.HyperthreadingState.EfficiencyFactor)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 9: PARALLEL PROCESSING

func (suite *CPUTestSuite) TestParallelProcessing() error {
	fmt.Println("  🔄 Testing Parallel Processing...")

	// Test 1: Parallel processing configuration
	if err := suite.runTest("Parallel Processing Configuration", func() error {
		if !suite.CPU.ParallelProcessingState.Enabled {
			return fmt.Errorf("parallel processing should be enabled")
		}

		if suite.CPU.ParallelProcessingState.MaxParallelizableRatio <= 0.0 ||
		   suite.CPU.ParallelProcessingState.MaxParallelizableRatio > 1.0 {
			return fmt.Errorf("invalid max parallelizable ratio: %.2f",
				suite.CPU.ParallelProcessingState.MaxParallelizableRatio)
		}

		if len(suite.CPU.ParallelProcessingState.ParallelizabilityMap) == 0 {
			return fmt.Errorf("parallelizability map should not be empty")
		}

		if len(suite.CPU.ParallelProcessingState.EfficiencyCurve) == 0 {
			return fmt.Errorf("efficiency curve should not be empty")
		}

		return nil
	}); err != nil {
		return err
	}

	// Test 2: Parallel speedup validation
	if err := suite.runTest("Parallel Speedup Validation", func() error {
		// Test single-threaded operation
		singleOp := &engines.Operation{
			ID:         "test_parallel_single",
			Type:       "compute",
			Complexity: "O(1)",
			Language:   "cpp",
			DataSize:   1024,
		}

		// Test highly parallelizable operation
		parallelOp := &engines.Operation{
			ID:         "test_parallel_multi",
			Type:       "compute",
			Complexity: "O(n²)",
			Language:   "cpp",
			DataSize:   1024000,
		}

		_ = suite.CPU.ProcessOperation(singleOp, 8000)
		_ = suite.CPU.ProcessOperation(parallelOp, 8001)

		// Parallel operation should use more cores
		if suite.CPU.ActiveCores <= 1 {
			return fmt.Errorf("parallel operation should use multiple cores")
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 10: MEMORY BANDWIDTH

func (suite *CPUTestSuite) TestMemoryBandwidth() error {
	fmt.Println("  🚌 Testing Memory Bandwidth...")

	// Test 1: Memory bandwidth configuration
	if err := suite.runTest("Memory Bandwidth Configuration", func() error {
		if suite.CPU.MemoryBandwidthState.TotalBandwidthGBps <= 0 {
			return fmt.Errorf("invalid total bandwidth: %.2f",
				suite.CPU.MemoryBandwidthState.TotalBandwidthGBps)
		}

		if suite.CPU.MemoryBandwidthState.PerCoreDegradation < 0 {
			return fmt.Errorf("invalid per-core degradation: %.3f",
				suite.CPU.MemoryBandwidthState.PerCoreDegradation)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 11: BRANCH PREDICTION

func (suite *CPUTestSuite) TestBranchPrediction() error {
	fmt.Println("  🌿 Testing Branch Prediction...")

	// Test 1: Branch prediction configuration
	if err := suite.runTest("Branch Prediction Configuration", func() error {
		if suite.CPU.BranchPredictionState.BaseAccuracy <= 0.0 ||
		   suite.CPU.BranchPredictionState.BaseAccuracy > 1.0 {
			return fmt.Errorf("invalid base accuracy: %.3f",
				suite.CPU.BranchPredictionState.BaseAccuracy)
		}

		if suite.CPU.BranchPredictionState.PipelineDepth <= 0 {
			return fmt.Errorf("invalid pipeline depth: %d",
				suite.CPU.BranchPredictionState.PipelineDepth)
		}

		if suite.CPU.BranchPredictionState.MispredictionPenalty < 0 {
			return fmt.Errorf("invalid misprediction penalty: %.3f",
				suite.CPU.BranchPredictionState.MispredictionPenalty)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 12: ADVANCED PREFETCHING

func (suite *CPUTestSuite) TestAdvancedPrefetching() error {
	fmt.Println("  📡 Testing Advanced Prefetching...")

	// Test 1: Prefetch configuration
	if err := suite.runTest("Prefetch Configuration", func() error {
		if suite.CPU.AdvancedPrefetchState.HardwarePrefetchers <= 0 {
			return fmt.Errorf("invalid hardware prefetchers: %d",
				suite.CPU.AdvancedPrefetchState.HardwarePrefetchers)
		}

		if suite.CPU.AdvancedPrefetchState.SequentialAccuracy <= 0.0 ||
		   suite.CPU.AdvancedPrefetchState.SequentialAccuracy > 1.0 {
			return fmt.Errorf("invalid sequential accuracy: %.3f",
				suite.CPU.AdvancedPrefetchState.SequentialAccuracy)
		}

		if suite.CPU.AdvancedPrefetchState.PrefetchDistance <= 0 {
			return fmt.Errorf("invalid prefetch distance: %d",
				suite.CPU.AdvancedPrefetchState.PrefetchDistance)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 13: INTEGRATION TESTS

func (suite *CPUTestSuite) TestIntegration() error {
	fmt.Println("  🎯 Testing Integration...")

	// Test 1: Complex workload simulation
	if err := suite.runTest("Complex Workload Simulation", func() error {
		workloads := []*engines.Operation{
			{ID: "integration_1", Type: "matrix_multiply", Complexity: "O(n³)", Language: "cpp", DataSize: 1024000},
			{ID: "integration_2", Type: "database_query", Complexity: "O(n log n)", Language: "java", DataSize: 102400},
			{ID: "integration_3", Type: "image_process", Complexity: "O(n)", Language: "python", DataSize: 512000},
			{ID: "integration_4", Type: "array_sum", Complexity: "O(n)", Language: "rust", DataSize: 204800},
			{ID: "integration_5", Type: "string_process", Complexity: "O(n)", Language: "go", DataSize: 51200},
		}

		var results []*engines.OperationResult
		for i, op := range workloads {
			result := suite.CPU.ProcessOperation(op, int64(9000+i))
			if result == nil {
				return fmt.Errorf("operation %s returned nil result", op.ID)
			}
			results = append(results, result)
		}

		// Verify all operations completed successfully
		for _, result := range results {
			if result.ProcessingTime <= 0 {
				return fmt.Errorf("invalid processing time for %s: %v",
					result.OperationID, result.ProcessingTime)
			}
			if len(result.Metrics) == 0 {
				return fmt.Errorf("no metrics for operation %s", result.OperationID)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 14: PERFORMANCE VALIDATION

func (suite *CPUTestSuite) TestPerformanceValidation() error {
	fmt.Println("  📈 Testing Performance Validation...")

	// Test 1: Performance consistency
	if err := suite.runTest("Performance Consistency", func() error {
		op := &engines.Operation{
			ID:         "perf_test",
			Type:       "compute",
			Complexity: "O(n)",
			Language:   "cpp",
			DataSize:   102400,
		}

		var times []time.Duration
		for i := 0; i < 5; i++ {
			result := suite.CPU.ProcessOperation(op, int64(10000+i))
			times = append(times, result.ProcessingTime)
		}

		// Check for reasonable consistency (within 50% variance)
		minTime := times[0]
		maxTime := times[0]
		for _, t := range times {
			if t < minTime {
				minTime = t
			}
			if t > maxTime {
				maxTime = t
			}
		}

		variance := float64(maxTime-minTime) / float64(minTime)
		if variance > 0.5 {
			return fmt.Errorf("performance variance too high: %.2f", variance)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PHASE 15: EDGE CASES AND STRESS TESTS

func (suite *CPUTestSuite) TestEdgeCases() error {
	fmt.Println("  🔍 Testing Edge Cases...")

	// Test 1: Zero data size
	if err := suite.runTest("Zero Data Size", func() error {
		op := &engines.Operation{
			ID:         "edge_zero_data",
			Type:       "compute",
			Complexity: "O(1)",
			Language:   "cpp",
			DataSize:   0,
		}

		result := suite.CPU.ProcessOperation(op, 11000)
		if result == nil {
			return fmt.Errorf("zero data size should still return a result")
		}
		if result.ProcessingTime <= 0 {
			return fmt.Errorf("zero data size should still have positive processing time")
		}

		return nil
	}); err != nil {
		return err
	}

	// Test 2: Very large data size
	if err := suite.runTest("Very Large Data Size", func() error {
		op := &engines.Operation{
			ID:         "edge_large_data",
			Type:       "compute",
			Complexity: "O(n)",
			Language:   "cpp",
			DataSize:   1024 * 1024 * 1024, // 1GB
		}

		result := suite.CPU.ProcessOperation(op, 11001)
		if result == nil {
			return fmt.Errorf("large data size should return a result")
		}
		if result.ProcessingTime <= 0 {
			return fmt.Errorf("large data size should have positive processing time")
		}

		return nil
	}); err != nil {
		return err
	}

	// Test 3: Unknown language
	if err := suite.runTest("Unknown Language", func() error {
		op := &engines.Operation{
			ID:         "edge_unknown_lang",
			Type:       "compute",
			Complexity: "O(n)",
			Language:   "unknown_language",
			DataSize:   10240,
		}

		result := suite.CPU.ProcessOperation(op, 11002)
		if result == nil {
			return fmt.Errorf("unknown language should still return a result")
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// HELPER METHODS

func (suite *CPUTestSuite) runTest(testName string, testFunc func() error) error {
	suite.TotalTests++
	fmt.Printf("    ▶ %s... ", testName)

	if err := testFunc(); err != nil {
		suite.FailedTests++
		fmt.Printf("❌ FAILED: %v\n", err)
		return fmt.Errorf("test '%s' failed: %v", testName, err)
	}

	suite.PassedTests++
	fmt.Println("✅ PASSED")
	return nil
}

func (suite *CPUTestSuite) loadTestProfile(path string) (*engines.EngineProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file: %v", err)
	}

	var profile engines.EngineProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile JSON: %v", err)
	}

	return &profile, nil
}

func (suite *CPUTestSuite) validateMetrics(result *engines.OperationResult, expectedMetrics []string) error {
	for _, metric := range expectedMetrics {
		if _, exists := result.Metrics[metric]; !exists {
			return fmt.Errorf("missing expected metric: %s", metric)
		}
	}
	return nil
}

func (suite *CPUTestSuite) validateNumericRange(value float64, min, max float64, name string) error {
	if value < min || value > max {
		return fmt.Errorf("%s out of range: %.3f (expected %.3f-%.3f)", name, value, min, max)
	}
	return nil
}

func (suite *CPUTestSuite) compareProcessingTimes(results []*engines.OperationResult, expectedOrder []string) error {
	if len(results) != len(expectedOrder) {
		return fmt.Errorf("result count mismatch: got %d, expected %d", len(results), len(expectedOrder))
	}

	for i := 1; i < len(results); i++ {
		if results[i-1].ProcessingTime >= results[i].ProcessingTime {
			return fmt.Errorf("processing time order incorrect: %s (%.3fms) should be faster than %s (%.3fms)",
				expectedOrder[i-1], float64(results[i-1].ProcessingTime)/float64(time.Millisecond),
				expectedOrder[i], float64(results[i].ProcessingTime)/float64(time.Millisecond))
		}
	}

	return nil
}

func (suite *CPUTestSuite) printTestSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🧪 CPU ENGINE TEST SUITE SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("📊 Total Tests: %d\n", suite.TotalTests)
	fmt.Printf("✅ Passed: %d\n", suite.PassedTests)
	fmt.Printf("❌ Failed: %d\n", suite.FailedTests)

	successRate := float64(suite.PassedTests) / float64(suite.TotalTests) * 100
	fmt.Printf("📈 Success Rate: %.1f%%\n", successRate)

	if suite.FailedTests == 0 {
		fmt.Println("🎉 ALL TESTS PASSED!")
	} else {
		fmt.Printf("⚠️  %d tests failed\n", suite.FailedTests)
	}

	fmt.Println(strings.Repeat("=", 80))
}

// Test data generators for stress testing

func (suite *CPUTestSuite) generateStressTestOperations(count int) []*engines.Operation {
	operations := make([]*engines.Operation, count)

	types := []string{"compute", "matrix_multiply", "image_process", "database_query", "array_sum"}
	complexities := []string{"O(1)", "O(log n)", "O(n)", "O(n log n)", "O(n²)"}
	languages := []string{"cpp", "python", "go", "java", "rust"}

	for i := 0; i < count; i++ {
		operations[i] = &engines.Operation{
			ID:         fmt.Sprintf("stress_test_%d", i),
			Type:       types[i%len(types)],
			Complexity: complexities[i%len(complexities)],
			Language:   languages[i%len(languages)],
			DataSize:   int64(1024 * (1 + i%1000)), // 1KB to 1MB
		}
	}

	return operations
}

func (suite *CPUTestSuite) benchmarkOperation(op *engines.Operation, iterations int) (time.Duration, error) {
	var totalTime time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		result := suite.CPU.ProcessOperation(op, int64(20000+i))
		elapsed := time.Since(start)

		if result == nil {
			return 0, fmt.Errorf("operation returned nil result")
		}

		totalTime += elapsed
	}

	return totalTime / time.Duration(iterations), nil
}
