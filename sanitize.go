package bullets

import "regexp"

// ansiEscapeRegex matches ANSI escape sequences (CSI sequences).
// Covers color codes, cursor movement, line clearing, and other control sequences.
var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from a string.
// This is useful for sanitizing user-provided input that may contain
// malicious ANSI control characters (terminal injection).
//
// Example:
//
//	clean := bullets.StripANSI("\x1b[31mred text\x1b[0m")
//	// clean == "red text"
func StripANSI(s string) string {
	return ansiEscapeRegex.ReplaceAllString(s, "")
}

// sanitizeMsg strips ANSI escape sequences from a message.
// Used internally at entry points when sanitizeInput is enabled.
func sanitizeMsg(msg string) string {
	return StripANSI(msg)
}
