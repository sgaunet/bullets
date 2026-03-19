// Package main demonstrates context.Context support for spinner cancellation and timeouts.
//
// Run with: BULLETS_FORCE_TTY=1 go run examples/context/main.go
package main

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/sgaunet/bullets"
)

func main() {
	logger := bullets.New(os.Stdout)
	logger.Info("Context support demonstrations")

	// Demo 1: Timeout cancellation
	demoTimeout(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause

	// Demo 2: Manual cancellation
	demoManualCancel(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause

	// Demo 3: Shared context across multiple spinners
	demoSharedContext(logger)

	time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo pause

	// Demo 4: Mixed manual and context completion
	demoMixedCompletion(logger)

	logger.Success("All context demos completed")
}

// demoTimeout shows a spinner that auto-cancels after a timeout.
func demoTimeout(logger *bullets.Logger) {
	logger.Info("Demo: Timeout cancellation (2s timeout)")
	logger.IncreasePadding()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) //nolint:mnd // Demo timeout
	defer cancel()

	_ = logger.Spinner(ctx, "Long operation (will timeout after 2s)")

	// Wait for the timeout to trigger
	<-ctx.Done()
	time.Sleep(200 * time.Millisecond) //nolint:mnd // Allow rendering to complete

	logger.DecreasePadding()
	logger.Info("Timeout demo complete")
}

// demoManualCancel shows explicit context cancellation.
func demoManualCancel(logger *bullets.Logger) {
	logger.Info("Demo: Manual context cancellation")
	logger.IncreasePadding()

	ctx, cancel := context.WithCancel(context.Background())

	_ = logger.SpinnerCircle(ctx, "Waiting for cancel signal")

	// Simulate some work then cancel
	time.Sleep(1500 * time.Millisecond) //nolint:mnd // Demo timing
	cancel()
	time.Sleep(200 * time.Millisecond) //nolint:mnd // Allow rendering to complete

	logger.DecreasePadding()
	logger.Info("Manual cancel demo complete")
}

// demoSharedContext shows multiple spinners sharing one context.
func demoSharedContext(logger *bullets.Logger) {
	logger.Info("Demo: Shared context (3s timeout, 3 spinners)")
	logger.IncreasePadding()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) //nolint:mnd // Demo timeout
	defer cancel()

	s1 := logger.SpinnerDots(ctx, "Database migration")
	s2 := logger.SpinnerCircle(ctx, "API sync")
	s3 := logger.SpinnerBounce(ctx, "Cache rebuild")

	// Complete s1 manually before timeout
	var wg sync.WaitGroup
	wg.Go(func() {
		time.Sleep(1 * time.Second)
		s1.Success("Database migration complete")
	})

	wg.Wait()

	// s2 and s3 will be auto-cancelled when the 3s timeout expires
	<-ctx.Done()
	time.Sleep(200 * time.Millisecond) //nolint:mnd // Allow rendering to complete

	// Safe to call after cancellation
	s2.Success("late call")
	s3.Success("late call")

	logger.DecreasePadding()
	logger.Info("Shared context demo complete")
}

// demoMixedCompletion shows spinners with mixed manual and context-driven completion.
func demoMixedCompletion(logger *bullets.Logger) {
	logger.Info("Demo: Mixed completion (manual + context)")
	logger.IncreasePadding()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) //nolint:mnd // Demo timeout
	defer cancel()

	s1 := logger.Spinner(ctx, "Fast task (manual success)")
	s2 := logger.Spinner(ctx, "Slow task (will timeout)")
	s3 := logger.Spinner(ctx, "Medium task (manual error)")

	var wg sync.WaitGroup
	wg.Add(2) //nolint:mnd // Number of manual completions

	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond) //nolint:mnd // Demo timing
		s1.Success("Fast task done")
	}()

	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second)
		s3.Error("Medium task failed")
	}()

	wg.Wait()

	// s2 will be auto-cancelled when the 2s timeout expires
	<-ctx.Done()
	time.Sleep(200 * time.Millisecond) //nolint:mnd // Allow rendering to complete

	s2.Success("late call")

	logger.DecreasePadding()
	logger.Info("Mixed completion demo complete")
}
