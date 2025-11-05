package bullets

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestCursorPositionAfterCompletions verifies that spinner completions
// don't leave blank lines or overwrite each other due to cursor positioning bugs.
func TestCursorPositionAfterCompletions(t *testing.T) {
	// Force TTY mode for this test
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	// Create 4 spinners
	s1 := logger.Spinner("Task 1")
	s2 := logger.Spinner("Task 2")
	s3 := logger.Spinner("Task 3")
	s4 := logger.Spinner("Task 4")

	// Let spinners animate a bit
	time.Sleep(100 * time.Millisecond)

	// Complete them in order (last to first to test reallocation)
	s4.Success("Task 4 done")
	time.Sleep(50 * time.Millisecond)

	s3.Success("Task 3 done")
	time.Sleep(50 * time.Millisecond)

	s2.Success("Task 2 done")
	time.Sleep(50 * time.Millisecond)

	s1.Success("Task 1 done")
	time.Sleep(50 * time.Millisecond)

	// Print something after all spinners complete
	logger.Info("All tasks completed")

	output := buf.String()

	// Strip ANSI codes for analysis
	stripped := stripAnsiCodes(output)

	// With ANSI cursor movement, all content is on one "line" in the buffer
	// but visually appears on separate lines in the terminal.
	// Verify all 4 completion messages are present in the output
	expectedMessages := []string{
		"Task 1 done",
		"Task 2 done",
		"Task 3 done",
		"Task 4 done",
		"All tasks completed",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(stripped, msg) {
			t.Errorf("Expected message %q not found in output", msg)
			t.Logf("Full output:\n%s", stripped)
		}
	}

	// Count how many times each completion message appears (should be exactly once)
	for i := 1; i <= 4; i++ {
		msg := fmt.Sprintf("Task %d done", i)
		count := strings.Count(stripped, msg)
		if count != 1 {
			t.Errorf("Message %q appears %d times, expected 1", msg, count)
		}
	}

	// Verify proper cursor positioning by checking the raw output
	// Count the number of ansiMoveDown sequences after completions
	// There should be one per completion
	moveDownPattern := "\033["
	moveDownCount := strings.Count(output, moveDownPattern+"B") + // Move down variations
		strings.Count(output, moveDownPattern+"1B") +
		strings.Count(output, moveDownPattern+"2B") +
		strings.Count(output, moveDownPattern+"3B")

	debugLog("TEST", "Found %d cursor move-down sequences in output", moveDownCount)

	// The exact count depends on the animation frames and completions,
	// but we should have at least a few cursor movements
	if moveDownCount == 0 {
		t.Error("No cursor move-down sequences found; cursor positioning may be broken")
	}
}

// TestCursorPositionOutOfOrder tests cursor positioning with out-of-order completions
func TestCursorPositionOutOfOrder(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	s1 := logger.Spinner("Task 1")
	s2 := logger.Spinner("Task 2")
	s3 := logger.Spinner("Task 3")

	time.Sleep(100 * time.Millisecond)

	// Complete in reverse order
	s3.Success("Task 3 done")
	time.Sleep(50 * time.Millisecond)

	s1.Success("Task 1 done")
	time.Sleep(50 * time.Millisecond)

	s2.Success("Task 2 done")
	time.Sleep(50 * time.Millisecond)

	logger.Info("Done")

	output := stripAnsiCodes(buf.String())

	// Verify all completions are visible
	if !strings.Contains(output, "Task 1 done") {
		t.Error("Task 1 completion not found in output")
	}
	if !strings.Contains(output, "Task 2 done") {
		t.Error("Task 2 completion not found in output")
	}
	if !strings.Contains(output, "Task 3 done") {
		t.Error("Task 3 completion not found in output")
	}
	if !strings.Contains(output, "Done") {
		t.Error("Final message not found in output")
	}
}

// stripAnsiCodes removes ANSI escape sequences from a string
func stripAnsiCodes(s string) string {
	// Simple ANSI code stripper (handles common codes)
	result := s
	// Remove color codes
	result = strings.ReplaceAll(result, "\033[36m", "")
	result = strings.ReplaceAll(result, "\033[32m", "")
	result = strings.ReplaceAll(result, "\033[31m", "")
	result = strings.ReplaceAll(result, "\033[33m", "")
	result = strings.ReplaceAll(result, "\033[0m", "")
	result = strings.ReplaceAll(result, "\033[2m", "")

	// Remove cursor movement codes with regex-like patterns
	// This is a simplified version - we'll just remove common patterns
	for {
		// Look for ANSI codes like \033[NNN or \033[NNNM
		idx := strings.Index(result, "\033[")
		if idx == -1 {
			break
		}
		// Find the end of the sequence (letter)
		endIdx := idx + 2
		for endIdx < len(result) && !((result[endIdx] >= 'A' && result[endIdx] <= 'Z') || (result[endIdx] >= 'a' && result[endIdx] <= 'z')) {
			endIdx++
		}
		if endIdx < len(result) {
			endIdx++ // Include the letter
		}
		// Remove this sequence
		result = result[:idx] + result[endIdx:]
	}

	return result
}
