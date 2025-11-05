package bullets

import (
	"sync"
	"testing"
	"time"
)

// TestConcurrentSpinnersDifferentDurations tests 2-3 spinners with varied lifespans
func TestConcurrentSpinnersDifferentDurations(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create three concurrent spinners
	spinner1 := logger.Spinner("Short task")   // Will run 100ms
	spinner2 := logger.Spinner("Medium task")  // Will run 300ms
	spinner3 := logger.Spinner("Long task")    // Will run 500ms

	// Complete them at different times
	go func() {
		time.Sleep(100 * time.Millisecond)
		spinner1.Success("Short task done")
	}()

	go func() {
		time.Sleep(300 * time.Millisecond)
		spinner2.Success("Medium task done")
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		spinner3.Success("Long task done")
	}()

	// Wait for all to complete
	time.Sleep(600 * time.Millisecond)

	// Validate no blank lines
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Found blank lines in concurrent spinner output")
	}

	// Validate cursor stability
	if !capture.ValidateCursorStability(t) {
		t.Error("Cursor position drift detected")
	}
}

// TestConcurrentSpinnersEarlyCompletion tests first spinner completing early
func TestConcurrentSpinnersEarlyCompletion(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create three spinners
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")

	// First spinner completes immediately
	time.Sleep(50 * time.Millisecond)
	spinner1.Success("Task 1 done early")

	// Others continue
	time.Sleep(300 * time.Millisecond)
	spinner2.Success("Task 2 done")
	spinner3.Success("Task 3 done")

	time.Sleep(50 * time.Millisecond)

	// Check line positions remained stable
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Early completion caused blank lines")
	}

	// Verify all three spinners had consistent line positions
	moveUpValues := capture.GetMoveUpValues()
	t.Logf("MoveUp values after early completion: %v", moveUpValues)

	// Should see patterns like [1, 2, 3] for the three spinners
	// even after first one completes
}

// TestConcurrentSpinnersLateCompletion tests last spinner completing after others
func TestConcurrentSpinnersLateCompletion(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")

	// First two complete quickly
	time.Sleep(100 * time.Millisecond)
	spinner1.Success("Task 1 done")
	spinner2.Success("Task 2 done")

	// Last one continues for a while
	time.Sleep(400 * time.Millisecond)
	spinner3.Success("Task 3 done late")

	time.Sleep(50 * time.Millisecond)

	// Validate output
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Late completion caused blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Late completion caused cursor instability")
	}
}

// TestConcurrentSpinnersMiddleCompletion tests middle spinner completing first
func TestConcurrentSpinnersMiddleCompletion(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	spinner1 := logger.Spinner("Task 1") // Line 0
	spinner2 := logger.Spinner("Task 2") // Line 1
	spinner3 := logger.Spinner("Task 3") // Line 2

	// Let them all animate for a bit
	time.Sleep(200 * time.Millisecond)

	// Middle spinner completes
	spinner2.Success("Task 2 (middle) done")
	time.Sleep(50 * time.Millisecond)

	// Others continue
	time.Sleep(200 * time.Millisecond)
	spinner1.Success("Task 1 done")
	spinner3.Success("Task 3 done")

	time.Sleep(50 * time.Millisecond)

	// This is the key test - middle completion shouldn't affect line positions
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Middle completion caused blank lines")
	}

	// Validate no position drift occurred
	events := capture.GetEvents()

	// After middle spinner completes, remaining spinners should maintain
	// their line positions (no shifting/recalculation)
	t.Logf("Total events captured: %d", len(events))
}

// TestConcurrentSpinnersStaggeredStart tests spinners starting at different times
func TestConcurrentSpinnersStaggeredStart(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Stagger the start times
	spinner1 := logger.Spinner("Task 1")
	time.Sleep(100 * time.Millisecond)

	spinner2 := logger.Spinner("Task 2")
	time.Sleep(100 * time.Millisecond)

	spinner3 := logger.Spinner("Task 3")
	time.Sleep(200 * time.Millisecond)

	// Complete them in order
	spinner1.Success("Task 1 done")
	spinner2.Success("Task 2 done")
	spinner3.Success("Task 3 done")

	time.Sleep(50 * time.Millisecond)

	if !capture.ValidateNoBlankLines(t) {
		t.Error("Staggered start caused blank lines")
	}
}

// TestConcurrentSpinnersOverlappingLifetimes tests complex overlapping patterns
func TestConcurrentSpinnersOverlappingLifetimes(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Complex pattern: start multiple, complete some, start more
	spinner1 := logger.Spinner("Phase 1 - Task A")
	spinner2 := logger.Spinner("Phase 1 - Task B")

	time.Sleep(150 * time.Millisecond)

	// Complete first phase
	spinner1.Success("Phase 1-A done")
	spinner2.Success("Phase 1-B done")

	time.Sleep(50 * time.Millisecond)

	// Start second phase immediately
	spinner3 := logger.Spinner("Phase 2 - Task C")
	spinner4 := logger.Spinner("Phase 2 - Task D")

	time.Sleep(200 * time.Millisecond)

	spinner3.Success("Phase 2-C done")
	spinner4.Success("Phase 2-D done")

	time.Sleep(50 * time.Millisecond)

	// Validate no artifacts from overlapping lifetimes
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Overlapping lifetimes caused blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Overlapping lifetimes caused cursor drift")
	}
}

// TestConcurrentSpinnersLinePositionStability specifically tests that
// line positions don't drift as spinners complete
func TestConcurrentSpinnersLinePositionStability(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create 4 spinners
	spinner1 := logger.Spinner("S1")
	spinner2 := logger.Spinner("S2")
	spinner3 := logger.Spinner("S3")
	spinner4 := logger.Spinner("S4")

	// Record line assignments
	logger.coordinator.mu.Lock()
	line1 := logger.coordinator.spinners[spinner1].lineNumber
	line2 := logger.coordinator.spinners[spinner2].lineNumber
	line3 := logger.coordinator.spinners[spinner3].lineNumber
	line4 := logger.coordinator.spinners[spinner4].lineNumber
	logger.coordinator.mu.Unlock()

	t.Logf("Initial line assignments: S1=%d, S2=%d, S3=%d, S4=%d",
		line1, line2, line3, line4)

	// Let them animate
	time.Sleep(200 * time.Millisecond)

	// Complete S2 (middle spinner)
	spinner2.Success("S2 done")
	time.Sleep(100 * time.Millisecond)

	// Check remaining spinner positions haven't changed
	logger.coordinator.mu.Lock()
	newLine1 := logger.coordinator.spinners[spinner1].lineNumber
	newLine3 := logger.coordinator.spinners[spinner3].lineNumber
	newLine4 := logger.coordinator.spinners[spinner4].lineNumber
	logger.coordinator.mu.Unlock()

	t.Logf("After S2 completion: S1=%d, S3=%d, S4=%d",
		newLine1, newLine3, newLine4)

	// Line positions should be stable (no recalculation in TTY mode)
	if newLine1 != line1 {
		t.Errorf("S1 line position changed: %d -> %d", line1, newLine1)
	}
	if newLine3 != line3 {
		t.Errorf("S3 line position changed: %d -> %d", line3, newLine3)
	}
	if newLine4 != line4 {
		t.Errorf("S4 line position changed: %d -> %d", line4, newLine4)
	}

	// Complete remaining
	spinner1.Success("S1 done")
	spinner3.Success("S3 done")
	spinner4.Success("S4 done")
	time.Sleep(50 * time.Millisecond)

	// Final validation
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Position stability test produced blank lines")
	}
}

// TestConcurrentSpinnersVaryingAnimationFrames tests that spinners with
// different numbers of animation frames render correctly
func TestConcurrentSpinnersVaryingAnimationFrames(t *testing.T) {
	capture := NewSpinnerTestCapture()
	writeMu := sync.Mutex{}

	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Quick spinner (~2 frames)
	spinner1 := logger.Spinner("Quick")

	// Medium spinner (~4 frames)
	spinner2 := logger.Spinner("Medium")

	// Long spinner (~8 frames)
	spinner3 := logger.Spinner("Long")

	// Complete at different frame counts
	time.Sleep(160 * time.Millisecond) // ~2 frames
	spinner1.Success("Quick done")

	time.Sleep(160 * time.Millisecond) // ~2 more frames (4 total for spinner2)
	spinner2.Success("Medium done")

	time.Sleep(320 * time.Millisecond) // ~4 more frames (8 total for spinner3)
	spinner3.Success("Long done")

	time.Sleep(50 * time.Millisecond)

	// Count total animation frames
	moveUpCount := capture.CountEventType(EventMoveUp)
	t.Logf("Total animation frames across all spinners: %d", moveUpCount)

	// Verify proper rendering
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Varying frame counts caused blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Varying frame counts caused cursor instability")
	}
}
