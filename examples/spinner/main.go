// Package main demonstrates concurrent spinner usage with the bullets logger.
// This example shows how multiple spinners can run simultaneously with automatic coordination.
//
// Run with: BULLETS_FORCE_TTY=1 go run examples/spinner/main.go
package main

import (
	"os"
	"sync"
	"time"

	"github.com/sgaunet/bullets"
)

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)

	logger.Info("Starting parallel operations demo")
	logger.IncreasePadding()

	// Demo 1: Concurrent spinners with different completion methods
	demoBasicConcurrent(logger)

	logger.DecreasePadding()
	logger.Success("All demos completed")
}

// demoBasicConcurrent demonstrates multiple spinners running concurrently
func demoBasicConcurrent(logger *bullets.Logger) {
	logger.Info("Demo: Concurrent operations with different outcomes")
	logger.IncreasePadding()

	// Start multiple spinners with different animation styles
	dbSpinner := logger.SpinnerDots("Connecting to database")
	apiSpinner := logger.SpinnerCircle("Fetching API data")
	fileSpinner := logger.SpinnerBounce("Processing files")
	cacheSpinner := logger.Spinner("Warming up cache")

	// Use WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(4)

	// Simulate database connection (completes successfully after 2s)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		dbSpinner.Success("Database connected")
	}()

	// Simulate API fetch (completes successfully after 3s)
	go func() {
		defer wg.Done()
		time.Sleep(3 * time.Second)
		apiSpinner.Success("API data fetched (1.2MB)")
	}()

	// Simulate file processing (fails after 1s)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		fileSpinner.Error("File processing failed: permission denied")
	}()

	// Simulate cache warming (completes with custom message after 1.5s)
	go func() {
		defer wg.Done()
		time.Sleep(1500 * time.Millisecond)
		cacheSpinner.Replace("Cache warmed: 1000 entries loaded")
	}()

	// Wait for all operations to complete
	wg.Wait()

	logger.DecreasePadding()
	logger.Info("Concurrent operations demo complete")
	logger.Info("")
}
