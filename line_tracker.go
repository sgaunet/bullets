package bullets

import (
	"sync"
	"time"
)

// lineState represents the state of a terminal line in the spinner system.
type lineState int

const (
	// lineStateAvailable indicates the line is free and can be allocated.
	lineStateAvailable lineState = iota
	// lineStateActive indicates the line is currently being used by an active spinner.
	lineStateActive
	// lineStateReserved indicates the line was used by a completed spinner
	// and is reserved to prevent overwriting completion messages.
	lineStateReserved
)

// reservedLine tracks the state and metadata of a terminal line.
type reservedLine struct {
	lineNumber     int
	state          lineState
	spinnerID      *Spinner      // Reference to the spinner using this line (nil if not active)
	reservedAt     time.Time     // Timestamp when line was reserved
	lastAccessedAt time.Time     // Last time this line was accessed
}

// lineTracker manages line allocation and tracks line states to prevent position drift.
//
// The tracker implements a "reserved lines" concept where completed spinner lines
// are marked as reserved rather than immediately available for reuse. This prevents
// new spinners from overwriting completion messages in TTY mode.
//
// Architecture:
//   - Lines have three states: Available, Active, Reserved
//   - Allocation prefers Available lines, creates new lines if none available
//   - Deallocation marks lines as Reserved (not Available)
//   - Cleanup phase reclaims Reserved lines as Available when safe
//   - Line position validation detects and corrects drift
//
// Thread Safety:
//   - All methods are thread-safe and protected by internal mutex
type lineTracker struct {
	mu              sync.Mutex
	lines           map[int]*reservedLine // Map of line number to line state
	activeSpinners  map[*Spinner]int      // Map of spinner to assigned line number
	nextLineNumber  int                   // Next line number to allocate (grows monotonically)
	cleanupInterval time.Duration         // How long to wait before reclaiming reserved lines
	isTTY           bool                  // Whether we're in TTY mode
}

// newLineTracker creates a new line tracker with the specified cleanup interval.
func newLineTracker(isTTY bool, cleanupInterval time.Duration) *lineTracker {
	return &lineTracker{
		lines:           make(map[int]*reservedLine),
		activeSpinners:  make(map[*Spinner]int),
		nextLineNumber:  0,
		cleanupInterval: cleanupInterval,
		isTTY:           isTTY,
	}
}

// allocateLine allocates a line for the given spinner.
// It prefers reusing available lines before creating new ones.
// Returns the allocated line number.
func (lt *lineTracker) allocateLine(spinner *Spinner) int {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	// In non-TTY mode, we can always reuse lines since there's no cursor positioning
	if !lt.isTTY {
		lineNum := lt.nextLineNumber
		lt.nextLineNumber++
		lt.activeSpinners[spinner] = lineNum
		lt.lines[lineNum] = &reservedLine{
			lineNumber:     lineNum,
			state:          lineStateActive,
			spinnerID:      spinner,
			lastAccessedAt: time.Now(),
		}
		return lineNum
	}

	// TTY mode: Look for an available line to reuse
	for lineNum, line := range lt.lines {
		if line.state == lineStateAvailable {
			// Reuse this line
			line.state = lineStateActive
			line.spinnerID = spinner
			line.lastAccessedAt = time.Now()
			lt.activeSpinners[spinner] = lineNum
			return lineNum
		}
	}

	// No available lines, allocate a new one
	lineNum := lt.nextLineNumber
	lt.nextLineNumber++
	lt.activeSpinners[spinner] = lineNum
	lt.lines[lineNum] = &reservedLine{
		lineNumber:     lineNum,
		state:          lineStateActive,
		spinnerID:      spinner,
		lastAccessedAt: time.Now(),
	}

	return lineNum
}

// deallocateLine marks a spinner's line as reserved (not immediately available).
// The line will be reclaimed during cleanup phase.
func (lt *lineTracker) deallocateLine(spinner *Spinner) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lineNum, exists := lt.activeSpinners[spinner]
	if !exists {
		return
	}

	delete(lt.activeSpinners, spinner)

	if line, ok := lt.lines[lineNum]; ok {
		// In TTY mode, mark as reserved to prevent immediate reuse
		// In non-TTY mode, mark as available immediately
		if lt.isTTY {
			line.state = lineStateReserved
			line.spinnerID = nil
			line.reservedAt = time.Now()
		} else {
			line.state = lineStateAvailable
			line.spinnerID = nil
		}
	}
}

// getLineNumber returns the line number for a given spinner.
// Returns -1 if the spinner is not registered.
// This is the authoritative method for querying spinner line positions.
func (lt *lineTracker) getLineNumber(spinner *Spinner) int {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if lineNum, exists := lt.activeSpinners[spinner]; exists {
		return lineNum
	}
	return -1
}

// reclaimReservedLines converts reserved lines to available lines if they've been
// reserved for longer than the cleanup interval. This is the "cleanup phase".
// Returns the number of lines reclaimed.
func (lt *lineTracker) reclaimReservedLines() int {
	if !lt.isTTY {
		return 0 // No need to reclaim in non-TTY mode
	}

	lt.mu.Lock()
	defer lt.mu.Unlock()

	now := time.Now()
	reclaimed := 0

	for _, line := range lt.lines {
		if line.state == lineStateReserved {
			// Check if line has been reserved long enough
			if now.Sub(line.reservedAt) >= lt.cleanupInterval {
				line.state = lineStateAvailable
				reclaimed++
			}
		}
	}

	return reclaimed
}

// validateLinePositions checks that all active spinners have valid line assignments.
// Returns a slice of spinners with invalid positions (for debugging).
func (lt *lineTracker) validateLinePositions() []*Spinner {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	var invalid []*Spinner

	for spinner, lineNum := range lt.activeSpinners {
		line, exists := lt.lines[lineNum]
		if !exists {
			invalid = append(invalid, spinner)
			continue
		}
		if line.state != lineStateActive {
			invalid = append(invalid, spinner)
			continue
		}
		if line.spinnerID != spinner {
			invalid = append(invalid, spinner)
			continue
		}
	}

	return invalid
}

// getActiveLineCount returns the number of lines currently in active state.
// This function is primarily used for testing and debugging.
func (lt *lineTracker) getActiveLineCount() int { //nolint:unused // Used in tests
	lt.mu.Lock()
	defer lt.mu.Unlock()

	count := 0
	for _, line := range lt.lines {
		if line.state == lineStateActive {
			count++
		}
	}
	return count
}

// getReservedLineCount returns the number of lines currently in reserved state.
// This function is primarily used for testing and debugging.
func (lt *lineTracker) getReservedLineCount() int { //nolint:unused // Used in tests
	lt.mu.Lock()
	defer lt.mu.Unlock()

	count := 0
	for _, line := range lt.lines {
		if line.state == lineStateReserved {
			count++
		}
	}
	return count
}

// getTotalLineCount returns the total number of lines allocated.
// This function is primarily used for testing and debugging.
func (lt *lineTracker) getTotalLineCount() int { //nolint:unused // Used in tests
	lt.mu.Lock()
	defer lt.mu.Unlock()

	return len(lt.lines)
}

// linePositionLedger provides detailed tracking and audit trail for line allocations.
type linePositionLedger struct {
	tracker       *lineTracker
	mu            sync.Mutex
	history       []ledgerEntry // Chronological history of all operations
	maxHistory    int           // Maximum history entries to keep
}

// ledgerEntry represents a single operation in the line allocation history.
type ledgerEntry struct {
	timestamp  time.Time
	operation  string    // "allocate", "deallocate", "reclaim", "validate"
	spinnerID  *Spinner
	lineNumber int
	details    string
}

// newLinePositionLedger creates a new line position ledger.
func newLinePositionLedger(tracker *lineTracker, maxHistory int) *linePositionLedger {
	return &linePositionLedger{
		tracker:    tracker,
		history:    make([]ledgerEntry, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// recordAllocation records a line allocation operation.
func (lpl *linePositionLedger) recordAllocation(spinner *Spinner, lineNumber int) {
	lpl.mu.Lock()
	defer lpl.mu.Unlock()

	lpl.addEntry(ledgerEntry{
		timestamp:  time.Now(),
		operation:  "allocate",
		spinnerID:  spinner,
		lineNumber: lineNumber,
		details:    "Line allocated to spinner",
	})
}

// recordDeallocation records a line deallocation operation.
func (lpl *linePositionLedger) recordDeallocation(spinner *Spinner, lineNumber int) {
	lpl.mu.Lock()
	defer lpl.mu.Unlock()

	lpl.addEntry(ledgerEntry{
		timestamp:  time.Now(),
		operation:  "deallocate",
		spinnerID:  spinner,
		lineNumber: lineNumber,
		details:    "Line deallocated, marked as reserved",
	})
}

// recordReclaim records a cleanup/reclaim operation.
func (lpl *linePositionLedger) recordReclaim(linesReclaimed int) {
	lpl.mu.Lock()
	defer lpl.mu.Unlock()

	lpl.addEntry(ledgerEntry{
		timestamp:  time.Now(),
		operation:  "reclaim",
		lineNumber: linesReclaimed,
		details:    "Reserved lines reclaimed as available",
	})
}

// recordValidation records a validation check.
func (lpl *linePositionLedger) recordValidation(invalidCount int) {
	lpl.mu.Lock()
	defer lpl.mu.Unlock()

	lpl.addEntry(ledgerEntry{
		timestamp:  time.Now(),
		operation:  "validate",
		lineNumber: invalidCount,
		details:    "Line position validation check",
	})
}

// addEntry adds an entry to the history, maintaining the maximum size.
func (lpl *linePositionLedger) addEntry(entry ledgerEntry) {
	lpl.history = append(lpl.history, entry)

	// Trim history if it exceeds max size
	if len(lpl.history) > lpl.maxHistory {
		// Keep the most recent maxHistory entries
		lpl.history = lpl.history[len(lpl.history)-lpl.maxHistory:]
	}
}

// getHistory returns a copy of the ledger history.
// This function is primarily used for testing and debugging.
func (lpl *linePositionLedger) getHistory() []ledgerEntry { //nolint:unused // Used in tests
	lpl.mu.Lock()
	defer lpl.mu.Unlock()

	// Return a copy to prevent external modification
	history := make([]ledgerEntry, len(lpl.history))
	copy(history, lpl.history)
	return history
}

// reconcile checks that the ledger's view of the world matches the tracker's actual state.
// Returns true if consistent, false if discrepancies found.
// This function is primarily used for testing and debugging.
func (lpl *linePositionLedger) reconcile() bool { //nolint:unused // Used in tests
	// For now, we rely on the tracker's validateLinePositions
	invalid := lpl.tracker.validateLinePositions()
	consistent := len(invalid) == 0

	if !consistent {
		lpl.recordValidation(len(invalid))
	}

	return consistent
}
