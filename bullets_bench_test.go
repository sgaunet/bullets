package bullets_test

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sgaunet/bullets"
)

// Benchmark basic logging operations
func BenchmarkLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message")
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithField("key", "value").Info("Benchmark message")
	}
}

func BenchmarkLogger_InfoWithMultipleFields(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	fields := map[string]interface{}{
		"request_id": "12345",
		"user_id":    67890,
		"action":     "login",
		"timestamp":  time.Now().Unix(),
		"status":     "success",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields).Info("Benchmark message")
	}
}

// Benchmark formatted logging
func BenchmarkLogger_Infof(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof("Benchmark message %d with %s", i, "formatting")
	}
}

// Benchmark different log levels
func BenchmarkLogger_AllLevels(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	logger.SetLevel(bullets.DebugLevel)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 5 {
		case 0:
			logger.Debug("Debug message")
		case 1:
			logger.Info("Info message")
		case 2:
			logger.Warn("Warning message")
		case 3:
			logger.Error("Error message")
		case 4:
			logger.Success("Success message")
		}
	}
}

// Benchmark concurrent logging
func BenchmarkLogger_Concurrent(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Concurrent benchmark message")
		}
	})
}

func BenchmarkLogger_ConcurrentWithFields(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.WithField("goroutine_id", i).Info("Concurrent message")
			i++
		}
	})
}

// Benchmark padding operations
func BenchmarkLogger_Padding(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.IncreasePadding()
		logger.Info("Indented message")
		logger.DecreasePadding()
	}
}

func BenchmarkLogger_DeepPadding(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Set up deep padding
	for i := 0; i < 10; i++ {
		logger.IncreasePadding()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Deeply indented message")
	}

	// Clean up
	for i := 0; i < 10; i++ {
		logger.DecreasePadding()
	}
}

// Benchmark special bullets
func BenchmarkLogger_SpecialBullets(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	logger.SetUseSpecialBullets(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Success("Success with special bullet")
		logger.Error("Error with special bullet")
		logger.Warn("Warning with special bullet")
	}
}

// Benchmark updatable logger
func BenchmarkUpdatableLogger_Create(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger := bullets.NewUpdatable(&buf)
		_ = logger
	}
}

func BenchmarkUpdatableLogger_InfoHandle(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle := logger.InfoHandle("Benchmark handle")
		_ = handle
	}
}

func BenchmarkBulletHandle_Update(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Initial message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle.Update(bullets.InfoLevel, fmt.Sprintf("Updated message %d", i))
	}
}

func BenchmarkBulletHandle_Progress(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Progress test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle.Progress(i%101, 100)
	}
}

func BenchmarkBulletHandle_WithFields(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)
	handle := logger.InfoHandle("Field test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle.WithField("key", i).WithField("value", fmt.Sprintf("val_%d", i))
	}
}

// Benchmark handle groups
func BenchmarkHandleGroup_UpdateAll(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create group with multiple handles
	handles := make([]*bullets.BulletHandle, 10)
	for i := 0; i < 10; i++ {
		handles[i] = logger.InfoHandle(fmt.Sprintf("Handle %d", i))
	}
	group := bullets.NewHandleGroup(handles...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group.UpdateAll(bullets.InfoLevel, fmt.Sprintf("Update %d", i))
	}
}

func BenchmarkHandleGroup_Large(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create large group
	handles := make([]*bullets.BulletHandle, 100)
	for i := 0; i < 100; i++ {
		handles[i] = logger.InfoHandle(fmt.Sprintf("Handle %d", i))
	}
	group := bullets.NewHandleGroup(handles...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		group.UpdateAll(bullets.InfoLevel, "Mass update")
	}
}

// Benchmark batch operations
func BenchmarkBatchUpdate(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create handles
	handles := make([]*bullets.BulletHandle, 10)
	for i := 0; i < 10; i++ {
		handles[i] = logger.InfoHandle(fmt.Sprintf("Handle %d", i))
	}

	// Prepare updates
	updates := make(map[*bullets.BulletHandle]struct {
		Level   bullets.Level
		Message string
	})
	for i, h := range handles {
		updates[h] = struct {
			Level   bullets.Level
			Message string
		}{
			Level:   bullets.InfoLevel,
			Message: fmt.Sprintf("Batch update %d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bullets.BatchUpdate(handles, updates)
	}
}

// Benchmark spinners
func BenchmarkLogger_Spinner(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spinner := logger.Spinner("Benchmark spinner")
		spinner.Stop()
	}
}

func BenchmarkSpinner_LifeCycle(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spinner := logger.Spinner("Loading")
		time.Sleep(1 * time.Millisecond)
		spinner.Success("Complete")
	}
}

func BenchmarkSpinner_CustomFrames(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spinner := logger.SpinnerWithFrames("Custom", frames)
		spinner.Stop()
	}
}

// Benchmark memory allocations
func BenchmarkLogger_MemoryAllocations(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithField("id", i).
			WithField("status", "active").
			Info("Allocation test")
	}
}

func BenchmarkUpdatable_MemoryAllocations(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handle := logger.InfoHandle("Test")
		handle.Update(bullets.WarnLevel, "Updated")
		handle.Success("Done")
	}
}

// Benchmark large messages
func BenchmarkLogger_LargeMessages(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create a large message (1KB)
	largeMsg := make([]byte, 1024)
	for i := range largeMsg {
		largeMsg[i] = 'a'
	}
	msg := string(largeMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(msg)
	}
}

func BenchmarkLogger_VeryLargeMessages(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create a very large message (10KB)
	veryLargeMsg := make([]byte, 10240)
	for i := range veryLargeMsg {
		veryLargeMsg[i] = 'b'
	}
	msg := string(veryLargeMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(msg)
	}
}

// Benchmark writer performance
func BenchmarkLogger_FastWriter(b *testing.B) {
	writer := &nullWriter{}
	logger := bullets.New(writer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark with fast writer")
	}
}

func BenchmarkLogger_SlowWriter(b *testing.B) {
	writer := &slowWriter{delay: 1 * time.Microsecond}
	logger := bullets.New(writer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark with slow writer")
	}
}

// Benchmark concurrent updatable operations
func BenchmarkUpdatable_Concurrent(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create handles
	handles := make([]*bullets.BulletHandle, 10)
	for i := 0; i < 10; i++ {
		handles[i] = logger.InfoHandle(fmt.Sprintf("Handle %d", i))
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			handle := handles[i%10]
			handle.Update(bullets.InfoLevel, fmt.Sprintf("Update %d", i))
			i++
		}
	})
}

// Benchmark step function
func BenchmarkLogger_Step(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		step := logger.Step("Processing")
		// Simulate some work
		time.Sleep(100 * time.Nanosecond)
		step()
	}
}

// Benchmark chain operations
func BenchmarkChain_Operations(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.NewUpdatable(&buf)

	// Create handles
	h1 := logger.InfoHandle("Chain 1")
	h2 := logger.InfoHandle("Chain 2")
	h3 := logger.InfoHandle("Chain 3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain := bullets.Chain(h1, h2, h3)
		chain.Update(bullets.InfoLevel, fmt.Sprintf("Chain update %d", i))
	}
}

// Helper types for benchmarks
type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type slowWriter struct {
	delay time.Duration
}

func (w *slowWriter) Write(p []byte) (n int, err error) {
	time.Sleep(w.delay)
	return len(p), nil
}

// Benchmark mixed operations
func BenchmarkMixedOperations(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	updatable := bullets.NewUpdatable(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Regular logging
		logger.Info("Regular log")
		logger.WithField("id", i).Debug("Debug log")

		// Updatable handle
		handle := updatable.InfoHandle("Updatable")
		handle.Update(bullets.WarnLevel, "Updated")

		// Spinner
		spinner := logger.Spinner("Processing")
		spinner.Stop()
	}
}

// Benchmark worst-case scenarios
func BenchmarkWorstCase_ManyFields(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create logger with many fields
	fields := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		fields[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(fields).Info("Many fields")
	}
}

func BenchmarkWorstCase_DeepNesting(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	// Create deep nesting
	for i := 0; i < 20; i++ {
		logger.IncreasePadding()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Deeply nested")
	}

	// Clean up
	for i := 0; i < 20; i++ {
		logger.DecreasePadding()
	}
}

func BenchmarkWorstCase_ConcurrentStress(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var wg sync.WaitGroup
			for j := 0; j < 10; j++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					logger.WithField("goroutine", id).Info("Stress test")
				}(j)
			}
			wg.Wait()
		}
	})
}

// Benchmark comparison: standard vs special bullets
func BenchmarkLogger_StandardBullets(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	logger.SetUseSpecialBullets(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Success("Success with standard bullet")
		logger.Error("Error with standard bullet")
		logger.Warn("Warning with standard bullet")
	}
}

func BenchmarkLogger_SpecialBulletsComparison(b *testing.B) {
	var buf bytes.Buffer
	logger := bullets.New(&buf)
	logger.SetUseSpecialBullets(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Success("Success with special bullet")
		logger.Error("Error with special bullet")
		logger.Warn("Warning with special bullet")
	}
}

// Benchmark throughput
func BenchmarkLogger_Throughput(b *testing.B) {
	writer := io.Discard
	logger := bullets.New(writer)

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("Throughput test message")
	}

	b.StopTimer()
	elapsed := time.Since(start)
	throughput := float64(b.N) / elapsed.Seconds()
	b.Logf("Throughput: %.0f messages/second", throughput)
}

func BenchmarkUpdatable_Throughput(b *testing.B) {
	writer := io.Discard
	logger := bullets.NewUpdatable(writer)

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handle := logger.InfoHandle("Throughput test")
		handle.Update(bullets.InfoLevel, "Updated")
	}

	b.StopTimer()
	elapsed := time.Since(start)
	throughput := float64(b.N) / elapsed.Seconds()
	b.Logf("Throughput: %.0f updates/second", throughput)
}