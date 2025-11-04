package bullets

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestSpinnerStopIdempotent verifies Stop() can be called multiple times safely.
func TestSpinnerStopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)

	// Call Stop() multiple times concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			spinner.Stop()
		}()
	}

	wg.Wait()

	// Verify spinner is stopped
	if !spinner.stopped {
		t.Error("Spinner should be stopped")
	}
}

// TestConcurrentSpinnerOperations tests multiple spinners with concurrent operations.
func TestConcurrentSpinnerOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	const numSpinners = 20
	spinners := make([]*Spinner, numSpinners)

	// Create spinners concurrently
	var wg sync.WaitGroup
	for i := 0; i < numSpinners; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			spinners[idx] = logger.SpinnerCircle("concurrent test")
		}(i)
	}
	wg.Wait()

	// Let them run briefly
	time.Sleep(200 * time.Millisecond)

	// Stop them all concurrently with different completion methods
	for i := 0; i < numSpinners; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 4 {
			case 0:
				spinners[idx].Success("done")
			case 1:
				spinners[idx].Error("failed")
			case 2:
				spinners[idx].Replace("replaced")
			case 3:
				spinners[idx].Stop()
			}
		}(i)
	}
	wg.Wait()

	// Verify all spinners are stopped
	for i, spinner := range spinners {
		if !spinner.stopped {
			t.Errorf("Spinner %d should be stopped", i)
		}
	}
}

// TestNoGoroutineLeaks verifies spinners don't leak goroutines.
func TestNoGoroutineLeaks(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	var buf bytes.Buffer
	logger := New(&buf)

	// Create and destroy many spinners
	for i := 0; i < 100; i++ {
		spinner := logger.SpinnerCircle("leak test")
		time.Sleep(10 * time.Millisecond)
		spinner.Stop()
	}

	// Give time for cleanup
	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	// Check goroutine count - allow small delta for system goroutines
	final := runtime.NumGoroutine()
	delta := final - baseline

	if delta > 5 {
		t.Errorf("Potential goroutine leak: baseline=%d, final=%d, delta=%d", baseline, final, delta)
	}
}

// TestRapidCreateDestroyCycles tests rapid creation and destruction.
func TestRapidCreateDestroyCycles(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	for i := 0; i < 1000; i++ {
		spinner := logger.SpinnerCircle("rapid test")
		spinner.Stop()
	}

	// Should complete without panic or deadlock
}

// TestConcurrentSuccessError tests concurrent Success() and Error() calls.
func TestConcurrentSuccessError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)

	// Try to call Success and Error concurrently
	// Only one should succeed, both should be safe
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		spinner.Success("success")
	}()

	go func() {
		defer wg.Done()
		spinner.Error("error")
	}()

	wg.Wait()

	// Should not panic or deadlock
	if !spinner.stopped {
		t.Error("Spinner should be stopped")
	}
}

// TestSpinnerStopWaitsForAnimation verifies Stop() waits for animation goroutine.
func TestSpinnerStopWaitsForAnimation(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("wait test")
	time.Sleep(100 * time.Millisecond)

	// This should block until animation goroutine exits
	done := make(chan bool)
	go func() {
		spinner.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success - Stop() completed
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not complete within timeout - possible deadlock")
	}
}

// TestMultipleSpinnersCleanup verifies all spinners clean up properly.
func TestMultipleSpinnersCleanup(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinners := make([]*Spinner, 10)
	for i := 0; i < 10; i++ {
		spinners[i] = logger.SpinnerCircle("cleanup test")
	}

	time.Sleep(200 * time.Millisecond)

	// Stop all
	for _, spinner := range spinners {
		spinner.Stop()
	}

	// Verify logger has no active spinners
	if len(logger.activeSpinners) != 0 {
		t.Errorf("Expected 0 active spinners, got %d", len(logger.activeSpinners))
	}
}

// TestCoordinatorUnregister verifies coordinator cleanup.
func TestCoordinatorUnregister(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner1 := logger.SpinnerCircle("test 1")
	spinner2 := logger.SpinnerCircle("test 2")

	// Coordinator should have 2 spinners
	logger.coordinator.mu.Lock()
	count := len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()

	if count != 2 {
		t.Errorf("Expected 2 registered spinners, got %d", count)
	}

	spinner1.Stop()

	// Should have 1 spinner now
	logger.coordinator.mu.Lock()
	count = len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()

	if count != 1 {
		t.Errorf("Expected 1 registered spinner after Stop(), got %d", count)
	}

	spinner2.Stop()

	// Should have 0 spinners now
	logger.coordinator.mu.Lock()
	count = len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()

	if count != 0 {
		t.Errorf("Expected 0 registered spinners after all Stop(), got %d", count)
	}
}

// TestStopAfterCompletion verifies Stop() is safe after Success/Error/Replace.
func TestStopAfterCompletion(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)

	spinner.Success("done")

	// Should be safe to call Stop() after Success()
	spinner.Stop()
	spinner.Stop() // Multiple times even

	// Should not panic
}

// TestConcurrentSpinnersRaceDetector runs with -race flag.
func TestConcurrentSpinnersRaceDetector(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			spinner := logger.SpinnerCircle("race test")
			time.Sleep(50 * time.Millisecond)

			switch idx % 3 {
			case 0:
				spinner.Success("success")
			case 1:
				spinner.Error("error")
			case 2:
				spinner.Stop()
			}
		}(i)
	}

	wg.Wait()
}
