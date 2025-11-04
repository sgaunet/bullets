package bullets

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
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
// It provides thread-safe spinner management and serialized output operations.
type SpinnerCoordinator struct {
	mu        sync.Mutex
	spinners  map[*Spinner]*spinnerState
	updateCh  chan spinnerUpdate
	doneCh    chan struct{}
	isTTY     bool
	writer    io.Writer
	writeMu   *sync.Mutex // Shared write mutex from Logger
	nextLine  int
	running   bool
	startOnce sync.Once
}

// newSpinnerCoordinator creates a new spinner coordinator.
func newSpinnerCoordinator(writer io.Writer, writeMu *sync.Mutex, isTTY bool) *SpinnerCoordinator {
	return &SpinnerCoordinator{
		spinners: make(map[*Spinner]*spinnerState),
		updateCh: make(chan spinnerUpdate, 100), // Buffered channel to prevent blocking
		doneCh:   make(chan struct{}),
		writer:   writer,
		writeMu:  writeMu,
		isTTY:    isTTY,
		nextLine: 0,
		running:  false,
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

// stop gracefully shuts down the coordinator.
func (c *SpinnerCoordinator) stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	close(c.doneCh)
}

// register adds a new spinner to the coordinator and returns its line number.
func (c *SpinnerCoordinator) register(s *Spinner) int {
	c.start() // Ensure coordinator is running

	c.mu.Lock()
	defer c.mu.Unlock()

	lineNum := c.nextLine
	c.nextLine++ // Always allocate line numbers for tracking

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
	defer c.mu.Unlock()

	if state, exists := c.spinners[s]; exists {
		state.stopped = true
		delete(c.spinners, s)

		// Always recalculate line numbers to maintain correct ordering
		c.recalculateLineNumbers()
	}
}

// recalculateLineNumbers reassigns line numbers after a spinner is removed.
// Must be called with c.mu locked.
func (c *SpinnerCoordinator) recalculateLineNumbers() {
	// Build sorted list of spinners by creation time
	type spinnerWithState struct {
		spinner *Spinner
		state   *spinnerState
	}

	spinnerList := make([]spinnerWithState, 0, len(c.spinners))
	for spinner, state := range c.spinners {
		spinnerList = append(spinnerList, spinnerWithState{spinner, state})
	}

	// Sort by creation time (earlier created = lower line number)
	for i := 0; i < len(spinnerList); i++ {
		for j := i + 1; j < len(spinnerList); j++ {
			if spinnerList[i].state.createdAt.After(spinnerList[j].state.createdAt) {
				spinnerList[i], spinnerList[j] = spinnerList[j], spinnerList[i]
			}
		}
	}

	// Reassign line numbers
	for i, item := range spinnerList {
		item.state.lineNumber = i
		// Lock spinner to safely update lineNumber (prevents race with newSpinner)
		item.spinner.mu.Lock()
		item.spinner.lineNumber = i
		item.spinner.mu.Unlock()
	}

	c.nextLine = len(spinnerList)
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
	ticker := time.NewTicker(80 * time.Millisecond) // Default animation interval
	defer ticker.Stop()

	for {
		select {
		case <-c.doneCh:
			return

		case update := <-c.updateCh:
			c.handleUpdate(update)

		case <-ticker.C:
			// Periodic tick for animations in TTY mode
			if c.isTTY {
				c.updateAnimations()
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
func (c *SpinnerCoordinator) renderCompletion(spinner *Spinner, state *spinnerState, message, color, bullet string) {
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

		// Clear line, write completion message
		fmt.Fprintf(c.writer, "%s%s%s%s\n", ansiClearLine, ansiMoveToCol, indent, formatted)

		if linesToMove > 1 {
			// Move cursor back down (adjusted for the newline we just printed)
			fmt.Fprintf(c.writer, ansiMoveDown, linesToMove-1)
		}
	} else {
		// Non-TTY mode: just print the completion message as a new line
		fmt.Fprintf(c.writer, "%s%s\n", indent, formatted)
	}
}
