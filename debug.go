package bullets

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// debugLevel represents the debug verbosity level.
type debugLevel int

const (
	// debugOff indicates debugging is disabled.
	debugOff debugLevel = 0
	// debugBasic enables basic debugging output.
	debugBasic debugLevel = 1
	// debugVerbose enables verbose debugging output with periodic state dumps.
	debugVerbose debugLevel = 2
)

var (
	// currentDebugLevel caches the debug level to avoid repeated env var lookups.
	currentDebugLevel debugLevel
	// debugLevelOnce ensures debug level is initialized only once.
	debugLevelOnce sync.Once
	// debugStartTime is the timestamp when debug mode was initialized.
	debugStartTime time.Time
	// debugInitialized tracks if debug level has been initialized.
	debugInitialized bool
	// debugMu protects debug initialization state and reads/writes of debug level.
	debugMu sync.RWMutex
	// debugTestMode disables Once caching for testing (always re-read env var).
	debugTestMode bool
)

// resetDebugLevel resets the debug level initialization state.
// This is intended for testing purposes only.
// In test mode, getDebugLevel() will always re-read the environment variable
// instead of using cached values, making it safe for tests to change BULLETS_DEBUG.
func resetDebugLevel() {
	debugMu.Lock()
	defer debugMu.Unlock()

	// Enable test mode to bypass Once caching
	debugTestMode = true
	debugInitialized = false
	currentDebugLevel = debugOff
	debugStartTime = time.Time{}
}

// getDebugLevel returns the current debug level based on BULLETS_DEBUG environment variable.
// BULLETS_DEBUG=0 or unset: debugging disabled
// BULLETS_DEBUG=1: basic debugging enabled
// BULLETS_DEBUG=2: verbose debugging with periodic state dumps
func getDebugLevel() debugLevel {
	// Check if we're in test mode (bypass Once caching for thread safety)
	debugMu.RLock()
	testMode := debugTestMode
	debugMu.RUnlock()

	if testMode {
		// In test mode, always re-read environment variable
		debugMu.Lock()
		defer debugMu.Unlock()

		if !debugInitialized {
			debugStartTime = time.Now()
			debugInitialized = true
		}

		envVal := os.Getenv("BULLETS_DEBUG")
		switch envVal {
		case "1":
			currentDebugLevel = debugBasic
		case "2":
			currentDebugLevel = debugVerbose
		default:
			currentDebugLevel = debugOff
		}
		return currentDebugLevel
	}

	// Normal mode: use Once pattern for caching
	debugLevelOnce.Do(func() {
		debugMu.Lock()
		defer debugMu.Unlock()

		debugStartTime = time.Now()
		envVal := os.Getenv("BULLETS_DEBUG")
		switch envVal {
		case "1":
			currentDebugLevel = debugBasic
		case "2":
			currentDebugLevel = debugVerbose
		default:
			currentDebugLevel = debugOff
		}
		debugInitialized = true
	})

	debugMu.RLock()
	defer debugMu.RUnlock()
	return currentDebugLevel
}

// isDebugEnabled returns true if any level of debugging is enabled.
func isDebugEnabled() bool {
	return getDebugLevel() > debugOff
}

// isVerboseDebug returns true if verbose debugging is enabled.
func isVerboseDebug() bool {
	return getDebugLevel() >= debugVerbose
}

// debugLog writes a debug message to stderr with timestamp and component information.
// Only outputs if debugging is enabled. Format: [HH:MM:SS.mmm] [component] message
func debugLog(component, format string, args ...interface{}) {
	if !isDebugEnabled() {
		return
	}

	debugMu.RLock()
	startTime := debugStartTime
	debugMu.RUnlock()

	elapsed := time.Since(startTime)
	timestamp := fmt.Sprintf("%02d:%02d:%02d.%03d",
		int(elapsed.Hours()),
		int(elapsed.Minutes())%60,
		int(elapsed.Seconds())%60,
		elapsed.Milliseconds()%1000)

	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[%s] [%s] %s\n", timestamp, component, message)
}

// debugLogVerbose writes a verbose debug message, only if verbose debugging is enabled.
func debugLogVerbose(component, format string, args ...interface{}) {
	if !isVerboseDebug() {
		return
	}
	debugLog(component, format, args...)
}

// formatANSISequence returns a human-readable description of an ANSI escape sequence.
func formatANSISequence(sequence string, args ...interface{}) string {
	formatted := fmt.Sprintf(sequence, args...)

	// Decode common ANSI sequences
	description := ""
	switch sequence {
	case ansiMoveUp:
		description = fmt.Sprintf("move cursor up %d lines", args[0])
	case ansiMoveDown:
		description = fmt.Sprintf("move cursor down %d lines", args[0])
	case ansiClearLine:
		description = "clear current line"
	case ansiMoveToCol:
		description = "move to column 0"
	default:
		description = "unknown sequence"
	}

	return fmt.Sprintf("%q (%s)", formatted, description)
}

// debugTimer helps measure operation timing for performance debugging.
type debugTimer struct {
	component string
	operation string
	start     time.Time
}

// startDebugTimer creates a timer for measuring operation duration.
// Returns nil if debugging is disabled (zero cost when disabled).
func startDebugTimer(component, operation string) *debugTimer {
	if !isDebugEnabled() {
		return nil
	}
	return &debugTimer{
		component: component,
		operation: operation,
		start:     time.Now(),
	}
}

// stop logs the elapsed time for the operation.
func (dt *debugTimer) stop() {
	if dt == nil {
		return
	}
	elapsed := time.Since(dt.start)
	debugLog(dt.component, "%s took %v", dt.operation, elapsed)
}

// debugState represents a snapshot of coordinator state for debugging.
type debugState struct {
	timestamp      time.Time
	activeSpinners int
	lineAllocations map[int]string // line number -> spinner description
}

// captureDebugState captures the current coordinator state for debugging.
func (c *SpinnerCoordinator) captureDebugState() *debugState {
	if !isDebugEnabled() {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	state := &debugState{
		timestamp:       time.Now(),
		activeSpinners:  len(c.spinners),
		lineAllocations: make(map[int]string),
	}

	for spinner := range c.spinners {
		lineNum := c.lineTracker.getLineNumber(spinner)
		if lineNum != -1 {
			state.lineAllocations[lineNum] = fmt.Sprintf("Spinner@%p (msg: %q)", spinner, spinner.msg)
		}
	}

	return state
}

// renderDebugMap outputs a visual representation of current spinner positions.
// Format:
//   ┌─────────────────────────────────────────┐
//   │ Spinner Position Map                    │
//   ├─────────────────────────────────────────┤
//   │ Line 0: Spinner@0x... (msg: "Loading") │
//   │ Line 1: Spinner@0x... (msg: "Saving")  │
//   │ Line 2: <empty>                         │
//   │ Cursor: Line 3                          │
//   └─────────────────────────────────────────┘
func (c *SpinnerCoordinator) renderDebugMap() {
	if !isDebugEnabled() {
		return
	}

	state := c.captureDebugState()
	if state == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "\n┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(os.Stderr, "│ Spinner Position Map (active: %d)                            \n", state.activeSpinners)
	fmt.Fprintf(os.Stderr, "├─────────────────────────────────────────────────────────────┤\n")

	// Find the highest line number
	maxLine := -1
	for lineNum := range state.lineAllocations {
		if lineNum > maxLine {
			maxLine = lineNum
		}
	}

	// Show all lines from 0 to max
	for i := 0; i <= maxLine; i++ {
		if desc, ok := state.lineAllocations[i]; ok {
			fmt.Fprintf(os.Stderr, "│ Line %d: %-52s│\n", i, desc)
		} else {
			fmt.Fprintf(os.Stderr, "│ Line %d: <empty>                                             │\n", i)
		}
	}

	// Show cursor position (always at bottom in our model)
	fmt.Fprintf(os.Stderr, "│ Cursor: Line %d                                              │\n", maxLine+1)
	fmt.Fprintf(os.Stderr, "└─────────────────────────────────────────────────────────────┘\n\n")
}

// validateDebugMode runs validation checks when debug mode is enabled.
// Panics if critical inconsistencies are detected in debug mode.
func (c *SpinnerCoordinator) validateDebugMode() {
	if !isDebugEnabled() {
		return
	}

	inconsistencies := c.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		errorCount := 0
		for _, inc := range inconsistencies {
			if inc.severity == "error" {
				debugLog("VALIDATION", "ERROR: %s", inc.description)
				errorCount++
			} else {
				debugLogVerbose("VALIDATION", "WARNING: %s", inc.description)
			}
		}

		// In debug mode, panic on errors to catch issues early
		if errorCount > 0 {
			panic(fmt.Sprintf("DEBUG MODE: %d state consistency errors detected", errorCount))
		}
	}

	if !c.checkStateInvariants() {
		debugLog("VALIDATION", "ERROR: State invariants violated")
		panic("DEBUG MODE: State invariants violated")
	}
}
