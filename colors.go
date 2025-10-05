package bullets

import "fmt"

// ANSI color codes for terminal output
const (
	reset = "\033[0m"

	// Text colors
	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"

	// Bright colors
	brightBlack   = "\033[90m"
	brightRed     = "\033[91m"
	brightGreen   = "\033[92m"
	brightYellow  = "\033[93m"
	brightBlue    = "\033[94m"
	brightMagenta = "\033[95m"
	brightCyan    = "\033[96m"
	brightWhite   = "\033[97m"

	// Text styles
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	underline = "\033[4m"
)

// Bullet symbols
const (
	bulletInfo    = "•"
	bulletSuccess = "✓"
	bulletError   = "✗"
	bulletWarn    = "⚠"
	bulletDebug   = "◦"
)

// colorize wraps text in ANSI color codes
func colorize(color, text string) string {
	return color + text + reset
}

// getBulletStyle returns the colored bullet and color for a given level
func getBulletStyle(level Level) (string, string) {
	switch level {
	case DebugLevel:
		return colorize(dim, bulletDebug), dim
	case InfoLevel:
		return colorize(cyan, bulletInfo), cyan
	case WarnLevel:
		return colorize(yellow, bulletWarn), yellow
	case ErrorLevel:
		return colorize(red, bulletError), red
	case FatalLevel:
		return colorize(brightRed+bold, bulletError), brightRed + bold
	default:
		return bulletInfo, reset
	}
}

// formatMessage formats a message with the appropriate style for the level
func formatMessage(level Level, msg string) string {
	bullet, color := getBulletStyle(level)
	return fmt.Sprintf("%s %s", bullet, colorize(color, msg))
}
