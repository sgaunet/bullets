package bullets

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// testDebugMu serializes tests that modify debug state to prevent data races.
// resetDebugLevel() is not safe for concurrent use, so all tests using it
// must be serialized.
var testDebugMu sync.Mutex

// TestDebugMode_Disabled verifies no debug output when debug mode is off.
func TestDebugMode_Disabled(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	// Ensure debug mode is off
	os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	if isDebugEnabled() {
		t.Error("Debug should be disabled when BULLETS_DEBUG is unset")
	}

	// debugLog should not output anything
	debugLog("TEST", "This should not appear")
}

// TestDebugMode_Basic verifies basic debug output when BULLETS_DEBUG=1.
func TestDebugMode_Basic(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	if !isDebugEnabled() {
		t.Error("Debug should be enabled when BULLETS_DEBUG=1")
	}

	if isVerboseDebug() {
		t.Error("Verbose debug should not be enabled when BULLETS_DEBUG=1")
	}
}

// TestDebugMode_Verbose verifies verbose debug output when BULLETS_DEBUG=2.
func TestDebugMode_Verbose(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "2")
	defer os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	if !isDebugEnabled() {
		t.Error("Debug should be enabled when BULLETS_DEBUG=2")
	}

	if !isVerboseDebug() {
		t.Error("Verbose debug should be enabled when BULLETS_DEBUG=2")
	}
}

// TestDebugOutput_WithSpinners verifies debug output is generated for spinner operations.
func TestDebugOutput_WithSpinners(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	// Create and complete a spinner
	spinner := logger.Spinner("Test spinner")
	time.Sleep(50 * time.Millisecond)
	spinner.Success("Done")
	time.Sleep(50 * time.Millisecond)

	// Note: Debug output goes to stderr, not the buffer
	// So we can't directly test the debug output here
	// But we can verify the code runs without errors
}

// TestDebugTimer verifies timer functionality.
func TestDebugTimer(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	timer := startDebugTimer("TEST", "test operation")
	if timer == nil {
		t.Error("Timer should not be nil when debug is enabled")
	}
	time.Sleep(10 * time.Millisecond)
	timer.stop()

	// Timer should be nil when debug is disabled
	os.Setenv("BULLETS_DEBUG", "0")
	resetDebugLevel()
	timer = startDebugTimer("TEST", "test operation")
	if timer != nil {
		t.Error("Timer should be nil when debug is disabled")
	}
}

// TestDebugState_Capture verifies state capture functionality.
func TestDebugState_Capture(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	spinner1 := logger.Spinner("Spinner 1")
	spinner2 := logger.Spinner("Spinner 2")

	state := logger.coordinator.captureDebugState()
	if state == nil {
		t.Fatal("Debug state should not be nil when debug is enabled")
	}

	if state.activeSpinners != 2 {
		t.Errorf("Expected 2 active spinners, got %d", state.activeSpinners)
	}

	if len(state.lineAllocations) != 2 {
		t.Errorf("Expected 2 line allocations, got %d", len(state.lineAllocations))
	}

	spinner1.Success("Done")
	spinner2.Success("Done")
	time.Sleep(50 * time.Millisecond)
}

// TestDebugMap_Rendering verifies debug map can be rendered without panics.
func TestDebugMap_Rendering(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	spinner1 := logger.Spinner("Spinner 1")
	spinner2 := logger.Spinner("Spinner 2")
	spinner3 := logger.Spinner("Spinner 3")

	// Render debug map (goes to stderr)
	logger.coordinator.renderDebugMap()

	spinner1.Success("Done")
	spinner2.Success("Done")
	spinner3.Success("Done")
	time.Sleep(50 * time.Millisecond)
}

// TestDebugValidation verifies validation runs in debug mode.
func TestDebugValidation(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("Test")

	// Validation should run without panicking
	logger.coordinator.validateDebugMode()

	spinner.Success("Done")
	time.Sleep(50 * time.Millisecond)
}

// TestDebugPerformance_Disabled verifies minimal overhead when debug is disabled.
func TestDebugPerformance_Disabled(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Unsetenv("BULLETS_DEBUG")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	start := time.Now()

	// Create and complete many spinners
	for i := 0; i < 50; i++ {
		spinner := logger.Spinner("Test")
		spinner.Success("Done")
	}

	elapsed := time.Since(start)
	t.Logf("50 spinners without debug: %v", elapsed)

	// Should be reasonably fast
	if elapsed > 5*time.Second {
		t.Errorf("Performance degraded: took %v for 50 spinners", elapsed)
	}
}

// TestDebugPerformance_Enabled verifies reasonable performance with debug enabled.
func TestDebugPerformance_Enabled(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	start := time.Now()

	// Create and complete many spinners with debug enabled
	for i := 0; i < 50; i++ {
		spinner := logger.Spinner("Test")
		spinner.Success("Done")
	}

	elapsed := time.Since(start)
	t.Logf("50 spinners with debug: %v", elapsed)

	// Should still be reasonably fast (debug adds overhead but not excessive)
	if elapsed > 10*time.Second {
		t.Errorf("Debug mode has excessive overhead: took %v for 50 spinners", elapsed)
	}
}

// BenchmarkDebugLog_Disabled benchmarks debugLog when disabled.
func BenchmarkDebugLog_Disabled(b *testing.B) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debugLog("BENCH", "test message %d", i)
	}
}

// BenchmarkDebugLog_Enabled benchmarks debugLog when enabled.
func BenchmarkDebugLog_Enabled(b *testing.B) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debugLog("BENCH", "test message %d", i)
	}
}

// TestDebugLog_Format verifies debug log output format (manual inspection test).
func TestDebugLog_Format(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	if testing.Short() {
		t.Skip("Skipping manual inspection test in short mode")
	}

	os.Setenv("BULLETS_DEBUG", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	resetDebugLevel()

	t.Log("Debug output should appear on stderr with format: [HH:MM:SS.mmm] [COMPONENT] message")
	debugLog("TEST", "This is a test message")
	debugLog("COORDINATOR", "Registering spinner at line %d", 5)
	debugLogVerbose("VERBOSE", "This should not appear (verbose disabled)")
}

// TestDebugModeIntegration verifies full integration of debug mode.
func TestDebugModeIntegration(t *testing.T) {
	testDebugMu.Lock()
	defer testDebugMu.Unlock()

	os.Setenv("BULLETS_DEBUG", "1")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_DEBUG")
	defer os.Unsetenv("BULLETS_FORCE_TTY")
	resetDebugLevel()

	var buf bytes.Buffer
	logger := New(&buf)

	t.Log("Creating spinners with debug output...")

	s1 := logger.Spinner("Task 1")
	s2 := logger.Spinner("Task 2")
	s3 := logger.Spinner("Task 3")

	time.Sleep(200 * time.Millisecond)

	t.Log("Completing middle spinner...")
	s2.Success("Task 2 done")

	time.Sleep(100 * time.Millisecond)

	t.Log("Rendering debug map...")
	logger.coordinator.renderDebugMap()

	t.Log("Completing remaining spinners...")
	s1.Success("Task 1 done")
	s3.Error("Task 3 failed")

	time.Sleep(100 * time.Millisecond)

	// Verify output contains expected content
	output := buf.String()
	if !strings.Contains(output, "Task 1") {
		t.Error("Output should contain Task 1")
	}
	if !strings.Contains(output, "Task 2") {
		t.Error("Output should contain Task 2")
	}
	if !strings.Contains(output, "Task 3") {
		t.Error("Output should contain Task 3")
	}
}
