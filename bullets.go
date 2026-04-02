// Package bullets provides a colorful terminal logger with bullet-style output.
// Inspired by goreleaser's logging output.
package bullets

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	// Default timing thresholds and intervals.
	spinnerIntervalDefault = 80 * time.Millisecond
	spinnerIntervalSlow    = 100 * time.Millisecond
	stepDurationThreshold  = 10 * time.Second
)

// Logger represents a logger with configurable level and output.
type Logger struct {
	mu                sync.Mutex
	writer            io.Writer
	level             Level
	padding           int
	fields            map[string]any
	useSpecialBullets bool
	customBullets     map[Level]string
	sanitizeInput     bool
	progressBarWidth  int
	writeMu           *sync.Mutex
	coordinator       *SpinnerCoordinator
}

// New creates a new logger that writes to the given writer.
func New(w io.Writer) *Logger {
	// Detect TTY capability for coordinator
	isTTY := false
	if os.Getenv("BULLETS_FORCE_TTY") == "1" {
		isTTY = true
	} else if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd())) //nolint:gosec // G115: fd fits int on all supported platforms
	}

	writeMu := &sync.Mutex{}
	l := &Logger{
		writer:            w,
		level:             InfoLevel,
		padding:           0,
		fields:            make(map[string]any),
		useSpecialBullets:  false,
		customBullets:      make(map[Level]string),
		progressBarWidth:   defaultProgressBarWidth,
		writeMu:            writeMu,
	}

	// Initialize coordinator with shared write mutex
	l.coordinator = newSpinnerCoordinator(w, writeMu, isTTY)

	return l
}

// Default returns a logger that writes to stderr.
func Default() *Logger {
	return New(os.Stderr)
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
func (l *Logger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetUseSpecialBullets enables or disables special bullet symbols (✓, ✗, ⚠).
// When disabled (default), all levels use the circle bullet (●) with level-specific colors.
func (l *Logger) SetUseSpecialBullets(use bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.useSpecialBullets = use
}

// SetSanitizeInput enables or disables ANSI escape sequence sanitization of user messages.
// When enabled, all messages passed to logging methods will have ANSI escape sequences
// stripped before storage and rendering. This protects against terminal injection attacks.
// Default is false (disabled) for backward compatibility.
func (l *Logger) SetSanitizeInput(sanitize bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sanitizeInput = sanitize
}

// SetProgressBarWidth sets the default width for progress bars.
// Width is clamped to the range [5, 100]. Default is 20.
func (l *Logger) SetProgressBarWidth(width int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.progressBarWidth = clampProgressBarWidth(width)
}

// SetBullet sets a custom bullet symbol for a specific log level.
// Custom bullets take priority over special bullets.
func (l *Logger) SetBullet(level Level, bullet string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.customBullets[level] = bullet
}

// SetBullets sets custom bullet symbols for multiple log levels.
// Custom bullets take priority over special bullets.
func (l *Logger) SetBullets(bullets map[Level]string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	maps.Copy(l.customBullets, bullets)
}

// IncreasePadding increases the indentation level.
func (l *Logger) IncreasePadding() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.padding++
}

// DecreasePadding decreases the indentation level.
func (l *Logger) DecreasePadding() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.padding > 0 {
		l.padding--
	}
}

// ResetPadding resets the indentation to zero.
func (l *Logger) ResetPadding() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.padding = 0
}

// WithField returns a new logger with the given field added.
func (l *Logger) WithField(key string, value any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		writer:            l.writer,
		level:             l.level,
		padding:           l.padding,
		fields:            make(map[string]any),
		useSpecialBullets: l.useSpecialBullets,
		sanitizeInput:     l.sanitizeInput,
		progressBarWidth:  l.progressBarWidth,
		customBullets:     make(map[Level]string),
	}

	maps.Copy(newLogger.fields, l.fields)
	newLogger.fields[key] = value

	maps.Copy(newLogger.customBullets, l.customBullets)

	return newLogger
}

// WithFields returns a new logger with the given fields added.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		writer:            l.writer,
		level:             l.level,
		padding:           l.padding,
		fields:            make(map[string]any),
		useSpecialBullets: l.useSpecialBullets,
		sanitizeInput:     l.sanitizeInput,
		progressBarWidth:  l.progressBarWidth,
		customBullets:     make(map[Level]string),
	}

	maps.Copy(newLogger.fields, l.fields)
	maps.Copy(newLogger.fields, fields)

	maps.Copy(newLogger.customBullets, l.customBullets)

	return newLogger
}

// WithError returns a new logger with an error field.
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string) {
	l.log(DebugLevel, msg)
}

// Debugf logs a formatted debug message.
func (l *Logger) Debugf(format string, args ...any) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info logs an info message.
func (l *Logger) Info(msg string) {
	l.log(InfoLevel, msg)
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...any) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.log(WarnLevel, msg)
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...any) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.log(ErrorLevel, msg)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...any) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string) {
	l.log(FatalLevel, msg)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits.
func (l *Logger) Fatalf(format string, args ...any) {
	l.log(FatalLevel, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Success logs a success message (using info level with success bullet).
func (l *Logger) Success(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if InfoLevel < l.level {
		return
	}

	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}

	indent := strings.Repeat("  ", l.padding)

	// Determine bullet symbol for success
	var bullet string
	if custom, ok := l.customBullets[InfoLevel]; ok {
		bullet = custom
	} else if l.useSpecialBullets {
		bullet = bulletSuccess
	} else {
		bullet = bulletInfo // Default circle
	}

	formatted := fmt.Sprintf("%s %s", colorize(green, bullet), msg)

	fmt.Fprintf(l.writer, "%s%s\n", indent, formatted)
}

// Successf logs a formatted success message.
func (l *Logger) Successf(format string, args ...any) {
	l.Success(fmt.Sprintf(format, args...))
}

// Ln prints a blank line without indentation.
// This method always outputs regardless of the log level.
func (l *Logger) Ln() {
	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintln(l.writer)
}

// Step logs a step message with timing information.
// It returns a function that should be called when the step is complete.
func (l *Logger) Step(msg string) func() {
	start := time.Now()
	l.Info(msg)
	l.IncreasePadding()

	return func() {
		l.DecreasePadding()
		duration := time.Since(start)
		if duration > stepDurationThreshold {
			l.WithField("duration", duration.Round(time.Millisecond)).Success("completed")
		} else {
			l.Success("completed")
		}
	}
}

// Spinner creates and starts a spinner with default Braille dots animation.
//
// The context controls the spinner's lifetime. When the context is cancelled or
// its deadline expires, the spinner automatically stops with an error message
// from ctx.Err().Error() (e.g., "context canceled" or "context deadline exceeded").
//
// Multiple spinners can run concurrently with automatic coordination. The spinner
// animates until stopped with Stop(), Success(), Error(), Replace(), or context cancellation.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	spinner := logger.Spinner(ctx, "Processing data")
//	// ... do work ...
//	spinner.Success("Processing complete") // or context cancels automatically
//
// In TTY mode, the spinner animates in-place. In non-TTY mode (logs, CI/CD),
// it displays as a static message.
//
// Thread-safe: Multiple spinners can be created from different goroutines.
func (l *Logger) Spinner(ctx context.Context, msg string) *Spinner {
	l.mu.Lock()
	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}
	color := cyan // Default info level color
	l.mu.Unlock()

	// Default Braille spinner
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return newSpinner(ctx, l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerDots creates a spinner with rotating Braille dots pattern.
//
// This is the default spinner style with smooth dot transitions. Identical to Spinner().
//
// Thread-safe: Multiple spinners can be created from different goroutines.
func (l *Logger) SpinnerDots(ctx context.Context, msg string) *Spinner {
	l.mu.Lock()
	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}
	color := cyan
	l.mu.Unlock()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return newSpinner(ctx, l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerCircle creates a spinner with growing/shrinking circle pattern.
//
// Creates a glassy circular rotation effect using quarter-circle characters.
// The animation is slower than the default Braille dots for a more relaxed feel.
//
// Thread-safe: Multiple spinners can be created from different goroutines.
func (l *Logger) SpinnerCircle(ctx context.Context, msg string) *Spinner {
	l.mu.Lock()
	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}
	color := cyan
	l.mu.Unlock()

	frames := []string{"◐", "◓", "◑", "◒"}
	return newSpinner(ctx, l, msg, frames, color, spinnerIntervalSlow)
}

// SpinnerBounce creates a spinner with bouncing dot pattern.
//
// Creates a smooth bouncing animation effect using Braille dots that appear
// to bounce vertically. The animation uses the default speed.
//
// Thread-safe: Multiple spinners can be created from different goroutines.
func (l *Logger) SpinnerBounce(ctx context.Context, msg string) *Spinner {
	l.mu.Lock()
	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}
	color := cyan
	l.mu.Unlock()

	frames := []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	return newSpinner(ctx, l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerWithFrames creates and starts a spinner with custom animation frames.
//
// Frames will cycle through the provided slice of strings. If frames is empty,
// defaults to the standard Braille dots pattern.
//
// Example:
//
//	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
//	spinner := logger.SpinnerWithFrames(context.Background(), "Compiling", frames)
//	// ... do work ...
//	spinner.Success("Compilation complete")
//
// Thread-safe: Multiple spinners can be created from different goroutines.
func (l *Logger) SpinnerWithFrames(ctx context.Context, msg string, frames []string) *Spinner {
	l.mu.Lock()
	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}
	color := cyan // Default info level color
	l.mu.Unlock()

	return newSpinner(ctx, l, msg, frames, color, spinnerIntervalSlow)
}

// log is the internal logging method.
func (l *Logger) log(level Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	if l.sanitizeInput {
		msg = sanitizeMsg(msg)
	}

	// Build padding
	indent := strings.Repeat("  ", l.padding)

	// Format message
	formatted := formatMessage(level, msg, l.useSpecialBullets, l.customBullets)

	// Add fields if present
	if len(l.fields) > 0 {
		var parts []string
		for k, v := range l.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		formatted += colorize(dim, fmt.Sprintf(" (%s)", strings.Join(parts, ", ")))
	}

	// Write to output
	fmt.Fprintf(l.writer, "%s%s\n", indent, formatted)
}

// registerSpinner adds a spinner to the active spinners list and returns its line number.
func (l *Logger) registerSpinner(s *Spinner) int {
	// Use coordinator for spinner registration and line allocation
	return l.coordinator.register(s)
}

// unregisterSpinner removes a spinner from the coordinator.
func (l *Logger) unregisterSpinner(s *Spinner) {
	// Use coordinator for spinner cleanup
	l.coordinator.unregister(s)
}
