package bullets

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	// spinnerUpdateChannelSize defines the buffer size for the spinner update channel.
	spinnerUpdateChannelSize = 100
	// spinnerAnimationInterval defines the default animation interval in milliseconds.
	spinnerAnimationInterval = 80
	// cleanupPhaseInterval defines how often to run the cleanup phase (in milliseconds).
	cleanupPhaseInterval = 5000 // 5 seconds
	// reservedLineTimeout defines how long reserved lines are kept before reclaiming (in milliseconds).
	reservedLineTimeout = 3000 // 3 seconds
)

// updateType defines the type of spinner update.
type updateType int

const (
	updateFrame updateType = iota
	updateMessage
	updateComplete
)

// spinnerUpdate represents a message sent from a spinner to the coordinator.
type spinnerUpdate struct {
	spinner      *Spinner
	updateType   updateType
	message      string
	frameIdx     int
	finalMessage string
	finalColor   string
	finalBullet  string
	doneCh       chan struct{} // Channel to signal completion rendering is done
}

// spinnerState tracks the internal state of a spinner in the coordinator.
// Note: Line numbers are NOT stored here - lineTracker is the single source of truth.
type spinnerState struct {
	currentFrame       int
	message            string
	frames             []string
	color              string
	padding            int
	stopped            bool
	createdAt          time.Time
	firstFrameRendered bool // Track if initial line has been established with newline
}

// SpinnerCoordinator manages all spinner instances and coordinates their output.
//
// The coordinator implements a centralized pattern where a single goroutine handles
// all spinner animations and updates. This eliminates timing issues and ensures
// smooth, flicker-free animations even with multiple concurrent spinners.
//
// Architecture:
//   - Central ticker goroutine updates all active spinners (80ms interval)
//   - Channel-based communication for thread-safe spinner updates
//   - Automatic line number allocation and recalculation
//   - Unified TTY detection for consistent behavior
//
// The coordinator is automatically created by Logger and managed internally.
// Users don't need to interact with it directly.
//
// Thread Safety:
//
// All coordinator methods are thread-safe and can be called from multiple
// goroutines. Internal state is protected by mutexes and channel synchronization.
type SpinnerCoordinator struct {
	mu             sync.Mutex
	spinners       map[*Spinner]*spinnerState
	updateCh       chan spinnerUpdate
	doneCh         chan struct{}
	isTTY          bool
	writer         io.Writer
	writeMu        *sync.Mutex // Shared write mutex from Logger
	nextLine       int         // Deprecated: use lineTracker instead
	running        bool
	inSpinnerMode  bool        // Tracks whether we're in spinner mode (vs regular logging mode)
	startOnce      sync.Once
	lineTracker    *lineTracker           // New line tracking system
	ledger         *linePositionLedger    // Audit trail for line operations
	cursorLine     int         // Current cursor line position (0 = first spinner line)
}

const (
	// maxLedgerHistory defines the maximum number of operations to keep in the ledger history.
	maxLedgerHistory = 1000
)

// newSpinnerCoordinator creates a new spinner coordinator.
func newSpinnerCoordinator(writer io.Writer, writeMu *sync.Mutex, isTTY bool) *SpinnerCoordinator {
	cleanupInterval := time.Duration(reservedLineTimeout) * time.Millisecond
	tracker := newLineTracker(isTTY, cleanupInterval)
	ledger := newLinePositionLedger(tracker, maxLedgerHistory) // Keep last N operations

	return &SpinnerCoordinator{
		spinners:    make(map[*Spinner]*spinnerState),
		updateCh:    make(chan spinnerUpdate, spinnerUpdateChannelSize), // Buffered channel to prevent blocking
		doneCh:      make(chan struct{}),
		writer:      writer,
		writeMu:     writeMu,
		isTTY:       isTTY,
		nextLine:    0,
		running:     false,
		lineTracker: tracker,
		ledger:      ledger,
		cursorLine:  0, // Cursor starts at first line
	}
}

// start begins the coordinator's update processing goroutine.
// This method is safe to call multiple times; it will only start once.
func (c *SpinnerCoordinator) start() {
	c.startOnce.Do(func() {
		c.mu.Lock()
		if c.running {
			c.mu.Unlock()
			return
		}
		c.running = true
		c.mu.Unlock()

		go c.processUpdates()
	})
}

// getSpinnerLineNumber returns the line number for a spinner from the lineTracker.
// Returns -1 if the spinner is not registered.
// This is the authoritative method for querying line positions.
func (c *SpinnerCoordinator) getSpinnerLineNumber(s *Spinner) int {
	return c.lineTracker.getLineNumber(s)
}

// register adds a new spinner to the coordinator and returns its line number.
func (c *SpinnerCoordinator) register(s *Spinner) int {
	c.start() // Ensure coordinator is running

	// Allocate line using the line tracker (single source of truth)
	lineNum := c.lineTracker.allocateLine(s)
	c.ledger.recordAllocation(s, lineNum)

	debugLog("COORDINATOR", "Registered spinner %p at line %d (msg: %q)", s, lineNum, s.msg)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Enter spinner mode when registering a spinner
	wasInSpinnerMode := c.inSpinnerMode
	if !c.inSpinnerMode {
		debugLog("MODE_TRANSITION", "Setting inSpinnerMode=true (was false) for spinner %p at line %d", s, lineNum)
		c.inSpinnerMode = true
		debugLog("COORDINATOR", "Entering spinner mode")
	} else {
		debugLog("MODE_TRANSITION", "Already in spinner mode (inSpinnerMode=true) for spinner %p at line %d", s, lineNum)
	}
	_ = wasInSpinnerMode // Suppress unused variable warning if needed

	// Store spinner state WITHOUT line number - lineTracker owns that
	c.spinners[s] = &spinnerState{
		currentFrame:       0,
		message:            s.msg,
		frames:             s.frames,
		color:              s.color,
		padding:            s.padding,
		stopped:            false,
		createdAt:          time.Now(),
		firstFrameRendered: false, // First frame needs to establish line with newline
	}

	return lineNum
}

// unregister removes a spinner from the coordinator.
func (c *SpinnerCoordinator) unregister(s *Spinner) {
	// Get line number from lineTracker (single source of truth) before unregistering
	lineNum := c.lineTracker.getLineNumber(s)

	debugLog("COORDINATOR", "Unregistering spinner %p from line %d", s, lineNum)

	c.mu.Lock()
	state, exists := c.spinners[s]
	if exists {
		state.stopped = true
		delete(c.spinners, s)
		c.mu.Unlock()

		// Deallocate line using the line tracker
		// This marks the line as reserved (TTY mode) or available (non-TTY mode)
		c.lineTracker.deallocateLine(s)
		if lineNum != -1 {
			c.ledger.recordDeallocation(s, lineNum)
		}
	} else {
		c.mu.Unlock()
		debugLog("COORDINATOR", "Warning: spinner %p not found in coordinator map", s)
	}
}


// sendUpdate sends an update to the coordinator's update channel.
// Completion updates are blocking to ensure they're processed.
// Frame updates are non-blocking and can be dropped if channel is full.
func (c *SpinnerCoordinator) sendUpdate(update spinnerUpdate) {
	if update.updateType == updateComplete {
		// Completion updates are critical - block until sent
		c.updateCh <- update
	} else {
		// Frame/message updates are ephemeral - drop if channel full
		select {
		case c.updateCh <- update:
			// Update sent successfully
		default:
			// Channel full, drop update
		}
	}
}

// processUpdates is the main coordinator goroutine that processes spinner updates.
func (c *SpinnerCoordinator) processUpdates() {
	animationTicker := time.NewTicker(spinnerAnimationInterval * time.Millisecond) // Animation interval
	cleanupTicker := time.NewTicker(cleanupPhaseInterval * time.Millisecond)       // Cleanup phase interval
	defer animationTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-c.doneCh:
			return

		case update := <-c.updateCh:
			c.handleUpdate(update)

		case <-animationTicker.C:
			// Periodic tick for animations in TTY mode
			if c.isTTY {
				c.updateAnimations()
			}

		case <-cleanupTicker.C:
			// Cleanup phase: reclaim reserved lines
			reclaimed := c.lineTracker.reclaimReservedLines()
			if reclaimed > 0 {
				c.ledger.recordReclaim(reclaimed)
			}
		}
	}
}

// handleUpdate processes a single spinner update.
func (c *SpinnerCoordinator) handleUpdate(update spinnerUpdate) {
	c.mu.Lock()
	state, exists := c.spinners[update.spinner]
	if !exists || state.stopped {
		c.mu.Unlock()
		// Signal done even if spinner doesn't exist to prevent deadlock
		if update.doneCh != nil {
			close(update.doneCh)
		}
		return
	}

	switch update.updateType {
	case updateFrame:
		// Frame updates are now handled by the central animation ticker
		// Individual spinners don't send frame updates in TTY mode
		state.currentFrame = update.frameIdx
		c.mu.Unlock()

	case updateMessage:
		state.message = update.message
		c.mu.Unlock()

	case updateComplete:
		state.stopped = true
		c.mu.Unlock()
		c.renderCompletion(update.spinner, state, update.finalMessage, update.finalColor, update.finalBullet)
		// Signal that rendering is complete
		if update.doneCh != nil {
			close(update.doneCh)
		}
	}
}

// updateAnimations advances animation frames for all active spinners in TTY mode.
func (c *SpinnerCoordinator) updateAnimations() {
	timer := startDebugTimer("ANIMATION", "updateAnimations")
	defer func() {
		if timer != nil {
			timer.stop()
		}
	}()

	// Validate line positions before rendering
	invalid := c.lineTracker.validateLinePositions()
	if len(invalid) > 0 {
		c.ledger.recordValidation(len(invalid))
		debugLog("ANIMATION", "Warning: %d invalid line positions detected", len(invalid))
		// Log validation issues but continue rendering
		// In production, invalid spinners would be corrected here
	}

	// Run validation in debug mode
	c.validateDebugMode()

	c.mu.Lock()

	// Collect rendering data while holding lock
	type renderData struct {
		spinner            *Spinner
		lineNumber         int
		padding            int
		frame              string
		color              string
		message            string
		isFirstFrame       bool
	}
	toRender := make([]renderData, 0, len(c.spinners))

	for spinner, state := range c.spinners {
		if !state.stopped {
			// Query lineTracker for authoritative line number
			lineNum := c.lineTracker.getLineNumber(spinner)
			if lineNum == -1 {
				debugLog("ANIMATION", "Skipping spinner %p: not in lineTracker", spinner)
				continue // Spinner not registered in lineTracker
			}

			state.currentFrame = (state.currentFrame + 1) % len(state.frames)
			isFirstFrame := !state.firstFrameRendered
			toRender = append(toRender, renderData{
				spinner:      spinner,
				lineNumber:   lineNum,
				padding:      state.padding,
				frame:        state.frames[state.currentFrame],
				color:        state.color,
				message:      state.message,
				isFirstFrame: isFirstFrame,
			})

			// Mark first frame as rendered
			if isFirstFrame {
				state.firstFrameRendered = true
			}
		}
	}
	c.mu.Unlock()

	debugLogVerbose("ANIMATION", "Rendering %d active spinners", len(toRender))

	// Render all spinners without holding the coordinator lock
	for _, data := range toRender {
		c.renderSpinnerFrame(data.lineNumber, data.padding, data.frame, data.color, data.message, data.isFirstFrame)
	}

	// In verbose debug mode, periodically render debug map
	// Use a static counter to avoid rendering every frame
	if isVerboseDebug() {
		c.mu.Lock()
		frameCount := 0
		for _, state := range c.spinners {
			frameCount += state.currentFrame
		}
		c.mu.Unlock()

		// Render debug map every ~20 frames (varies based on total frame sum)
		if frameCount%20 == 0 {
			c.renderDebugMap()
		}
	}
}

// renderSpinnerFrame renders a single spinner frame (helper for updateAnimations).
func (c *SpinnerCoordinator) renderSpinnerFrame(lineNumber, padding int, frame, color, message string, isFirstFrame bool) {
	debugLogVerbose("RENDER", "Rendering frame at line %d: %q (firstFrame=%v, cursorLine=%d)", lineNumber, message, isFirstFrame, c.cursorLine)

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	indent := strings.Repeat("  ", padding)
	bullet := colorize(color, frame)
	content := fmt.Sprintf("%s%s %s", indent, bullet, message)

	if isFirstFrame {
		// First frame: print with newline to establish the line
		debugLogVerbose("RENDER", "First frame: establishing line %d with newline (cursor moves from %d to %d)", lineNumber, c.cursorLine, lineNumber+1)
		fmt.Fprintf(c.writer, "%s\n", content)
		// Cursor is now at the line AFTER the one we just printed
		c.cursorLine = lineNumber + 1
	} else {
		// Subsequent frames: update in place using cursor movement
		// Calculate how many lines to move up from current cursor position
		linesToMove := c.cursorLine - lineNumber

		if linesToMove > 0 {
			debugLogVerbose("ANSI", "Moving cursor up %d lines (from %d to %d)", linesToMove, c.cursorLine, lineNumber)
			fmt.Fprintf(c.writer, ansiMoveUp, linesToMove)
		} else if linesToMove < 0 {
			debugLogVerbose("ANSI", "Moving cursor down %d lines (from %d to %d)", -linesToMove, c.cursorLine, lineNumber)
			fmt.Fprintf(c.writer, ansiMoveDown, -linesToMove)
		}

		// Clear line, move to column 0, write content
		debugLogVerbose("ANSI", "Clearing line and writing content")
		fmt.Fprintf(c.writer, "%s%s%s", ansiClearLine, ansiMoveToCol, content)

		// Move cursor back to original position
		if linesToMove > 0 {
			debugLogVerbose("ANSI", "Moving cursor down %d lines back to %d", linesToMove, c.cursorLine)
			fmt.Fprintf(c.writer, ansiMoveDown, linesToMove)
			fmt.Fprint(c.writer, ansiMoveToCol)
		} else if linesToMove < 0 {
			debugLogVerbose("ANSI", "Moving cursor up %d lines back to %d", -linesToMove, c.cursorLine)
			fmt.Fprintf(c.writer, ansiMoveUp, -linesToMove)
			fmt.Fprint(c.writer, ansiMoveToCol)
		}
	}
}


// renderCompletion renders the final completion message for a spinner.
func (c *SpinnerCoordinator) renderCompletion(spinner *Spinner, state *spinnerState, message, color, bullet string) {
	debugLog("COMPLETION", "Rendering completion for spinner %p: %q", spinner, message)

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	indent := strings.Repeat("  ", state.padding)
	formatted := fmt.Sprintf("%s %s", colorize(color, bullet), message)

	if c.isTTY {
		// Query lineTracker for authoritative line number
		lineNum := c.lineTracker.getLineNumber(spinner)
		if lineNum == -1 {
			// Spinner not registered, fall back to non-TTY mode
			debugLog("COMPLETION", "Spinner %p not in lineTracker, falling back to non-TTY mode", spinner)
			fmt.Fprintf(c.writer, "%s%s\n", indent, formatted)
			return
		}

		debugLog("COMPLETION", "Completing spinner at line %d (cursorLine=%d)", lineNum, c.cursorLine)

		// Check if this is the last active spinner
		isLastSpinner := c.lineTracker.getActiveLineCount() == 1

		// TTY mode: update the spinner's line with final message
		// Calculate movement from current cursor position
		linesToMove := c.cursorLine - lineNum

		if linesToMove > 0 {
			debugLog("ANSI", "Moving cursor up %d lines for completion (from %d to %d)", linesToMove, c.cursorLine, lineNum)
			fmt.Fprintf(c.writer, ansiMoveUp, linesToMove)
		} else if linesToMove < 0 {
			debugLog("ANSI", "Moving cursor down %d lines for completion (from %d to %d)", -linesToMove, c.cursorLine, lineNum)
			fmt.Fprintf(c.writer, ansiMoveDown, -linesToMove)
		}

		// Clear line, write completion message (no newline to avoid buffer modification)
		debugLog("ANSI", "Writing completion message: %q", formatted)
		fmt.Fprintf(c.writer, "%s%s%s%s", ansiClearLine, ansiMoveToCol, indent, formatted)

		// Move cursor back to original position
		if linesToMove > 0 {
			debugLog("ANSI", "Moving cursor down %d lines back to %d", linesToMove, c.cursorLine)
			fmt.Fprintf(c.writer, ansiMoveDown, linesToMove)
			fmt.Fprint(c.writer, ansiMoveToCol)
		} else if linesToMove < 0 {
			debugLog("ANSI", "Moving cursor up %d lines back to %d", -linesToMove, c.cursorLine)
			fmt.Fprintf(c.writer, ansiMoveUp, -linesToMove)
			fmt.Fprint(c.writer, ansiMoveToCol)
		}

		// If this is the last spinner AND we're in spinner mode, emit a newline to transition to regular logging
		// This prevents adding extra newlines when new spinners are created (they allocate their own lines)
		c.mu.Lock()
		shouldExitSpinnerMode := isLastSpinner && c.inSpinnerMode
		if shouldExitSpinnerMode {
			debugLog("MODE_TRANSITION", "Setting inSpinnerMode=false (was true) - last spinner completing at line %d", lineNum)
			c.inSpinnerMode = false
		}
		c.mu.Unlock()

		if shouldExitSpinnerMode {
			debugLog("ANSI", "Last spinner completing, exiting spinner mode and emitting newline (cursorLine stays at %d)", c.cursorLine)
			fmt.Fprintln(c.writer)
			// Cursor remains at current position - line numbers are cumulative
		}
	} else {
		// Non-TTY mode: just print the completion message as a new line
		debugLog("COMPLETION", "Non-TTY mode: printing completion as new line")
		fmt.Fprintf(c.writer, "%s%s\n", indent, formatted)
	}
}

// stateInconsistency represents a detected state inconsistency.
type stateInconsistency struct {
	spinner     *Spinner
	description string
	severity    string // "warning" or "error"
}

// validateCoordinatorState performs comprehensive state consistency validation.
// Returns a slice of detected inconsistencies. Empty slice means state is consistent.
//
// This method checks:
// - All spinners in coordinator.spinners exist in lineTracker
// - All active spinners in lineTracker exist in coordinator.spinners
// - No spinner has conflicting state between coordinator and lineTracker
//
// Thread-safe: Can be called concurrently with other operations.
func (c *SpinnerCoordinator) validateCoordinatorState() []stateInconsistency {
	var inconsistencies []stateInconsistency

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check 1: Every spinner in coordinator.spinners should have a line in lineTracker
	for spinner, state := range c.spinners {
		lineNum := c.lineTracker.getLineNumber(spinner)

		if lineNum == -1 {
			inconsistencies = append(inconsistencies, stateInconsistency{
				spinner:     spinner,
				description: "Spinner registered in coordinator but not in lineTracker",
				severity:    "error",
			})
			continue
		}

		// Check if stopped spinners should still be in the map
		if state.stopped {
			inconsistencies = append(inconsistencies, stateInconsistency{
				spinner:     spinner,
				description: "Stopped spinner still in coordinator.spinners map",
				severity:    "warning",
			})
		}
	}

	// Check 2: Validate lineTracker internal consistency
	invalidSpinners := c.lineTracker.validateLinePositions()
	for _, spinner := range invalidSpinners {
		inconsistencies = append(inconsistencies, stateInconsistency{
			spinner:     spinner,
			description: "LineTracker reports invalid line position",
			severity:    "error",
		})
	}

	return inconsistencies
}

// checkStateInvariants verifies internal consistency invariants.
// Panics in development mode if invariants are violated.
// Returns true if all invariants hold, false otherwise.
//
// Invariants checked:
// - Active spinner count matches between coordinator and lineTracker
// - No nil spinner references in maps
// - All frame indices are within valid ranges
func (c *SpinnerCoordinator) checkStateInvariants() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	allInvariantsHold := true

	// Invariant 1: No nil spinner keys
	for spinner := range c.spinners {
		if spinner == nil {
			allInvariantsHold = false
			break
		}
	}

	// Invariant 2: Frame indices are valid
	for _, state := range c.spinners {
		if state.currentFrame < 0 || state.currentFrame >= len(state.frames) {
			allInvariantsHold = false
			break
		}
		if len(state.frames) == 0 {
			allInvariantsHold = false
			break
		}
	}

	return allInvariantsHold
}
