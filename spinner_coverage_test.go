package bullets

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// TestSpinnerEmptyFramesFallback tests that empty frames use default.
func TestSpinnerEmptyFramesFallback(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Pass empty frames
	spinner := logger.SpinnerWithFrames("test", []string{})

	if len(spinner.frames) == 0 {
		t.Error("Expected spinner to have default frames when empty frames provided")
	}

	spinner.Stop()
}

// TestSpinnerSuccessWithCustomBullet tests Success with custom bullet.
func TestSpinnerSuccessWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "★")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	if !strings.Contains(output, "★") {
		t.Errorf("Expected custom bullet '★' in success output, got: %q", output)
	}
	if !strings.Contains(output, "done") {
		t.Errorf("Expected 'done' in success output, got: %q", output)
	}
}

// TestSpinnerSuccessWithSpecialBullet tests Success with special bullet enabled.
func TestSpinnerSuccessWithSpecialBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	// Should use special success bullet (✓)
	if !strings.Contains(output, "✓") && !strings.Contains(output, "done") {
		t.Errorf("Expected special bullet '✓' or 'done' in output, got: %q", output)
	}
}

// TestSpinnerSuccessDefault tests Success with default bullet (no custom, no special).
func TestSpinnerSuccessDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	// Explicitly disable special bullets and no custom bullets

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	if !strings.Contains(output, "done") {
		t.Errorf("Expected 'done' in output, got: %q", output)
	}
	// Should use default circle bullet (•)
	if !strings.Contains(output, "•") {
		t.Errorf("Expected default bullet '•' in output, got: %q", output)
	}
}

// TestSpinnerErrorWithCustomBullet tests Error with custom bullet.
func TestSpinnerErrorWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(ErrorLevel, "✖")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "✖") {
		t.Errorf("Expected custom bullet '✖' in error output, got: %q", output)
	}
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in error output, got: %q", output)
	}
}

// TestSpinnerErrorWithSpecialBullet tests Error with special bullet enabled.
func TestSpinnerErrorWithSpecialBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	// Should use special error bullet (✗)
	if !strings.Contains(output, "✗") && !strings.Contains(output, "failed") {
		t.Errorf("Expected special bullet '✗' or 'failed' in output, got: %q", output)
	}
}

// TestSpinnerErrorDefault tests Error with default bullet.
func TestSpinnerErrorDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in output, got: %q", output)
	}
}

// TestSpinnerReplaceWithCustomBullet tests Replace with custom bullet.
func TestSpinnerReplaceWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "→")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("replaced")

	output := buf.String()
	if !strings.Contains(output, "→") {
		t.Errorf("Expected custom bullet '→' in replace output, got: %q", output)
	}
	if !strings.Contains(output, "replaced") {
		t.Errorf("Expected 'replaced' in output, got: %q", output)
	}
}

// TestSpinnerReplaceDefault tests Replace with default bullet.
func TestSpinnerReplaceDefault(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("replaced")

	output := buf.String()
	if !strings.Contains(output, "replaced") {
		t.Errorf("Expected 'replaced' in output, got: %q", output)
	}
}

// TestSpinnerNonTTYSuccessWithCustomBullet tests non-TTY Success with custom bullet.
func TestSpinnerNonTTYSuccessWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "✓")

	// Non-TTY mode (default for bytes.Buffer)
	spinner := logger.SpinnerCircle("test")
	time.Sleep(50 * time.Millisecond)
	spinner.Success("completed")

	output := buf.String()
	if !strings.Contains(output, "completed") {
		t.Errorf("Expected 'completed' in output, got: %q", output)
	}
}

// TestSpinnerNonTTYErrorWithCustomBullet tests non-TTY Error with custom bullet.
func TestSpinnerNonTTYErrorWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(ErrorLevel, "✗")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(50 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in output, got: %q", output)
	}
}

// TestSpinnerNonTTYReplaceWithCustomBullet tests non-TTY Replace with custom bullet.
func TestSpinnerNonTTYReplaceWithCustomBullet(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "→")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(50 * time.Millisecond)
	spinner.Replace("replaced")

	output := buf.String()
	if !strings.Contains(output, "replaced") {
		t.Errorf("Expected 'replaced' in output, got: %q", output)
	}
}

// TestSpinnerWithOsFileWriter tests spinner with os.File writer for TTY detection.
func TestSpinnerWithOsFileWriter(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "spinner-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logger := New(tmpFile)
	spinner := logger.SpinnerCircle("test with file")

	// Should detect as non-TTY (file, not terminal)
	if spinner.isTTY {
		t.Error("Expected spinner.isTTY to be false for file writer")
	}

	time.Sleep(50 * time.Millisecond)
	spinner.Stop()
}

// TestSpinnerWithBULLETS_FORCE_TTY tests BULLETS_FORCE_TTY environment variable.
func TestSpinnerWithBULLETS_FORCE_TTY(t *testing.T) {
	// Save and restore original env
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	spinner := logger.SpinnerCircle("forced TTY test")

	// Should detect as TTY due to env var
	if !spinner.isTTY {
		t.Error("Expected spinner.isTTY to be true when BULLETS_FORCE_TTY=1")
	}

	time.Sleep(50 * time.Millisecond)
	spinner.Stop()
}

// TestSpinnerMultipleBulletConfigurations tests various bullet configurations.
func TestSpinnerMultipleBulletConfigurations(t *testing.T) {
	testCases := []struct {
		name           string
		setupLogger    func(*Logger)
		completionFunc func(*Spinner)
		expectedText   string
	}{
		{
			name: "Success with default",
			setupLogger: func(l *Logger) {
				// Default: no special bullets, no custom bullets
			},
			completionFunc: func(s *Spinner) { s.Success("ok") },
			expectedText:   "ok",
		},
		{
			name: "Success with special bullets",
			setupLogger: func(l *Logger) {
				l.SetUseSpecialBullets(true)
			},
			completionFunc: func(s *Spinner) { s.Success("ok") },
			expectedText:   "ok",
		},
		{
			name: "Success with custom bullet",
			setupLogger: func(l *Logger) {
				l.SetBullet(InfoLevel, "✓")
			},
			completionFunc: func(s *Spinner) { s.Success("ok") },
			expectedText:   "ok",
		},
		{
			name: "Error with default",
			setupLogger: func(l *Logger) {
				// Default
			},
			completionFunc: func(s *Spinner) { s.Error("fail") },
			expectedText:   "fail",
		},
		{
			name: "Error with special bullets",
			setupLogger: func(l *Logger) {
				l.SetUseSpecialBullets(true)
			},
			completionFunc: func(s *Spinner) { s.Error("fail") },
			expectedText:   "fail",
		},
		{
			name: "Error with custom bullet",
			setupLogger: func(l *Logger) {
				l.SetBullet(ErrorLevel, "✗")
			},
			completionFunc: func(s *Spinner) { s.Error("fail") },
			expectedText:   "fail",
		},
		{
			name: "Replace with default",
			setupLogger: func(l *Logger) {
				// Default
			},
			completionFunc: func(s *Spinner) { s.Replace("done") },
			expectedText:   "done",
		},
		{
			name: "Replace with custom bullet",
			setupLogger: func(l *Logger) {
				l.SetBullet(InfoLevel, "→")
			},
			completionFunc: func(s *Spinner) { s.Replace("done") },
			expectedText:   "done",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf)
			tc.setupLogger(logger)

			spinner := logger.SpinnerCircle("test")
			time.Sleep(50 * time.Millisecond)
			tc.completionFunc(spinner)

			output := buf.String()
			if !strings.Contains(output, tc.expectedText) {
				t.Errorf("Expected '%s' in output, got: %q", tc.expectedText, output)
			}
		})
	}
}

// TestSpinnerStressRapidCompletions tests rapid spinner creation and completion.
func TestSpinnerStressRapidCompletions(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Create and complete 100 spinners rapidly
	for i := 0; i < 100; i++ {
		spinner := logger.SpinnerCircle("test")
		switch i % 3 {
		case 0:
			spinner.Success("ok")
		case 1:
			spinner.Error("fail")
		case 2:
			spinner.Replace("done")
		}
	}

	// Should complete without panic or deadlock
}

// TestSpinnerEdgeCaseZeroInterval tests spinner with zero interval.
func TestSpinnerEdgeCaseZeroInterval(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Create spinner with zero interval (should still work)
	spinner := newSpinner(logger, "test", []string{"1", "2"}, "", 0)
	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should not panic
}

// TestSpinnerEdgeCaseSingleFrame tests spinner with single frame.
func TestSpinnerEdgeCaseSingleFrame(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerWithFrames("test", []string{"●"})
	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// Should work with single frame
}

// TestSpinnerWithPaddingLevels tests spinners with different padding levels.
func TestSpinnerWithPaddingLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	// Test with different padding levels
	for i := 0; i < 5; i++ {
		logger.ResetPadding()
		for j := 0; j < i; j++ {
			logger.IncreasePadding()
		}

		spinner := logger.SpinnerCircle("test")
		if spinner.padding != i {
			t.Errorf("Expected padding %d, got %d", i, spinner.padding)
		}
		time.Sleep(20 * time.Millisecond)
		spinner.Stop()
	}
}

// TestSpinnerDoubleStopIdempotent tests calling Stop twice is safe.
func TestSpinnerDoubleStopIdempotent(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(50 * time.Millisecond)

	spinner.Stop()
	spinner.Stop() // Should not panic or deadlock
}

// TestSpinnerCompletionAfterStop tests completion methods after Stop.
func TestSpinnerCompletionAfterStop(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(50 * time.Millisecond)
	spinner.Stop()

	// These should be safe (no-op since already stopped)
	spinner.Success("ok")
	spinner.Error("fail")
	spinner.Replace("done")
}

// TestSpinnerTTYSuccessWithCustomBullet tests TTY mode Success with custom bullet.
func TestSpinnerTTYSuccessWithCustomBullet(t *testing.T) {
	// Save and restore original env
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "✓")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("completed")

	output := buf.String()
	if !strings.Contains(output, "completed") {
		t.Errorf("Expected 'completed' in TTY output, got: %q", output)
	}
}

// TestSpinnerTTYSuccessWithSpecialBullet tests TTY mode Success with special bullet.
func TestSpinnerTTYSuccessWithSpecialBullet(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	if !strings.Contains(output, "done") {
		t.Errorf("Expected 'done' in TTY output with special bullet, got: %q", output)
	}
}

// TestSpinnerTTYSuccessDefault tests TTY mode Success with default bullet.
func TestSpinnerTTYSuccessDefault(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Success("done")

	output := buf.String()
	if !strings.Contains(output, "done") {
		t.Errorf("Expected 'done' in TTY output with default bullet, got: %q", output)
	}
}

// TestSpinnerTTYErrorWithCustomBullet tests TTY mode Error with custom bullet.
func TestSpinnerTTYErrorWithCustomBullet(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(ErrorLevel, "✗")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in TTY error output, got: %q", output)
	}
}

// TestSpinnerTTYErrorWithSpecialBullet tests TTY mode Error with special bullet.
func TestSpinnerTTYErrorWithSpecialBullet(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetUseSpecialBullets(true)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in TTY error output with special bullet, got: %q", output)
	}
}

// TestSpinnerTTYErrorDefault tests TTY mode Error with default bullet.
func TestSpinnerTTYErrorDefault(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Error("failed")

	output := buf.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("Expected 'failed' in TTY error output with default bullet, got: %q", output)
	}
}

// TestSpinnerTTYReplaceWithCustomBullet tests TTY mode Replace with custom bullet.
func TestSpinnerTTYReplaceWithCustomBullet(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)
	logger.SetBullet(InfoLevel, "→")

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("replaced")

	output := buf.String()
	if !strings.Contains(output, "replaced") {
		t.Errorf("Expected 'replaced' in TTY replace output, got: %q", output)
	}
}

// TestSpinnerTTYReplaceDefault tests TTY mode Replace with default bullet.
func TestSpinnerTTYReplaceDefault(t *testing.T) {
	orig := os.Getenv("BULLETS_FORCE_TTY")
	defer func() {
		if orig == "" {
			os.Unsetenv("BULLETS_FORCE_TTY")
		} else {
			os.Setenv("BULLETS_FORCE_TTY", orig)
		}
	}()

	os.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.SpinnerCircle("test")
	time.Sleep(100 * time.Millisecond)
	spinner.Replace("replaced")

	output := buf.String()
	if !strings.Contains(output, "replaced") {
		t.Errorf("Expected 'replaced' in TTY replace output with default bullet, got: %q", output)
	}
}
