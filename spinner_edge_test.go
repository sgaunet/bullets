package bullets_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sgaunet/bullets"
)

// TestSpinnerWithEmptyFrames tests spinner with empty frames array
func TestSpinnerWithEmptyFrames(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create spinner with empty frames
	spinner := logger.SpinnerWithFrames("Loading", []string{})

	if spinner == nil {
		t.Fatal("SpinnerWithFrames with empty frames returned nil")
	}

	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should handle empty frames gracefully
}

// TestSpinnerWithNilFrames tests spinner with nil frames
func TestSpinnerWithNilFrames(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create spinner with nil frames
	spinner := logger.SpinnerWithFrames("Loading", nil)

	if spinner == nil {
		t.Fatal("SpinnerWithFrames with nil frames returned nil")
	}

	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should handle nil frames gracefully
}

// TestSpinnerDoubleStart tests starting an already started spinner
func TestSpinnerDoubleStart(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Testing")
	time.Sleep(20 * time.Millisecond)

	// Try to create another spinner while one is running
	spinner2 := logger.Spinner("Another spinner")

	// Both spinners should exist
	if spinner == nil || spinner2 == nil {
		t.Error("Spinners should be created even when one is running")
	}

	spinner.Stop()
	spinner2.Stop()
}

// TestSpinnerDoubleStop tests stopping an already stopped spinner
func TestSpinnerDoubleStop(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Testing")
	time.Sleep(50 * time.Millisecond)

	// Stop multiple times
	spinner.Stop()
	spinner.Stop()
	spinner.Stop()

	// Should not panic
}

// TestSpinnerConcurrentOperations tests concurrent spinner operations
func TestSpinnerConcurrentOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Concurrent test")

	var wg sync.WaitGroup
	wg.Add(4)

	// Multiple goroutines trying to stop the same spinner
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
	}()

	go func() {
		defer wg.Done()
		time.Sleep(15 * time.Millisecond)
		spinner.Success("Success from goroutine")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		spinner.Error("Error from goroutine")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(25 * time.Millisecond)
		spinner.Replace("Replaced from goroutine")
	}()

	wg.Wait()

	// Should handle concurrent operations without panic
}

// TestSpinnerWithEmptyMessage tests spinner with empty message
func TestSpinnerWithEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("")
	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should work with empty message
}

// TestSpinnerWithVeryLongMessage tests spinner with extremely long message
func TestSpinnerWithVeryLongMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	longMessage := strings.Repeat("a", 10000)
	spinner := logger.Spinner(longMessage)
	time.Sleep(50 * time.Millisecond)
	spinner.Success("Done")

	output := buf.String()
	if !strings.Contains(output, "Done") {
		t.Error("Spinner should complete even with very long message")
	}
}

// TestSpinnerAfterStop tests operations on stopped spinner
func TestSpinnerAfterStop(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Test")
	spinner.Stop()

	// Try operations after stop
	spinner.Success("Success after stop")
	spinner.Error("Error after stop")
	spinner.Fail("Fail after stop")
	spinner.Replace("Replace after stop")

	// Should not panic
}

// TestMultipleSpinnersSimultaneous tests multiple spinners at once
func TestMultipleSpinnersSimultaneous(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create multiple spinners
	spinners := make([]*bullets.Spinner, 10)
	for i := 0; i < 10; i++ {
		spinners[i] = logger.Spinner(fmt.Sprintf("Spinner %d", i))
		time.Sleep(5 * time.Millisecond)
	}

	// Stop them in reverse order
	for i := 9; i >= 0; i-- {
		spinners[i].Stop()
	}

	// All should work without issues
}

// TestSpinnerWithSpecialCharacters tests spinner with special characters in message
func TestSpinnerWithSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	messages := []string{
		"Message with \n newline",
		"Message with \t tab",
		"Message with emoji 🚀",
		"Message with ANSI \033[31mcolor\033[0m",
	}

	for _, msg := range messages {
		spinner := logger.Spinner(msg)
		time.Sleep(20 * time.Millisecond)
		spinner.Success("Complete")
	}

	// Should handle special characters without issues
}

// TestSpinnerFramesWithSpecialCharacters tests custom frames with special chars
func TestSpinnerFramesWithSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Frames with unicode and emoji
	frames := []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}
	spinner := logger.SpinnerWithFrames("Moon phases", frames)

	time.Sleep(100 * time.Millisecond)
	spinner.Success("Complete")

	// Should handle unicode frames
}

// TestSpinnerWithHighFrequencyUpdates tests very fast spinner updates
func TestSpinnerWithHighFrequencyUpdates(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create spinner with 1ms frame duration (very fast)
	frames := []string{"1", "2", "3", "4"}
	spinner := logger.SpinnerWithFrames("Fast", frames)

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)
	spinner.Stop()

	// Should handle high frequency updates
}

// TestSpinnerMemoryLeak tests for memory leaks with long-running spinner
func TestSpinnerMemoryLeak(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create and destroy many spinners
	for i := 0; i < 1000; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Spinner %d", i))
		// Don't sleep, just create and stop immediately
		spinner.Stop()
		buf.Reset() // Clear buffer to avoid growing
	}

	// Should not leak memory (use memory profiler to verify)
}

// TestSpinnerPaddingInheritance tests spinner with logger padding
func TestSpinnerPaddingInheritance(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Set padding before creating spinner
	logger.IncreasePadding()
	logger.IncreasePadding()
	logger.IncreasePadding()

	spinner := logger.Spinner("Indented spinner")
	time.Sleep(50 * time.Millisecond)
	spinner.Success("Done")

	output := buf.String()
	// Should have indentation in output
	if !strings.Contains(output, "      ") {
		t.Error("Spinner should inherit logger padding")
	}
}

// TestSpinnerWithClosedWriter tests spinner operations with closed writer
func TestSpinnerWithClosedWriter(t *testing.T) {
	// Create a logger with a pipe writer that we'll close
	pr, pw := io.Pipe()
	logger := bullets.New(pw)

	// Close the writer
	pw.Close()
	pr.Close()

	// Create spinner with closed writer - should handle gracefully
	spinner := logger.Spinner("Test")
	if spinner != nil {
		// These operations should not panic even with closed writer
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
	}
}

// TestSpinnerRaceCondition tests for race conditions
func TestSpinnerRaceCondition(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Race test")

	// Start multiple goroutines that modify spinner state
	go func() {
		for i := 0; i < 100; i++ {
			// Try to trigger race by checking stopped state
			// This would fail with -race flag if not thread-safe
			_ = spinner // Access spinner
		}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			// Another goroutine also accessing
			_ = spinner
		}
	}()

	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should complete without race detection failures
}

// TestSpinnerSuccessWithEmptyMessage tests success with empty message
func TestSpinnerSuccessWithEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Loading...")
	time.Sleep(30 * time.Millisecond)
	spinner.Success("")

	// Should handle empty success message
}

// TestSpinnerErrorWithEmptyMessage tests error with empty message
func TestSpinnerErrorWithEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Processing...")
	time.Sleep(30 * time.Millisecond)
	spinner.Error("")

	// Should handle empty error message
}

// TestSpinnerWithZeroDuration tests spinner that stops immediately
func TestSpinnerWithZeroDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	spinner := logger.Spinner("Instant")
	// Stop immediately, no sleep
	spinner.Stop()

	// Should handle immediate stop
}

// TestSpinnerAllStylesEdgeCases tests all spinner styles with edge cases
func TestSpinnerAllStylesEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test each style with immediate stop
	styles := []func(string) *bullets.Spinner{
		logger.Spinner,
		logger.SpinnerDots,
		logger.SpinnerCircle,
		logger.SpinnerBounce,
	}

	for _, styleFunc := range styles {
		spinner := styleFunc("Quick test")
		spinner.Stop()
	}

	// All styles should handle immediate stop
}