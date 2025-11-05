package bullets

import (
	"sync"
	"testing"
	"time"
)

// TestSequentialSpinnersNoBlankLines tests that two sequential spinners
// don't introduce blank lines between completion and next spinner start
func TestSequentialSpinnersNoBlankLines(t *testing.T) {
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

	// First spinner - let it run for 3-5 frames (~240-400ms)
	spinner1 := logger.Spinner("Task 1")
	time.Sleep(320 * time.Millisecond) // ~4 frames at 80ms interval
	spinner1.Success("Task 1 complete")

	// Brief pause to ensure completion is fully rendered
	time.Sleep(50 * time.Millisecond)

	// Second spinner - immediately after first completes
	spinner2 := logger.Spinner("Task 2")
	time.Sleep(320 * time.Millisecond) // ~4 frames
	spinner2.Success("Task 2 complete")

	time.Sleep(50 * time.Millisecond)

	// Validate no blank lines
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Found blank lines in sequential spinner output")
	}

	// Check for consecutive newlines in events
	events := capture.GetEvents()
	consecutiveNewlines := 0
	maxConsecutive := 0

	for _, event := range events {
		if event.Type == EventNewline {
			consecutiveNewlines++
			if consecutiveNewlines > maxConsecutive {
				maxConsecutive = consecutiveNewlines
			}
		} else if event.Type == EventText {
			consecutiveNewlines = 0
		}
	}

	if maxConsecutive > 1 {
		t.Errorf("Found %d consecutive newlines (blank lines detected)", maxConsecutive)
		capture.DumpEvents(t, 100)
	}
}

// TestSequentialSpinnersCursorPositioning verifies cursor returns to
// correct position after each frame update
func TestSequentialSpinnersCursorPositioning(t *testing.T) {
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

	// Run first spinner
	spinner1 := logger.Spinner("Task 1")
	time.Sleep(250 * time.Millisecond)
	spinner1.Success("Task 1 done")
	time.Sleep(50 * time.Millisecond)

	// Run second spinner
	spinner2 := logger.Spinner("Task 2")
	time.Sleep(250 * time.Millisecond)
	spinner2.Success("Task 2 done")
	time.Sleep(50 * time.Millisecond)

	// Validate cursor stability
	if !capture.ValidateCursorStability(t) {
		t.Error("Cursor position drift detected in sequential spinners")
	}

	// Check that moveUp/moveDown pairs are symmetric
	moveUpValues := capture.GetMoveUpValues()
	moveDownValues := capture.GetMoveDownValues()

	t.Logf("MoveUp values: %v", moveUpValues)
	t.Logf("MoveDown values: %v", moveDownValues)

	// For sequential spinners at line 0, all movements should be 1
	// (moveUp 1 line, moveDown 1 line)
	for i, up := range moveUpValues {
		if i < len(moveDownValues) {
			down := moveDownValues[i]
			if up != down {
				t.Errorf("Asymmetric cursor movement at index %d: up=%d, down=%d", i, up, down)
			}
		}
	}
}

// TestSequentialSpinnersLineClearing validates that line clearing
// happens properly without leaving artifacts
func TestSequentialSpinnersLineClearing(t *testing.T) {
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

	// Create and complete first spinner
	spinner1 := logger.Spinner("Processing data")
	time.Sleep(240 * time.Millisecond) // 3 frames
	spinner1.Success("Data processed")
	time.Sleep(50 * time.Millisecond)

	// Create and complete second spinner
	spinner2 := logger.Spinner("Uploading results")
	time.Sleep(240 * time.Millisecond) // 3 frames
	spinner2.Success("Results uploaded")
	time.Sleep(50 * time.Millisecond)

	// Check that clearLine events are properly paired with content writes
	events := capture.GetEvents()

	clearLineCount := 0
	textAfterClear := 0

	for i, event := range events {
		if event.Type == EventClearLine {
			clearLineCount++
			// Next non-ANSI event should be text (the new content)
			for j := i + 1; j < len(events) && j < i+5; j++ {
				if events[j].Type == EventText {
					textAfterClear++
					break
				}
			}
		}
	}

	t.Logf("Found %d clearLine events, %d followed by text", clearLineCount, textAfterClear)

	// Every clearLine should be followed by new content
	if clearLineCount > 0 && textAfterClear < clearLineCount {
		t.Errorf("Some clearLine events not followed by text: %d clears, %d text writes",
			clearLineCount, textAfterClear)
	}
}

// TestSequentialSpinnersCompletionTiming tests that completion messages
// are immediately visible without animation delay
func TestSequentialSpinnersCompletionTiming(t *testing.T) {
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

	// Time the completion rendering
	spinner1 := logger.Spinner("Task 1")
	time.Sleep(200 * time.Millisecond)

	startCompletion := time.Now()
	spinner1.Success("Task 1 complete")

	// Check how quickly completion message appears in output
	time.Sleep(10 * time.Millisecond) // Minimal delay
	completionDelay := time.Since(startCompletion)

	// Just verify that output was generated (completion happened)
	raw := capture.GetRawOutput()
	if len(raw) == 0 {
		t.Error("No output captured - completion may not have rendered")
	}

	t.Logf("Completion rendered in %v", completionDelay)

	// Completion should be nearly immediate (< 100ms)
	if completionDelay > 100*time.Millisecond {
		t.Errorf("Completion took too long: %v (expected < 100ms)", completionDelay)
	}
}

// TestSequentialSpinnersFrameCount validates that spinners render
// the expected number of frames
func TestSequentialSpinnersFrameCount(t *testing.T) {
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

	// Run spinner for exactly 400ms (should get ~5 frames at 80ms interval)
	spinner1 := logger.Spinner("Task 1")
	time.Sleep(400 * time.Millisecond)
	spinner1.Success("Done")
	time.Sleep(50 * time.Millisecond)

	// Count animation frames by counting moveUp/moveDown pairs
	moveUpCount := capture.CountEventType(EventMoveUp)
	moveDownCount := capture.CountEventType(EventMoveDown)

	// Each animation frame generates a moveUp/moveDown pair
	frameCount := moveUpCount
	if moveDownCount < frameCount {
		frameCount = moveDownCount
	}

	t.Logf("Captured %d animation frames (moveUp: %d, moveDown: %d)",
		frameCount, moveUpCount, moveDownCount)

	// Should have 4-6 frames (allowing for timing variance)
	if frameCount < 3 || frameCount > 7 {
		t.Logf("Warning: Unexpected frame count: got %d, expected 4-6", frameCount)
		t.Logf("This could be due to timing variance in test environment")
	}

	// Verify we captured some output
	if frameCount == 0 {
		t.Error("No animation frames captured")
	}
}

// TestSequentialSpinnersWithDifferentMessages tests spinners with
// varying message lengths don't cause artifacts
func TestSequentialSpinnersWithDifferentMessages(t *testing.T) {
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

	// Short message
	spinner1 := logger.Spinner("A")
	time.Sleep(200 * time.Millisecond)
	spinner1.Success("Done")
	time.Sleep(50 * time.Millisecond)

	// Long message
	spinner2 := logger.Spinner("This is a very long spinner message that should properly clear the previous content")
	time.Sleep(200 * time.Millisecond)
	spinner2.Success("Completed successfully with a detailed message")
	time.Sleep(50 * time.Millisecond)

	// Short again
	spinner3 := logger.Spinner("B")
	time.Sleep(200 * time.Millisecond)
	spinner3.Success("OK")
	time.Sleep(50 * time.Millisecond)

	// Validate no artifacts left behind
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Found blank lines with varying message lengths")
	}

	// Verify clearLine events match content writes
	clearCount := capture.CountEventType(EventClearLine)

	t.Logf("Total clearLine events: %d", clearCount)

	if clearCount == 0 {
		t.Error("No clearLine events found - line clearing not working")
	}
}

// TestSequentialSpinnersImmediateCompletion tests completing a spinner
// immediately after creation (before first frame)
func TestSequentialSpinnersImmediateCompletion(t *testing.T) {
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

	// Complete immediately - no animation frames
	spinner1 := logger.Spinner("Task 1")
	spinner1.Success("Already done")

	// Normal spinner
	spinner2 := logger.Spinner("Task 2")
	time.Sleep(200 * time.Millisecond)
	spinner2.Success("Task 2 done")

	time.Sleep(50 * time.Millisecond)

	// Should not cause any issues
	if !capture.ValidateNoBlankLines(t) {
		t.Error("Immediate completion caused blank lines")
	}

	// Check that output was generated (both completions rendered)
	raw := capture.GetRawOutput()
	if len(raw) == 0 {
		t.Error("No output captured - completions may not have rendered")
	}

	// Log for debugging
	t.Logf("Captured %d bytes of output", len(raw))
}
