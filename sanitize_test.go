package bullets

import (
	"bytes"
	"sync"
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no ANSI sequences",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "color code red",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "bold and color",
			input:    "\x1b[1m\x1b[32mbold green\x1b[0m",
			expected: "bold green",
		},
		{
			name:     "cursor movement up",
			input:    "\x1b[5Amoved up",
			expected: "moved up",
		},
		{
			name:     "cursor movement down",
			input:    "\x1b[3Bmoved down",
			expected: "moved down",
		},
		{
			name:     "clear line",
			input:    "\x1b[2Kcleared",
			expected: "cleared",
		},
		{
			name:     "clear screen",
			input:    "\x1b[2Jscreen cleared",
			expected: "screen cleared",
		},
		{
			name:     "move to column",
			input:    "\x1b[0Gstart of line",
			expected: "start of line",
		},
		{
			name:     "mixed sequences",
			input:    "\x1b[31m\x1b[1mERROR\x1b[0m: \x1b[33mwarning\x1b[0m text",
			expected: "ERROR: warning text",
		},
		{
			name:     "unicode content preserved",
			input:    "こんにちは \x1b[31m世界\x1b[0m ✓",
			expected: "こんにちは 世界 ✓",
		},
		{
			name:     "SGR with multiple params",
			input:    "\x1b[38;5;196mextended color\x1b[0m",
			expected: "extended color",
		},
		{
			name:     "partial escape not matched",
			input:    "\x1b not a sequence",
			expected: "\x1b not a sequence",
		},
		{
			name:     "injection attempt: fake success",
			input:    "\x1b[2K\x1b[0G\x1b[32m● Done!\x1b[0m",
			expected: "● Done!",
		},
		{
			name:     "injection attempt: cursor overwrite",
			input:    "\x1b[10A\x1b[2KMalicious overwrite",
			expected: "Malicious overwrite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeMsg(t *testing.T) {
	// sanitizeMsg delegates to StripANSI, so a basic check is sufficient
	input := "\x1b[31mhello\x1b[0m"
	expected := "hello"
	result := sanitizeMsg(input)
	if result != expected {
		t.Errorf("sanitizeMsg(%q) = %q, want %q", input, result, expected)
	}
}

func TestLoggerSanitizeInput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	logger.Info("\x1b[2J\x1b[Hinjected clear screen")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[2J")) {
		t.Error("ANSI clear screen sequence was not sanitized in Info()")
	}
	if !bytes.Contains([]byte(output), []byte("injected clear screen")) {
		t.Error("message text was lost during sanitization")
	}
}

func TestLoggerSanitizeInputDisabledByDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// With sanitization disabled (default), ANSI should pass through
	logger.Info("\x1b[31mred\x1b[0m")
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("\x1b[31m")) {
		t.Error("ANSI sequences should pass through when sanitizeInput is disabled")
	}
}

func TestLoggerSuccessSanitize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	logger.Success("\x1b[5Aoverwrite\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[5A")) {
		t.Error("ANSI cursor movement was not sanitized in Success()")
	}
	if !bytes.Contains([]byte(output), []byte("overwrite")) {
		t.Error("message text was lost during sanitization")
	}
}

func TestLoggerDebugfSanitize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetLevel(DebugLevel)
	logger.SetSanitizeInput(true)

	logger.Debugf("value: %s", "\x1b[2Khidden")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[2K")) {
		t.Error("ANSI clear line was not sanitized in Debugf()")
	}
}

func TestWithFieldPreservesSanitizeInput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	derived := logger.WithField("key", "val")
	derived.Info("\x1b[31mshould be stripped\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[31m")) {
		t.Error("sanitizeInput was not preserved through WithField()")
	}
}

func TestWithFieldsPreservesSanitizeInput(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	derived := logger.WithFields(map[string]interface{}{"a": 1})
	derived.Info("\x1b[31mshould be stripped\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[31m")) {
		t.Error("sanitizeInput was not preserved through WithFields()")
	}
}

func TestSpinnerSanitize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	// Non-TTY mode: spinner prints static message
	s := logger.Spinner("\x1b[2Jclear screen attack")
	s.Success("\x1b[10Aoverwrite attack")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[2J")) {
		t.Error("ANSI was not sanitized in Spinner() creation message")
	}
	if bytes.Contains([]byte(output), []byte("\x1b[10A")) {
		t.Error("ANSI was not sanitized in Spinner.Success()")
	}
}

func TestSpinnerErrorSanitize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	s := logger.Spinner("task")
	s.Error("\x1b[2Kfake error\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[2K")) {
		t.Error("ANSI was not sanitized in Spinner.Error()")
	}
}

func TestSpinnerReplaceSanitize(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	s := logger.Spinner("task")
	s.Replace("\x1b[0G\x1b[2Kreplaced\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[0G")) {
		t.Error("ANSI was not sanitized in Spinner.Replace()")
	}
}

func TestUpdatableLoggerSanitize(t *testing.T) {
	var buf bytes.Buffer
	ul := NewUpdatable(&buf)
	ul.SetSanitizeInput(true)

	ul.Success("\x1b[5Ainjected\x1b[0m")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[5A")) {
		t.Error("ANSI was not sanitized in UpdatableLogger.Success()")
	}
}

func TestBulletHandleSanitize(t *testing.T) {
	var buf bytes.Buffer
	ul := NewUpdatable(&buf)
	ul.SetSanitizeInput(true)

	handle := ul.InfoHandle("\x1b[2Jinjected")
	output := buf.String()

	if bytes.Contains([]byte(output), []byte("\x1b[2J")) {
		t.Error("ANSI was not sanitized in logHandle()")
	}

	// Test Update
	buf.Reset()
	handle.Update(InfoLevel, "\x1b[31mred\x1b[0m")
	state := handle.GetState()
	if state.Message != "red" {
		t.Errorf("BulletHandle.Update() did not sanitize: got %q, want %q", state.Message, "red")
	}

	// Test UpdateMessage
	handle.UpdateMessage("\x1b[2Kupdated\x1b[0m")
	state = handle.GetState()
	if state.Message != "updated" {
		t.Errorf("BulletHandle.UpdateMessage() did not sanitize: got %q, want %q", state.Message, "updated")
	}

	// Test Success
	handle.Success("\x1b[5Asuccess\x1b[0m")
	state = handle.GetState()
	if state.Message != "success" {
		t.Errorf("BulletHandle.Success() did not sanitize: got %q, want %q", state.Message, "success")
	}
}

func TestBulletHandleSetStateSanitize(t *testing.T) {
	var buf bytes.Buffer
	ul := NewUpdatable(&buf)
	ul.SetSanitizeInput(true)

	handle := ul.InfoHandle("initial")
	handle.SetState(HandleState{
		Level:   WarnLevel,
		Message: "\x1b[31minjected\x1b[0m",
	})

	state := handle.GetState()
	if state.Message != "injected" {
		t.Errorf("SetState() did not sanitize: got %q, want %q", state.Message, "injected")
	}
}

func TestConcurrentSanitization(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetSanitizeInput(true)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Infof("msg %d: \x1b[31mred\x1b[0m", n)
		}(i)
	}
	wg.Wait()

	output := buf.String()
	if bytes.Contains([]byte(output), []byte("\x1b[31m")) {
		t.Error("ANSI sequences leaked through under concurrent access")
	}
}

func BenchmarkStripANSI_NoSequences(b *testing.B) {
	input := "this is a plain message with no ANSI sequences at all"
	for b.Loop() {
		StripANSI(input)
	}
}

func BenchmarkStripANSI_WithSequences(b *testing.B) {
	input := "\x1b[31m\x1b[1mERROR\x1b[0m: \x1b[33mwarning\x1b[0m text \x1b[2Kclear"
	for b.Loop() {
		StripANSI(input)
	}
}

func BenchmarkStripANSI_LongString(b *testing.B) {
	input := ""
	for range 100 {
		input += "\x1b[31mred\x1b[0m normal text "
	}
	for b.Loop() {
		StripANSI(input)
	}
}
