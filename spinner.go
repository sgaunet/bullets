package bullets

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Spinner represents an animated spinner that can be stopped and replaced.
type Spinner struct {
	logger   *Logger
	writer   io.Writer
	frames   []string
	color    string
	msg      string
	interval time.Duration
	padding  int
	stopCh   chan bool
	doneCh   chan bool
	mu       sync.Mutex
	stopped  bool
}

// newSpinner creates a new spinner with the given parameters.
func newSpinner(logger *Logger, msg string, frames []string, color string, interval time.Duration) *Spinner {
	if len(frames) == 0 {
		frames = []string{bulletInfo, "○"} // Default: full and empty circle
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
	}

	go s.animate()

	return s
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
			fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", 100))
			return
		case <-ticker.C:
			bullet := colorize(s.color, s.frames[frameIdx])
			fmt.Fprintf(s.writer, "\r%s%s %s", indent, bullet, s.msg)
			frameIdx = (frameIdx + 1) % len(s.frames)
		}
	}
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
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
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
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
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
	} else if s.logger.useSpecialBullets {
		bullet = bulletInfo
	} else {
		bullet = bulletInfo // Default circle
	}

	formatted := fmt.Sprintf("%s %s", colorize(cyan, bullet), msg)
	fmt.Fprintf(s.writer, "%s%s\n", indent, formatted)
}
