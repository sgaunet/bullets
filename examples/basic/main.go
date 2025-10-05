package main

import (
	"errors"
	"os"
	"time"

	"github.com/sgaunet/bullets"
)

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)

	// Basic logging
	logger.Info("starting build process")
	logger.IncreasePadding()

	// Simulating a build process similar to goreleaser
	simulateTask(logger, "building", []string{
		"binary=dist/app_linux_amd64",
		"binary=dist/app_darwin_amd64",
		"binary=dist/app_windows_amd64.exe",
	})

	simulateTask(logger, "archiving", []string{
		"binary=app name=app_0.2.1_linux_amd64",
		"binary=app name=app_0.2.1_darwin_amd64",
		"binary=app name=app_0.2.1_windows_amd64",
	})

	logger.Info("calculating checksums")
	time.Sleep(500 * time.Millisecond)

	logger.Info("writing artifacts metadata")
	time.Sleep(300 * time.Millisecond)

	logger.DecreasePadding()
	logger.Success("release succeeded after 3s")

	// Demonstrate different log levels
	logger.Info("")
	logger.Info("demonstrating log levels:")
	logger.IncreasePadding()

	logger.Debug("this is a debug message")
	logger.Info("this is an info message")
	logger.Warn("this is a warning message")
	logger.Error("this is an error message")

	logger.DecreasePadding()

	// Demonstrate structured logging with fields
	logger.Info("")
	logger.Info("structured logging example:")
	logger.IncreasePadding()

	logger.WithField("user", "john").Info("user logged in")
	logger.WithFields(map[string]interface{}{
		"version": "1.2.3",
		"arch":    "amd64",
	}).Info("building package")

	err := errors.New("connection timeout")
	logger.WithError(err).Error("upload failed")

	logger.DecreasePadding()

	// Demonstrate Step function with timing
	logger.Info("")
	logger.Info("step function with timing:")
	done := logger.Step("processing large dataset")
	time.Sleep(2 * time.Second) // Simulate work
	done()                       // This will log completion

	// Another step that takes longer
	done = logger.Step("running integration tests")
	time.Sleep(11 * time.Second) // Simulate longer work
	done()                        // This will include duration info

	logger.Info("")
	logger.Success("all examples completed!")
}

func simulateTask(logger *bullets.Logger, taskName string, items []string) {
	logger.Info(taskName)
	logger.IncreasePadding()
	for _, item := range items {
		logger.Info(item)
		time.Sleep(300 * time.Millisecond)
	}
	logger.DecreasePadding()
}
