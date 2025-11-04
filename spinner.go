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
//
// Spinners provide visual feedback for long-running operations. Multiple spinners
// can run concurrently with automatic coordination via SpinnerCoordinator.
//
// In TTY mode, spinners animate in-place using ANSI escape codes. In non-TTY mode,
// they display as static messages for compatibility with logs and CI/CD systems.
//
// Example usage:
//
//	logger := bullets.New(os.Stdout)
//	spinner := logger.SpinnerCircle("Processing data")
//	// ... do work ...
//	spinner.Success("Processing complete")
//
// Thread Safety:
//
// All spinner methods are thread-safe and can be called from multiple goroutines.
// The coordinator ensures proper serialization of terminal updates.
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

// Stop stops the spinner and clears the line without displaying a completion message.
//
// This method immediately halts the animation and unregisters the spinner from the
// coordinator. The spinner cannot be reused after calling Stop().
//
// Thread-safe: Can be called from any goroutine.
func (s *Spinner) Stop() {
	s.stopAnimation()

	// Unregister spinner from logger (without holding spinner lock to avoid deadlock)
	s.logger.unregisterSpinner(s)
}

// Success stops the spinner and replaces it with a success message.
//
// The completion message is displayed with a success bullet (green color).
// The bullet symbol respects custom bullets and special bullet settings.
//
// In TTY mode, the spinner line is overwritten with the final message.
// In non-TTY mode, a new line is printed with the completion message.
//
// Example:
//
//	spinner := logger.SpinnerCircle("Connecting")
//	// ... do work ...
//	spinner.Success("Connected successfully")
//
// Thread-safe: Can be called from any goroutine.
func (s *Spinner) Success(msg string) {
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

	s.completeSpinner(msg, green, bullet)
}

// Error stops the spinner and replaces it with an error message.
//
// The completion message is displayed with an error bullet (red color).
// The bullet symbol respects custom bullets and special bullet settings.
//
// In TTY mode, the spinner line is overwritten with the final message.
// In non-TTY mode, a new line is printed with the completion message.
//
// Example:
//
//	spinner := logger.SpinnerCircle("Connecting")
//	// ... do work ...
//	spinner.Error("Connection failed: timeout")
//
// Thread-safe: Can be called from any goroutine.
func (s *Spinner) Error(msg string) {
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

	s.completeSpinner(msg, red, bullet)
}

// Fail is an alias for Error.
//
// This method behaves identically to Error() and is provided for convenience.
// Thread-safe: Can be called from any goroutine.
func (s *Spinner) Fail(msg string) {
	s.Error(msg)
}

// Replace stops the spinner and replaces it with a custom message at info level.
//
// The completion message is displayed with an info bullet (cyan color).
// The bullet symbol respects custom bullets and special bullet settings.
//
// Use Replace when the operation completes but you want to show a custom message
// that doesn't imply success or failure.
//
// Example:
//
//	spinner := logger.SpinnerCircle("Processing")
//	// ... do work ...
//	spinner.Replace("Processed 1000 records in 5.2s")
//
// Thread-safe: Can be called from any goroutine.
func (s *Spinner) Replace(msg string) {
	s.logger.mu.Lock()
	// Determine bullet symbol for info
	var bullet string
	if custom, ok := s.logger.customBullets[InfoLevel]; ok {
		bullet = custom
	} else {
		bullet = bulletInfo
	}
	s.logger.mu.Unlock()

	s.completeSpinner(msg, cyan, bullet)
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

// completeSpinner is a helper method that handles spinner completion logic.
// It stops the animation and renders the completion message with the specified color and bullet.
func (s *Spinner) completeSpinner(msg, color, bullet string) {
	// Stop animation but don't unregister yet
	s.stopAnimation()

	// Use coordinator's TTY detection for consistency
	if s.logger.coordinator.isTTY {
		// TTY mode: Send completion to coordinator (while still registered)
		// Don't unregister to maintain stable line positions for other spinners
		doneCh := make(chan struct{})
		s.logger.coordinator.sendUpdate(spinnerUpdate{
			spinner:      s,
			updateType:   updateComplete,
			finalMessage: msg,
			finalColor:   color,
			finalBullet:  bullet,
			doneCh:       doneCh,
		})
		// Wait for rendering to complete before returning
		<-doneCh
	} else {
		// Non-TTY mode: Print completion message as new line, then unregister
		indent := strings.Repeat("  ", s.padding)
		formatted := fmt.Sprintf("%s %s", colorize(color, bullet), msg)

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

