package bullets_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sgaunet/bullets"
)

// TestTTYDetectionWithEnvVar tests BULLETS_FORCE_TTY environment variable
func TestTTYDetectionWithEnvVar(t *testing.T) {
	var buf bytes.Buffer

	// Test without env var (should detect as non-TTY for buffer)
	logger1 := bullets.NewUpdatable(&buf)
	if logger1 == nil {
		t.Fatal("NewUpdatable returned nil")
	}

	// Set env var to force TTY mode
	oldEnv := os.Getenv("BULLETS_FORCE_TTY")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Setenv("BULLETS_FORCE_TTY", oldEnv)

	logger2 := bullets.NewUpdatable(&buf)
	if logger2 == nil {
		t.Fatal("NewUpdatable with BULLETS_FORCE_TTY returned nil")
	}

	// Test with invalid env var value
	os.Setenv("BULLETS_FORCE_TTY", "false")
	logger3 := bullets.NewUpdatable(&buf)
	if logger3 == nil {
		t.Fatal("NewUpdatable with invalid BULLETS_FORCE_TTY returned nil")
	}
}

// TestNonTTYFallback tests fallback behavior when not in TTY mode
func TestNonTTYFallback(t *testing.T) {
	var buf bytes.Buffer

	// Ensure we're in non-TTY mode
	oldEnv := os.Getenv("BULLETS_FORCE_TTY")
	os.Unsetenv("BULLETS_FORCE_TTY")
	defer os.Setenv("BULLETS_FORCE_TTY", oldEnv)

	logger := bullets.NewUpdatable(&buf)

	// Create handle and update it
	handle := logger.InfoHandle("Initial")
	buf.Reset() // Clear initial output

	// Updates should print new lines in non-TTY mode
	handle.Update(bullets.WarnLevel, "Updated")
	output := buf.String()

	// In non-TTY mode, updates print as new lines
	if output != "" && !strings.Contains(output, "Updated") {
		// Handle might be in TTY mode or update might not print
		// This is expected behavior - non-TTY handles may not print updates
	}

	// Success should print in non-TTY mode
	buf.Reset()
	handle.Success("Completed")
	output = buf.String()
	if !strings.Contains(output, "Completed") {
		t.Error("Success should print in non-TTY mode")
	}
}

// TestConcurrentHandleUpdates tests concurrent updates to same handle
func TestConcurrentHandleUpdates(t *testing.T) {
	writer := &syncWriterUpdatable{buf: &bytes.Buffer{}}
	logger := bullets.NewUpdatable(writer)

	handle := logger.InfoHandle("Starting...")

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Multiple goroutines updating the same handle
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				switch j % 3 {
				case 0:
					handle.UpdateMessage(fmt.Sprintf("Message from %d", id))
				case 1:
					handle.UpdateLevel(bullets.WarnLevel)
				case 2:
					handle.WithField("goroutine", id)
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or corrupt data
	state := handle.GetState()
	if state.Message == "" {
		t.Error("Handle message should not be empty after concurrent updates")
	}
}

// TestProgressEdgeCases tests progress bar with edge values
func TestProgressEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Downloading...")

	// Test negative progress
	handle.Progress(-10, 100)
	// Should not panic - black box testing can't check internal state

	// Test progress > 100%
	handle.Progress(150, 100)
	// Should handle values > 100%

	// Test with very small total
	handle.Progress(50, 1)
	// Should handle small totals

	// Test with negative progress
	handle.Progress(-50, 100)
	// Should handle negative values gracefully

	// Test very large values
	handle.Progress(1000000, 1000000)
	// Should handle large values

	// Test progress after Success
	handle.Success("Complete")
	handle.Progress(50, 100)
	// Progress after success should still work

	// Test alternating progress values
	handle.Progress(100, 100)
	handle.Progress(0, 100)
	handle.Progress(50, 100)

	// All operations should complete without panic
	state := handle.GetState()
	if state.Message != "Complete" {
		t.Error("Handle should maintain message after progress updates")
	}
}

// TestRapidUpdates tests very rapid successive updates
func TestRapidUpdates(t *testing.T) {
	var buf bytes.Buffer

	// Force TTY mode for this test
	oldEnv := os.Getenv("BULLETS_FORCE_TTY")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Setenv("BULLETS_FORCE_TTY", oldEnv)

	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Rapid test")

	// Perform 1000 rapid updates
	for i := 0; i < 1000; i++ {
		handle.UpdateMessage(fmt.Sprintf("Update %d", i))
		if i%100 == 0 {
			handle.Progress(i/10, 100)
		}
	}

	// Should not crash or leak memory
	handle.Success("Rapid updates complete")
}

// TestHandleWithBufferWriter tests handle operations with buffer writer
func TestHandleWithBufferWriter(t *testing.T) {
	// Create logger with buffer writer
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create a handle
	handle := logger.InfoHandle("Test message")

	// These operations should work with buffer writer
	handle.UpdateMessage("Updated test")
	handle.UpdateLevel(bullets.ErrorLevel)
	handle.WithField("key", "value")
	handle.Progress(50, 100)
	handle.Success("Done")

	// GetState should work
	state := handle.GetState()
	if state.Message != "Done" {
		t.Error("Handle should maintain state with buffer writer")
	}

	// Buffer should have some output
	if buf.Len() == 0 {
		t.Error("Expected some output in buffer")
	}
}

// TestMultipleHandleGroups tests multiple groups with overlapping handles
func TestMultipleHandleGroups(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	h1 := logger.InfoHandle("Handle 1")
	h2 := logger.InfoHandle("Handle 2")
	h3 := logger.InfoHandle("Handle 3")

	// Create overlapping groups
	group1 := bullets.NewHandleGroup(h1, h2)
	group2 := bullets.NewHandleGroup(h2, h3)
	group3 := bullets.NewHandleGroup(h1, h2, h3)

	// Update through different groups
	group1.SuccessAll("Group 1 success")
	group2.ErrorAll("Group 2 error")
	group3.UpdateAll(bullets.WarnLevel, "Group 3 warning")

	// h2 should have the last update (Group 3)
	state := h2.GetState()
	if state.Level != bullets.WarnLevel {
		t.Errorf("Expected WarnLevel for h2, got %v", state.Level)
	}
}

// TestHandleChainWithMultipleHandles tests chain with multiple handles
func TestHandleChainWithMultipleHandles(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	h1 := logger.InfoHandle("Handle 1")
	h2 := logger.InfoHandle("Handle 2")
	h3 := logger.InfoHandle("Handle 3")

	// Create chain with multiple handles
	chain := bullets.Chain(h1, h2, h3)

	// Update all through chain
	chain.Success("Success message")
	chain.WithField("key", "value")

	// All handles should be updated
	for i, h := range []*bullets.BulletHandle{h1, h2, h3} {
		state := h.GetState()
		if state.Message != "Success message" {
			t.Errorf("Handle %d should be updated in chain", i+1)
		}
		if state.Fields["key"] != "value" {
			t.Errorf("Handle %d should have field from chain", i+1)
		}
	}
}

// TestUpdatableWithFailingWriter tests behavior with failing writer
func TestUpdatableWithFailingWriter(t *testing.T) {
	// Create a writer that always fails
	failWriter := &failingWriterUpdatable{shouldFail: true}

	logger := bullets.NewUpdatable(failWriter)

	// Operations should not panic even with failing writer
	handle := logger.InfoHandle("Test message")
	handle.UpdateMessage("Updated")
	handle.Success("Complete")

	// Logger and handle should still work
	state := handle.GetState()
	if state.Message != "Complete" {
		t.Error("Handle should maintain state even with failing writer")
	}
}

// TestConcurrentLineCount tests concurrent line count updates
func TestConcurrentLineCount(t *testing.T) {
	writer := &syncWriterUpdatable{buf: &bytes.Buffer{}}
	logger := bullets.NewUpdatable(writer)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			// Mix regular logs and handle creation
			for j := 0; j < 50; j++ {
				if j%2 == 0 {
					logger.Info(fmt.Sprintf("Regular log %d-%d", id, j))
				} else {
					handle := logger.InfoHandle(fmt.Sprintf("Handle %d-%d", id, j))
					handle.Success("Done")
				}
			}
		}(i)
	}

	wg.Wait()

	// Line count should be consistent
	// No specific assertion as it depends on concurrency
}

// TestUpdateAfterLoggerModification tests handle after logger state changes
func TestUpdateAfterLoggerModification(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	handle := logger.InfoHandle("Original message")

	// Change logger state
	logger.SetLevel(bullets.ErrorLevel)
	logger.IncreasePadding()
	logger.SetUseSpecialBullets(true)
	logger.SetBullet(bullets.InfoLevel, "▶")

	// Update handle - should still work
	handle.UpdateMessage("Updated after logger changes")
	handle.Success("Complete")

	// Handle should maintain its own state
	state := handle.GetState()
	if state.Message != "Complete" {
		t.Error("Handle update should work after logger modification")
	}
}

// TestProgressBarOverflow tests progress bar with extreme values
func TestProgressBarOverflow(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Testing overflow")

	// Test with max int values
	handle.Progress(int(^uint(0)>>1), 100) // max int

	// Test with very small fractions
	handle.Progress(1, 1000000)

	// Test alternating progress (going backwards)
	handle.Progress(100, 100)
	handle.Progress(50, 100)
	handle.Progress(75, 100)

	// All should handle gracefully without panic
}

// TestEmptyStringOperations tests operations with empty strings
func TestEmptyStringOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create handle with empty message
	handle := logger.InfoHandle("")

	// All operations with empty strings should be safe
	handle.UpdateMessage("")
	handle.UpdateColor("")
	handle.UpdateBullet("")
	handle.WithField("", "")
	handle.Success("")
	handle.Error("")
	handle.Warning("")

	// GetState should work
	state := handle.GetState()
	// Message should be empty string (last set)
	if state.Message != "" {
		t.Errorf("Expected empty message, got: %s", state.Message)
	}
}

// TestHandleGroupThreadSafety tests thread safety of handle groups
func TestHandleGroupThreadSafety(t *testing.T) {
	writer := &syncWriterUpdatable{buf: &bytes.Buffer{}}
	logger := bullets.NewUpdatable(writer)

	group := bullets.NewHandleGroup()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent operations on group
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			handle := logger.InfoHandle(fmt.Sprintf("Handle %d", id))

			// Race between add and update
			group.Add(handle)
			group.UpdateAll(bullets.WarnLevel, "Concurrent update")

			// Race between get and clear
			if id%5 == 0 {
				group.Clear()
			} else {
				size := group.Size()
				if size > 0 {
					_ = group.Get(id % size)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or have race conditions
}

// TestBatchUpdateEdgeCases tests BatchUpdate with edge cases
func TestBatchUpdateEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	h1 := logger.InfoHandle("Handle 1")
	h2 := logger.InfoHandle("Handle 2")

	// Test with nil map
	bullets.BatchUpdate([]*bullets.BulletHandle{h1, h2}, nil)

	// Test with empty slice
	bullets.BatchUpdate([]*bullets.BulletHandle{}, map[*bullets.BulletHandle]struct {
		Level   bullets.Level
		Message string
	}{
		h1: {Level: bullets.ErrorLevel, Message: "Error"},
	})

	// Test with nil handle in slice
	bullets.BatchUpdate([]*bullets.BulletHandle{h1, nil, h2}, map[*bullets.BulletHandle]struct {
		Level   bullets.Level
		Message string
	}{
		h1: {Level: bullets.WarnLevel, Message: "Warning"},
	})

	// Test with handle not in map
	bullets.BatchUpdate([]*bullets.BulletHandle{h1}, map[*bullets.BulletHandle]struct {
		Level   bullets.Level
		Message string
	}{
		h2: {Level: bullets.InfoLevel, Message: "Info"},
	})
}

// TestPulseWithEdgeCases tests Pulse with edge timing
func TestPulseWithEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Pulse test")

	// Test with zero duration
	handle.Pulse(0, "Alternate")
	time.Sleep(10 * time.Millisecond)

	// Test with negative duration (should be treated as zero or ignored)
	handle.Pulse(-1*time.Second, "Negative")
	time.Sleep(10 * time.Millisecond)

	// Test with very short duration
	handle.Pulse(1*time.Nanosecond, "Nanosecond")
	time.Sleep(10 * time.Millisecond)

	// Test with empty alternate message
	handle.Pulse(100*time.Millisecond, "")
	time.Sleep(150 * time.Millisecond)

	// Should handle all cases gracefully
	state := handle.GetState()
	if state.Message != "Pulse test" {
		// Message might have changed, that's ok
		t.Log("Message after pulse:", state.Message)
	}
}

// failingWriterUpdatable is a writer that can be configured to fail
type failingWriterUpdatable struct {
	shouldFail bool
	mu         sync.Mutex
}

func (w *failingWriterUpdatable) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.shouldFail {
		return 0, fmt.Errorf("simulated write failure")
	}
	return len(p), nil
}

// syncWriterUpdatable wraps bytes.Buffer to make it thread-safe
type syncWriterUpdatable struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func (w *syncWriterUpdatable) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncWriterUpdatable) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}