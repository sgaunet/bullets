package bullets

import "fmt"

// ANSI color codes for terminal output.
const (
	reset = "\033[0m"
	Reset = reset // Exported version

	// Text colors.
	black   = "\033[30m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"

	// Exported color constants for public use.
	ColorBlack   = black
	ColorRed     = red
	ColorGreen   = green
	ColorYellow  = yellow
	ColorBlue    = blue
	ColorMagenta = magenta
	ColorCyan    = cyan
	ColorWhite   = white

	// Bright colors.
	brightBlack   = "\033[90m"
	brightRed     = "\033[91m"
	brightGreen   = "\033[92m"
	brightYellow  = "\033[93m"
	brightBlue    = "\033[94m"
	brightMagenta = "\033[95m"
	brightCyan    = "\033[96m"
	brightWhite   = "\033[97m"

	// Exported bright color constants.
	ColorBrightBlack   = brightBlack
	ColorBrightRed     = brightRed
	ColorBrightGreen   = brightGreen
	ColorBrightYellow  = brightYellow
	ColorBrightBlue    = brightBlue
	ColorBrightMagenta = brightMagenta
	ColorBrightCyan    = brightCyan
	ColorBrightWhite   = brightWhite

	// Text styles.
	bold      = "\033[1m"
	dim       = "\033[2m"
	italic    = "\033[3m"
	underline = "\033[4m"

	// Exported style constants.
	StyleBold      = bold
	StyleDim       = dim
	StyleItalic    = italic
	StyleUnderline = underline
)

// Bullet symbols.
const (
	bulletInfo    = "•"
	bulletSuccess = "✓"
	bulletError   = "✗"
	bulletWarn    = "⚠"
	bulletDebug   = "○"
)

// colorize wraps text in ANSI color codes (internal use).
func colorize(color, text string) string {
	return color + text + reset
}

// Colorize wraps text in ANSI color codes (exported for public use).
func Colorize(color, text string) string {
	return colorize(color, text)
}

// getBulletStyle returns the colored bullet and color for a given level.
func getBulletStyle(level Level, useSpecialBullets bool, customBullets map[Level]string) (string, string) {
	bullet := getBulletSymbol(level, useSpecialBullets, customBullets)
	color := getColorForLevel(level)
	return colorize(color, bullet), color
}

// getBulletSymbol determines the bullet symbol based on level and configuration.
func getBulletSymbol(level Level, useSpecialBullets bool, customBullets map[Level]string) string {
	// Priority: custom bullets > special bullets > default circle
	if custom, ok := customBullets[level]; ok {
		return custom
	}

	if useSpecialBullets {
		return getSpecialBullet(level)
	}

	return bulletInfo
}

// getSpecialBullet returns the special bullet symbol for a level.
func getSpecialBullet(level Level) string {
	switch level {
	case DebugLevel:
		return bulletDebug
	case InfoLevel:
		return bulletInfo
	case WarnLevel:
		return bulletWarn
	case ErrorLevel, FatalLevel:
		return bulletError
	default:
		return bulletInfo
	}
}

// getColorForLevel returns the color for a level.
func getColorForLevel(level Level) string {
	switch level {
	case DebugLevel:
		return dim
	case InfoLevel:
		return cyan
	case WarnLevel:
		return yellow
	case ErrorLevel:
		return red
	case FatalLevel:
		return brightRed + bold
	default:
		return reset
	}
}

// formatMessage formats a message with the appropriate style for the level.
func formatMessage(level Level, msg string, useSpecialBullets bool, customBullets map[Level]string) string {
	bullet, _ := getBulletStyle(level, useSpecialBullets, customBullets)
	return fmt.Sprintf("%s %s", bullet, msg)
}
