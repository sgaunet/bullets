package bullets

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	if logger == nil {
		t.Fatal("New() returned nil")
	}

	if logger.level != InfoLevel {
		t.Errorf("Expected default level InfoLevel, got %v", logger.level)
	}

	if logger.padding != 0 {
		t.Errorf("Expected default padding 0, got %d", logger.padding)
	}

	if logger.useSpecialBullets != false {
		t.Error("Expected useSpecialBullets to be false by default")
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name     string
		logLevel Level
		logFunc  func(*Logger)
		want     bool // should output
	}{
		{"Debug at Debug level", DebugLevel, func(l *Logger) { l.Debug("test") }, true},
		{"Debug at Info level", InfoLevel, func(l *Logger) { l.Debug("test") }, false},
		{"Info at Info level", InfoLevel, func(l *Logger) { l.Info("test") }, true},
		{"Info at Warn level", WarnLevel, func(l *Logger) { l.Info("test") }, false},
		{"Warn at Warn level", WarnLevel, func(l *Logger) { l.Warn("test") }, true},
		{"Error at Warn level", WarnLevel, func(l *Logger) { l.Error("test") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf)
			logger.SetLevel(tt.logLevel)

			tt.logFunc(logger)

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.want {
				t.Errorf("Expected output=%v, got output=%v (output: %q)", tt.want, hasOutput, output)
			}
		})
	}
}

func TestFormatted(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.Infof("count: %d", 42)

	output := buf.String()
	if !strings.Contains(output, "count: 42") {
		t.Errorf("Expected formatted output to contain 'count: 42', got %q", output)
	}
}

func TestPadding(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.Info("level 0")
	logger.IncreasePadding()
	logger.Info("level 1")
	logger.IncreasePadding()
	logger.Info("level 2")
	logger.DecreasePadding()
	logger.Info("level 1 again")
	logger.ResetPadding()
	logger.Info("level 0 again")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 5 {
		t.Fatalf("Expected 5 lines, got %d", len(lines))
	}

	// Check indentation
	if strings.HasPrefix(lines[0], " ") {
		t.Error("First line should not be indented")
	}

	if !strings.HasPrefix(lines[1], "  ") {
		t.Error("Second line should have 1 level of indentation")
	}

	if !strings.HasPrefix(lines[2], "    ") {
		t.Error("Third line should have 2 levels of indentation")
	}
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.WithField("user", "john").Info("logged in")

	output := buf.String()
	if !strings.Contains(output, "user=john") {
		t.Errorf("Expected output to contain 'user=john', got %q", output)
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.WithFields(map[string]interface{}{
		"version": "1.0",
		"arch":    "amd64",
	}).Info("building")

	output := buf.String()
	if !strings.Contains(output, "version=1.0") || !strings.Contains(output, "arch=amd64") {
		t.Errorf("Expected output to contain version and arch, got %q", output)
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	err := errors.New("connection timeout")
	logger.WithError(err).Error("failed")

	output := buf.String()
	if !strings.Contains(output, "connection timeout") {
		t.Errorf("Expected output to contain error message, got %q", output)
	}
}

func TestSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.Success("completed")

	output := buf.String()
	if !strings.Contains(output, "completed") {
		t.Errorf("Expected output to contain 'completed', got %q", output)
	}
}

func TestStep(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	done := logger.Step("processing")
	time.Sleep(50 * time.Millisecond)
	done()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines of output, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "processing") {
		t.Errorf("Expected first line to contain 'processing', got %q", lines[0])
	}

	if !strings.Contains(lines[1], "completed") {
		t.Errorf("Expected second line to contain 'completed', got %q", lines[1])
	}
}

func TestSetUseSpecialBullets(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.SetUseSpecialBullets(true)
	logger.Success("done")

	output := buf.String()
	// When special bullets are enabled, Success should use ✓
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected output to contain special bullet '✓', got %q", output)
	}
}

func TestSetBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.SetBullet(InfoLevel, "→")
	logger.Info("custom bullet")

	output := buf.String()
	if !strings.Contains(output, "→") {
		t.Errorf("Expected output to contain custom bullet '→', got %q", output)
	}
}

func TestSetBullets(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	logger.SetBullets(map[Level]string{
		InfoLevel: "▶",
		WarnLevel: "⚡",
	})

	logger.Info("info message")
	output1 := buf.String()
	if !strings.Contains(output1, "▶") {
		t.Errorf("Expected info to use '▶', got %q", output1)
	}

	buf.Reset()
	logger.Warn("warn message")
	output2 := buf.String()
	if !strings.Contains(output2, "⚡") {
		t.Errorf("Expected warn to use '⚡', got %q", output2)
	}
}

func TestConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	done := make(chan bool)

	// Test concurrent logging
	for i := 0; i < 10; i++ {
		go func(n int) {
			logger.Infof("message %d", n)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 10 {
		t.Errorf("Expected 10 lines of output, got %d", len(lines))
	}
}

func TestWithFieldPreservesSettings(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)
	logger.SetBullet(InfoLevel, "★")
	logger.IncreasePadding()

	newLogger := logger.WithField("key", "value")

	// Test that settings are preserved
	if newLogger.useSpecialBullets != logger.useSpecialBullets {
		t.Error("WithField should preserve useSpecialBullets setting")
	}

	if len(newLogger.customBullets) != len(logger.customBullets) {
		t.Error("WithField should preserve custom bullets")
	}

	if newLogger.padding != logger.padding {
		t.Error("WithField should preserve padding")
	}
}
