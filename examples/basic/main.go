// Package main demonstrates the basic usage of the bullets logger.
package main

import (
	"errors"
	"os"
	"time"

	"github.com/sgaunet/bullets"
)

const (
	sleepShort  = 300 * time.Millisecond
	sleepMedium = 500 * time.Millisecond
	sleep2      = 2 * time.Second
	sleep3      = 3 * time.Second
	sleep11     = 11 * time.Second
)

var errConnectionTimeout = errors.New("connection timeout")

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
	time.Sleep(sleepMedium)

	logger.Info("writing artifacts metadata")
	time.Sleep(sleepShort)

	logger.DecreasePadding()
	logger.Success("release succeeded after 3s")

	// Demonstrate different log levels
	logger.Info("demonstrating log levels:")
	logger.IncreasePadding()

	logger.Debug("this is a debug message")
	logger.Info("this is an info message")
	logger.Warn("this is a warning message")
	logger.Error("this is an error message")

	logger.DecreasePadding()

	// Demonstrate structured logging with fields
	logger.Info("structured logging example:")
	logger.IncreasePadding()

	logger.WithField("user", "john").Info("user logged in")
	logger.WithFields(map[string]interface{}{
		"version": "1.2.3",
		"arch":    "amd64",
	}).Info("building package")

	logger.WithError(errConnectionTimeout).Error("upload failed")

	logger.DecreasePadding()

	// Demonstrate Step function with timing
	logger.Info("step function with timing:")
	done := logger.Step("processing large dataset")
	time.Sleep(sleep2) // Simulate work
	done()             // This will log completion

	// Another step that takes longer
	done = logger.Step("running integration tests")
	time.Sleep(sleep11) // Simulate longer work
	done()              // This will include duration info

	// Demonstrate spinner functionality
	logger.Info("")
	logger.Info("spinner examples:")
	logger.IncreasePadding()

	// Default Braille spinner with success
	spinner := logger.Spinner("downloading files")
	time.Sleep(sleep3)
	spinner.Success("downloaded 10 files")

	// Circle spinner with error
	spinner = logger.SpinnerCircle("connecting to database")
	spinner2 := logger.SpinnerCircle("connecting to database2")
	time.Sleep(sleep2)
	spinner.Error("connection failed")
	spinner2.Success("connection OK")

	// Bounce spinner with custom completion
	spinner = logger.SpinnerBounce("processing data")
	time.Sleep(sleep2)
	spinner.Replace("processed 1000 records")

	// Dots spinner (same as default)
	spinner = logger.SpinnerDots("installing packages")
	time.Sleep(sleep2)
	spinner.Success("packages installed")

	// Custom frames spinner
	spinner = logger.SpinnerWithFrames("compiling code", []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"})
	time.Sleep(sleep2)
	spinner.Success("compilation complete")

	logger.DecreasePadding()

	logger.Info("")
	logger.Success("all examples completed!")
}

func simulateTask(logger *bullets.Logger, taskName string, items []string) {
	logger.Info(taskName)
	logger.IncreasePadding()
	for _, item := range items {
		logger.Info(item)
		time.Sleep(sleepShort)
	}
	logger.DecreasePadding()
}
