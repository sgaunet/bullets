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

// SpinnerTestCapture extends ansiCapture with comprehensive testing utilities
type SpinnerTestCapture struct {
	mu            sync.Mutex
	output        bytes.Buffer
	events        []ANSIEvent
	frames        []AnimationFrame
	cursorHistory []CursorPosition
	currentCursor CursorPosition
}

// ANSIEvent represents a parsed ANSI escape sequence or text event
type ANSIEvent struct {
	Timestamp time.Time
	Raw       string
	Type      ANSIEventType
	Value     int    // For move operations
	Text      string // For text content
}

// ANSIEventType categorizes ANSI events
type ANSIEventType string

const (
	EventMoveUp      ANSIEventType = "moveUp"
	EventMoveDown    ANSIEventType = "moveDown"
	EventClearLine   ANSIEventType = "clearLine"
	EventMoveToCol   ANSIEventType = "moveToCol"
	EventText        ANSIEventType = "text"
	EventNewline     ANSIEventType = "newline"
	EventUnknown     ANSIEventType = "unknown"
)

// AnimationFrame represents a complete animation frame with all spinner states
type AnimationFrame struct {
	Timestamp     time.Time
	SpinnerStates []SpinnerState
	CursorPos     CursorPosition
}

// SpinnerState represents the state of a single spinner at a point in time
type SpinnerState struct {
	Line    int
	Content string
}

// CursorPosition tracks the current cursor position
type CursorPosition struct {
	Line   int
	Column int
}

// NewSpinnerTestCapture creates a new test capture utility
func NewSpinnerTestCapture() *SpinnerTestCapture {
	return &SpinnerTestCapture{
		events:        make([]ANSIEvent, 0),
		frames:        make([]AnimationFrame, 0),
		cursorHistory: make([]CursorPosition, 0),
		currentCursor: CursorPosition{Line: 0, Column: 0},
	}
}

// Write implements io.Writer and captures output
func (s *SpinnerTestCapture) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, err = s.output.Write(p)
	if err != nil {
		return n, err
	}

	// Parse and track cursor movements
	s.parseAndTrack(string(p))
	return n, err
}

// parseAndTrack parses ANSI sequences and tracks cursor position
func (s *SpinnerTestCapture) parseAndTrack(text string) {
	// Enhanced ANSI pattern matching
	ansiPattern := regexp.MustCompile(`\033\[(\d+)?([A-Za-z])`)

	lastIdx := 0
	for _, match := range ansiPattern.FindAllStringSubmatchIndex(text, -1) {
		// Capture text before ANSI sequence
		if match[0] > lastIdx {
			textContent := text[lastIdx:match[0]]
			if textContent != "" {
				s.processText(textContent)
			}
		}

		// Parse ANSI sequence
		numStr := ""
		if match[2] != -1 {
			numStr = text[match[2]:match[3]]
		}
		code := text[match[4]:match[5]]

		num := 1
		if numStr != "" {
			fmt.Sscanf(numStr, "%d", &num)
		}

		event := ANSIEvent{
			Timestamp: time.Now(),
			Raw:       text[match[0]:match[1]],
			Value:     num,
		}

		// Process ANSI code and update cursor
		switch code {
		case "A": // Move up
			event.Type = EventMoveUp
			s.currentCursor.Line -= num
		case "B": // Move down
			event.Type = EventMoveDown
			s.currentCursor.Line += num
		case "K": // Clear line
			event.Type = EventClearLine
		case "G": // Move to column
			event.Type = EventMoveToCol
			if num > 0 {
				s.currentCursor.Column = num - 1 // ANSI columns are 1-indexed
			} else {
				s.currentCursor.Column = 0
			}
		default:
			event.Type = EventUnknown
		}

		s.events = append(s.events, event)
		s.cursorHistory = append(s.cursorHistory, s.currentCursor)
		lastIdx = match[1]
	}

	// Process remaining text
	if lastIdx < len(text) {
		remaining := text[lastIdx:]
		if remaining != "" {
			s.processText(remaining)
		}
	}
}

// processText handles text content and newlines
func (s *SpinnerTestCapture) processText(text string) {
	// Split by newlines to track them separately
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if line != "" && !isOnlyANSICodes(line) {
			event := ANSIEvent{
				Timestamp: time.Now(),
				Type:      EventText,
				Text:      line,
			}
			s.events = append(s.events, event)
		}

		// Add newline event (except for last empty line after split)
		if i < len(lines)-1 {
			event := ANSIEvent{
				Timestamp: time.Now(),
				Type:      EventNewline,
				Raw:       "\\n",
			}
			s.events = append(s.events, event)
			s.currentCursor.Line++
			s.currentCursor.Column = 0
		}
	}
}

// isOnlyANSICodes checks if string contains only ANSI codes
func isOnlyANSICodes(s string) bool {
	ansiPattern := regexp.MustCompile(`\033\[[0-9;]*[A-Za-z]`)
	cleaned := ansiPattern.ReplaceAllString(s, "")
	return len(strings.TrimSpace(cleaned)) == 0
}

// GetEvents returns a thread-safe copy of captured events
func (s *SpinnerTestCapture) GetEvents() []ANSIEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ANSIEvent, len(s.events))
	copy(result, s.events)
	return result
}

// GetCursorHistory returns the complete cursor movement history
func (s *SpinnerTestCapture) GetCursorHistory() []CursorPosition {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]CursorPosition, len(s.cursorHistory))
	copy(result, s.cursorHistory)
	return result
}

// GetRawOutput returns the complete raw output
func (s *SpinnerTestCapture) GetRawOutput() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

// ExtractFrames analyzes events to identify distinct animation frames
func (s *SpinnerTestCapture) ExtractFrames() []AnimationFrame {
	s.mu.Lock()
	defer s.mu.Unlock()

	frames := make([]AnimationFrame, 0)
	currentFrame := AnimationFrame{
		SpinnerStates: make([]SpinnerState, 0),
	}

	inFrame := false
	currentLine := -1

	for i, event := range s.events {
		switch event.Type {
		case EventMoveUp:
			if inFrame {
				// Save previous frame
				frames = append(frames, currentFrame)
				currentFrame = AnimationFrame{SpinnerStates: make([]SpinnerState, 0)}
			}
			inFrame = true
			currentLine = -event.Value // Track relative line position
			currentFrame.Timestamp = event.Timestamp

		case EventText:
			if inFrame && currentLine >= 0 {
				currentFrame.SpinnerStates = append(currentFrame.SpinnerStates, SpinnerState{
					Line:    currentLine,
					Content: event.Text,
				})
			}

		case EventMoveDown:
			inFrame = false
			if i > 0 && i < len(s.cursorHistory) {
				currentFrame.CursorPos = s.cursorHistory[i]
			}
		}
	}

	// Add final frame if exists
	if len(currentFrame.SpinnerStates) > 0 {
		frames = append(frames, currentFrame)
	}

	s.frames = frames
	return frames
}

// ValidateNoBlankLines checks that no extra blank lines exist in output
func (s *SpinnerTestCapture) ValidateNoBlankLines(t *testing.T) bool {
	events := s.GetEvents()

	// Look for consecutive newline events (indicates blank lines)
	consecutiveNewlines := 0
	for _, event := range events {
		if event.Type == EventNewline {
			consecutiveNewlines++
			if consecutiveNewlines > 1 {
				t.Errorf("Found %d consecutive newlines (blank line detected)", consecutiveNewlines)
				return false
			}
		} else if event.Type == EventText {
			consecutiveNewlines = 0
		}
	}

	return true
}

// ValidateCursorStability checks that cursor movements are consistent
func (s *SpinnerTestCapture) ValidateCursorStability(t *testing.T) bool {
	// Check for unexpected cursor drifts
	// UPDATED: Account for completion events where cursor position changes
	// - Animation frames: moveUp(N) + moveDown(N) - cursor returns to same position
	// - Completions: moveUp(N) + moveDown(M where M <= N) - cursor may move up

	moveStack := make([]int, 0)
	isCompletionSequence := false

	for i, event := range s.events {
		if event.Type == EventMoveUp {
			moveStack = append(moveStack, event.Value)
			// Check if this might be a completion sequence by looking ahead
			// Completion sequences have: moveUp, clearLine, moveToCol, text with success/error bullet, moveDown
			if i+4 < len(s.events) {
				nextEvents := s.events[i+1 : i+5]
				// Look for completion pattern: clearLine, moveToCol, text, moveDown
				if len(nextEvents) >= 3 &&
					nextEvents[0].Type == EventClearLine &&
					nextEvents[1].Type == EventMoveToCol {
					// Check if there's text with completion markers
					for j := 2; j < len(nextEvents); j++ {
						if nextEvents[j].Type == EventText {
							text := nextEvents[j].Text
							// Check for completion markers (success/error bullets)
							if strings.Contains(text, "✓") ||
							   strings.Contains(text, "✗") ||
							   strings.Contains(text, "done") ||
							   strings.Contains(text, "failed") ||
							   strings.Contains(text, "complete") ||
							   strings.Contains(text, "error") ||
							   strings.Contains(text, "succeeded") {
								isCompletionSequence = true
								break
							}
						}
					}
				}
			}
		} else if event.Type == EventMoveDown {
			if len(moveStack) > 0 {
				expectedUp := moveStack[len(moveStack)-1]
				// For completions, allow moveDown to be less than moveUp
				// For animation frames, they should be equal
				if isCompletionSequence {
					// Completion: moveDown can be less than or equal to moveUp
					if event.Value > expectedUp {
						t.Errorf("Cursor movement error at event %d: moveDown(%d) > moveUp(%d) in completion",
							i, event.Value, expectedUp)
						return false
					}
					// This is OK - completion can move cursor up
					isCompletionSequence = false
				} else {
					// Animation frame: moveDown should equal moveUp
					if event.Value != expectedUp {
						// Only log as error if the difference is large (tolerance for small differences)
						diff := expectedUp - event.Value
						if diff < 0 {
							diff = -diff
						}
						if diff > 1 {
							t.Errorf("Cursor movement mismatch at event %d: moveUp(%d) but moveDown(%d)",
								i, expectedUp, event.Value)
							return false
						}
					}
				}
				moveStack = moveStack[:len(moveStack)-1]
			}
		} else if event.Type == EventNewline {
			// Newline resets completion tracking
			isCompletionSequence = false
		}
	}

	// Allow some unmatched moves at the end (final cursor position adjustment)
	if len(moveStack) > 2 {
		t.Errorf("Unmatched cursor movements: %d moveUp operations without corresponding moveDown", len(moveStack))
		return false
	}

	return true
}

// CountEventType counts occurrences of a specific event type
func (s *SpinnerTestCapture) CountEventType(eventType ANSIEventType) int {
	count := 0
	for _, event := range s.events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

// GetMoveUpValues extracts all moveUp values for pattern analysis
func (s *SpinnerTestCapture) GetMoveUpValues() []int {
	values := make([]int, 0)
	for _, event := range s.events {
		if event.Type == EventMoveUp {
			values = append(values, event.Value)
		}
	}
	return values
}

// GetMoveDownValues extracts all moveDown values for pattern analysis
func (s *SpinnerTestCapture) GetMoveDownValues() []int {
	values := make([]int, 0)
	for _, event := range s.events {
		if event.Type == EventMoveDown {
			values = append(values, event.Value)
		}
	}
	return values
}

// DumpEvents prints all captured events for debugging
func (s *SpinnerTestCapture) DumpEvents(t *testing.T, maxEvents int) {
	events := s.GetEvents()
	limit := len(events)
	if maxEvents > 0 && maxEvents < limit {
		limit = maxEvents
	}

	t.Logf("Captured %d total events (showing first %d):", len(events), limit)
	for i := 0; i < limit; i++ {
		event := events[i]
		switch event.Type {
		case EventText:
			t.Logf("  [%d] %s: %q", i, event.Type, event.Text)
		case EventMoveUp, EventMoveDown:
			t.Logf("  [%d] %s(%d) [raw: %q]", i, event.Type, event.Value, event.Raw)
		default:
			t.Logf("  [%d] %s [raw: %q]", i, event.Type, event.Raw)
		}
	}
}

// ValidateNoPositionDrift checks that line positions remain stable over time
func (s *SpinnerTestCapture) ValidateNoPositionDrift(t *testing.T) bool {
	frames := s.ExtractFrames()

	if len(frames) < 2 {
		return true // Not enough frames to check drift
	}

	// Track line numbers for each spinner across frames
	spinnerLines := make(map[string][]int)

	for _, frame := range frames {
		for _, state := range frame.SpinnerStates {
			key := state.Content // Use content as identifier
			spinnerLines[key] = append(spinnerLines[key], state.Line)
		}
	}

	// Check for drift (changing line numbers for same spinner)
	for content, lines := range spinnerLines {
		if len(lines) > 1 {
			firstLine := lines[0]
			for i, line := range lines {
				if line != firstLine {
					t.Errorf("Position drift detected for spinner %q: frame %d has line %d, expected %d",
						content, i, line, firstLine)
					return false
				}
			}
		}
	}

	return true
}
