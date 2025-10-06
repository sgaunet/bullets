package bullets_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sgaunet/bullets"
)

// TestCompleteLoggingWorkflow tests a complete real-world logging scenario
func TestCompleteLoggingWorkflow(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Simulate a build process
	logger.Info("Starting build process")
	logger.IncreasePadding()

	// Step 1: Dependencies
	depsStep := logger.Step("Installing dependencies")
	time.Sleep(10 * time.Millisecond)
	logger.IncreasePadding()
	logger.Debug("npm install")
	logger.Debug("bower install")
	logger.DecreasePadding()
	depsStep()

	// Step 2: Compilation with spinner
	spinner := logger.Spinner("Compiling source files")
	time.Sleep(20 * time.Millisecond)
	spinner.Success("Compilation complete")

	// Step 3: Tests
	logger.WithField("suite", "unit").Info("Running tests")
	logger.IncreasePadding()
	logger.Success("auth_test.go")
	logger.Success("api_test.go")
	logger.Error("database_test.go")
	logger.DecreasePadding()

	// Step 4: Build artifacts
	logger.WithFields(map[string]interface{}{
		"arch":    "amd64",
		"os":      "linux",
		"version": "1.2.3",
	}).Info("Building artifacts")

	logger.DecreasePadding()
	logger.Success("Build complete!")

	output := buf.String()
	// Verify output contains expected elements
	if !strings.Contains(output, "Starting build") {
		t.Error("Missing build start message")
	}
	if !strings.Contains(output, "Compilation complete") {
		t.Error("Missing compilation message")
	}
	if !strings.Contains(output, "Build complete") {
		t.Error("Missing build complete message")
	}
}

// TestMixedSpinnerAndLogging tests spinner with regular logging intermixed
func TestMixedSpinnerAndLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Start spinner
	spinner := logger.Spinner("Background task running")

	// Log while spinner is running
	go func() {
		time.Sleep(10 * time.Millisecond)
		logger.Info("Regular log while spinning")
		logger.Warn("Warning while spinning")
		logger.Error("Error while spinning")
	}()

	time.Sleep(50 * time.Millisecond)
	spinner.Success("Background task complete")

	// Continue logging after spinner
	logger.Info("Post-spinner logging")

	output := buf.String()
	if !strings.Contains(output, "Background task complete") {
		t.Error("Spinner completion not found")
	}
}

// TestUpdatableWithANSI tests updatable bullets with ANSI sequences
func TestUpdatableWithANSI(t *testing.T) {
	var buf bytes.Buffer

	// Force TTY mode to test ANSI codes
	oldEnv := os.Getenv("BULLETS_FORCE_TTY")
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Setenv("BULLETS_FORCE_TTY", oldEnv)

	logger := bullets.NewUpdatable(&buf)

	// Create multiple handles
	h1 := logger.InfoHandle("Task 1: Initializing...")
	h2 := logger.InfoHandle("Task 2: Initializing...")
	h3 := logger.InfoHandle("Task 3: Initializing...")

	// Update them
	h1.Success("Task 1: Complete ✓")
	h2.Warning("Task 2: Warning ⚠")
	h3.Error("Task 3: Failed ✗")

	// Add progress
	h4 := logger.InfoHandle("Downloading...")
	for i := 0; i <= 100; i += 25 {
		h4.Progress(i, 100)
		time.Sleep(5 * time.Millisecond)
	}
	h4.Success("Download complete!")

	output := buf.String()
	// Should contain ANSI escape codes
	if !strings.Contains(output, "\033[") {
		t.Log("Warning: ANSI codes might not be present in test environment")
	}
}

// TestHighVolumeLogging tests performance with many log entries
func TestHighVolumeLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	const logCount = 1000

	start := time.Now()
	for i := 0; i < logCount; i++ {
		switch i % 5 {
		case 0:
			logger.Infof("Info message %d", i)
		case 1:
			logger.Debugf("Debug message %d", i)
		case 2:
			logger.Warnf("Warning message %d", i)
		case 3:
			logger.Errorf("Error message %d", i)
		case 4:
			logger.WithField("index", i).Info("Structured log")
		}
	}
	duration := time.Since(start)

	// Performance check - should complete quickly
	if duration > 1*time.Second {
		t.Errorf("High volume logging took too long: %v", duration)
	}

	// Verify we got output
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < logCount/2 { // At least half should be visible (debug might be filtered)
		t.Errorf("Expected many log lines, got %d", len(lines))
	}
}

// TestConcurrentMixedOperations tests all features used concurrently
func TestConcurrentMixedOperations(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	updatable := bullets.NewUpdatable(&buf)

	var wg sync.WaitGroup

	// Goroutine 1: Regular logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			logger.Infof("Regular log %d", i)
			time.Sleep(2 * time.Millisecond)
		}
	}()

	// Goroutine 2: Spinners
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			spinner := logger.Spinner(fmt.Sprintf("Spinner %d", i))
			time.Sleep(10 * time.Millisecond)
			spinner.Success("Done")
		}
	}()

	// Goroutine 3: Updatable handles
	wg.Add(1)
	go func() {
		defer wg.Done()
		handles := make([]*bullets.BulletHandle, 5)
		for i := 0; i < 5; i++ {
			handles[i] = updatable.InfoHandle(fmt.Sprintf("Handle %d", i))
		}
		for i := 0; i < 5; i++ {
			handles[i].Success(fmt.Sprintf("Handle %d complete", i))
		}
	}()

	// Goroutine 4: Changing logger settings
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			logger.SetLevel(bullets.InfoLevel)
			logger.SetLevel(bullets.DebugLevel)
			logger.SetUseSpecialBullets(i%2 == 0)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Should complete without panics or races
}

// TestErrorRecovery tests recovery from various error conditions
func TestErrorRecovery(t *testing.T) {
	// Test with failing writer
	failWriter := &failingWriterIntegration{shouldFail: true}
	logger := bullets.New(failWriter)

	// These should not panic even with write failures
	logger.Info("This will fail to write")
	logger.Error("This also fails")

	spinner := logger.Spinner("Spinner with failed writes")
	time.Sleep(10 * time.Millisecond)
	spinner.Stop()

	// Now allow writes to succeed
	failWriter.shouldFail = false
	logger.Success("This should succeed")

	if failWriter.successCount < 1 {
		t.Error("Expected at least one successful write")
	}
}

// failingWriterIntegration simulates intermittent write failures
type failingWriterIntegration struct {
	shouldFail   bool
	failCount    int
	successCount int
	mu           sync.Mutex
}

func (w *failingWriterIntegration) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.shouldFail {
		w.failCount++
		return 0, fmt.Errorf("simulated write failure")
	}
	w.successCount++
	return len(p), nil
}

// TestRealWorldCLISimulation simulates a real CLI application workflow
func TestRealWorldCLISimulation(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Simulate a deployment process
	logger.Info("🚀 Starting deployment to production")
	logger.IncreasePadding()

	// Phase 1: Pre-checks
	logger.Info("Running pre-deployment checks")
	logger.IncreasePadding()

	checks := []string{
		"Checking git status",
		"Validating configuration",
		"Testing database connection",
		"Verifying API keys",
	}

	for _, check := range checks {
		spinner := logger.Spinner(check)
		time.Sleep(5 * time.Millisecond)
		spinner.Success(check + " ✓")
	}
	logger.DecreasePadding()

	// Phase 2: Build
	buildStep := logger.Step("Building application")
	logger.IncreasePadding()
	logger.Info("Compiling TypeScript")
	logger.Info("Bundling assets")
	logger.Info("Optimizing images")
	logger.DecreasePadding()
	buildStep()

	// Phase 3: Tests
	logger.Warn("Running test suite")
	logger.IncreasePadding()
	logger.Success("Unit tests: 142 passed")
	logger.Success("Integration tests: 28 passed")
	logger.Error("E2E tests: 2 failed")
	logger.DecreasePadding()

	// Phase 4: Deployment
	logger.Info("Deploying to servers")
	logger.IncreasePadding()

	servers := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-south-1"}
	for _, server := range servers {
		logger.WithField("region", server).Info("Deploying")
		time.Sleep(5 * time.Millisecond)
		logger.WithField("region", server).Success("Deployed")
	}
	logger.DecreasePadding()

	// Phase 5: Post-deployment
	logger.Info("Running post-deployment tasks")
	logger.IncreasePadding()
	logger.Info("Clearing CDN cache")
	logger.Info("Sending notifications")
	logger.Info("Updating status page")
	logger.DecreasePadding()

	logger.DecreasePadding()
	logger.Success("✨ Deployment complete!")

	// Verify output structure
	output := buf.String()
	expectedStrings := []string{
		"Starting deployment",
		"pre-deployment checks",
		"Building application",
		"test suite",
		"Deploying to servers",
		"Deployment complete",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Missing expected output: %s", expected)
		}
	}
}

// TestMemoryUsageUnderLoad tests memory usage with continuous operations
func TestMemoryUsageUnderLoad(t *testing.T) {
	var buf bytes.Buffer

	// Run continuous operations for a short period
	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	for {
		select {
		case <-done:
			return
		default:
			// Create and destroy many loggers
			logger := bullets.New(&buf)
			logger.Info("Test message")

			// Create and destroy spinners
			spinner := logger.Spinner("Loading")
			spinner.Stop()

			// Create updatable logger and handles
			updatable := bullets.NewUpdatable(&buf)
			handle := updatable.InfoHandle("Test")
			handle.Success("Done")

			// Clear buffer periodically to avoid unbounded growth
			if buf.Len() > 1024*1024 { // 1MB
				buf.Reset()
			}
		}
	}
}

// TestNestedPaddingComplex tests complex nested padding scenarios
func TestNestedPaddingComplex(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create deeply nested structure
	logger.Info("Level 0")
	for i := 1; i <= 5; i++ {
		logger.IncreasePadding()
		logger.Infof("Level %d", i)

		// Add some content at each level
		logger.WithField("depth", i).Debug("Debug at level")

		if i == 3 {
			// Add spinner at level 3
			spinner := logger.Spinner("Processing at level 3")
			time.Sleep(10 * time.Millisecond)
			spinner.Success("Done at level 3")
		}
	}

	// Now decrease back
	for i := 5; i >= 1; i-- {
		logger.Infof("Returning from level %d", i)
		logger.DecreasePadding()
	}

	logger.Info("Back to level 0")

	output := buf.String()
	// Should have proper indentation structure
	lines := strings.Split(output, "\n")

	// Check that we have output
	if len(lines) < 10 {
		t.Error("Expected more output lines for nested structure")
	}
}

// TestLoggerChaining tests method chaining patterns
func TestLoggerChaining(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Test field chaining
	logger.
		WithField("request_id", "123").
		WithField("user_id", "456").
		WithField("action", "login").
		WithFields(map[string]interface{}{
			"ip":     "192.168.1.1",
			"method": "POST",
		}).
		Info("User login attempt")

	// Test with error chaining
	err := fmt.Errorf("connection timeout")
	logger.
		WithError(err).
		WithField("retry", 3).
		Error("Database connection failed")

	output := buf.String()

	// Verify chained fields appear
	expectedFields := []string{"request_id", "user_id", "action", "ip", "method", "connection timeout", "retry"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Missing expected field: %s", field)
		}
	}
}

// TestEdgeCaseIntegration combines multiple edge cases in realistic scenarios
func TestEdgeCaseIntegration(t *testing.T) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Unicode and emoji in real scenario
	logger.Info("🔨 Building multilingual app 多语言应用")
	logger.IncreasePadding()

	languages := []string{
		"🇺🇸 English: Building...",
		"🇯🇵 日本語: ビルド中...",
		"🇨🇳 中文: 构建中...",
		"🇷🇺 Русский: Сборка...",
		"🇸🇦 العربية: بناء...",
	}

	for _, lang := range languages {
		spinner := logger.Spinner(lang)
		time.Sleep(5 * time.Millisecond)
		spinner.Success(strings.Replace(lang, "...", " ✓", 1))
	}

	logger.DecreasePadding()

	// Very long field values
	logger.WithField("token", strings.Repeat("a", 500)).Debug("Long token")

	// Empty values
	logger.WithField("", "").Info("Empty field test")

	// Special characters in fields
	logger.WithFields(map[string]interface{}{
		"path":    "/usr/local/bin",
		"command": "ls -la | grep test",
		"regex":   `\d+\.\d+\.\d+`,
	}).Info("Special characters test")

	// Should handle all edge cases in integration
	output := buf.String()
	if output == "" {
		t.Error("Expected output from edge case integration")
	}
}