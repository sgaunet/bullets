package bullets

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// ansiCapture captures and parses ANSI escape sequences from output
type ansiCapture struct {
	mu     sync.Mutex
	output bytes.Buffer
	events []ansiEvent
}

type ansiEvent struct {
	timestamp time.Time
	raw       string
	eventType string // "moveUp", "moveDown", "clearLine", "text", "newline"
	value     int    // For move operations, number of lines
}

// Write implements io.Writer and captures output
func (a *ansiCapture) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	n, err = a.output.Write(p)
	if err != nil {
		return n, err
	}

	// Parse ANSI sequences from the written data
	a.parseANSI(string(p))
	return n, err
}

// parseANSI extracts ANSI escape sequences and text
func (a *ansiCapture) parseANSI(s string) {
	// Pattern: \033[<number><letter>
	ansiPattern := regexp.MustCompile(`\033\[(\d+)?([A-Za-z])`)

	lastIdx := 0
	for _, match := range ansiPattern.FindAllStringSubmatchIndex(s, -1) {
		// Capture any text before this ANSI sequence
		if match[0] > lastIdx {
			text := s[lastIdx:match[0]]
			if text != "" {
				// Check for newlines
				for _, line := range strings.Split(text, "\n") {
					if line != "" && line != "\033[2K\033[0G" {
						a.events = append(a.events, ansiEvent{
							timestamp: time.Now(),
							raw:       line,
							eventType: "text",
						})
					}
				}
				if strings.Contains(text, "\n") {
					a.events = append(a.events, ansiEvent{
						timestamp: time.Now(),
						raw:       "\\n",
						eventType: "newline",
					})
				}
			}
		}

		// Parse the ANSI sequence
		numStr := ""
		if match[2] != -1 {
			numStr = s[match[2]:match[3]]
		}
		code := s[match[4]:match[5]]

		num := 1
		if numStr != "" {
			fmt.Sscanf(numStr, "%d", &num)
		}

		event := ansiEvent{
			timestamp: time.Now(),
			raw:       s[match[0]:match[1]],
			value:     num,
		}

		switch code {
		case "A":
			event.eventType = "moveUp"
		case "B":
			event.eventType = "moveDown"
		case "K":
			event.eventType = "clearLine"
		case "G":
			event.eventType = "moveToCol"
		default:
			event.eventType = "unknown"
		}

		a.events = append(a.events, event)
		lastIdx = match[1]
	}

	// Capture any remaining text
	if lastIdx < len(s) {
		remaining := s[lastIdx:]
		if remaining != "" && remaining != "\033[2K" && remaining != "\033[0G" {
			a.events = append(a.events, ansiEvent{
				timestamp: time.Now(),
				raw:       remaining,
				eventType: "text",
			})
		}
	}
}

// getEvents returns captured events (thread-safe)
func (a *ansiCapture) getEvents() []ansiEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]ansiEvent, len(a.events))
	copy(result, a.events)
	return result
}

// TestConcurrentSpinnersANSISequences reproduces the exact ANSI sequence bug
// This test captures the [2A, [3A inconsistency mentioned in the PRD
func TestConcurrentSpinnersANSISequences(t *testing.T) {
	capture := &ansiCapture{}
	writeMu := sync.Mutex{}
	logger := &Logger{
		writer:            capture,
		writeMu:           writeMu,
		level:             InfoLevel,
		fields:            make(map[string]interface{}),
		useSpecialBullets: true,
		customBullets:     make(map[Level]string),
		coordinator:       newSpinnerCoordinator(capture, &writeMu, true), // Force TTY mode
	}
	logger.coordinator.isTTY = true // Force TTY

	// Create 3 spinners (will be assigned lines 0, 1, 2)
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")

	// Let them animate for a bit
	time.Sleep(200 * time.Millisecond)

	// Complete the middle spinner
	spinner2.Success("Task 2 complete")

	// Wait for completion to render
	time.Sleep(50 * time.Millisecond)

	// Let remaining spinners animate
	time.Sleep(200 * time.Millisecond)

	// Complete remaining spinners
	spinner1.Success("Task 1 complete")
	spinner3.Success("Task 3 complete")

	time.Sleep(100 * time.Millisecond)

	// Analyze captured ANSI sequences
	events := capture.getEvents()

	// Find moveUp events and check for inconsistencies
	moveUpValues := []int{}
	for _, event := range events {
		if event.eventType == "moveUp" {
			moveUpValues = append(moveUpValues, event.value)
		}
	}

	t.Logf("Captured %d ANSI events", len(events))
	t.Logf("MoveUp values: %v", moveUpValues)

	// Look for the specific bug: after middle spinner completes,
	// remaining spinners should use consistent line numbers
	// If bug exists, we'll see inconsistent moveUp values (e.g., [2A then [3A for same spinner)

	// Log all events for debugging
	for i, event := range events {
		if i < 50 { // Limit output
			t.Logf("Event %d: %s - %s (value: %d)", i, event.eventType, event.raw, event.value)
		}
	}
}

// TestRenderCompletionNewlineBug specifically tests the newline issue
func TestRenderCompletionNewlineBug(t *testing.T) {
	capture := &ansiCapture{}
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

	// Create 2 spinners
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")

	time.Sleep(100 * time.Millisecond)

	// Complete first spinner and check for newline
	spinner1.Success("Task 1 done")

	time.Sleep(50 * time.Millisecond)

	events := capture.getEvents()

	// Look for newline events (these are the bug)
	newlineCount := 0
	for _, event := range events {
		if event.eventType == "newline" {
			newlineCount++
			t.Logf("Found newline event at position %d", len(events))
		}
	}

	// Check for the specific pattern: moveUp, clearLine, text, NEWLINE, moveDown
	// This pattern indicates the bug where renderCompletion adds a newline
	for i := 0; i < len(events)-4; i++ {
		if events[i].eventType == "moveUp" &&
			events[i+1].eventType == "clearLine" &&
			events[i+3].eventType == "newline" &&
			events[i+4].eventType == "moveDown" {
			t.Logf("BUG CONFIRMED: Found newline pattern at event %d", i)
			t.Logf("  moveUp(%d) -> clearLine -> text -> NEWLINE -> moveDown(%d)",
				events[i].value, events[i+4].value)

			// The bug is when moveDown value = moveUp value - 1
			// This is the compensation for the newline
			if events[i+4].value == events[i].value-1 {
				t.Logf("  Confirmed: moveDown is compensating (moveUp=%d, moveDown=%d)",
					events[i].value, events[i+4].value)
			}
		}
	}

	spinner2.Stop()
}

// TestLineNumberDriftAfterCompletion tests the race condition
func TestLineNumberDriftAfterCompletion(t *testing.T) {
	capture := &ansiCapture{}
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

	// Create 3 spinners at lines 0, 1, 2
	spinner1 := logger.Spinner("Task 1")
	spinner2 := logger.Spinner("Task 2")
	spinner3 := logger.Spinner("Task 3")

	// Let them animate for a couple frames
	time.Sleep(200 * time.Millisecond)

	// Record spinner3's line number before completion
	logger.coordinator.mu.Lock()
	lineBeforeCompletion := logger.coordinator.spinners[spinner3].lineNumber
	logger.coordinator.mu.Unlock()

	// Complete middle spinner (spinner2 at line 1)
	spinner2.Success("Task 2 complete")

	// After completion, spinner3 should MAINTAIN its original line number (line 2)
	// This is the fix: spinners no longer shift to fill gaps
	time.Sleep(50 * time.Millisecond)

	logger.coordinator.mu.Lock()
	lineAfterCompletion := logger.coordinator.spinners[spinner3].lineNumber
	logger.coordinator.mu.Unlock()

	t.Logf("Spinner3 line number: before=%d, after=%d", lineBeforeCompletion, lineAfterCompletion)

	// Check that line number stayed the same (no recalculation in TTY mode)
	if lineAfterCompletion != lineBeforeCompletion {
		t.Errorf("Expected spinner3 to maintain line %d, but shifted to line %d",
			lineBeforeCompletion, lineAfterCompletion)
	}

	// Now check ANSI sequences for spinner3's next frame
	// It should continue using its ORIGINAL line number (lineBeforeCompletion)
	time.Sleep(100 * time.Millisecond) // Wait for at least one animation frame

	events := capture.getEvents()

	// Look for moveUp events after the completion
	// We expect spinner3 to keep using its original line position
	expectedMoveUp := lineBeforeCompletion + 1  // Should be 3 (line 2 + 1)
	foundCorrectMoveUp := false

	for _, event := range events {
		if event.eventType == "moveUp" && event.value == expectedMoveUp {
			foundCorrectMoveUp = true
			t.Logf("Found correct moveUp(%d) - spinner3 maintains original position", expectedMoveUp)
			break
		}
	}

	if !foundCorrectMoveUp {
		t.Logf("Warning: Did not find expected moveUp(%d) in ANSI sequence", expectedMoveUp)
		t.Logf("Spinner3 should maintain its original line position after spinner2 completes")
	}

	spinner1.Stop()
	spinner3.Stop()
}
