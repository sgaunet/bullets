package bullets

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"golang.org/x/term"
)

// UpdatableLogger wraps a regular Logger and provides updatable bullet functionality
type UpdatableLogger struct {
	*Logger
	mu        sync.RWMutex
	writeMu   sync.Mutex  // Mutex for terminal write operations
	handles   []*BulletHandle
	lineCount int  // Track total lines written
	isTTY     bool // Whether output is a terminal
}

// BulletHandle represents a handle to an updatable bullet
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

// ANSI escape codes for cursor control
const (
	ansiSaveCursor    = "\033[s"
	ansiRestoreCursor = "\033[u"
	ansiClearLine     = "\033[2K"
	ansiMoveUp        = "\033[%dA"
	ansiMoveDown      = "\033[%dB"
	ansiMoveToCol     = "\033[0G"
)

// NewUpdatable creates a new updatable logger
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

// InfoHandle logs an info message and returns a handle for updates
func (ul *UpdatableLogger) InfoHandle(msg string) *BulletHandle {
	return ul.logHandle(InfoLevel, msg)
}

// DebugHandle logs a debug message and returns a handle for updates
func (ul *UpdatableLogger) DebugHandle(msg string) *BulletHandle {
	return ul.logHandle(DebugLevel, msg)
}

// WarnHandle logs a warning message and returns a handle for updates
func (ul *UpdatableLogger) WarnHandle(msg string) *BulletHandle {
	return ul.logHandle(WarnLevel, msg)
}

// ErrorHandle logs an error message and returns a handle for updates
func (ul *UpdatableLogger) ErrorHandle(msg string) *BulletHandle {
	return ul.logHandle(ErrorLevel, msg)
}

// logHandle creates a bullet and returns a handle to it
func (ul *UpdatableLogger) logHandle(level Level, msg string) *BulletHandle {
	ul.mu.Lock()
	defer ul.mu.Unlock()

	// Log the message normally
	ul.Logger.log(level, msg)

	// If not a TTY, return a handle that prints updates as new lines
	if !ul.isTTY {
		return &BulletHandle{
			logger:          ul,
			lineNum:         -1,
			level:           level,
			message:         msg,
			originalMessage: msg,
			padding:         ul.Logger.padding,
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
		padding:         ul.Logger.padding,
		fields:          make(map[string]interface{}),
	}

	// Copy fields from logger
	ul.Logger.mu.Lock()
	for k, v := range ul.Logger.fields {
		handle.fields[k] = v
	}
	ul.Logger.mu.Unlock()

	ul.handles = append(ul.handles, handle)
	ul.lineCount++

	return handle
}

// Update updates the bullet with a new message and level
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
		h.logger.Logger.log(level, msg)
	}
	return h
}

// UpdateMessage updates just the message text
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

// UpdateLevel updates just the level (and thus color/bullet)
func (h *BulletHandle) UpdateLevel(level Level) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.level = level

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// Success updates the bullet to show success
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
		h.logger.Logger.Success(msg)
	}
	return h
}

// Error updates the bullet to show an error
func (h *BulletHandle) Error(msg string) *BulletHandle {
	return h.Update(ErrorLevel, msg)
}

// Warning updates the bullet to show a warning
func (h *BulletHandle) Warning(msg string) *BulletHandle {
	return h.Update(WarnLevel, msg)
}

// WithField adds a field to this bullet
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

// WithFields adds multiple fields to this bullet
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

// redraw redraws the bullet at its original position
func (h *BulletHandle) redraw() {
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
		h.renderInPlace()

		// Move back down to the last line
		fmt.Fprintf(h.logger.writer, ansiMoveDown, linesToMoveUp)
		// Move to start of next line
		fmt.Fprint(h.logger.writer, "\r")
	} else {
		// Current line, just clear and redraw
		fmt.Fprint(h.logger.writer, "\r")
		fmt.Fprint(h.logger.writer, ansiClearLine)
		h.renderInPlace()
	}
}

// redrawSuccess redraws the bullet as a success message
func (h *BulletHandle) redrawSuccess() {
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

		// Render as success without newline
		h.renderSuccessInPlace()

		// Move back down to the last line
		fmt.Fprintf(h.logger.writer, ansiMoveDown, linesToMoveUp)
		// Move to start of next line
		fmt.Fprint(h.logger.writer, "\r")
	} else {
		// Current line, just clear and redraw
		fmt.Fprint(h.logger.writer, "\r")
		fmt.Fprint(h.logger.writer, ansiClearLine)
		h.renderSuccessInPlace()
	}
}

// render outputs the bullet without cursor manipulation (with newline)
func (h *BulletHandle) render() {
	h.renderInPlace()
	fmt.Fprint(h.logger.writer, "\n")
}

// renderInPlace outputs the bullet without cursor manipulation and without newline
func (h *BulletHandle) renderInPlace() {
	indent := strings.Repeat("  ", h.padding)

	// Get bullet style
	h.logger.Logger.mu.Lock()
	useSpecial := h.logger.Logger.useSpecialBullets
	customBullets := h.logger.Logger.customBullets
	h.logger.Logger.mu.Unlock()

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

// renderSuccess outputs the bullet as a success message (with newline)
func (h *BulletHandle) renderSuccess() {
	h.renderSuccessInPlace()
	fmt.Fprint(h.logger.writer, "\n")
}

// renderSuccessInPlace outputs the bullet as a success message without newline
func (h *BulletHandle) renderSuccessInPlace() {
	indent := strings.Repeat("  ", h.padding)

	h.logger.Logger.mu.Lock()
	useSpecial := h.logger.Logger.useSpecialBullets
	customBullets := h.logger.Logger.customBullets
	h.logger.Logger.mu.Unlock()

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

// BatchUpdate allows updating multiple handles at once
func BatchUpdate(handles []*BulletHandle, updates map[*BulletHandle]struct {
	Level   Level
	Message string
}) {
	for handle, update := range updates {
		handle.Update(update.Level, update.Message)
	}
}

// IncrementLineCount increments the line count (called by regular log methods)
func (ul *UpdatableLogger) IncrementLineCount() {
	ul.mu.Lock()
	defer ul.mu.Unlock()
	ul.lineCount++
}

// Override regular logging methods to track line count
func (ul *UpdatableLogger) Info(msg string) {
	ul.Logger.Info(msg)
	ul.IncrementLineCount()
}

func (ul *UpdatableLogger) Debug(msg string) {
	ul.Logger.Debug(msg)
	ul.IncrementLineCount()
}

func (ul *UpdatableLogger) Warn(msg string) {
	ul.Logger.Warn(msg)
	ul.IncrementLineCount()
}

func (ul *UpdatableLogger) Error(msg string) {
	ul.Logger.Error(msg)
	ul.IncrementLineCount()
}

func (ul *UpdatableLogger) Success(msg string) {
	ul.Logger.Success(msg)
	ul.IncrementLineCount()
}