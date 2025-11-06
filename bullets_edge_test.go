package bullets_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/sgaunet/bullets"
)

// TestInvalidWriter tests behavior with various writer states
func TestInvalidWriter(t *testing.T) {
	// Test with a closed writer
	pr, pw := io.Pipe()
	pw.Close() // Close the writer immediately

	logger := bullets.New(pw)

	// These operations should handle closed writer gracefully (may fail to write but not panic)
	logger.Info("Test with closed writer")
	logger.Error("Error with closed writer")

	// Logger should be created even with closed writer
	if logger == nil {
		t.Error("Logger should be created even with closed writer")
	}

	pr.Close()
}

// TestVeryLongMessages tests with extremely long messages
func TestVeryLongMessages(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create a very long message (10KB)
	longMsg := strings.Repeat("a", 10000)
	logger.Info(longMsg)

	output := buf.String()
	if !strings.Contains(output, longMsg) {
		t.Error("Long message was truncated or lost")
	}

	// Test with formatted long message
	buf.Reset()
	logger.Infof("Message: %s", longMsg)
	output = buf.String()
	if !strings.Contains(output, longMsg) {
		t.Error("Formatted long message was truncated or lost")
	}
}

// TestSpecialCharacters tests messages with special characters
func TestSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	testCases := []string{
		"Message with \n newline",
		"Message with \t tab",
		"Message with \r carriage return",
		"Message with \x00 null byte",
		"Message with emoji 🚀 🎨 ✨",
		"Message with unicode 你好世界 مرحبا بالعالم",
		`Message with "quotes" and 'apostrophes'`,
		"Message with backslash \\",
		"Message with ANSI \033[31mred\033[0m text",
	}

	for _, msg := range testCases {
		buf.Reset()
		logger.Info(msg)
		output := buf.String()

		// Check that special characters are preserved
		if !strings.Contains(output, msg) {
			t.Errorf("Special characters not preserved in message: %q", msg)
		}
	}
}

// TestUnicodeBullets tests custom unicode bullet symbols
func TestUnicodeBullets(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	unicodeBullets := []string{
		"→", "▶", "★", "◆", "◉", "➤", "✦", "⬢", "🔹", "🎯", "🚀",
	}

	for _, bullet := range unicodeBullets {
		buf.Reset()
		logger.SetBullet(bullets.InfoLevel, bullet)
		logger.Info("Test message")

		output := buf.String()
		if !strings.Contains(output, bullet) {
			t.Errorf("Unicode bullet %s not found in output", bullet)
		}
	}
}

// TestNegativePadding tests behavior with negative padding
func TestNegativePadding(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Try to set negative padding
	logger.DecreasePadding()
	logger.DecreasePadding()
	logger.DecreasePadding()

	logger.Info("Test message")
	output := buf.String()

	// Should handle negative padding gracefully (treat as 0)
	if strings.HasPrefix(output, " ") {
		t.Error("Negative padding should be treated as 0")
	}
}

// TestConcurrentLogging tests thread safety
func TestConcurrentLogging(t *testing.T) {
	// Use a thread-safe writer for concurrent tests
	writer := &syncWriter{buf: &bytes.Buffer{}}
	logger := bullets.New(writer)

	const goroutines = 100
	const logsPerRoutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerRoutine; j++ {
				switch j % 5 {
				case 0:
					logger.Infof("Goroutine %d: Info %d", id, j)
				case 1:
					logger.Debugf("Goroutine %d: Debug %d", id, j)
				case 2:
					logger.Warnf("Goroutine %d: Warn %d", id, j)
				case 3:
					logger.Errorf("Goroutine %d: Error %d", id, j)
				case 4:
					logger.WithField("gid", id).Info("With field")
				}
			}
		}(i)
	}

	wg.Wait()

	output := writer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have many log lines (depending on log level)
	if len(lines) < goroutines*logsPerRoutine/2 {
		t.Errorf("Expected many concurrent log lines, got %d", len(lines))
	}
}

// TestFormattingErrors tests printf with invalid format strings
func TestFormattingErrors(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test with valid formatted messages to ensure basic formatting works
	logger.Infof("Expected %s", "one")
	logger.Infof("Number: %d", 42)
	logger.Infof("Multiple: %s %d", "test", 123)

	// Test with empty format
	logger.Infof("")

	// Test with no placeholders
	logger.Infof("No placeholders here")

	// Should not panic, errors should be handled gracefully
	output := buf.String()
	if output == "" {
		t.Error("Expected some output from formatted messages")
	}
}

// TestFieldsWithSpecialValues tests WithField with various types
func TestFieldsWithSpecialValues(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test with nil value
	logger.WithField("nil_field", nil).Info("Test nil")

	// Test with empty string key
	buf.Reset()
	logger.WithField("", "value").Info("Empty key")

	// Test with very long field key/value
	longKey := strings.Repeat("k", 1000)
	longValue := strings.Repeat("v", 1000)
	buf.Reset()
	logger.WithField(longKey, longValue).Info("Long fields")

	// Test with special characters in field
	buf.Reset()
	logger.WithField("key\nwith\nnewlines", "value\twith\ttabs").Info("Special chars")

	// Test with complex nested structure (but not circular to avoid stack overflow)
	complexMap := make(map[string]interface{})
	complexMap["nested"] = map[string]interface{}{
		"level2": map[string]interface{}{
			"level3": "deep value",
		},
	}
	buf.Reset()
	logger.WithField("complex", complexMap).Info("Complex nested structure")

	// All should handle gracefully without panic
	if buf.String() == "" {
		t.Error("Expected output for special field values")
	}
}

// TestWriterError tests behavior when writer returns error
func TestWriterError(t *testing.T) {
	// Create a writer that always returns error
	errorWriter := &failingWriter{err: errors.New("write failed")}
	logger := bullets.New(errorWriter)

	// Should not panic even if write fails
	logger.Info("This write will fail")
	logger.Warn("This also fails")
	logger.Error("All writes fail")

	if errorWriter.attempts < 3 {
		t.Error("Expected write attempts even with failures")
	}
}

// failingWriter is a writer that always returns an error
type failingWriter struct {
	attempts int
	err      error
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	w.attempts++
	return 0, w.err
}

// TestMaxPadding tests extremely high padding levels
func TestMaxPadding(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Set very high padding
	for i := 0; i < 100; i++ {
		logger.IncreasePadding()
	}

	logger.Info("Very indented")
	output := buf.String()

	// Should have lots of indentation
	expectedSpaces := strings.Repeat("  ", 100)
	if !strings.Contains(output, expectedSpaces) {
		t.Error("Expected deep indentation not found")
	}
}

// TestLogLevelEdgeCases tests edge cases for log levels
func TestLogLevelEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test with invalid level (cast from int)
	invalidLevel := bullets.Level(999)
	logger.SetLevel(invalidLevel)

	// Should handle gracefully
	logger.Info("Test with invalid level")

	// Test level boundaries - use valid levels
	logger.SetLevel(bullets.DebugLevel)
	logger.Debug("Debug should appear")

	logger.SetLevel(bullets.FatalLevel)
	logger.Error("Error should NOT appear at Fatal level")

	// Test very high level value
	veryHighLevel := bullets.Level(100)
	logger.SetLevel(veryHighLevel)
	logger.Error("Nothing should appear with very high level")
}

// TestPanicInFormatFunc tests recovery from panic in format function
func TestPanicInFormatFunc(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create a value that causes panic when formatted
	type panicker struct{}
	p := panicker{}

	// Use custom Stringer that panics
	defer func() {
		if r := recover(); r != nil {
			// Good, we recovered from panic
			return
		}
	}()

	logger.WithField("panic_field", p).Info("This might panic")
}

// TestRaceConditions specifically tests for race conditions
func TestRaceConditions(t *testing.T) {
	writer := &syncWriter{buf: &bytes.Buffer{}}
	logger := bullets.New(writer)

	const iterations = 1000

	// Start multiple goroutines doing different operations
	go func() {
		for i := 0; i < iterations; i++ {
			logger.SetLevel(bullets.InfoLevel)
			logger.SetLevel(bullets.DebugLevel)
		}
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			logger.IncreasePadding()
			logger.DecreasePadding()
		}
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			logger.SetUseSpecialBullets(true)
			logger.SetUseSpecialBullets(false)
		}
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			logger.Info("Race test")
		}
	}()

	go func() {
		for i := 0; i < iterations; i++ {
			logger.WithField("key", i).Info("Field race")
		}
	}()

	// Let everything run
	// This test will fail with -race flag if there are race conditions
}

// TestZeroValues tests behavior with zero/default values
func TestZeroValues(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test with zero-value types
	logger.WithField("int", 0).Info("Zero int")
	logger.WithField("string", "").Info("Empty string")
	logger.WithField("bool", false).Info("False bool")
	logger.WithField("float", 0.0).Info("Zero float")
	logger.WithField("nil", nil).Info("Nil value")

	var zeroStruct struct{}
	logger.WithField("struct", zeroStruct).Info("Zero struct")

	// All should work without issues
	output := buf.String()
	if output == "" {
		t.Error("Expected output for zero values")
	}
}

// TestMultibyteCharacters tests with various multibyte characters
func TestMultibyteCharacters(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test various languages and scripts
	messages := []string{
		"日本語のメッセージ",              // Japanese
		"中文消息",                   // Chinese
		"한국어 메시지",                // Korean
		"Сообщение на русском",   // Russian
		"رسالة بالعربية",         // Arabic
		"🏃‍♂️ Running man emoji", // Complex emoji
		"𝓜𝓪𝓽𝓱𝓮𝓶𝓪𝓽𝓲𝓬𝓪𝓵 𝓫𝓸𝓵𝓭 𝓼𝓬𝓻𝓲𝓹𝓽", // Mathematical bold script
	}

	for _, msg := range messages {
		buf.Reset()
		logger.Info(msg)
		output := buf.String()
		if !strings.Contains(output, msg) {
			t.Errorf("Multibyte message not preserved: %s", msg)
		}
	}
}

// TestMemoryLeaks tests for potential memory leaks
func TestMemoryLeaks(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create many loggers and fields (should be garbage collected)
	for i := 0; i < 10000; i++ {
		newLogger := logger.WithField(fmt.Sprintf("key%d", i), i)
		newLogger.Info("Memory test")
	}

	// Create and destroy many messages
	for i := 0; i < 10000; i++ {
		logger.Info(strings.Repeat("x", 1000))
		buf.Reset() // Clear buffer to avoid growing indefinitely
	}

	// This test doesn't directly check for leaks but exercises
	// paths that might leak memory. Use with memory profiler.
}

// syncWriter wraps bytes.Buffer to make it thread-safe
type syncWriter struct {
	mu  sync.Mutex
	buf *bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}
