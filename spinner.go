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
	logger                  *Logger
	writer                  io.Writer
	frames                  []string
	color                   string
	msg                     string
	interval                time.Duration
	padding                 int
	stopCh                  chan bool
	doneCh                  chan bool
	mu                      sync.Mutex
	stopped                 bool
	lineNumber              int  // Line position (0 = bottom, 1 = second from bottom, etc.)
	isTTY                   bool // Whether output is a TTY
	disableAnimation        bool // Set to true when transitioning to multiple spinners
	disableAnimationMu      sync.RWMutex
	wasAnimating            bool // Track if this spinner ever animated
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
	s.lineNumber = logger.registerSpinner(s)

	if isTTY {
		// TTY mode: allocate a line for this spinner
		fmt.Fprintln(logger.writer)
	} else {
		// Non-TTY mode: handle multiple spinner detection
		logger.spinnerMu.Lock()
		spinnerCount := len(logger.activeSpinners)
		logger.spinnerMu.Unlock()

		if spinnerCount == 2 {
			// Transition from 1 to 2 spinners
			logger.spinnerMu.Lock()
			firstSpinner := logger.activeSpinners[0]
			logger.spinnerMu.Unlock()

			// Stop first spinner's animation completely
			firstSpinner.disableAnimationMu.Lock()
			firstSpinner.disableAnimation = true
			firstSpinner.disableAnimationMu.Unlock()

			// Wait longer to ensure animation loop has stopped printing
			time.Sleep(50 * time.Millisecond)

			// Now print both spinners with a single write lock
			logger.writeMu.Lock()
			// Move to a new line (don't try to clear the animated line)
			fmt.Fprintln(logger.writer)

			// Print first spinner's message
			indent1 := strings.Repeat("  ", firstSpinner.padding)
			bullet1 := colorize(firstSpinner.color, firstSpinner.frames[0])
			fmt.Fprintf(logger.writer, "%s%s %s\n", indent1, bullet1, firstSpinner.msg)

			// Print second spinner's message
			indent2 := strings.Repeat("  ", s.padding)
			bullet2 := colorize(color, s.frames[0])
			fmt.Fprintf(logger.writer, "%s%s %s\n", indent2, bullet2, msg)
			logger.writeMu.Unlock()

			// Disable this spinner's animation too
			s.disableAnimationMu.Lock()
			s.disableAnimation = true
			s.disableAnimationMu.Unlock()
		} else if spinnerCount > 2 {
			// More than 2 spinners: just print this one
			s.disableAnimationMu.Lock()
			s.disableAnimation = true
			s.disableAnimationMu.Unlock()

			logger.writeMu.Lock()
			indent := strings.Repeat("  ", s.padding)
			bullet := colorize(color, s.frames[0])
			fmt.Fprintf(logger.writer, "%s%s %s\n", indent, bullet, msg)
			logger.writeMu.Unlock()
		}
		// Single spinner: don't print yet, will animate with \r
	}

	go s.animate()

	return s
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return
	}

	s.stopped = true
	close(s.stopCh)
	<-s.doneCh // Wait for animation to finish

	// Unregister spinner from logger
	s.logger.unregisterSpinner(s)
}

// Success stops the spinner and replaces it with a success message.
func (s *Spinner) Success(msg string) {
	s.Stop()

	s.logger.mu.Lock()
	defer s.logger.mu.Unlock()

	indent := strings.Repeat("  ", s.padding)

	// Determine bullet symbol for success
	var bullet string
	if custom, ok := s.logger.customBullets[InfoLevel]; ok {
		bullet = custom
	} else if s.logger.useSpecialBullets {
		bullet = bulletSuccess
	} else {
		bullet = bulletInfo // Default circle
	}

	formatted := fmt.Sprintf("%s %s", colorize(green, bullet), msg)

	s.logger.writeMu.Lock()
	if !s.isTTY && s.wasAnimating {
		// Non-TTY mode: clear the animated line only if this spinner was animating
		const clearWidth = 100
		fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", clearWidth))
	}
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
	s.logger.writeMu.Unlock()
}

// Error stops the spinner and replaces it with an error message.
func (s *Spinner) Error(msg string) {
	s.Stop()

	s.logger.mu.Lock()
	defer s.logger.mu.Unlock()

	indent := strings.Repeat("  ", s.padding)

	// Determine bullet symbol for error
	var bullet string
	if custom, ok := s.logger.customBullets[ErrorLevel]; ok {
		bullet = custom
	} else if s.logger.useSpecialBullets {
		bullet = bulletError
	} else {
		bullet = bulletInfo // Default circle
	}

	formatted := fmt.Sprintf("%s %s", colorize(red, bullet), msg)

	s.logger.writeMu.Lock()
	if !s.isTTY && s.wasAnimating {
		// Non-TTY mode: clear the animated line only if this spinner was animating
		const clearWidth = 100
		fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", clearWidth))
	}
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
	s.logger.writeMu.Unlock()
}

// Fail is an alias for Error.
func (s *Spinner) Fail(msg string) {
	s.Error(msg)
}

// Replace stops the spinner and replaces it with a custom message at info level.
func (s *Spinner) Replace(msg string) {
	s.Stop()

	s.logger.mu.Lock()
	defer s.logger.mu.Unlock()

	indent := strings.Repeat("  ", s.padding)

	// Determine bullet symbol for info
	var bullet string
	if custom, ok := s.logger.customBullets[InfoLevel]; ok {
		bullet = custom
	} else {
		bullet = bulletInfo
	}

	formatted := fmt.Sprintf("%s %s", colorize(cyan, bullet), msg)

	s.logger.writeMu.Lock()
	if !s.isTTY && s.wasAnimating {
		// Non-TTY mode: clear the animated line only if this spinner was animating
		const clearWidth = 100
		fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", clearWidth))
	}
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
	s.logger.writeMu.Unlock()
}

// animate runs the spinner animation in a goroutine.
func (s *Spinner) animate() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer close(s.doneCh)

	frameIdx := 0
	indent := strings.Repeat("  ", s.padding)

	for {
		select {
		case <-s.stopCh:
			// Clear the spinner line
			if s.isTTY {
				s.clearLine()
			}
			return
		case <-ticker.C:
			bullet := colorize(s.color, s.frames[frameIdx])
			content := fmt.Sprintf("%s%s %s", indent, bullet, s.msg)

			if s.isTTY {
				// Multi-line mode: use ANSI cursor positioning
				s.updateLine(content)
			} else {
				// Non-TTY mode: check if animation is disabled
				s.disableAnimationMu.RLock()
				shouldAnimate := !s.disableAnimation
				s.disableAnimationMu.RUnlock()

				if shouldAnimate {
					// Single spinner: use carriage return for animation
					s.wasAnimating = true
					s.logger.writeMu.Lock()
					fmt.Fprintf(s.writer, "\r%s", content)
					s.logger.writeMu.Unlock()
				}
				// Multiple spinners or disabled: don't animate (already printed initial message)
			}

			frameIdx = (frameIdx + 1) % len(s.frames)
		}
	}
}

// updateLine updates the spinner's line using ANSI cursor positioning.
func (s *Spinner) updateLine(content string) {
	s.logger.writeMu.Lock()
	defer s.logger.writeMu.Unlock()

	// Calculate how many lines to move up
	// lineNumber 0 = most recent (bottom), need to move up by 1
	// lineNumber 1 = second from bottom, need to move up by 2, etc.
	linesToMove := s.lineNumber + 1

	if linesToMove > 0 {
		// Move cursor up to the spinner's line
		fmt.Fprintf(s.writer, ansiMoveUp, linesToMove)
	}

	// Clear line, move to column 0, write content
	fmt.Fprintf(s.writer, "%s%s%s", ansiClearLine, ansiMoveToCol, content)

	if linesToMove > 0 {
		// Move cursor back down to the bottom
		fmt.Fprintf(s.writer, ansiMoveDown, linesToMove)
	}
}

// clearLine clears the spinner's line using ANSI cursor positioning.
func (s *Spinner) clearLine() {
	s.logger.writeMu.Lock()
	defer s.logger.writeMu.Unlock()

	linesToMove := s.lineNumber + 1

	if linesToMove > 0 {
		fmt.Fprintf(s.writer, ansiMoveUp, linesToMove)
	}

	fmt.Fprintf(s.writer, "%s%s", ansiClearLine, ansiMoveToCol)

	if linesToMove > 0 {
		fmt.Fprintf(s.writer, ansiMoveDown, linesToMove)
	}
}
