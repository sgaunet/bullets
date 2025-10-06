package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sgaunet/bullets"
)

func main() {
	// IMPORTANT: If updates are not working in-place (just printing new lines),
	// set the environment variable BULLETS_FORCE_TTY=1 before running this example.
	// This is often needed when running via 'go run' or in certain terminals.
	//
	// Example: export BULLETS_FORCE_TTY=1 && go run main.go

	// Create an updatable logger
	logger := bullets.NewUpdatable(os.Stdout)

	logger.Info("Starting updatable bullets demonstration")
	logger.IncreasePadding()

	// Example 1: Simple status updates
	logger.Info("Example 1: Simple status updates")
	logger.IncreasePadding()

	task1 := logger.InfoHandle("Task 1: Initializing...")
	task2 := logger.InfoHandle("Task 2: Waiting...")
	task3 := logger.InfoHandle("Task 3: Pending...")

	time.Sleep(1 * time.Second)
	task1.Success("Task 1: Completed ✓")

	time.Sleep(1 * time.Second)
	task2.UpdateLevel(bullets.WarnLevel).UpdateMessage("Task 2: Warning - retrying...")

	time.Sleep(1 * time.Second)
	task2.Success("Task 2: Completed after retry ✓")
	task3.Error("Task 3: Failed ✗")

	logger.DecreasePadding()
	fmt.Println()

	// Example 2: Progress tracking
	logger.Info("Example 2: Download progress tracking")
	logger.IncreasePadding()

	download1 := logger.InfoHandle("Downloading package-1.tar.gz...")
	download2 := logger.InfoHandle("Downloading package-2.tar.gz...")
	download3 := logger.InfoHandle("Downloading package-3.tar.gz...")

	var wg sync.WaitGroup
	// Simulate different download speeds
	wg.Add(3)
	go func() {
		defer wg.Done()
		updateDownload(download1, "package-1.tar.gz", 50*time.Millisecond)
	}()
	go func() {
		defer wg.Done()
		updateDownload(download2, "package-2.tar.gz", 100*time.Millisecond)
	}()
	go func() {
		defer wg.Done()
		updateDownload(download3, "package-3.tar.gz", 75*time.Millisecond)
	}()

	wg.Wait()
	time.Sleep(500 * time.Millisecond)
	logger.DecreasePadding()
	fmt.Println()

	// Example 3: Batch operations
	logger.Info("Example 3: Batch test execution")
	logger.IncreasePadding()

	tests := []*bullets.BulletHandle{
		logger.InfoHandle("Running test: auth_test.go"),
		logger.InfoHandle("Running test: api_test.go"),
		logger.InfoHandle("Running test: db_test.go"),
		logger.InfoHandle("Running test: cache_test.go"),
		logger.InfoHandle("Running test: integration_test.go"),
	}

	// Create a handle group (not used in this example, but available for batch operations)
	// testGroup := bullets.NewHandleGroup(tests...)

	time.Sleep(2 * time.Second)

	// Update individual tests
	tests[0].Success("Test passed: auth_test.go ✓")
	tests[1].Success("Test passed: api_test.go ✓")
	tests[2].Error("Test failed: db_test.go ✗")
	tests[3].Success("Test passed: cache_test.go ✓")
	tests[4].Warning("Test skipped: integration_test.go ⚠")

	logger.DecreasePadding()
	fmt.Println()

	// Example 4: Build pipeline simulation
	logger.Info("Example 4: CI/CD Pipeline")
	logger.IncreasePadding()

	pipeline := []struct {
		name   string
		handle *bullets.BulletHandle
		delay  time.Duration
		status string
	}{
		{"Checkout code", nil, 500 * time.Millisecond, "success"},
		{"Install dependencies", nil, 1 * time.Second, "success"},
		{"Run linter", nil, 800 * time.Millisecond, "warning"},
		{"Run tests", nil, 2 * time.Second, "success"},
		{"Build application", nil, 1500 * time.Millisecond, "success"},
		{"Deploy to staging", nil, 1 * time.Second, "error"},
	}

	// Start all pipeline steps as pending
	for i := range pipeline {
		pipeline[i].handle = logger.InfoHandle(pipeline[i].name + " ⏳")
	}

	// Execute pipeline steps
	for i, step := range pipeline {
		time.Sleep(step.delay)

		switch step.status {
		case "success":
			step.handle.Success(step.name + " ✓")
		case "warning":
			step.handle.Warning(step.name + " ⚠ (with warnings)")
		case "error":
			step.handle.Error(step.name + " ✗ (failed)")
			// Stop pipeline on error
			for j := i + 1; j < len(pipeline); j++ {
				pipeline[j].handle.UpdateColor(bullets.ColorBrightBlack).
					UpdateMessage(pipeline[j].name + " (skipped)")
			}
			break
		}

		if step.status == "error" {
			break
		}
	}

	logger.DecreasePadding()
	fmt.Println()

	// Example 5: Parallel operations with different outcomes
	logger.Info("Example 5: Parallel service health checks")
	logger.IncreasePadding()

	services := map[string]*bullets.BulletHandle{
		"Database":      logger.InfoHandle("Checking database..."),
		"Redis":         logger.InfoHandle("Checking Redis..."),
		"API Gateway":   logger.InfoHandle("Checking API Gateway..."),
		"Auth Service":  logger.InfoHandle("Checking Auth Service..."),
		"Message Queue": logger.InfoHandle("Checking Message Queue..."),
	}

	// Simulate health checks
	go func() {
		time.Sleep(500 * time.Millisecond)
		services["Database"].Success("Database: Healthy (15ms)")
	}()

	go func() {
		time.Sleep(700 * time.Millisecond)
		services["Redis"].Success("Redis: Healthy (8ms)")
	}()

	go func() {
		time.Sleep(1200 * time.Millisecond)
		services["API Gateway"].Warning("API Gateway: Degraded (high latency)")
	}()

	go func() {
		time.Sleep(900 * time.Millisecond)
		services["Auth Service"].Success("Auth Service: Healthy (22ms)")
	}()

	go func() {
		time.Sleep(1500 * time.Millisecond)
		services["Message Queue"].Error("Message Queue: Unreachable")
	}()

	time.Sleep(2 * time.Second)

	logger.DecreasePadding()
	fmt.Println()

	// Example 6: Using HandleChain for coordinated updates
	logger.Info("Example 6: Deployment across regions")
	logger.IncreasePadding()

	usEast := logger.InfoHandle("Deploying to US-East...")
	usWest := logger.InfoHandle("Deploying to US-West...")
	euWest := logger.InfoHandle("Deploying to EU-West...")
	apSouth := logger.InfoHandle("Deploying to AP-South...")

	// Create a chain for all regions
	regions := bullets.Chain(usEast, usWest, euWest, apSouth)

	time.Sleep(1 * time.Second)
	regions.WithField("version", "v2.1.0")

	time.Sleep(1 * time.Second)
	usEast.Success("US-East: Deployed successfully")
	usWest.Success("US-West: Deployed successfully")

	time.Sleep(500 * time.Millisecond)
	euWest.Success("EU-West: Deployed successfully")
	apSouth.Success("AP-South: Deployed successfully")

	logger.DecreasePadding()

	logger.Success("All examples completed!")
}

// updateDownload simulates a download with progress updates
func updateDownload(handle *bullets.BulletHandle, filename string, speed time.Duration) {
	// Progress updates will be shown with the original message
	for i := 0; i <= 100; i += 10 {
		handle.Progress(i, 100)
		time.Sleep(speed)
	}
	handle.Success(filename + " downloaded successfully ✓")
}
