package bullets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"golang.org/x/term"
)

// UpdatableLogger wraps a regular Logger and provides updatable bullet functionality.
type UpdatableLogger struct {
	*Logger

	mu        sync.RWMutex
	writeMu   sync.Mutex  // Mutex for terminal write operations
	handles   []*BulletHandle
	lineCount int  // Track total lines written
	isTTY     bool // Whether output is a terminal
}

// BulletHandle represents a handle to an updatable bullet.
type BulletHandle struct {
	logger         *UpdatableLogger
	lineNum        int     // Line number relative to start
	level          Level
	message        string
	originalMessage string  // Store original message for progress updates
	progressBar    string  // Store progress bar separately
	color          string
	bullet         string
	padding        int
	fields         map[string]interface{}
	mu             sync.Mutex
}

// NewUpdatable creates a new updatable logger.
func NewUpdatable(w io.Writer) *UpdatableLogger {
	// Check if output is a terminal
	isTTY := false

	// Allow forcing TTY mode via environment variable for testing
	if os.Getenv("BULLETS_FORCE_TTY") == "1" {
		isTTY = true
	} else if f, ok := w.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	return &UpdatableLogger{
		Logger:  New(w),
		handles: make([]*BulletHandle, 0),
		isTTY:   isTTY,
	}
}

// InfoHandle logs an info message and returns a handle for updates.
func (ul *UpdatableLogger) InfoHandle(msg string) *BulletHandle {
	return ul.logHandle(InfoLevel, msg)
}

// DebugHandle logs a debug message and returns a handle for updates.
func (ul *UpdatableLogger) DebugHandle(msg string) *BulletHandle {
	return ul.logHandle(DebugLevel, msg)
}

// WarnHandle logs a warning message and returns a handle for updates.
func (ul *UpdatableLogger) WarnHandle(msg string) *BulletHandle {
	return ul.logHandle(WarnLevel, msg)
}

// ErrorHandle logs an error message and returns a handle for updates.
func (ul *UpdatableLogger) ErrorHandle(msg string) *BulletHandle {
	return ul.logHandle(ErrorLevel, msg)
}

// Info logs an info message and increments line count.
func (ul *UpdatableLogger) Info(msg string) {
	ul.log(InfoLevel, msg)
	ul.mu.Lock()
	ul.lineCount++
	ul.mu.Unlock()
}

// Debug logs a debug message and increments line count.
func (ul *UpdatableLogger) Debug(msg string) {
	ul.log(DebugLevel, msg)
	ul.mu.Lock()
	ul.lineCount++
	ul.mu.Unlock()
}

// Warn logs a warning message and increments line count.
func (ul *UpdatableLogger) Warn(msg string) {
	ul.log(WarnLevel, msg)
	ul.mu.Lock()
	ul.lineCount++
	ul.mu.Unlock()
}

// Error logs an error message and increments line count.
func (ul *UpdatableLogger) Error(msg string) {
	ul.log(ErrorLevel, msg)
	ul.mu.Lock()
	ul.lineCount++
	ul.mu.Unlock()
}

// Success logs a success message and increments line count.
func (ul *UpdatableLogger) Success(msg string) {
	ul.mu.Lock()
	defer ul.mu.Unlock()

	if InfoLevel < ul.level {
		ul.lineCount++
		return
	}

	indent := strings.Repeat("  ", ul.padding)

	// Determine bullet symbol for success
	var bullet string
	if custom, ok := ul.customBullets[InfoLevel]; ok {
		bullet = custom
	} else if ul.useSpecialBullets {
		bullet = bulletSuccess
	} else {
		bullet = bulletInfo
	}

	formatted := fmt.Sprintf("%s %s", colorize(green, bullet), msg)
	fmt.Fprintf(ul.writer, "%s%s\n", indent, formatted)
	ul.lineCount++
}

// IncrementLineCount increments the line count (called by regular log methods).
func (ul *UpdatableLogger) IncrementLineCount() {
	ul.mu.Lock()
	defer ul.mu.Unlock()
	ul.lineCount++
}

// logHandle creates a bullet and returns a handle to it.
func (ul *UpdatableLogger) logHandle(level Level, msg string) *BulletHandle {
	ul.mu.Lock()
	defer ul.mu.Unlock()

	// Log the message normally
	ul.log(level, msg)

	// If not a TTY, return a handle that prints updates as new lines
	if !ul.isTTY {
		return &BulletHandle{
			logger:          ul,
			lineNum:         -1,
			level:           level,
			message:         msg,
			originalMessage: msg,
			padding:         ul.padding,
			fields:          make(map[string]interface{}),
		}
	}

	// Create and register handle
	handle := &BulletHandle{
		logger:          ul,
		lineNum:         ul.lineCount,
		level:           level,
		message:         msg,
		originalMessage: msg,
		padding:         ul.padding,
		fields:          make(map[string]interface{}),
	}

	// Copy fields from logger (already have the lock)
	for k, v := range ul.fields {
		handle.fields[k] = v
	}

	ul.handles = append(ul.handles, handle)
	ul.lineCount++

	return handle
}

// Update updates the bullet with a new message and level.
func (h *BulletHandle) Update(level Level, msg string) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.level = level
	h.message = msg
	h.progressBar = ""  // Clear progress bar on update

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	} else if h.lineNum == -1 && !h.logger.isTTY {
		// Fallback: print as new line when not in TTY mode
		h.logger.log(level, msg)
	}
	return h
}

// UpdateMessage updates just the message text.
func (h *BulletHandle) UpdateMessage(msg string) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.message = msg
	h.originalMessage = msg  // Also update original message

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// UpdateLevel updates just the level (and thus color/bullet).
func (h *BulletHandle) UpdateLevel(level Level) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.level = level

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// Success updates the bullet to show success.
func (h *BulletHandle) Success(msg string) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.message = msg
	h.progressBar = ""  // Clear progress bar on success

	if h.lineNum != -1 && h.logger.isTTY {
		// Success uses a special rendering
		h.redrawSuccess()
	} else if h.lineNum == -1 && !h.logger.isTTY {
		// Fallback: print as new success line when not in TTY mode
		h.logger.Success(msg)
	}
	return h
}

// Error updates the bullet to show an error.
func (h *BulletHandle) Error(msg string) *BulletHandle {
	return h.Update(ErrorLevel, msg)
}

// Warning updates the bullet to show a warning.
func (h *BulletHandle) Warning(msg string) *BulletHandle {
	return h.Update(WarnLevel, msg)
}

// WithField adds a field to this bullet.
func (h *BulletHandle) WithField(key string, value interface{}) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.fields == nil {
		h.fields = make(map[string]interface{})
	}
	h.fields[key] = value
	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// WithFields adds multiple fields to this bullet.
func (h *BulletHandle) WithFields(fields map[string]interface{}) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	for k, v := range fields {
		h.fields[k] = v
	}
	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// redraw redraws the bullet at its original position.
func (h *BulletHandle) redraw() {
	h.redrawWithRenderer(h.renderInPlace)
}

// redrawSuccess redraws the bullet as a success message.
func (h *BulletHandle) redrawSuccess() {
	h.redrawWithRenderer(h.renderSuccessInPlace)
}

// redrawWithRenderer performs the redraw operation with a custom renderer.
func (h *BulletHandle) redrawWithRenderer(renderer func()) {
	h.logger.mu.RLock()
	currentLine := h.logger.lineCount
	h.logger.mu.RUnlock()

	linesToMoveUp := currentLine - h.lineNum

	// Lock for terminal write operations
	h.logger.writeMu.Lock()
	defer h.logger.writeMu.Unlock()

	if linesToMoveUp > 0 {
		// Move up to the target line
		fmt.Fprintf(h.logger.writer, ansiMoveUp, linesToMoveUp)

		// Clear the line and move to start of line
		fmt.Fprint(h.logger.writer, ansiClearLine)
		fmt.Fprint(h.logger.writer, ansiMoveToCol)

		// Render the updated bullet without newline
		renderer()

		// Move back down to the last line
		fmt.Fprintf(h.logger.writer, ansiMoveDown, linesToMoveUp)
		// Move to start of next line
		fmt.Fprint(h.logger.writer, "\r")
	} else {
		// Current line, just clear and redraw
		fmt.Fprint(h.logger.writer, "\r")
		fmt.Fprint(h.logger.writer, ansiClearLine)
		renderer()
	}
}

// renderInPlace outputs the bullet without cursor manipulation and without newline.
func (h *BulletHandle) renderInPlace() {
	indent := strings.Repeat("  ", h.padding)

	// Get bullet style
	h.logger.mu.Lock()
	useSpecial := h.logger.useSpecialBullets
	customBullets := h.logger.customBullets
	h.logger.mu.Unlock()

	formatted := formatMessage(h.level, h.message, useSpecial, customBullets)

	// Add progress bar if present
	if h.progressBar != "" {
		formatted += " " + colorize(cyan, h.progressBar)
	}

	// Add fields if present
	if len(h.fields) > 0 {
		var parts []string
		for k, v := range h.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		formatted += colorize(dim, fmt.Sprintf(" (%s)", strings.Join(parts, ", ")))
	}

	fmt.Fprintf(h.logger.writer, "%s%s", indent, formatted)
}

// renderSuccessInPlace outputs the bullet as a success message without newline.
func (h *BulletHandle) renderSuccessInPlace() {
	indent := strings.Repeat("  ", h.padding)

	h.logger.mu.Lock()
	useSpecial := h.logger.useSpecialBullets
	customBullets := h.logger.customBullets
	h.logger.mu.Unlock()

	// Determine bullet symbol for success
	var bullet string
	if custom, ok := customBullets[InfoLevel]; ok {
		bullet = custom
	} else if useSpecial {
		bullet = bulletSuccess
	} else {
		bullet = bulletInfo
	}

	formatted := fmt.Sprintf("%s %s", colorize(green, bullet), h.message)

	// Add fields if present
	if len(h.fields) > 0 {
		var parts []string
		for k, v := range h.fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		formatted += colorize(dim, fmt.Sprintf(" (%s)", strings.Join(parts, ", ")))
	}

	fmt.Fprintf(h.logger.writer, "%s%s", indent, formatted)
}

// BatchUpdate allows updating multiple handles at once.
// Note: The handles parameter is currently unused but kept for API compatibility.
func BatchUpdate(_ []*BulletHandle, updates map[*BulletHandle]struct {
	Level   Level
	Message string
}) {
	for handle, update := range updates {
		handle.Update(update.Level, update.Message)
	}
}