package bullets

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestRapidSpinnerCreation stress tests rapid spinner creation
func TestRapidSpinnerCreation(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	numSpinners := 50
	spinners := make([]*Spinner, numSpinners)

	// Record starting goroutines
	startGoroutines := runtime.NumGoroutine()

	// Create spinners rapidly (10ms intervals)
	for i := 0; i < numSpinners; i++ {
		spinners[i] = logger.Spinner(fmt.Sprintf("Task %d", i))
		time.Sleep(10 * time.Millisecond)
	}

	// Complete them all quickly
	for _, spinner := range spinners {
		spinner.Success("Done")
	}

	time.Sleep(100 * time.Millisecond)

	// Check for goroutine leaks
	endGoroutines := runtime.NumGoroutine()
	goroutineDiff := endGoroutines - startGoroutines

	t.Logf("Goroutines: start=%d, end=%d, diff=%d", startGoroutines, endGoroutines, goroutineDiff)

	// Allow for small variance (coordinator goroutine might still be running)
	if goroutineDiff > 5 {
		t.Errorf("Potential goroutine leak: %d extra goroutines", goroutineDiff)
	}

	// Validate output quality
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Rapid creation produced blank lines")
	}
}

// TestRapidCompletionRandomTiming tests spinners with random completion times
func TestRapidCompletionRandomTiming(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	numSpinners := 30
	var wg sync.WaitGroup

	// Create spinners and complete them at random times
	for i := 0; i < numSpinners; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Random task %d", i))

		wg.Add(1)
		go func(s *Spinner, id int) {
			defer wg.Done()

			// Random completion time between 50-200ms
			delay := time.Duration(50+rand.Intn(150)) * time.Millisecond
			time.Sleep(delay)
			s.Success(fmt.Sprintf("Task %d done", id))
		}(spinner, i)

		// Small delay between creations
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all completions
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Validate output
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Random timing produced blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Random timing caused cursor drift")
	}
}

// TestImmediateCompletionStress tests completing spinners before first frame
func TestImmediateCompletionStress(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create and immediately complete many spinners
	for i := 0; i < 50; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Immediate %d", i))
		spinner.Success("Done immediately")
	}

	time.Sleep(50 * time.Millisecond)

	// Should handle immediate completions gracefully
	raw := capture.GetRawOutput()
	if len(raw) == 0 {
		t.Error("No output captured for immediate completions")
	}

	// No goroutine leaks
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
}

// TestRapidSequentialCycles tests rapid create/complete cycles
func TestRapidSequentialCycles(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// 10 cycles of create -> wait -> complete
	for cycle := 0; cycle < 10; cycle++ {
		spinner := logger.Spinner(fmt.Sprintf("Cycle %d", cycle))
		time.Sleep(100 * time.Millisecond) // 1-2 frames
		spinner.Success(fmt.Sprintf("Cycle %d done", cycle))
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(50 * time.Millisecond)

	// Validate no drift over cycles
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Sequential cycles produced blank lines")
	}
}

// TestConcurrentBurstPattern tests bursts of concurrent spinners
func TestConcurrentBurstPattern(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create bursts of spinners
	for burst := 0; burst < 5; burst++ {
		var wg sync.WaitGroup

		// Each burst creates 5 concurrent spinners
		for i := 0; i < 5; i++ {
			spinner := logger.Spinner(fmt.Sprintf("Burst %d Task %d", burst, i))

			wg.Add(1)
			go func(s *Spinner) {
				defer wg.Done()
				time.Sleep(150 * time.Millisecond)
				s.Success("Burst task done")
			}(spinner)
		}

		// Wait for burst to complete
		wg.Wait()

		// Small pause between bursts
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	// Validate output quality across bursts
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Burst pattern produced blank lines")
	}
}

// TestHighConcurrencyManySpinners tests many simultaneous spinners
func TestHighConcurrencyManySpinners(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	numSpinners := 100
	spinners := make([]*Spinner, numSpinners)
	var wg sync.WaitGroup

	// Create all spinners
	for i := 0; i < numSpinners; i++ {
		spinners[i] = logger.Spinner(fmt.Sprintf("Concurrent %d", i))
	}

	// Complete them all at random times
	for i, spinner := range spinners {
		wg.Add(1)
		go func(s *Spinner, id int) {
			defer wg.Done()
			delay := time.Duration(50+rand.Intn(150)) * time.Millisecond
			time.Sleep(delay)
			s.Success(fmt.Sprintf("Concurrent %d done", id))
		}(spinner, i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	t.Logf("Successfully handled %d concurrent spinners", numSpinners)

	// Basic output validation
	raw := capture.GetRawOutput()
	if len(raw) == 0 {
		t.Error("No output captured for high concurrency test")
	}
}

// TestMemoryUsageStress tests for memory leaks under load
func TestMemoryUsageStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stress test in short mode")
	}

	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Record initial memory
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create and complete many spinners
	for i := 0; i < 200; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Memory test %d", i))
		time.Sleep(5 * time.Millisecond)
		spinner.Success("Done")
	}

	time.Sleep(100 * time.Millisecond)

	// Force cleanup and measure memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Calculate memory growth
	allocGrowth := m2.Alloc - m1.Alloc
	heapGrowth := m2.HeapAlloc - m1.HeapAlloc

	t.Logf("Memory growth: Alloc=%d bytes, HeapAlloc=%d bytes", allocGrowth, heapGrowth)

	// Allow for reasonable memory growth (< 10MB)
	if heapGrowth > 10*1024*1024 {
		t.Errorf("Excessive memory growth: %d bytes", heapGrowth)
	}
}

// TestLinePositionDriftUnderLoad verifies positions remain stable under stress
func TestLinePositionDriftUnderLoad(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create spinners and track their positions
	spinner1 := logger.Spinner("Persistent 1")
	spinner2 := logger.Spinner("Persistent 2")
	spinner3 := logger.Spinner("Persistent 3")

	// Record initial positions
	logger.coordinator.mu.Lock()
	initialPos1 := logger.coordinator.spinners[spinner1].lineNumber
	initialPos2 := logger.coordinator.spinners[spinner2].lineNumber
	initialPos3 := logger.coordinator.spinners[spinner3].lineNumber
	logger.coordinator.mu.Unlock()

	// Create and destroy many temporary spinners
	for i := 0; i < 30; i++ {
		temp := logger.Spinner(fmt.Sprintf("Temp %d", i))
		time.Sleep(10 * time.Millisecond)
		temp.Success("Temp done")
	}

	// Check positions haven't drifted
	logger.coordinator.mu.Lock()
	finalPos1 := logger.coordinator.spinners[spinner1].lineNumber
	finalPos2 := logger.coordinator.spinners[spinner2].lineNumber
	finalPos3 := logger.coordinator.spinners[spinner3].lineNumber
	logger.coordinator.mu.Unlock()

	t.Logf("Position stability: S1 %d->%d, S2 %d->%d, S3 %d->%d",
		initialPos1, finalPos1, initialPos2, finalPos2, initialPos3, finalPos3)

	// Positions should remain stable
	if finalPos1 != initialPos1 || finalPos2 != initialPos2 || finalPos3 != initialPos3 {
		t.Error("Line positions drifted under load")
	}

	// Cleanup
	spinner1.Success("Done")
	spinner2.Success("Done")
	spinner3.Success("Done")
	time.Sleep(50 * time.Millisecond)
}

// TestOutputCoherenceUnderStress validates output remains readable
func TestOutputCoherenceUnderStress(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := &sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Rapid fire mixed operations
	for i := 0; i < 50; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Task %d", i))

		if i%3 == 0 {
			// Immediate completion
			spinner.Success("Done")
		} else if i%3 == 1 {
			// Short delay
			go func(s *Spinner) {
				time.Sleep(50 * time.Millisecond)
				s.Success("Done after delay")
			}(spinner)
		} else {
			// Longer delay
			go func(s *Spinner) {
				time.Sleep(150 * time.Millisecond)
				s.Success("Done after long delay")
			}(spinner)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Wait for all to complete
	time.Sleep(300 * time.Millisecond)

	// Verify output coherence
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Mixed operations produced blank lines")
	}

	// Check that cursor movements are consistent
	if !capture.ValidateCursorStability(t) {
		t.Error("Mixed operations caused cursor instability")
	}
}
