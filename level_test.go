package bullets

import "testing"

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelComparison(t *testing.T) {
	if !(DebugLevel < InfoLevel) {
		t.Error("DebugLevel should be less than InfoLevel")
	}

	if !(InfoLevel < WarnLevel) {
		t.Error("InfoLevel should be less than WarnLevel")
	}

	if !(WarnLevel < ErrorLevel) {
		t.Error("WarnLevel should be less than ErrorLevel")
	}

	if !(ErrorLevel < FatalLevel) {
		t.Error("ErrorLevel should be less than FatalLevel")
	}
}
