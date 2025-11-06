// Package main demonstrates high-concurrency spinner usage with the bullets logger.
// This stress test shows the coordinator handling 15 simultaneous spinners with
// varying completion times and outcomes.
//
// Run with: BULLETS_FORCE_TTY=1 go run examples/spinner_stress/main.go
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sgaunet/bullets"
)

// Task represents a simulated work item with duration and outcome.
type Task struct {
	ID       int
	Name     string
	Duration time.Duration
	Success  bool
}

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)

	logger.Info("High-Concurrency Spinner Stress Test")
	logger.IncreasePadding()

	// Track overall timing
	startTime := time.Now()

	// Demo 1: 15 concurrent spinners with random durations
	logger.Info("Starting 15 concurrent operations...")
	logger.IncreasePadding()

	tasks := generateTasks(15) //nolint:mnd // Stress test with 15 spinners

	// BEST PRACTICE: Create all spinners upfront
	// This allows the coordinator to efficiently allocate line numbers
	spinners := make([]*bullets.Spinner, len(tasks))
	for i, task := range tasks {
		spinners[i] = logger.Spinner(task.Name)
	}

	logger.DecreasePadding()
	logger.Info("All spinners created, starting work...")

	// Execute all tasks concurrently
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for i, task := range tasks {
		go executeTask(task, spinners[i], &wg)
	}

	// Wait for all tasks to complete
	wg.Wait()

	elapsed := time.Since(startTime)
	logger.DecreasePadding()
	logger.Success(fmt.Sprintf("All operations completed in %v", elapsed.Round(time.Millisecond)))

	// Statistics
	logger.Info("Performance Statistics:")
	logger.IncreasePadding()
	successful := 0
	failed := 0
	for _, task := range tasks {
		if task.Success {
			successful++
		} else {
			failed++
		}
	}
	logger.Info(fmt.Sprintf("Total tasks: %d", len(tasks)))
	logger.Info(fmt.Sprintf("Successful: %d", successful))
	logger.Info(fmt.Sprintf("Failed: %d", failed))
	logger.Info(fmt.Sprintf("Total time: %v", elapsed.Round(time.Millisecond)))
	logger.DecreasePadding()
}

// generateTasks creates a set of simulated tasks with varying characteristics.
func generateTasks(count int) []Task {
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // Example code, not cryptographic

	tasks := make([]Task, count)
	for i := 0; i < count; i++ { //nolint:intrange // Backward compatibility with older Go versions
		// Random duration between 100ms and 3s
		duration := time.Duration(100+rng.Intn(2900)) * time.Millisecond //nolint:mnd // Random task duration

		// 80% success rate
		success := rng.Float32() < 0.8 //nolint:mnd // 80% success rate

		tasks[i] = Task{
			ID:       i + 1,
			Name:     fmt.Sprintf("Task %02d", i+1),
			Duration: duration,
			Success:  success,
		}
	}

	return tasks
}

// executeTask simulates work and completes the spinner with appropriate status.
func executeTask(task Task, spinner *bullets.Spinner, wg *sync.WaitGroup) {
	defer wg.Done()

	// Simulate work
	time.Sleep(task.Duration)

	// Complete with appropriate status
	if task.Success {
		spinner.Success(fmt.Sprintf("%s completed (%v)", task.Name, task.Duration.Round(10*time.Millisecond))) //nolint:mnd // Rounding milliseconds
	} else {
		spinner.Error(fmt.Sprintf("%s failed after %v", task.Name, task.Duration.Round(10*time.Millisecond))) //nolint:mnd // Rounding milliseconds
	}
}
