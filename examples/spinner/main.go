// Package main demonstrates concurrent spinner usage with the bullets logger.
// This example shows how multiple spinners can run simultaneously with automatic coordination.
//
// Run with: BULLETS_FORCE_TTY=1 go run examples/spinner/main.go
package main

import (
	"os"
	"sync"
	"time"

	"github.com/sgaunet/bullets"
)

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)
	logger.Info("Starting spinner demonstrations")

	// Demo 1: Concurrent spinners with different completion methods
	demoBasicConcurrent(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause between examples

	// Demo 2: Rapid spinner creation (edge case: stress test coordination)
	demoRapidCreation(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause between examples

	// Demo 3: Out-of-order completion (edge case: later spinners complete first)
	demoOutOfOrderCompletion(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause between examples

	// Demo 4: Mixed success/error scenarios
	demoMixedOutcomes(logger)

	logger.DecreasePadding()
	logger.Success("All demos completed")
}

// demoBasicConcurrent demonstrates multiple spinners running concurrently.
//
// KEY CONCEPTS:
// - SpinnerCoordinator manages all spinners centrally via channel-based updates
// - Each spinner gets a dedicated terminal line for in-place updates
// - Spinners can complete in any order; coordinator automatically reflows remaining spinners
// - Thread-safe by design: all updates go through coordinator channels
//
//nolint:dupl // Demo functions have similar structure but demonstrate different concepts
func demoBasicConcurrent(logger *bullets.Logger) {
	logger.Info("Demo: Concurrent operations with different outcomes")
	logger.IncreasePadding()

	// BEST PRACTICE: Create all spinners upfront before starting concurrent work
	// This allows the coordinator to allocate line numbers efficiently
	dbSpinner := logger.SpinnerDots("Connecting to database")
	apiSpinner := logger.SpinnerCircle("Fetching API data")
	fileSpinner := logger.SpinnerBounce("Processing files")
	cacheSpinner := logger.Spinner("Warming up cache")

	// Use WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(4) //nolint:mnd // Demo code: number of concurrent operations

	// Simulate database connection (completes successfully after 2s)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second) //nolint:mnd // Demo timing
		// COORDINATION: Success() sends update via channel, coordinator handles rendering
		dbSpinner.Success("Database connected")
	}()

	// Simulate API fetch (completes successfully after 3s)
	go func() {
		defer wg.Done()
		time.Sleep(3 * time.Second) //nolint:mnd // Demo timing
		apiSpinner.Success("API data fetched (1.2MB)")
	}()

	// Simulate file processing (fails after 1s)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		// COORDINATION: Error() completion triggers line reallocation for remaining spinners
		fileSpinner.Error("File processing failed: permission denied")
	}()

	// Simulate cache warming (completes with custom message after 1.5s)
	go func() {
		defer wg.Done()
		time.Sleep(1500 * time.Millisecond) //nolint:mnd // Demo timing
		// Replace() allows custom completion messages without status icons
		cacheSpinner.Replace("Cache warmed: 1000 entries loaded")
	}()

	// Wait for all operations to complete
	// THREAD SAFETY: Coordinator ensures all completions are properly synchronized
	wg.Wait()

	logger.DecreasePadding()
	logger.Info("Concurrent operations demo complete")
}

// demoRapidCreation demonstrates rapid spinner creation in quick succession.
//
// EDGE CASE TESTED: Rapid spinner registration
// This tests the coordinator's ability to handle spinners created with minimal delay.
// Previously could cause line position drift or blank line artifacts.
//
// WHAT'S HAPPENING:
// - 5 spinners created nearly simultaneously (10ms apart)
// - Coordinator allocates sequential line numbers (0, 1, 2, 3, 4)
// - Central animation ticker updates all spinners together
// - Spinners complete at different times, triggering line reallocation.
func demoRapidCreation(logger *bullets.Logger) {
	logger.Info("Demo: Rapid spinner creation")
	logger.IncreasePadding()

	// Create spinners in rapid succession with minimal delay
	var spinners []*bullets.Spinner
	for i := 1; i <= 5; i++ {
		spinner := logger.Spinner("Task " + string(rune('A'+i-1)))
		spinners = append(spinners, spinner)
		// Minimal delay between creations (simulates burst scenario)
		time.Sleep(10 * time.Millisecond) //nolint:mnd // Rapid creation timing
	}

	// Complete spinners with varying delays to test coordination
	var wg sync.WaitGroup
	wg.Add(len(spinners))

	for i, spinner := range spinners {
		go func(idx int, s *bullets.Spinner) {
			defer wg.Done()
			// Staggered completion times
			delay := time.Duration(200*(idx+1)) * time.Millisecond //nolint:mnd // Demo timing
			time.Sleep(delay)
			s.Success("Task completed")
		}(i, spinner)
	}

	wg.Wait()

	logger.DecreasePadding()
	logger.Info("Rapid creation demo complete")
}

// demoOutOfOrderCompletion demonstrates spinners completing in reverse order.
//
// EDGE CASE TESTED: Out-of-order completion
// Later spinners complete before earlier ones. This tests line reallocation logic.
// Previously could cause wrong-line updates or visual corruption.
//
// WHAT'S HAPPENING:
// - 4 spinners created in order (lines 0, 1, 2, 3)
// - Spinner 4 completes first (line 3), triggers reallocation
// - Spinner 3 completes second (now line 2), triggers reallocation
// - Spinner 2 completes third (now line 1), triggers reallocation
// - Spinner 1 completes last (now line 0)
// - LineTracker maintains consistency throughout
//
//nolint:dupl // Demo functions have similar structure but demonstrate different concepts
func demoOutOfOrderCompletion(logger *bullets.Logger) {
	logger.Info("Demo: Out-of-order completion")
	logger.IncreasePadding()

	// Create spinners in sequence
	spinner1 := logger.Spinner("Long operation (4s)")
	spinner2 := logger.Spinner("Medium operation (3s)")
	spinner3 := logger.Spinner("Short operation (2s)")
	spinner4 := logger.Spinner("Quick operation (1s)")

	var wg sync.WaitGroup
	wg.Add(4) //nolint:mnd // Number of spinners

	// Complete in REVERSE order (4, 3, 2, 1)
	go func() {
		defer wg.Done()
		time.Sleep(4 * time.Second) //nolint:mnd // Demo timing
		spinner1.Success("Long operation complete")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(3 * time.Second) //nolint:mnd // Demo timing
		spinner2.Success("Medium operation complete")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second) //nolint:mnd // Demo timing
		spinner3.Success("Short operation complete")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		// First to complete triggers line reallocation for remaining 3 spinners
		spinner4.Success("Quick operation complete")
	}()

	wg.Wait()

	logger.DecreasePadding()
	logger.Info("Out-of-order completion demo complete")
}

// demoMixedOutcomes demonstrates various success/error patterns.
//
// EDGE CASE TESTED: Mixed completion states
// Tests coordinator handling of Success(), Error(), Replace(), and Stop() in combination.
// Previously could cause inconsistent state or rendering artifacts.
//
// WHAT'S HAPPENING:
// - 6 spinners with different completion methods
// - Completions interleaved in time
// - Each completion type handled consistently by coordinator
// - Line positions maintained correctly regardless of outcome type.
func demoMixedOutcomes(logger *bullets.Logger) {
	logger.Info("Demo: Mixed success/error patterns")
	logger.IncreasePadding()

	// Create spinners that will have different outcomes
	s1 := logger.Spinner("Operation A")
	s2 := logger.Spinner("Operation B")
	s3 := logger.Spinner("Operation C")
	s4 := logger.Spinner("Operation D")
	s5 := logger.Spinner("Operation E")
	s6 := logger.Spinner("Operation F")

	var wg sync.WaitGroup
	wg.Add(6) //nolint:mnd // Number of operations

	// Mix of different completion types at different times
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo timing
		s1.Success("Operation A succeeded")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(800 * time.Millisecond) //nolint:mnd // Demo timing
		s2.Error("Operation B failed: timeout")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1100 * time.Millisecond) //nolint:mnd // Demo timing
		s3.Success("Operation C succeeded")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1400 * time.Millisecond) //nolint:mnd // Demo timing
		s4.Replace("Operation D: custom completion")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1700 * time.Millisecond) //nolint:mnd // Demo timing
		s5.Error("Operation E failed: connection refused")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(2000 * time.Millisecond) //nolint:mnd // Demo timing
		s6.Success("Operation F succeeded")
	}()

	wg.Wait()

	logger.DecreasePadding()
	logger.Info("Mixed outcomes demo complete")
}
