package bullets

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMixedCompletionStatuses tests Success, Error, and Stop mixed together
func TestMixedCompletionStatuses(t *testing.T) {
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

	// Create three spinners with different completion types
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")

	time.Sleep(200 * time.Millisecond)

	// Complete with different statuses
	spinner1.Success("Task 1 succeeded")
	spinner2.Error("Task 2 failed")
	spinner3.Stop() // Just stop without message

	time.Sleep(50 * time.Millisecond)

	// Validate output
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Mixed completion statuses produced blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Mixed completion statuses caused cursor instability")
	}

	// Check that output was generated
	raw := capture.GetRawOutput()
	if len(raw) == 0 {
		t.Error("No output captured")
	}

	t.Logf("Captured %d bytes with mixed completions", len(raw))
}

// TestAlternatingSuccessError tests alternating success and error patterns
func TestAlternatingSuccessError(t *testing.T) {
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

	// Create spinners and alternate success/error
	for i := 0; i < 10; i++ {
		spinner := logger.Spinner("Task")
		time.Sleep(50 * time.Millisecond)

		if i%2 == 0 {
			spinner.Success("Success")
		} else {
			spinner.Error("Failed")
		}
	}

	time.Sleep(50 * time.Millisecond)

	// Validate
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Alternating pattern produced blank lines")
	}
}

// TestAllSpinnersFailSimultaneously tests all spinners completing with errors
func TestAllSpinnersFailSimultaneously(t *testing.T) {
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

	// Create multiple spinners
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")
	spinner4 := logger.Spinner("Task 4")

	time.Sleep(200 * time.Millisecond)

	// All fail at once
	var wg sync.WaitGroup
	spinners := []*Spinner{spinner1, spinner2, spinner3, spinner4}

	for i, s := range spinners {
		wg.Add(1)
		go func(spinner *Spinner, id int) {
			defer wg.Done()
			spinner.Error("Failed")
		}(s, i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Validate
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Simultaneous failures produced blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Simultaneous failures caused cursor instability")
	}
}

// TestReplaceCompletionType tests the Replace completion method
func TestReplaceCompletionType(t *testing.T) {
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

	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")

	time.Sleep(150 * time.Millisecond)

	// Use Replace to change the message
	spinner1.Replace("Task 1 replaced with new message")
	spinner2.Success("Task 2 succeeded")

	time.Sleep(50 * time.Millisecond)

	// Validate
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Replace completion produced blank lines")
	}
}

// TestVaryingMessageLengths tests completions with different message sizes
func TestVaryingMessageLengths(t *testing.T) {
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

	// Short message
	spinner1 := logger.Spinner("S1")
	// Medium message
	spinner2 := logger.Spinner("S2")
	// Long message
	spinner3 := logger.Spinner("S3")

	time.Sleep(150 * time.Millisecond)

	// Complete with varying message lengths
	spinner1.Success("OK")
	spinner2.Error("This is a medium length error message that should render correctly")
	spinner3.Replace(strings.Repeat("Very long completion message ", 5))

	time.Sleep(50 * time.Millisecond)

	// Verify no visual artifacts
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Varying message lengths produced blank lines")
	}

	// Cursor should handle different lengths correctly
	if !capture.ValidateCursorStability(t) {
		t.Error("Varying message lengths caused cursor drift")
	}
}

// TestMixedCompletionsConcurrent tests mixed completions happening concurrently
func TestMixedCompletionsConcurrent(t *testing.T) {
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

	// Create multiple spinners
	spinners := make([]*Spinner, 12)
	for i := 0; i < 12; i++ {
		spinners[i] = logger.Spinner("Task")
	}

	time.Sleep(200 * time.Millisecond)

	// Complete concurrently with different methods
	var wg sync.WaitGroup
	for i, spinner := range spinners {
		wg.Add(1)
		go func(s *Spinner, id int) {
			defer wg.Done()
			switch id % 4 {
			case 0:
				s.Success("Success")
			case 1:
				s.Error("Error")
			case 2:
				s.Stop()
			case 3:
				s.Replace("Replaced")
			}
		}(spinner, i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Validate
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Mixed concurrent completions produced blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Mixed concurrent completions caused cursor instability")
	}
}

// TestErrorAfterSuccess tests error completion following success
func TestErrorAfterSuccess(t *testing.T) {
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

	// Sequential pattern: success then error
	spinner1 := logger.Spinner("First task")
	time.Sleep(100 * time.Millisecond)
	spinner1.Success("First succeeded")

	time.Sleep(50 * time.Millisecond)

	spinner2 := logger.Spinner("Second task")
	time.Sleep(100 * time.Millisecond)
	spinner2.Error("Second failed")

	time.Sleep(50 * time.Millisecond)

	// Validate
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Error after success produced blank lines")
	}
}

// TestStopWithoutMessage tests Stop() without any message
func TestStopWithoutMessage(t *testing.T) {
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

	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")

	time.Sleep(150 * time.Millisecond)

	// Stop without messages
	spinner1.Stop()
	spinner2.Stop()

	time.Sleep(50 * time.Millisecond)

	// Validate (Stop should clean up gracefully)
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Stop without message produced blank lines")
	}
}

// TestComplexMixedPattern tests a complex realistic pattern
func TestComplexMixedPattern(t *testing.T) {
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

	// Simulate a complex deployment scenario
	download := logger.Spinner("Downloading dependencies")
	compile := logger.Spinner("Compiling code")
	test := logger.Spinner("Running tests")
	deploy := logger.Spinner("Deploying to server")

	time.Sleep(100 * time.Millisecond)
	download.Success("Dependencies downloaded")

	time.Sleep(100 * time.Millisecond)
	compile.Success("Compilation successful")

	time.Sleep(100 * time.Millisecond)
	test.Error("Tests failed - 3 failures")

	time.Sleep(100 * time.Millisecond)
	deploy.Stop() // Deployment cancelled due to test failures

	time.Sleep(50 * time.Millisecond)

	// Validate complex pattern
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Complex mixed pattern produced blank lines")
	}

	if !capture.ValidateCursorStability(t) {
		t.Error("Complex mixed pattern caused cursor instability")
	}

	t.Log("Complex deployment scenario completed successfully")
}

// TestMixedWithColoredOutput validates ANSI color codes don't interfere
func TestMixedWithColoredOutput(t *testing.T) {
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

	spinner1 := logger.Spinner("Colorful task 1")
	spinner2 := logger.Spinner("Colorful task 2")
	spinner3 := logger.Spinner("Colorful task 3")

	time.Sleep(150 * time.Millisecond)

	// Different completions (each may have different color codes)
	spinner1.Success("Success with green")
	spinner2.Error("Error with red")
	spinner3.Replace("Replace with cyan")

	time.Sleep(50 * time.Millisecond)

	// Check raw output contains escape codes
	raw := capture.GetRawOutput()
	if !strings.Contains(raw, "\033[") {
		t.Log("Warning: No ANSI escape codes detected in output")
	}

	// Validate despite colors
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Colored output produced blank lines")
	}

	t.Logf("Captured %d bytes of colored output", len(raw))
}
