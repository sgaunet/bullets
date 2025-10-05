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
	bulletDebug   = "○"
)

// colorize wraps text in ANSI color codes
func colorize(color, text string) string {
	return color + text + reset
}

// getBulletStyle returns the colored bullet and color for a given level
func getBulletStyle(level Level, useSpecialBullets bool, customBullets map[Level]string) (string, string) {
	// Priority: custom bullets > special bullets > default circle
	var bullet string
	var color string

	// Determine bullet symbol
	if custom, ok := customBullets[level]; ok {
		bullet = custom
	} else if useSpecialBullets {
		switch level {
		case DebugLevel:
			bullet = bulletDebug
		case InfoLevel:
			bullet = bulletInfo
		case WarnLevel:
			bullet = bulletWarn
		case ErrorLevel:
			bullet = bulletError
		case FatalLevel:
			bullet = bulletError
		default:
			bullet = bulletInfo
		}
	} else {
		// Default: use circle for all levels
		bullet = bulletInfo
	}

	// Determine color
	switch level {
	case DebugLevel:
		color = dim
	case InfoLevel:
		color = cyan
	case WarnLevel:
		color = yellow
	case ErrorLevel:
		color = red
	case FatalLevel:
		color = brightRed + bold
	default:
		color = reset
	}

	return colorize(color, bullet), color
}

// formatMessage formats a message with the appropriate style for the level
func formatMessage(level Level, msg string, useSpecialBullets bool, customBullets map[Level]string) string {
	bullet, _ := getBulletStyle(level, useSpecialBullets, customBullets)
	return fmt.Sprintf("%s %s", bullet, msg)
}
