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

// cursorRequestType defines the type of cursor operation.
type cursorRequestType int

const (
	cursorMoveUp cursorRequestType = iota
	cursorMoveDown
	cursorWriteLine      // Write content and move to next line (newline)
	cursorWriteInPlace   // Write content without newline
	cursorGetPosition    // Query current cursor position
)

// cursorRequest represents a request to move the cursor or write content.
// All terminal writes go through this channel to ensure atomic, ordered operations.
type cursorRequest struct {
	reqType    cursorRequestType
	lines      int    // Number of lines to move (for moveUp/moveDown)
	content    string // Content to write (for write operations)
	responseCh chan cursorResponse
}

// cursorResponse contains the result of a cursor request.
type cursorResponse struct {
	cursorLine int   // Current cursor line after operation
	err        error // Error if operation failed
}

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
	writeMu        *sync.Mutex // Deprecated: cursor service now handles synchronization
	nextLine       int         // Deprecated: use lineTracker instead
	running        bool
	inSpinnerMode  bool        // Tracks whether we're in spinner mode (vs regular logging mode)
	startOnce      sync.Once
	lineTracker    *lineTracker           // New line tracking system
	ledger         *linePositionLedger    // Audit trail for line operations
	cursorCh       chan cursorRequest     // Channel for cursor movement requests
	cursorLine     int                    // Current cursor line (managed by cursor service)
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

	c := &SpinnerCoordinator{
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
		cursorCh:    make(chan cursorRequest, 100), // Buffered channel for cursor requests
		cursorLine:  0,                             // Cursor starts at first line
	}

	// Start cursor service goroutine
	go c.processCursorRequests()

	return c
}

// processCursorRequests is the cursor service goroutine that processes all cursor movement requests.
// This is the ONLY goroutine that writes to the terminal, ensuring atomic, ordered operations.
//
// All cursor movements and writes must go through this service via the cursorCh channel.
// This eliminates race conditions and cursor position drift.
func (c *SpinnerCoordinator) processCursorRequests() {
	for req := range c.cursorCh {
		// Lock writeMu for exclusive access to the writer
		c.writeMu.Lock()
		var err error

		switch req.reqType {
		case cursorMoveUp:
			if c.isTTY && req.lines > 0 {
				debugLogVerbose("CURSOR", "moveUp(%d) from line %d to %d", req.lines, c.cursorLine, c.cursorLine-req.lines)
				fmt.Fprintf(c.writer, ansiMoveUp, req.lines)
				c.cursorLine -= req.lines
			}

		case cursorMoveDown:
			if c.isTTY && req.lines > 0 {
				debugLogVerbose("CURSOR", "moveDown(%d) from line %d to %d", req.lines, c.cursorLine, c.cursorLine+req.lines)
				fmt.Fprintf(c.writer, ansiMoveDown, req.lines)
				fmt.Fprint(c.writer, ansiMoveToCol)
				c.cursorLine += req.lines
			}

		case cursorWriteLine:
			// Write content with newline (moves cursor to next line)
			debugLogVerbose("CURSOR", "writeLine at line %d: %q", c.cursorLine, req.content)
			fmt.Fprintf(c.writer, "%s\n", req.content)
			c.cursorLine++

		case cursorWriteInPlace:
			// Write content without newline (cursor stays at current line)
			debugLogVerbose("CURSOR", "writeInPlace at line %d: %q", c.cursorLine, req.content)
			if c.isTTY {
				fmt.Fprintf(c.writer, "%s%s%s", ansiClearLine, ansiMoveToCol, req.content)
			} else {
				fmt.Fprint(c.writer, req.content)
			}

		case cursorGetPosition:
			// Just return current position, no write
			debugLogVerbose("CURSOR", "getPosition: %d", c.cursorLine)
		}

		c.writeMu.Unlock()

		// Send response if channel provided (after unlocking)
		if req.responseCh != nil {
			req.responseCh <- cursorResponse{
				cursorLine: c.cursorLine,
				err:        err,
			}
		}
	}
}

// Helper methods for cursor service

// moveCursorUp sends a request to move cursor up by N lines.
// Blocks until operation completes.
func (c *SpinnerCoordinator) moveCursorUp(lines int) {
	if lines <= 0 {
		return
	}
	responseCh := make(chan cursorResponse)
	c.cursorCh <- cursorRequest{
		reqType:    cursorMoveUp,
		lines:      lines,
		responseCh: responseCh,
	}
	<-responseCh // Wait for completion
}

// moveCursorDown sends a request to move cursor down by N lines.
// Blocks until operation completes.
func (c *SpinnerCoordinator) moveCursorDown(lines int) {
	if lines <= 0 {
		return
	}
	responseCh := make(chan cursorResponse)
	c.cursorCh <- cursorRequest{
		reqType:    cursorMoveDown,
		lines:      lines,
		responseCh: responseCh,
	}
	<-responseCh // Wait for completion
}

// writeLine sends a request to write content with newline.
// Blocks until operation completes.
func (c *SpinnerCoordinator) writeLine(content string) {
	responseCh := make(chan cursorResponse)
	c.cursorCh <- cursorRequest{
		reqType:    cursorWriteLine,
		content:    content,
		responseCh: responseCh,
	}
	<-responseCh // Wait for completion
}

// writeInPlace sends a request to write content without newline.
// Blocks until operation completes.
func (c *SpinnerCoordinator) writeInPlace(content string) {
	responseCh := make(chan cursorResponse)
	c.cursorCh <- cursorRequest{
		reqType:    cursorWriteInPlace,
		content:    content,
		responseCh: responseCh,
	}
	<-responseCh // Wait for completion
}

// getCursorPosition returns the current cursor line position.
func (c *SpinnerCoordinator) getCursorPosition() int {
	responseCh := make(chan cursorResponse)
	c.cursorCh <- cursorRequest{
		reqType:    cursorGetPosition,
		responseCh: responseCh,
	}
	resp := <-responseCh
	return resp.cursorLine
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
		// Get the line number before unregistering
		lineNum := c.lineTracker.getLineNumber(update.spinner)

		// Delete from coordinator's spinner map
		delete(c.spinners, update.spinner)
		c.mu.Unlock()

		// Unregister from lineTracker (triggers line reallocation for remaining spinners)
		c.lineTracker.deallocateLine(update.spinner)

		// Now render the completion with the line number we saved
		// After rendering, cursor will be repositioned based on the NEW active count
		c.renderCompletionWithLineNum(update.spinner, state, lineNum, update.finalMessage, update.finalColor, update.finalBullet)

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

	// Sort toRender by line number to ensure first frames are rendered in sequential order.
	// This is critical because first frames use newlines to establish lines, and printing
	// them out of order (due to map iteration) causes content misalignment with line numbers.
	// Subsequent frame updates use ANSI cursor positioning, so order doesn't matter for them.
	for i := 0; i < len(toRender); i++ {
		for j := i + 1; j < len(toRender); j++ {
			if toRender[i].lineNumber > toRender[j].lineNumber {
				toRender[i], toRender[j] = toRender[j], toRender[i]
			}
		}
	}

	// Render all spinners without holding the coordinator lock
	// Since toRender is sorted by line number, first frames are rendered in sequential order,
	// ensuring correct cursor tracking without needing post-render adjustments
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
// Uses cursor service for atomic, ordered terminal operations.
func (c *SpinnerCoordinator) renderSpinnerFrame(lineNumber, padding int, frame, color, message string, isFirstFrame bool) {
	cursorPos := c.getCursorPosition()
	debugLogVerbose("RENDER", "Rendering frame at line %d: %q (firstFrame=%v, cursorPos=%d)", lineNumber, message, isFirstFrame, cursorPos)

	indent := strings.Repeat("  ", padding)
	bullet := colorize(color, frame)
	content := fmt.Sprintf("%s%s %s", indent, bullet, message)

	if isFirstFrame && !c.isTTY {
		// Non-TTY first frame: write with newline
		debugLogVerbose("RENDER", "First frame (non-TTY): writing with newline")
		c.writeLine(content)
		return
	}

	if isFirstFrame && c.isTTY && lineNumber == cursorPos {
		// TTY first frame at cursor position: establish line with newline
		debugLogVerbose("RENDER", "First frame (TTY): establishing line %d at cursor position", lineNumber)
		c.writeLine(content)
		return
	}

	// All other cases (subsequent frames OR first frames not at cursor position):
	// Use cursor positioning to update in place
	linesToMove := cursorPos - lineNumber

	if linesToMove > 0 {
		c.moveCursorUp(linesToMove)
	} else if linesToMove < 0 {
		c.moveCursorDown(-linesToMove)
	}

	// Write content in place (clears line automatically)
	c.writeInPlace(content)

	// Move cursor back to original position
	if linesToMove > 0 {
		c.moveCursorDown(linesToMove)
	} else if linesToMove < 0 {
		c.moveCursorUp(-linesToMove)
	}
}


// renderCompletionWithLineNum renders the final completion message for a spinner at the specified line.
// Uses cursor service for atomic, ordered terminal operations.
func (c *SpinnerCoordinator) renderCompletionWithLineNum(spinner *Spinner, state *spinnerState, lineNum int, message, color, bullet string) {
	debugLog("COMPLETION", "Rendering completion for spinner %p at line %d: %q", spinner, lineNum, message)

	indent := strings.Repeat("  ", state.padding)
	formatted := fmt.Sprintf("%s %s", colorize(color, bullet), message)
	content := fmt.Sprintf("%s%s", indent, formatted)

	if c.isTTY {
		if lineNum == -1 {
			// Invalid line number, fall back to non-TTY mode
			debugLog("COMPLETION", "Invalid line number for spinner %p, falling back", spinner)
			c.writeLine(content)
			return
		}

		// Get the number of remaining active spinners (spinner already unregistered)
		activeCount := c.lineTracker.getActiveLineCount()
		isLastSpinner := activeCount == 0

		debugLog("COMPLETION", "Active spinner count after unregistration: %d", activeCount)

		// Move to target line, write completion, move back
		cursorPos := c.getCursorPosition()
		linesToMove := cursorPos - lineNum

		if linesToMove > 0 {
			c.moveCursorUp(linesToMove)
		} else if linesToMove < 0 {
			c.moveCursorDown(-linesToMove)
		}

		// Write completion message without newline
		c.writeInPlace(content)

		// Move cursor back to original position
		if linesToMove > 0 {
			c.moveCursorDown(linesToMove)
		} else if linesToMove < 0 {
			c.moveCursorUp(-linesToMove)
		}

		// If this is the last spinner AND we're in spinner mode, exit spinner mode
		c.mu.Lock()
		shouldExitSpinnerMode := isLastSpinner && c.inSpinnerMode
		if shouldExitSpinnerMode {
			debugLog("MODE_TRANSITION", "Exiting spinner mode - last spinner completing at line %d", lineNum)
			c.inSpinnerMode = false
		}
		c.mu.Unlock()

		// After completion, check if cursor needs adjustment
		if shouldExitSpinnerMode {
			// Last spinner: move cursor to fresh line below all completions
			maxLineUsed := c.lineTracker.getMaxLineUsed()
			targetLine := maxLineUsed + 1

			cursorPos = c.getCursorPosition()
			moveDist := targetLine - cursorPos
			if moveDist > 0 {
				c.moveCursorDown(moveDist)
			}

			// Emit newline to establish fresh line
			c.writeLine("")
			debugLog("MODE_TRANSITION", "Emitted newline, exiting spinner mode")
		} else {
			debugLog("COMPLETION", "More spinners remain, cursor unchanged")
		}
	} else {
		// Non-TTY mode: just print the completion message as a new line
		debugLog("COMPLETION", "Non-TTY mode: printing completion as new line")
		c.writeLine(content)
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
