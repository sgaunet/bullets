// Package bullets provides a colorful terminal logger with bullet-style output.
// Inspired by goreleaser's logging output.
package bullets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
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
	fields            map[string]interface{}
	useSpecialBullets bool
	customBullets     map[Level]string
}

// New creates a new logger that writes to the given writer.
func New(w io.Writer) *Logger {
	return &Logger{
		writer:            w,
		level:             InfoLevel,
		padding:           0,
		fields:            make(map[string]interface{}),
		useSpecialBullets: false,
		customBullets:     make(map[Level]string),
	}
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
	for level, bullet := range bullets {
		l.customBullets[level] = bullet
	}
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
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		writer:            l.writer,
		level:             l.level,
		padding:           l.padding,
		fields:            make(map[string]interface{}),
		useSpecialBullets: l.useSpecialBullets,
		customBullets:     make(map[Level]string),
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value

	for k, v := range l.customBullets {
		newLogger.customBullets[k] = v
	}

	return newLogger
}

// WithFields returns a new logger with the given fields added.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		writer:            l.writer,
		level:             l.level,
		padding:           l.padding,
		fields:            make(map[string]interface{}),
		useSpecialBullets: l.useSpecialBullets,
		customBullets:     make(map[Level]string),
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	for k, v := range l.customBullets {
		newLogger.customBullets[k] = v
	}

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
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...))
}

// Info logs an info message.
func (l *Logger) Info(msg string) {
	l.log(InfoLevel, msg)
}

// Infof logs a formatted info message.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string) {
	l.log(WarnLevel, msg)
}

// Warnf logs a formatted warning message.
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...))
}

// Error logs an error message.
func (l *Logger) Error(msg string) {
	l.log(ErrorLevel, msg)
}

// Errorf logs a formatted error message.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string) {
	l.log(FatalLevel, msg)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits.
func (l *Logger) Fatalf(format string, args ...interface{}) {
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
func (l *Logger) Successf(format string, args ...interface{}) {
	l.Success(fmt.Sprintf(format, args...))
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

// Spinner creates and starts a spinner with default Braille animation.
// The spinner uses a smooth Braille dot pattern.
// Call Stop(), Success(), Error(), or Replace() on the returned spinner to stop it.
func (l *Logger) Spinner(msg string) *Spinner {
	l.mu.Lock()
	color := cyan // Default info level color
	l.mu.Unlock()

	// Default Braille spinner
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return newSpinner(l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerDots creates a spinner with rotating Braille dots pattern.
// This is the default spinner style with smooth dot transitions.
func (l *Logger) SpinnerDots(msg string) *Spinner {
	l.mu.Lock()
	color := cyan
	l.mu.Unlock()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return newSpinner(l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerCircle creates a spinner with growing/shrinking circle pattern.
// Creates a glassy circular rotation effect.
func (l *Logger) SpinnerCircle(msg string) *Spinner {
	l.mu.Lock()
	color := cyan
	l.mu.Unlock()

	frames := []string{"◐", "◓", "◑", "◒"}
	return newSpinner(l, msg, frames, color, spinnerIntervalSlow)
}

// SpinnerBounce creates a spinner with bouncing dot pattern.
// Creates a smooth bouncing animation effect.
func (l *Logger) SpinnerBounce(msg string) *Spinner {
	l.mu.Lock()
	color := cyan
	l.mu.Unlock()

	frames := []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	return newSpinner(l, msg, frames, color, spinnerIntervalDefault)
}

// SpinnerWithFrames creates and starts a spinner with custom animation frames.
// Frames will cycle through the provided slice of strings.
// Call Stop(), Success(), Error(), or Replace() on the returned spinner to stop it.
func (l *Logger) SpinnerWithFrames(msg string, frames []string) *Spinner {
	l.mu.Lock()
	color := cyan // Default info level color
	l.mu.Unlock()

	return newSpinner(l, msg, frames, color, spinnerIntervalSlow)
}

// log is the internal logging method.
func (l *Logger) log(level Level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
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
