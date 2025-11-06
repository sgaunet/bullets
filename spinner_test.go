package bullets

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinnerCreation(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("loading")

	if spinner == nil {
		t.Fatal("Spinner() returned nil")
	}

	if spinner.msg != "loading" {
		t.Errorf("Expected spinner message 'loading', got %q", spinner.msg)
	}

	spinner.Stop()
}

func TestSpinnerStop(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("processing")
	time.Sleep(100 * time.Millisecond)
	spinner.Stop()

	if !spinner.stopped {
		t.Error("Expected spinner to be stopped")
	}
}

func TestSpinnerSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("downloading")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("download complete")

	output := buf.String()
	if !strings.Contains(output, "download complete") {
		t.Errorf("Expected output to contain 'download complete', got %q", output)
	}
}

func TestSpinnerError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("connecting")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("connection failed")

	output := buf.String()
	if !strings.Contains(output, "connection failed") {
		t.Errorf("Expected output to contain 'connection failed', got %q", output)
	}
}

func TestSpinnerFail(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("uploading")
	time.Sleep(100 * time.Millisecond)
	spinner.Fail("upload failed")

	output := buf.String()
	if !strings.Contains(output, "upload failed") {
		t.Errorf("Expected output to contain 'upload failed', got %q", output)
	}
}

func TestSpinnerReplace(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("processing")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("processed 100 items")

	output := buf.String()
	if !strings.Contains(output, "processed 100 items") {
		t.Errorf("Expected output to contain 'processed 100 items', got %q", output)
	}
}

func TestSpinnerDots(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerDots("loading")

	if len(spinner.frames) == 0 {
		t.Error("Expected spinner to have frames")
	}

	spinner.Stop()
}

func TestSpinnerCircle(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("processing")

	expectedFrames := []string{"◐", "◓", "◑", "◒"}
	if len(spinner.frames) != len(expectedFrames) {
		t.Errorf("Expected %d frames, got %d", len(expectedFrames), len(spinner.frames))
	}

	spinner.Stop()
}

func TestSpinnerBounce(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerBounce("bouncing")

	if len(spinner.frames) == 0 {
		t.Error("Expected spinner to have frames")
	}

	spinner.Stop()
}

func TestSpinnerWithFrames(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	customFrames := []string{"1", "2", "3", "4"}
	spinner := logger.SpinnerWithFrames("custom", customFrames)

	if len(spinner.frames) != 4 {
		t.Errorf("Expected 4 frames, got %d", len(spinner.frames))
	}

	for i, frame := range customFrames {
		if spinner.frames[i] != frame {
			t.Errorf("Expected frame %d to be %q, got %q", i, frame, spinner.frames[i])
		}
	}

	spinner.Stop()
}

func TestSpinnerWithPadding(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.IncreasePadding()
	logger.IncreasePadding()

	spinner := logger.Spinner("indented")

	if spinner.padding != 2 {
		t.Errorf("Expected spinner padding to be 2, got %d", spinner.padding)
	}

	spinner.Stop()
}

func TestSpinnerMultipleStops(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("test")
	time.Sleep(50 * time.Millisecond)

	spinner.Stop()
	spinner.Stop() // Should not panic

	if !spinner.stopped {
		t.Error("Expected spinner to be stopped")
	}
}

func TestSpinnerWithSpecialBullets(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)

	spinner := logger.Spinner("loading")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	// Should use special success bullet
	if !strings.Contains(output, "✓") && !strings.Contains(output, "done") {
		t.Errorf("Expected special bullet or 'done' in output, got %q", output)
	}
}

func TestSpinnerWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "★")

	spinner := logger.Spinner("loading")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("finished")

	output := buf.String()
	if !strings.Contains(output, "★") {
		t.Errorf("Expected custom bullet '★' in output, got %q", output)
	}
}

func TestMultipleSimultaneousSpinners(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Create multiple spinners
	spinner1 := logger.SpinnerCircle("task 1")
	spinner2 := logger.SpinnerCircle("task 2")
	spinner3 := logger.SpinnerCircle("task 3")

	// Verify they are registered in coordinator
	logger.coordinator.mu.Lock()
	spinnerCount := len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()
	if spinnerCount != 3 {
		t.Errorf("Expected 3 active spinners, got %d", spinnerCount)
	}

	// Verify line numbers are assigned correctly
	if spinner1.lineNumber != 0 {
		t.Errorf("Expected spinner1 lineNumber 0, got %d", spinner1.lineNumber)
	}
	if spinner2.lineNumber != 1 {
		t.Errorf("Expected spinner2 lineNumber 1, got %d", spinner2.lineNumber)
	}
	if spinner3.lineNumber != 2 {
		t.Errorf("Expected spinner3 lineNumber 2, got %d", spinner3.lineNumber)
	}

	// Let them run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop them in different ways
	spinner1.Success("task 1 complete")
	spinner2.Error("task 2 failed")
	spinner3.Replace("task 3 done")

	// Verify they are all unregistered from coordinator
	logger.coordinator.mu.Lock()
	spinnerCount = len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()
	if spinnerCount != 0 {
		t.Errorf("Expected 0 active spinners after stopping, got %d", spinnerCount)
	}

	output := buf.String()

	// Verify all final messages appear in output
	if !strings.Contains(output, "task 1 complete") {
		t.Error("Expected 'task 1 complete' in output")
	}
	if !strings.Contains(output, "task 2 failed") {
		t.Error("Expected 'task 2 failed' in output")
	}
	if !strings.Contains(output, "task 3 done") {
		t.Error("Expected 'task 3 done' in output")
	}
}

func TestSpinnerLineNumberUpdate(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Create three spinners
	spinner1 := logger.SpinnerCircle("task 1")
	spinner2 := logger.SpinnerCircle("task 2")
	spinner3 := logger.SpinnerCircle("task 3")

	time.Sleep(50 * time.Millisecond)

	// Stop the middle spinner
	spinner2.Stop()

	// With the new stable line tracking system, spinner3's line number should NOT change
	// Line numbers remain stable to prevent position drift and overwriting completion messages
	if spinner3.lineNumber != 2 {
		t.Errorf("Expected spinner3 lineNumber to remain at 2, got %d", spinner3.lineNumber)
	}

	// Verify only 2 spinners remain in coordinator
	logger.coordinator.mu.Lock()
	spinnerCount := len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()
	if spinnerCount != 2 {
		t.Errorf("Expected 2 active spinners, got %d", spinnerCount)
	}

	spinner1.Stop()
	spinner3.Stop()
}
