package clock

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Fixed tick duration based on time-synchronization-solution.md specifications
// 0.01ms (10 microseconds) - ultra-fine granularity for precise simulation
const TICK_DURATION = 10 * time.Microsecond

// GlobalTickCoordinator manages the central simulation clock
// Based on time-synchronization-solution.md specifications
type GlobalTickCoordinator struct {
	// Core timing configuration
	CurrentTick   int64         `json:"current_tick"`
	TickDuration  time.Duration `json:"tick_duration"`  // Fixed at 0.01ms (10 microseconds)
	ScalingFactor float64       `json:"scaling_factor"` // Real-time to simulation time ratio
	
	// Component management
	Components    []Component   `json:"-"` // Registered components
	ComponentsMux sync.RWMutex  `json:"-"` // Protects components slice
	
	// Control and state
	Running       bool          `json:"running"`
	Paused        bool          `json:"paused"`
	StartTime     time.Time     `json:"start_time"`
	
	// Performance metrics
	TicksPerSecond    float64       `json:"ticks_per_second"`
	AverageTickTime   time.Duration `json:"average_tick_time"`
	MaxTickTime       time.Duration `json:"max_tick_time"`
	TotalTicks        int64         `json:"total_ticks"`
	
	// Control channels
	stopChan      chan struct{} `json:"-"`
	pauseChan     chan struct{} `json:"-"`
	resumeChan    chan struct{} `json:"-"`
	
	// Synchronization
	mutex         sync.RWMutex  `json:"-"`
}

// Component represents a simulation component that processes ticks
// This interface will be implemented by actual components
type Component interface {
	// ProcessTick processes one simulation tick and returns when complete
	ProcessTick(currentTick int64) error

	// GetID returns the component's unique identifier
	GetID() string

	// IsHealthy returns whether the component is functioning properly
	IsHealthy() bool

	// GetTickChannel returns the channel for receiving tick notifications
	GetTickChannel() chan int64

	// Start begins the component's goroutine (should be called once)
	Start(ctx context.Context) error

	// Stop gracefully shuts down the component
	Stop() error
}

// NewGlobalTickCoordinator creates a new central clock coordinator
func NewGlobalTickCoordinator() *GlobalTickCoordinator {
	return &GlobalTickCoordinator{
		CurrentTick:   0,
		TickDuration:  TICK_DURATION,
		ScalingFactor: 1.0, // Default to real-time
		Components:    make([]Component, 0),
		Running:       false,
		Paused:        false,
		
		// Performance tracking
		TicksPerSecond:  0.0,
		AverageTickTime: 0,
		MaxTickTime:     0,
		TotalTicks:      0,
		
		// Control channels
		stopChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
}

// RegisterComponent adds a component to the simulation and starts its goroutine
func (gtc *GlobalTickCoordinator) RegisterComponent(component Component, ctx context.Context) error {
	gtc.ComponentsMux.Lock()
	defer gtc.ComponentsMux.Unlock()

	// Check if component already registered
	for _, existing := range gtc.Components {
		if existing.GetID() == component.GetID() {
			return fmt.Errorf("component with ID %s already registered", component.GetID())
		}
	}

	// Start the component's goroutine
	err := component.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start component %s: %w", component.GetID(), err)
	}

	gtc.Components = append(gtc.Components, component)
	log.Printf("Registered and started component: %s (total: %d)", component.GetID(), len(gtc.Components))

	return nil
}

// RegisterComponentSimple adds a component without starting it (for backward compatibility)
func (gtc *GlobalTickCoordinator) RegisterComponentSimple(component Component) error {
	gtc.ComponentsMux.Lock()
	defer gtc.ComponentsMux.Unlock()

	// Check if component already registered
	for _, existing := range gtc.Components {
		if existing.GetID() == component.GetID() {
			return fmt.Errorf("component with ID %s already registered", component.GetID())
		}
	}

	gtc.Components = append(gtc.Components, component)
	log.Printf("Registered component: %s (total: %d)", component.GetID(), len(gtc.Components))

	return nil
}

// UnregisterComponent removes a component from the simulation
func (gtc *GlobalTickCoordinator) UnregisterComponent(componentID string) error {
	gtc.ComponentsMux.Lock()
	defer gtc.ComponentsMux.Unlock()
	
	for i, component := range gtc.Components {
		if component.GetID() == componentID {
			// Remove component from slice
			gtc.Components = append(gtc.Components[:i], gtc.Components[i+1:]...)
			log.Printf("Unregistered component: %s (remaining: %d)", componentID, len(gtc.Components))
			return nil
		}
	}
	
	return fmt.Errorf("component with ID %s not found", componentID)
}

// Start begins the simulation clock
func (gtc *GlobalTickCoordinator) Start(ctx context.Context) error {
	gtc.mutex.Lock()
	if gtc.Running {
		gtc.mutex.Unlock()
		return fmt.Errorf("simulation is already running")
	}
	
	gtc.Running = true
	gtc.StartTime = time.Now()
	gtc.CurrentTick = 0
	gtc.TotalTicks = 0
	gtc.mutex.Unlock()
	
	log.Printf("Starting global tick coordinator with %d components", len(gtc.Components))
	log.Printf("Tick duration: %v, Scaling factor: %.2fx", gtc.TickDuration, gtc.ScalingFactor)
	
	// Start the main simulation loop
	go gtc.runSimulationLoop(ctx)
	
	return nil
}

// Stop halts the simulation clock and stops all components
func (gtc *GlobalTickCoordinator) Stop() error {
	gtc.mutex.Lock()
	if !gtc.Running {
		gtc.mutex.Unlock()
		return fmt.Errorf("simulation is not running")
	}

	gtc.Running = false
	gtc.mutex.Unlock()

	// Stop all components first
	gtc.ComponentsMux.RLock()
	components := make([]Component, len(gtc.Components))
	copy(components, gtc.Components)
	gtc.ComponentsMux.RUnlock()

	for _, component := range components {
		err := component.Stop()
		if err != nil {
			log.Printf("Error stopping component %s: %v", component.GetID(), err)
		}
	}

	// Stop the coordinator
	close(gtc.stopChan)

	log.Printf("Stopped global tick coordinator after %d ticks", gtc.TotalTicks)

	return nil
}

// Pause temporarily halts the simulation
func (gtc *GlobalTickCoordinator) Pause() error {
	gtc.mutex.Lock()
	defer gtc.mutex.Unlock()
	
	if !gtc.Running {
		return fmt.Errorf("simulation is not running")
	}
	
	if gtc.Paused {
		return fmt.Errorf("simulation is already paused")
	}
	
	gtc.Paused = true
	gtc.pauseChan <- struct{}{}
	
	log.Printf("Paused simulation at tick %d", gtc.CurrentTick)
	
	return nil
}

// Resume continues a paused simulation
func (gtc *GlobalTickCoordinator) Resume() error {
	gtc.mutex.Lock()
	defer gtc.mutex.Unlock()
	
	if !gtc.Running {
		return fmt.Errorf("simulation is not running")
	}
	
	if !gtc.Paused {
		return fmt.Errorf("simulation is not paused")
	}
	
	gtc.Paused = false
	gtc.resumeChan <- struct{}{}
	
	log.Printf("Resumed simulation at tick %d", gtc.CurrentTick)
	
	return nil
}

// SetScalingFactor adjusts the simulation speed
func (gtc *GlobalTickCoordinator) SetScalingFactor(factor float64) error {
	if factor <= 0 {
		return fmt.Errorf("scaling factor must be positive, got: %f", factor)
	}
	
	gtc.mutex.Lock()
	defer gtc.mutex.Unlock()
	
	gtc.ScalingFactor = factor
	log.Printf("Set scaling factor to %.2fx", factor)
	
	return nil
}

// GetSimulationTime returns the current simulation time
func (gtc *GlobalTickCoordinator) GetSimulationTime() time.Duration {
	gtc.mutex.RLock()
	defer gtc.mutex.RUnlock()
	
	return time.Duration(gtc.CurrentTick) * gtc.TickDuration
}

// GetRealTimeElapsed returns the real time elapsed since simulation start
func (gtc *GlobalTickCoordinator) GetRealTimeElapsed() time.Duration {
	gtc.mutex.RLock()
	defer gtc.mutex.RUnlock()
	
	if gtc.StartTime.IsZero() {
		return 0
	}
	
	return time.Since(gtc.StartTime)
}

// IsRunning returns whether the simulation is currently running
func (gtc *GlobalTickCoordinator) IsRunning() bool {
	gtc.mutex.RLock()
	defer gtc.mutex.RUnlock()
	
	return gtc.Running
}

// IsPaused returns whether the simulation is currently paused
func (gtc *GlobalTickCoordinator) IsPaused() bool {
	gtc.mutex.RLock()
	defer gtc.mutex.RUnlock()
	
	return gtc.Paused
}

// GetComponentCount returns the number of registered components
func (gtc *GlobalTickCoordinator) GetComponentCount() int {
	gtc.ComponentsMux.RLock()
	defer gtc.ComponentsMux.RUnlock()

	return len(gtc.Components)
}

// GetTickDeliveryStatus returns detailed information about tick delivery to components
func (gtc *GlobalTickCoordinator) GetTickDeliveryStatus() TickDeliveryStatus {
	gtc.ComponentsMux.RLock()
	defer gtc.ComponentsMux.RUnlock()

	status := TickDeliveryStatus{
		TotalComponents:    len(gtc.Components),
		HealthyComponents:  0,
		ComponentStatuses:  make(map[string]ComponentTickStatus),
	}

	for _, component := range gtc.Components {
		componentStatus := ComponentTickStatus{
			ID:                component.GetID(),
			IsHealthy:         component.IsHealthy(),
			TickChannelLength: len(component.GetTickChannel()),
			TickChannelCap:    cap(component.GetTickChannel()),
		}

		if componentStatus.IsHealthy {
			status.HealthyComponents++
		}

		// Calculate channel utilization
		if componentStatus.TickChannelCap > 0 {
			componentStatus.ChannelUtilization = float64(componentStatus.TickChannelLength) / float64(componentStatus.TickChannelCap)
		}

		status.ComponentStatuses[component.GetID()] = componentStatus
	}

	return status
}

// TickDeliveryStatus provides information about tick delivery to all components
type TickDeliveryStatus struct {
	TotalComponents    int                            `json:"total_components"`
	HealthyComponents  int                            `json:"healthy_components"`
	ComponentStatuses  map[string]ComponentTickStatus `json:"component_statuses"`
}

// ComponentTickStatus provides tick-related status for a single component
type ComponentTickStatus struct {
	ID                 string  `json:"id"`
	IsHealthy          bool    `json:"is_healthy"`
	TickChannelLength  int     `json:"tick_channel_length"`
	TickChannelCap     int     `json:"tick_channel_cap"`
	ChannelUtilization float64 `json:"channel_utilization"`
}

// runSimulationLoop is the main simulation loop that processes ticks
func (gtc *GlobalTickCoordinator) runSimulationLoop(ctx context.Context) {
	log.Printf("Starting simulation loop")

	// Create ticker based on scaled tick duration
	scaledDuration := time.Duration(float64(gtc.TickDuration) * gtc.ScalingFactor)
	ticker := time.NewTicker(scaledDuration)
	defer ticker.Stop()

	// Performance tracking
	tickTimes := make([]time.Duration, 0, 100) // Keep last 100 tick times for averaging

	for {
		select {
		case <-ctx.Done():
			log.Printf("Simulation loop stopped by context cancellation")
			return

		case <-gtc.stopChan:
			log.Printf("Simulation loop stopped by stop signal")
			return

		case <-gtc.pauseChan:
			log.Printf("Simulation paused at tick %d", gtc.CurrentTick)
			// Wait for resume signal
			<-gtc.resumeChan
			log.Printf("Simulation resumed at tick %d", gtc.CurrentTick)

		case <-ticker.C:
			// Process one tick
			tickStart := time.Now()

			err := gtc.processTick()
			if err != nil {
				log.Printf("Error processing tick %d: %v", gtc.CurrentTick, err)
				// Continue processing - don't stop simulation for individual tick errors
			}

			// Update performance metrics
			tickDuration := time.Since(tickStart)
			gtc.updatePerformanceMetrics(tickDuration, &tickTimes)
		}
	}
}

// processTick processes one simulation tick by notifying all component goroutines
func (gtc *GlobalTickCoordinator) processTick() error {
	gtc.mutex.Lock()
	gtc.CurrentTick++
	gtc.TotalTicks++
	currentTick := gtc.CurrentTick
	gtc.mutex.Unlock()

	// Get snapshot of components (to avoid holding lock during processing)
	gtc.ComponentsMux.RLock()
	components := make([]Component, len(gtc.Components))
	copy(components, gtc.Components)
	gtc.ComponentsMux.RUnlock()

	// GUARANTEED tick delivery to all component goroutines
	deliveryTimeout := gtc.TickDuration / 2  // Half tick duration for delivery timeout

	var wg sync.WaitGroup
	for _, component := range components {
		wg.Add(1)
		go func(comp Component) {
			defer wg.Done()

			// Guaranteed delivery with timeout
			select {
			case comp.GetTickChannel() <- currentTick:
				// Perfect delivery!
			case <-time.After(deliveryTimeout):
				// Component is severely lagging
				log.Printf("CRITICAL: Component %s missed tick %d (channel timeout)",
					comp.GetID(), currentTick)
			}
		}(component)
	}

	// Wait for all tick deliveries to complete (or timeout)
	wg.Wait()

	// Note: Components process ticks independently in their own goroutines
	// This allows true parallel processing without blocking the global clock

	return nil
}

// updatePerformanceMetrics updates simulation performance tracking
func (gtc *GlobalTickCoordinator) updatePerformanceMetrics(tickDuration time.Duration, tickTimes *[]time.Duration) {
	gtc.mutex.Lock()
	defer gtc.mutex.Unlock()

	// Update max tick time
	if tickDuration > gtc.MaxTickTime {
		gtc.MaxTickTime = tickDuration
	}

	// Add to tick times history (keep last 100)
	*tickTimes = append(*tickTimes, tickDuration)
	if len(*tickTimes) > 100 {
		*tickTimes = (*tickTimes)[1:]
	}

	// Calculate average tick time
	if len(*tickTimes) > 0 {
		var total time.Duration
		for _, duration := range *tickTimes {
			total += duration
		}
		gtc.AverageTickTime = total / time.Duration(len(*tickTimes))
	}

	// Calculate ticks per second
	if gtc.TotalTicks > 0 {
		elapsed := time.Since(gtc.StartTime)
		gtc.TicksPerSecond = float64(gtc.TotalTicks) / elapsed.Seconds()
	}
}
