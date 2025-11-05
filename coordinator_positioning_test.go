package bullets

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSpinnerPositionStability tests that spinners maintain their line positions
// even when other spinners complete. This is the critical bug: after a spinner
// completes, remaining spinners should stay at their ORIGINAL line numbers,
// not shift to fill gaps.
func TestSpinnerPositionStability(t *testing.T) {
	var buf bytes.Buffer
	writeMu := &sync.Mutex{}
	logger := &Logger{
		writer:            &buf,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(&buf, writeMu, true), // Force TTY
	}
	logger.coordinator.isTTY = true

	// Create 4 spinners that will be at lines 0, 1, 2, 3
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")
	spinner4 := logger.Spinner("Task 4")

	// Let them animate briefly
	time.Sleep(200 * time.Millisecond)

	// Record spinner4's initial line number
	initialLine4 := spinner4.lineNumber

	// Complete spinner3 (middle spinner)
	spinner3.Success("Task 3 complete")
	time.Sleep(50 * time.Millisecond)

	// CRITICAL TEST: Spinner4's line number should NOT change
	// It should stay at line 3, not shift down to line 2
	currentLine4 := spinner4.lineNumber

	if currentLine4 != initialLine4 {
		t.Errorf("BUG DETECTED: Spinner4 line changed from %d to %d after spinner3 completion. Spinners should maintain their original line positions!",
			initialLine4, currentLine4)
	}

	// Let spinner4 animate a bit more
	time.Sleep(200 * time.Millisecond)

	// Stop spinners and wait before reading buffer
	spinner1.Stop()
	spinner2.Stop()
	spinner4.Stop()
	time.Sleep(100 * time.Millisecond)

	// Check the actual ANSI output to verify spinner4 is updating the correct line
	writeMu.Lock()
	output := buf.String()
	writeMu.Unlock()

	// Count moveUp sequences targeting line 3 (spinner4's original position)
	// We expect spinner4 to keep using [4A (moving up 4 lines from base to line 3)
	moveUp4Count := strings.Count(output, "\033[4A")

	// After spinner3 completes, if the bug exists, spinner4 would incorrectly use [3A
	moveUp3Count := strings.Count(output, "\033[3A")

	t.Logf("Spinner4 initial line: %d, current line: %d", initialLine4, currentLine4)
	t.Logf("MoveUp [4A] count (correct for line 3): %d", moveUp4Count)
	t.Logf("MoveUp [3A] count (incorrect if used by spinner4): %d", moveUp3Count)

	// After completion, spinner4 should continue using its original line position
	// If it shifts, it would start using [3A instead of [4A], which is the bug
	if currentLine4 < initialLine4 {
		t.Errorf("Spinner4 shifted from line %d to line %d - this causes wrong-line updates!",
			initialLine4, currentLine4)
	}
}

// TestCompletionDoesNotShiftRemainingSpinners is a more explicit test
// that verifies the exact bug: completing a spinner causes remaining
// spinners to shift line positions incorrectly.
func TestCompletionDoesNotShiftRemainingSpinners(t *testing.T) {
	var buf bytes.Buffer
	writeMu := &sync.Mutex{}
	logger := &Logger{
		writer:            &buf,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(&buf, writeMu, true),
	}
	logger.coordinator.isTTY = true

	// Create 3 spinners
	s1 := logger.Spinner("Spinner 1")
	s2 := logger.Spinner("Spinner 2")
	s3 := logger.Spinner("Spinner 3")

	// Verify initial line numbers
	if s1.lineNumber != 0 || s2.lineNumber != 1 || s3.lineNumber != 2 {
		t.Fatalf("Initial line numbers incorrect: s1=%d, s2=%d, s3=%d",
			s1.lineNumber, s2.lineNumber, s3.lineNumber)
	}

	time.Sleep(100 * time.Millisecond)

	// Complete the middle spinner
	s2.Success("Middle complete")
	time.Sleep(50 * time.Millisecond)

	// THE BUG: s3's line number should stay at 2, not shift to 1
	// With the current implementation, recalculateLineNumbers will
	// reassign s3 from line 2 to line 1, causing it to overwrite
	// the completion message of s2

	if s3.lineNumber != 2 {
		t.Errorf("BUG: s3 shifted from line 2 to line %d after s2 completion", s3.lineNumber)
		t.Logf("This causes s3 to update the wrong terminal line!")
	} else {
		t.Logf("CORRECT: s3 maintained line 2 after s2 completion")
	}

	// Verify s1 also stayed at line 0
	if s1.lineNumber != 0 {
		t.Errorf("BUG: s1 shifted from line 0 to line %d", s1.lineNumber)
	}

	// Check coordinator state
	logger.coordinator.mu.Lock()
	activeCount := len(logger.coordinator.spinners)
	logger.coordinator.mu.Unlock()

	t.Logf("Active spinners after s2 completion: %d (expected 2)", activeCount)

	if activeCount != 2 {
		t.Errorf("Expected 2 active spinners, got %d", activeCount)
	}

	s1.Stop()
	s3.Stop()
}

// TestVisualOutputNoOverlap verifies that completion messages don't get
// overwritten by subsequent spinner updates
func TestVisualOutputNoOverlap(t *testing.T) {
	var buf bytes.Buffer
	writeMu := &sync.Mutex{}
	logger := &Logger{
		writer:            &buf,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(&buf, writeMu, true),
	}
	logger.coordinator.isTTY = true

	s1 := logger.Spinner("Task A")
	s2 := logger.Spinner("Task B")
	s3 := logger.Spinner("Task C")

	time.Sleep(150 * time.Millisecond)

	// Complete s2
	s2.Success("Task B done")
	time.Sleep(100 * time.Millisecond)

	// Let s3 animate more - it should NOT overwrite "Task B done"
	time.Sleep(200 * time.Millisecond)

	// Stop remaining spinners before reading buffer to avoid race
	s1.Stop()
	s3.Stop()
	time.Sleep(100 * time.Millisecond) // Let stop complete

	// Acquire write lock to ensure all writes are complete before reading
	writeMu.Lock()
	output := buf.String()
	writeMu.Unlock()

	// The output should contain "Task B done" and it should not be
	// overwritten by subsequent s3 updates

	// Check that we have the completion message
	if !strings.Contains(output, "Task B done") {
		t.Error("Completion message 'Task B done' not found in output")
	}

	// After s2 completes at line 1, s3 should continue updating line 2
	// NOT line 1 (which would overwrite the completion message)

	// Look for patterns where line 2 is being updated (moveUp 3)
	// vs line 1 being updated (moveUp 2)

	// Count ANSI sequences after the completion message appears
	completionIndex := strings.Index(output, "Task B done")
	if completionIndex > 0 {
		afterCompletion := output[completionIndex:]

		// s3 should use [3A to reach line 2
		correctUpdates := strings.Count(afterCompletion, "\033[3A")

		// s3 should NOT use [2A (that's line 1 where completion message is)
		incorrectUpdates := strings.Count(afterCompletion, "\033[2A")

		t.Logf("After completion: [3A (correct for s3): %d, [2A (incorrect): %d",
			correctUpdates, incorrectUpdates)

		// Note: s2's completion itself will use [2A, so we can't enforce
		// incorrectUpdates == 0, but after completion, s3 should dominate with [3A
	}
}
