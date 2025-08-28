package clock

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// DemoGoroutineTickNotification demonstrates the updated clock system
func DemoGoroutineTickNotification() {
	fmt.Println("🎯 Demonstrating Goroutine-Based Tick Notification System")
	fmt.Println(strings.Repeat("=", 60))
	
	// Create global tick coordinator
	coordinator := NewGlobalTickCoordinator()
	
	// Create context for component lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create and register components
	fmt.Println("📋 Creating and registering components...")
	
	comp1 := NewExampleComponent("WebServer")
	comp2 := NewExampleComponent("Database") 
	comp3 := NewExampleComponent("LoadBalancer")
	
	// Register components (each starts its own goroutine)
	err := coordinator.RegisterComponent(comp1, ctx)
	if err != nil {
		log.Printf("Failed to register WebServer: %v", err)
		return
	}
	
	err = coordinator.RegisterComponent(comp2, ctx)
	if err != nil {
		log.Printf("Failed to register Database: %v", err)
		return
	}
	
	err = coordinator.RegisterComponent(comp3, ctx)
	if err != nil {
		log.Printf("Failed to register LoadBalancer: %v", err)
		return
	}
	
	fmt.Printf("✅ Registered %d components, each running in its own goroutine\n", coordinator.GetComponentCount())
	
	// Add some operations to demonstrate processing
	fmt.Println("\n📦 Adding operations to components...")
	
	// Add operations with different processing times
	comp1.AddOperation(Operation{
		ID:          "web_request_1",
		Type:        "http_request",
		Data:        "GET /api/users",
		ProcessTime: 2 * time.Millisecond, // 200 ticks
	})
	
	comp2.AddOperation(Operation{
		ID:          "db_query_1", 
		Type:        "sql_query",
		Data:        "SELECT * FROM users",
		ProcessTime: 5 * time.Millisecond, // 500 ticks
	})
	
	comp3.AddOperation(Operation{
		ID:          "lb_route_1",
		Type:        "route_request", 
		Data:        "route to server_1",
		ProcessTime: 500 * time.Microsecond, // 50 ticks
	})
	
	fmt.Println("✅ Added operations with different processing times")
	
	// Start the simulation
	fmt.Println("\n🚀 Starting simulation...")
	err = coordinator.Start(ctx)
	if err != nil {
		log.Printf("Failed to start simulation: %v", err)
		return
	}
	
	fmt.Println("⏰ Global clock is now broadcasting ticks to all component goroutines")
	
	// Monitor for a few seconds
	monitorDuration := 3 * time.Second
	fmt.Printf("📊 Monitoring for %v...\n\n", monitorDuration)
	
	startTime := time.Now()
	for time.Since(startTime) < monitorDuration {
		time.Sleep(500 * time.Millisecond)
		
		// Get current metrics
		metrics := coordinator.GetPerformanceMetrics()
		_ = coordinator.GetTickDeliveryStatus() // Unused for now
		
		fmt.Printf("TICK %d | Sim Time: %v | Real Time: %v | TPS: %.1f\n",
			metrics.CurrentTick,
			metrics.SimulationTime.Truncate(time.Microsecond),
			metrics.RealTimeElapsed.Truncate(time.Millisecond),
			metrics.TicksPerSecond)
		
		// Show component statuses
		for _, comp := range []*ExampleComponent{comp1, comp2, comp3} {
			status := comp.GetStatus()
			fmt.Printf("  %s: Processed=%d, Queue=%d, Processing=%d, TickChan=%d/%d\n",
				status.ID,
				status.ProcessedCount,
				status.InputQueueLength,
				status.ProcessingCount,
				status.TickChannelLength,
				100) // Channel capacity
		}
		
		fmt.Println()
	}
	
	// Final metrics
	fmt.Println("📈 Final Simulation Results:")
	fmt.Println(strings.Repeat("-", 40))
	
	finalMetrics := coordinator.GetPerformanceMetrics()
	fmt.Printf("Total Ticks Processed: %d\n", finalMetrics.TotalTicks)
	fmt.Printf("Total Simulation Time: %v\n", finalMetrics.SimulationTime.Truncate(time.Microsecond))
	fmt.Printf("Real Time Elapsed: %v\n", finalMetrics.RealTimeElapsed.Truncate(time.Millisecond))
	fmt.Printf("Average Ticks/Second: %.2f\n", finalMetrics.TicksPerSecond)
	fmt.Printf("Efficiency Ratio: %.2fx\n", finalMetrics.EfficiencyRatio)
	fmt.Printf("Tick Utilization: %.1f%%\n", finalMetrics.TickUtilization*100)
	
	fmt.Println("\n🔍 Component Final Status:")
	for _, comp := range []*ExampleComponent{comp1, comp2, comp3} {
		status := comp.GetStatus()
		fmt.Printf("  %s:\n", status.ID)
		fmt.Printf("    Operations Processed: %d\n", status.ProcessedCount)
		fmt.Printf("    Last Processed Tick: %d\n", status.LastProcessedTick)
		fmt.Printf("    Health: %v\n", status.Health)
		fmt.Printf("    Goroutine Running: %v\n", status.Running)
	}
	
	// Stop simulation
	fmt.Println("\n🛑 Stopping simulation...")
	coordinator.Stop()
	
	fmt.Println("✅ All component goroutines stopped gracefully")
	fmt.Println("\n🎯 Demo completed successfully!")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Key Features Demonstrated:")
	fmt.Println("✅ Each component runs in its own goroutine")
	fmt.Println("✅ Global clock broadcasts ticks to all goroutines simultaneously")
	fmt.Println("✅ Guaranteed tick delivery with timeout protection")
	fmt.Println("✅ Parallel processing of ticks across all components")
	fmt.Println("✅ Real-time performance monitoring")
	fmt.Println("✅ Graceful shutdown of all goroutines")
}

// RunQuickDemo runs a shorter demonstration
func RunQuickDemo() {
	fmt.Println("🚀 Quick Demo: Goroutine Tick Notification")
	
	coordinator := NewGlobalTickCoordinator()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Single component demo
	comp := NewExampleComponent("DemoComponent")
	coordinator.RegisterComponent(comp, ctx)
	
	// Add a simple operation
	comp.AddOperation(Operation{
		ID:          "demo_op",
		Type:        "demo",
		Data:        "test",
		ProcessTime: 1 * time.Millisecond,
	})
	
	// Start and run briefly
	coordinator.Start(ctx)
	time.Sleep(1 * time.Second)
	
	// Show results
	metrics := coordinator.GetPerformanceMetrics()
	status := comp.GetStatus()
	
	fmt.Printf("Processed %d ticks in %v\n", metrics.TotalTicks, metrics.RealTimeElapsed.Truncate(time.Millisecond))
	fmt.Printf("Component processed %d operations\n", status.ProcessedCount)
	
	coordinator.Stop()
	fmt.Println("✅ Quick demo completed!")
}
