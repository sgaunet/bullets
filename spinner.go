package bullets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Spinner represents an animated spinner that can be stopped and replaced.
type Spinner struct {
	logger     *Logger
	writer     io.Writer
	frames     []string
	color      string
	msg        string
	interval   time.Duration
	padding    int
	stopCh     chan bool
	doneCh     chan bool
	mu         sync.Mutex
	stopped    bool
	lineNumber int  // Line position (0 = bottom, 1 = second from bottom, etc.)
	isTTY      bool // Whether output is a TTY
}

// newSpinner creates a new spinner with the given parameters.
func newSpinner(logger *Logger, msg string, frames []string, _ string, interval time.Duration) *Spinner {
	color := cyan // Always use cyan for consistency
	if len(frames) == 0 {
		frames = []string{bulletInfo, "○"} // Default: full and empty circle
	}

	// Detect TTY capability
	isTTY := false
	if os.Getenv("BULLETS_FORCE_TTY") == "1" {
		isTTY = true
	} else if f, ok := logger.writer.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	s := &Spinner{
		logger:   logger,
		writer:   logger.writer,
		frames:   frames,
		color:    color,
		msg:      msg,
		interval: interval,
		padding:  logger.padding,
		stopCh:   make(chan bool),
		doneCh:   make(chan bool),
		stopped:  false,
		isTTY:    isTTY,
	}

	// Register spinner and get line number
	// Use mutex to prevent race with recalculateLineNumbers
	lineNum := logger.registerSpinner(s)
	s.mu.Lock()
	s.lineNumber = lineNum
	s.mu.Unlock()

	if isTTY {
		// TTY mode: allocate a line for this spinner and start animation
		logger.writeMu.Lock()
		fmt.Fprintln(logger.writer)
		logger.writeMu.Unlock()
		go s.animate()
	} else {
		// Non-TTY mode: print static message immediately (no animation)
		logger.writeMu.Lock()
		indent := strings.Repeat("  ", s.padding)
		bullet := colorize(color, s.frames[0])
		fmt.Fprintf(logger.writer, "%s%s %s\n", indent, bullet, msg)
		logger.writeMu.Unlock()

		// Start minimal goroutine to handle stop signal
		go s.animate()
	}

	return s
}

// stopAnimation stops the spinner animation goroutine without unregistering.
func (s *Spinner) stopAnimation() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}

	s.stopped = true
	close(s.stopCh)
	s.mu.Unlock()

	<-s.doneCh // Wait for animation to finish
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.stopAnimation()

	// Unregister spinner from logger (without holding spinner lock to avoid deadlock)
	s.logger.unregisterSpinner(s)
}

// Success stops the spinner and replaces it with a success message.
func (s *Spinner) Success(msg string) {
	// Stop animation but don't unregister yet
	s.stopAnimation()

	s.logger.mu.Lock()
	// Determine bullet symbol for success
	var bullet string
	if custom, ok := s.logger.customBullets[InfoLevel]; ok {
		bullet = custom
	} else if s.logger.useSpecialBullets {
		bullet = bulletSuccess
	} else {
		bullet = bulletInfo // Default circle
	}
	s.logger.mu.Unlock()

	// Use coordinator's TTY detection for consistency
	if s.logger.coordinator.isTTY {
		// TTY mode: Send completion to coordinator (while still registered)
		// Don't unregister to maintain stable line positions for other spinners
		doneCh := make(chan struct{})
		s.logger.coordinator.sendUpdate(spinnerUpdate{
			spinner:      s,
			updateType:   updateComplete,
			finalMessage: msg,
			finalColor:   green,
			finalBullet:  bullet,
			doneCh:       doneCh,
		})
		// Wait for rendering to complete before returning
		<-doneCh
	} else {
		// Non-TTY mode: Print completion message as new line, then unregister
		indent := strings.Repeat("  ", s.padding)
		formatted := fmt.Sprintf("%s %s", colorize(green, bullet), msg)

		s.logger.writeMu.Lock()
		fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
		s.logger.writeMu.Unlock()

		// Unregister in non-TTY mode since we don't need to maintain line positions
		s.logger.unregisterSpinner(s)
	}
}

// Error stops the spinner and replaces it with an error message.
func (s *Spinner) Error(msg string) {
	// Stop animation but don't unregister yet
	s.stopAnimation()

	s.logger.mu.Lock()
	// Determine bullet symbol for error
	var bullet string
	if custom, ok := s.logger.customBullets[ErrorLevel]; ok {
		bullet = custom
	} else if s.logger.useSpecialBullets {
		bullet = bulletError
	} else {
		bullet = bulletInfo // Default circle
	}
	s.logger.mu.Unlock()

	// Use coordinator's TTY detection for consistency
	if s.logger.coordinator.isTTY {
		// TTY mode: Send completion to coordinator (while still registered)
		// Don't unregister to maintain stable line positions for other spinners
		doneCh := make(chan struct{})
		s.logger.coordinator.sendUpdate(spinnerUpdate{
			spinner:      s,
			updateType:   updateComplete,
			finalMessage: msg,
			finalColor:   red,
			finalBullet:  bullet,
			doneCh:       doneCh,
		})
		// Wait for rendering to complete before returning
		<-doneCh
	} else {
		// Non-TTY mode: Print completion message as new line, then unregister
		indent := strings.Repeat("  ", s.padding)
		formatted := fmt.Sprintf("%s %s", colorize(red, bullet), msg)

		s.logger.writeMu.Lock()
		fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
		s.logger.writeMu.Unlock()

		// Unregister in non-TTY mode since we don't need to maintain line positions
		s.logger.unregisterSpinner(s)
	}
}

// Fail is an alias for Error.
func (s *Spinner) Fail(msg string) {
	s.Error(msg)
}

// Replace stops the spinner and replaces it with a custom message at info level.
func (s *Spinner) Replace(msg string) {
	// Stop animation but don't unregister yet
	s.stopAnimation()

	s.logger.mu.Lock()
	// Determine bullet symbol for info
	var bullet string
	if custom, ok := s.logger.customBullets[InfoLevel]; ok {
		bullet = custom
	} else {
		bullet = bulletInfo
	}
	s.logger.mu.Unlock()

	// Use coordinator's TTY detection for consistency
	if s.logger.coordinator.isTTY {
		// TTY mode: Send completion to coordinator (while still registered)
		// Don't unregister to maintain stable line positions for other spinners
		doneCh := make(chan struct{})
		s.logger.coordinator.sendUpdate(spinnerUpdate{
			spinner:      s,
			updateType:   updateComplete,
			finalMessage: msg,
			finalColor:   cyan,
			finalBullet:  bullet,
			doneCh:       doneCh,
		})
		// Wait for rendering to complete before returning
		<-doneCh
	} else {
		// Non-TTY mode: Print completion message as new line, then unregister
		indent := strings.Repeat("  ", s.padding)
		formatted := fmt.Sprintf("%s %s", colorize(cyan, bullet), msg)

		s.logger.writeMu.Lock()
		fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
		s.logger.writeMu.Unlock()

		// Unregister in non-TTY mode since we don't need to maintain line positions
		s.logger.unregisterSpinner(s)
	}
}

// animate runs the spinner animation in a goroutine.
// In TTY mode, coordinator handles rendering. In non-TTY mode, no animation occurs.
func (s *Spinner) animate() {
	defer close(s.doneCh)

	// Both TTY and non-TTY: just wait for stop signal
	// TTY animation is handled by coordinator's central ticker
	// Non-TTY has no animation (already printed static message)
	<-s.stopCh
}

