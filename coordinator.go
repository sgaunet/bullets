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
type spinnerState struct {
	lineNumber   int
	currentFrame int
	message      string
	frames       []string
	color        string
	padding      int
	stopped      bool
	createdAt    time.Time
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
	mu          sync.Mutex
	spinners    map[*Spinner]*spinnerState
	updateCh    chan spinnerUpdate
	doneCh      chan struct{}
	isTTY       bool
	writer      io.Writer
	writeMu     *sync.Mutex // Shared write mutex from Logger
	nextLine    int         // Deprecated: use lineTracker instead
	running     bool
	startOnce   sync.Once
	lineTracker *lineTracker           // New line tracking system
	ledger      *linePositionLedger    // Audit trail for line operations
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

// register adds a new spinner to the coordinator and returns its line number.
func (c *SpinnerCoordinator) register(s *Spinner) int {
	c.start() // Ensure coordinator is running

	// Allocate line using the line tracker
	lineNum := c.lineTracker.allocateLine(s)
	c.ledger.recordAllocation(s, lineNum)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.spinners[s] = &spinnerState{
		lineNumber:   lineNum,
		currentFrame: 0,
		message:      s.msg,
		frames:       s.frames,
		color:        s.color,
		padding:      s.padding,
		stopped:      false,
		createdAt:    time.Now(),
	}

	return lineNum
}

// unregister removes a spinner from the coordinator.
func (c *SpinnerCoordinator) unregister(s *Spinner) {
	c.mu.Lock()
	state, exists := c.spinners[s]
	if exists {
		state.stopped = true
		lineNum := state.lineNumber
		delete(c.spinners, s)
		c.mu.Unlock()

		// Deallocate line using the line tracker
		// This marks the line as reserved (TTY mode) or available (non-TTY mode)
		c.lineTracker.deallocateLine(s)
		c.ledger.recordDeallocation(s, lineNum)
	} else {
		c.mu.Unlock()
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
	// Validate line positions before rendering
	invalid := c.lineTracker.validateLinePositions()
	if len(invalid) > 0 {
		c.ledger.recordValidation(len(invalid))
		// Log validation issues but continue rendering
		// In production, invalid spinners would be corrected here
	}

	c.mu.Lock()

	// Collect rendering data while holding lock
	type renderData struct {
		lineNumber   int
		padding      int
		frame        string
		color        string
		message      string
	}
	toRender := make([]renderData, 0, len(c.spinners))

	for _, state := range c.spinners {
		if !state.stopped {
			state.currentFrame = (state.currentFrame + 1) % len(state.frames)
			toRender = append(toRender, renderData{
				lineNumber: state.lineNumber,
				padding:    state.padding,
				frame:      state.frames[state.currentFrame],
				color:      state.color,
				message:    state.message,
			})
		}
	}
	c.mu.Unlock()

	// Render all spinners without holding the coordinator lock
	for _, data := range toRender {
		c.renderSpinnerFrame(data.lineNumber, data.padding, data.frame, data.color, data.message)
	}
}

// renderSpinnerFrame renders a single spinner frame (helper for updateAnimations).
func (c *SpinnerCoordinator) renderSpinnerFrame(lineNumber, padding int, frame, color, message string) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	indent := strings.Repeat("  ", padding)
	bullet := colorize(color, frame)
	content := fmt.Sprintf("%s%s %s", indent, bullet, message)

	// Calculate how many lines to move up
	linesToMove := lineNumber + 1

	if linesToMove > 0 {
		fmt.Fprintf(c.writer, ansiMoveUp, linesToMove)
	}

	// Clear line, move to column 0, write content
	fmt.Fprintf(c.writer, "%s%s%s", ansiClearLine, ansiMoveToCol, content)

	if linesToMove > 0 {
		fmt.Fprintf(c.writer, ansiMoveDown, linesToMove)
	}
}


// renderCompletion renders the final completion message for a spinner.
func (c *SpinnerCoordinator) renderCompletion(_ *Spinner, state *spinnerState, message, color, bullet string) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	indent := strings.Repeat("  ", state.padding)
	formatted := fmt.Sprintf("%s %s", colorize(color, bullet), message)

	if c.isTTY {
		// TTY mode: update the spinner's line with final message
		linesToMove := state.lineNumber + 1

		if linesToMove > 0 {
			fmt.Fprintf(c.writer, ansiMoveUp, linesToMove)
		}

		// Clear line, write completion message (no newline to avoid buffer modification)
		fmt.Fprintf(c.writer, "%s%s%s%s", ansiClearLine, ansiMoveToCol, indent, formatted)

		if linesToMove > 0 {
			// Move cursor back down to original position
			fmt.Fprintf(c.writer, ansiMoveDown, linesToMove)
		}
	} else {
		// Non-TTY mode: just print the completion message as a new line
		fmt.Fprintf(c.writer, "%s%s\n", indent, formatted)
	}
}
