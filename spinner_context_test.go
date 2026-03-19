package bullets

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer for test assertions.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

var _ io.Writer = (*syncBuffer)(nil)

func TestSpinnerContextCancellation(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	spinner := logger.Spinner(ctx, "Working")

	// Cancel the context
	cancel()

	// Wait for the spinner to auto-stop — the animate goroutine processes ctx.Done()
	// and renders the cancellation message. Use a generous sleep for CI.
	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "context canceled") {
		t.Errorf("expected output to contain 'context canceled', got: %q", output)
	}

	// Calling Success after context cancellation should not panic
	spinner.Success("done")
}

func TestSpinnerContextTimeout(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = logger.Spinner(ctx, "Will timeout")

	// Wait for the timeout to trigger and render
	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "context deadline exceeded") {
		t.Errorf("expected output to contain 'context deadline exceeded', got: %q", output)
	}
}

func TestSpinnerContextAlreadyCancelled(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before creating spinner

	_ = logger.Spinner(ctx, "Already cancelled")

	// The spinner should stop almost immediately
	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "context canceled") {
		t.Errorf("expected output to contain 'context canceled', got: %q", output)
	}
}

func TestSpinnerManualStopBeforeContextCancel(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spinner := logger.Spinner(ctx, "Manual stop test")
	spinner.Success("completed manually")

	// Now cancel context — should have no effect (spinner already stopped)
	cancel()
	time.Sleep(200 * time.Millisecond)

	output := buf.String()
	if strings.Contains(output, "context canceled") {
		t.Errorf("expected no cancellation message after manual Success(), got: %q", output)
	}
	if !strings.Contains(output, "completed manually") {
		t.Errorf("expected 'completed manually' in output, got: %q", output)
	}
}

func TestSpinnerContextCancelConcurrentWithSuccess(t *testing.T) {
	// Race condition test: cancel context and call Success() concurrently.
	// Should not panic or deadlock.
	for range 10 {
		var buf syncBuffer
		logger := New(&buf)

		ctx, cancel := context.WithCancel(context.Background())
		spinner := logger.Spinner(ctx, "Race test")

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			cancel()
		}()

		go func() {
			defer wg.Done()
			spinner.Success("done")
		}()

		wg.Wait()
		// Give time for any background goroutines to settle
		time.Sleep(100 * time.Millisecond)
	}
}

func TestMultipleSpinnersSharedContext(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithCancel(context.Background())

	s1 := logger.Spinner(ctx, "Task 1")
	s2 := logger.Spinner(ctx, "Task 2")
	s3 := logger.Spinner(ctx, "Task 3")

	// Complete one manually
	s1.Success("Task 1 done")

	// Cancel context — should stop remaining spinners
	cancel()
	time.Sleep(500 * time.Millisecond)

	output := buf.String()

	// s1 should have success message
	if !strings.Contains(output, "Task 1 done") {
		t.Errorf("expected 'Task 1 done' in output, got: %q", output)
	}

	// s2 and s3 should have cancellation messages
	cancelCount := strings.Count(output, "context canceled")
	if cancelCount < 2 {
		t.Errorf("expected at least 2 'context canceled' messages, got %d in: %q", cancelCount, output)
	}

	// Verify no panics on subsequent calls
	s2.Success("late success")
	s3.Error("late error")
}

func TestSpinnerContextBackgroundNeverCancels(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	spinner := logger.Spinner(context.Background(), "Background context")

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Stop manually — should work normally
	spinner.Success("done")

	output := buf.String()
	if strings.Contains(output, "context canceled") {
		t.Errorf("context.Background() should never cancel, got: %q", output)
	}
	if !strings.Contains(output, "done") {
		t.Errorf("expected 'done' in output, got: %q", output)
	}
}

func TestSpinnerContextDeadlineExceeded(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	deadline := time.Now().Add(100 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	_ = logger.Spinner(ctx, "Deadline test")

	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "context deadline exceeded") {
		t.Errorf("expected 'context deadline exceeded' in output, got: %q", output)
	}
}

func TestPulseContextCancellation(t *testing.T) {
	logger := NewUpdatable(&bytes.Buffer{})
	handle := logger.InfoHandle("Test")

	ctx, cancel := context.WithCancel(context.Background())
	handle.Pulse(ctx, 5*time.Second, "Alternate")

	// Cancel the context — pulse should stop
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Verify no panic and message is restored
	state := handle.GetState()
	if state.Message != "Test" {
		t.Errorf("expected message to be restored to 'Test', got: %q", state.Message)
	}
}

func TestSpinnerContextWithMultipleStyles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var buf syncBuffer
	logger := New(&buf)

	_ = logger.SpinnerDots(ctx, "Dots")
	_ = logger.SpinnerCircle(ctx, "Circle")
	_ = logger.SpinnerBounce(ctx, "Bounce")
	_ = logger.SpinnerWithFrames(ctx, "Custom", []string{"a", "b"})

	cancel()
	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	cancelCount := strings.Count(output, "context canceled")
	if cancelCount < 4 {
		t.Errorf("expected 4 'context canceled' messages for 4 spinners, got %d in: %q", cancelCount, output)
	}
}

func TestSpinnerContextUpdateTextBeforeCancel(t *testing.T) {
	var buf syncBuffer
	logger := New(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	spinner := logger.Spinner(ctx, "Initial")

	spinner.UpdateText("Updated message")
	time.Sleep(50 * time.Millisecond)

	cancel()
	time.Sleep(500 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "context canceled") {
		t.Errorf("expected 'context canceled' in output, got: %q", output)
	}

	_ = fmt.Sprintf("spinner: %v", spinner) // ensure spinner is used
}
