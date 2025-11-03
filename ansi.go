package bullets

// ANSI escape codes for terminal cursor control and line manipulation.
const (
	ansiSaveCursor    = "\033[s"
	ansiRestoreCursor = "\033[u"
	ansiClearLine     = "\033[2K"   // Clear entire line
	ansiMoveUp        = "\033[%dA"  // Move cursor up N lines
	ansiMoveDown      = "\033[%dB"  // Move cursor down N lines
	ansiMoveToCol     = "\033[0G"   // Move to column 0
)
