package bullets

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// stripANSI removes all ANSI escape sequences from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences: ESC [ ... m (colors), ESC [ ... A/B/G/K (cursor movement)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mABCDEFGHJKSTfGK]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// setupTTYMode enables TTY mode for testing
func setupTTYMode() func() {
	original := os.Getenv("BULLETS_FORCE_TTY")
	os.Setenv("BULLETS_FORCE_TTY", "1")

	return func() {
		if original != "" {
			os.Setenv("BULLETS_FORCE_TTY", original)
		} else {
			os.Unsetenv("BULLETS_FORCE_TTY")
		}
	}
}

// captureOutput creates a buffer and logger for capturing output
func captureOutput() (*bytes.Buffer, *Logger) {
	buf := &bytes.Buffer{}
	logger := New(buf)
	return buf, logger
}

// countNonEmptyLines counts non-empty lines after stripping ANSI codes
func countNonEmptyLines(s string) int {
	stripped := stripANSI(s)
	lines := strings.Split(stripped, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// findBlankLines returns the indices of blank lines in the output
func findBlankLines(s string) []int {
	stripped := stripANSI(s)
	lines := strings.Split(stripped, "\n")
	var blankLineIndices []int

	for i, line := range lines {
		if strings.TrimSpace(line) == "" && i < len(lines)-1 {
			blankLineIndices = append(blankLineIndices, i)
		}
	}

	return blankLineIndices
}

// extractVisibleLines returns all visible non-empty lines after stripping ANSI
func extractVisibleLines(s string) []string {
	stripped := stripANSI(s)
	lines := strings.Split(stripped, "\n")
	var visibleLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			visibleLines = append(visibleLines, trimmed)
		}
	}

	return visibleLines
}

// TestSequentialSpinnerGroupsNoExtraLines tests that sequential spinner groups
// don't create extra blank lines in the output
func TestSequentialSpinnerGroupsNoExtraLines(t *testing.T) {
	cleanup := setupTTYMode()
	defer cleanup()

	buf, logger := captureOutput()

	// First group: Create 3 spinners and complete them
	s1 := logger.Spinner("Group 1 - Task 1")
	s2 := logger.Spinner("Group 1 - Task 2")
	s3 := logger.Spinner("Group 1 - Task 3")

	time.Sleep(200 * time.Millisecond) // Allow animation frames

	s1.Success("Group 1 - Task 1 done")
	s2.Success("Group 1 - Task 2 done")
	s3.Success("Group 1 - Task 3 done")

	time.Sleep(100 * time.Millisecond) // Ensure all updates are processed

	// Second group: Immediately create 3 more spinners and complete them
	s4 := logger.Spinner("Group 2 - Task 1")
	s5 := logger.Spinner("Group 2 - Task 2")
	s6 := logger.Spinner("Group 2 - Task 3")

	time.Sleep(200 * time.Millisecond) // Allow animation frames

	s4.Success("Group 2 - Task 1 done")
	s5.Success("Group 2 - Task 2 done")
	s6.Success("Group 2 - Task 3 done")

	time.Sleep(100 * time.Millisecond) // Ensure all updates are processed

	// Analyze output
	output := buf.String()
	stripped := stripANSI(output)
	lines := strings.Split(stripped, "\n")

	// Count visible lines and blank lines
	visibleLines := extractVisibleLines(output)
	blankLines := findBlankLines(output)

	// We expect exactly 6 completion messages (no extra blank lines)
	expectedVisibleLines := 6

	if len(visibleLines) != expectedVisibleLines {
		t.Errorf("Expected %d visible lines, got %d", expectedVisibleLines, len(visibleLines))
		t.Logf("Visible lines:\n%s", strings.Join(visibleLines, "\n"))
	}

	// Check for unexpected blank lines
	if len(blankLines) > 0 {
		t.Errorf("Found %d unexpected blank lines at indices: %v", len(blankLines), blankLines)
		t.Logf("Full output (with line numbers):")
		for i, line := range lines {
			t.Logf("  [%d] %q", i, line)
		}
	}

	// Verify all expected messages are present
	expectedMessages := []string{
		"Group 1 - Task 1 done",
		"Group 1 - Task 2 done",
		"Group 1 - Task 3 done",
		"Group 2 - Task 1 done",
		"Group 2 - Task 2 done",
		"Group 2 - Task 3 done",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(stripped, msg) {
			t.Errorf("Expected message not found in output: %q", msg)
		}
	}
}

// TestCompletionMessagesNotOverwritten ensures that spinner completion messages
// are not overwritten by subsequent operations
func TestCompletionMessagesNotOverwritten(t *testing.T) {
	cleanup := setupTTYMode()
	defer cleanup()

	buf, logger := captureOutput()

	// Create spinners with unique identifiers
	s1 := logger.Spinner("Processing task A")
	s2 := logger.Spinner("Processing task B")
	s3 := logger.Spinner("Processing task C")
	s4 := logger.Spinner("Processing task D")

	time.Sleep(200 * time.Millisecond) // Allow animation frames

	// Complete spinners with different completion types and unique messages
	s1.Success("Task A completed successfully [ID:A-SUCCESS]")
	time.Sleep(50 * time.Millisecond)

	s2.Error("Task B failed with error [ID:B-ERROR]")
	time.Sleep(50 * time.Millisecond)

	s3.Replace("Task C was replaced [ID:C-REPLACE]")
	time.Sleep(50 * time.Millisecond)

	s4.Success("Task D finished [ID:D-SUCCESS]")
	time.Sleep(100 * time.Millisecond) // Ensure all updates are processed

	// Analyze output
	output := buf.String()
	stripped := stripANSI(output)

	// Define expected completion messages with unique identifiers
	expectedMessages := map[string]string{
		"A-SUCCESS": "Task A completed successfully [ID:A-SUCCESS]",
		"B-ERROR":   "Task B failed with error [ID:B-ERROR]",
		"C-REPLACE": "Task C was replaced [ID:C-REPLACE]",
		"D-SUCCESS": "Task D finished [ID:D-SUCCESS]",
	}

	// Check that all completion messages are present in the output
	missingMessages := []string{}
	for id, msg := range expectedMessages {
		if !strings.Contains(stripped, msg) {
			missingMessages = append(missingMessages, fmt.Sprintf("%s: %q", id, msg))
		}
	}

	if len(missingMessages) > 0 {
		t.Errorf("The following completion messages were not found in output (may have been overwritten):")
		for _, msg := range missingMessages {
			t.Errorf("  - %s", msg)
		}
		t.Logf("Full stripped output:\n%s", stripped)
	}

	// Also verify that each message appears exactly once (not duplicated)
	for id, msg := range expectedMessages {
		count := strings.Count(stripped, msg)
		if count > 1 {
			t.Errorf("Message %s appears %d times (expected 1): %q", id, count, msg)
		}
	}

	// Check for unexpected blank lines
	blankLines := findBlankLines(output)
	if len(blankLines) > 0 {
		t.Logf("Warning: Found %d blank lines at indices: %v", len(blankLines), blankLines)
	}
}

// TestRapidSequentialSpinnerGroups stress-tests rapid creation and completion
// of spinner groups with minimal delays
func TestRapidSequentialSpinnerGroups(t *testing.T) {
	cleanup := setupTTYMode()
	defer cleanup()

	buf, logger := captureOutput()

	// Create and complete 3 groups rapidly
	for groupNum := 1; groupNum <= 3; groupNum++ {
		// Create 2 spinners per group
		s1 := logger.Spinner(fmt.Sprintf("Group %d - Task 1", groupNum))
		s2 := logger.Spinner(fmt.Sprintf("Group %d - Task 2", groupNum))

		time.Sleep(50 * time.Millisecond) // Minimal animation time

		// Complete them rapidly
		s1.Success(fmt.Sprintf("Group %d - Task 1 complete", groupNum))
		s2.Success(fmt.Sprintf("Group %d - Task 2 complete", groupNum))

		time.Sleep(20 * time.Millisecond) // Very short delay between groups
	}

	time.Sleep(100 * time.Millisecond) // Final processing time

	// Analyze output
	output := buf.String()
	stripped := stripANSI(output)
	visibleLines := extractVisibleLines(output)
	blankLines := findBlankLines(output)

	// We expect exactly 6 completion messages (3 groups * 2 tasks)
	expectedVisibleLines := 6

	if len(visibleLines) != expectedVisibleLines {
		t.Errorf("Expected %d visible lines, got %d", expectedVisibleLines, len(visibleLines))
		t.Logf("Visible lines:\n%s", strings.Join(visibleLines, "\n"))
	}

	// Check for unexpected blank lines
	if len(blankLines) > 0 {
		t.Errorf("Found %d unexpected blank lines at indices: %v", len(blankLines), blankLines)
		lines := strings.Split(stripped, "\n")
		t.Logf("Full output (with line numbers):")
		for i, line := range lines {
			t.Logf("  [%d] %q", i, line)
		}
	}

	// Verify all expected messages are present
	for groupNum := 1; groupNum <= 3; groupNum++ {
		for taskNum := 1; taskNum <= 2; taskNum++ {
			expectedMsg := fmt.Sprintf("Group %d - Task %d complete", groupNum, taskNum)
			if !strings.Contains(stripped, expectedMsg) {
				t.Errorf("Expected message not found: %q", expectedMsg)
			}
		}
	}
}

// TestMixedCompletionTypes tests various completion methods (Success, Error, Stop, Replace)
// in quick succession to ensure proper handling of different completion types
func TestMixedCompletionTypes(t *testing.T) {
	cleanup := setupTTYMode()
	defer cleanup()

	buf, logger := captureOutput()

	// Create spinners
	s1 := logger.Spinner("Task 1 (will succeed)")
	s2 := logger.Spinner("Task 2 (will error)")
	s3 := logger.Spinner("Task 3 (will succeed)")
	s4 := logger.Spinner("Task 4 (will replace)")
	s5 := logger.Spinner("Task 5 (will error)")
	s6 := logger.Spinner("Task 6 (will succeed)")

	time.Sleep(100 * time.Millisecond) // Allow animation

	// Complete with different types in quick succession
	s1.Success("Task 1 succeeded [TYPE:SUCCESS]")
	time.Sleep(20 * time.Millisecond)

	s2.Error("Task 2 failed [TYPE:ERROR]")
	time.Sleep(20 * time.Millisecond)

	s3.Success("Task 3 succeeded [TYPE:SUCCESS]")
	time.Sleep(20 * time.Millisecond)

	s4.Replace("Task 4 replaced [TYPE:REPLACE]")
	time.Sleep(20 * time.Millisecond)

	s5.Error("Task 5 failed [TYPE:ERROR]")
	time.Sleep(20 * time.Millisecond)

	s6.Success("Task 6 succeeded [TYPE:SUCCESS]")
	time.Sleep(100 * time.Millisecond)

	// Analyze output
	output := buf.String()
	stripped := stripANSI(output)
	visibleLines := extractVisibleLines(output)
	blankLines := findBlankLines(output)

	// Verify all completion messages are present
	expectedMessages := []string{
		"Task 1 succeeded [TYPE:SUCCESS]",
		"Task 2 failed [TYPE:ERROR]",
		"Task 3 succeeded [TYPE:SUCCESS]",
		"Task 4 replaced [TYPE:REPLACE]",
		"Task 5 failed [TYPE:ERROR]",
		"Task 6 succeeded [TYPE:SUCCESS]",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(stripped, msg) {
			t.Errorf("Expected message not found: %q", msg)
		}
	}

	// Check for extra blank lines
	if len(blankLines) > 0 {
		t.Errorf("Found %d unexpected blank lines at indices: %v", len(blankLines), blankLines)
	}

	// Verify expected number of visible lines
	expectedVisibleLines := len(expectedMessages)
	if len(visibleLines) != expectedVisibleLines {
		t.Errorf("Expected %d visible lines, got %d", expectedVisibleLines, len(visibleLines))
		t.Logf("Visible lines:\n%s", strings.Join(visibleLines, "\n"))
	}
}

// TestConcurrentGroupTransitions tests parallel spinner group creation
// to ensure thread safety and proper line management under concurrent load
func TestConcurrentGroupTransitions(t *testing.T) {
	cleanup := setupTTYMode()
	defer cleanup()

	buf, logger := captureOutput()

	var wg sync.WaitGroup

	// Group 1: Create and complete spinners in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		s1 := logger.Spinner("Concurrent Group 1 - Task 1")
		s2 := logger.Spinner("Concurrent Group 1 - Task 2")

		time.Sleep(80 * time.Millisecond)

		s1.Success("Concurrent Group 1 - Task 1 done")
		s2.Success("Concurrent Group 1 - Task 2 done")
	}()

	// Group 2: Create and complete spinners in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(40 * time.Millisecond) // Stagger start

		s3 := logger.Spinner("Concurrent Group 2 - Task 1")
		s4 := logger.Spinner("Concurrent Group 2 - Task 2")

		time.Sleep(80 * time.Millisecond)

		s3.Success("Concurrent Group 2 - Task 1 done")
		s4.Success("Concurrent Group 2 - Task 2 done")
	}()

	wg.Wait()
	time.Sleep(150 * time.Millisecond) // Allow final processing

	// Analyze output
	output := buf.String()
	stripped := stripANSI(output)
	visibleLines := extractVisibleLines(output)

	// Verify all expected messages are present
	expectedMessages := []string{
		"Concurrent Group 1 - Task 1 done",
		"Concurrent Group 1 - Task 2 done",
		"Concurrent Group 2 - Task 1 done",
		"Concurrent Group 2 - Task 2 done",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(stripped, msg) {
			t.Errorf("Expected message not found: %q", msg)
		}
	}

	// Verify expected number of visible lines
	expectedVisibleLines := len(expectedMessages)
	if len(visibleLines) != expectedVisibleLines {
		t.Errorf("Expected %d visible lines, got %d", expectedVisibleLines, len(visibleLines))
		t.Logf("Visible lines:\n%s", strings.Join(visibleLines, "\n"))
	}

	// Check for blank lines (may be acceptable in concurrent scenarios, so just log)
	blankLines := findBlankLines(output)
	if len(blankLines) > 0 {
		t.Logf("Found %d blank lines at indices: %v (may be expected in concurrent test)", len(blankLines), blankLines)
	}
}
