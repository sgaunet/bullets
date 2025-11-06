// Package main demonstrates the basic usage of the bullets logger.
package main

import (
	"os"
	"time"

	"github.com/sgaunet/bullets"
)

const (
	sleep2 = 2 * time.Second
	sleep3 = 3 * time.Second
)

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)

	// Demonstrate spinner functionality
	logger.Info("spinner examples:")
	// logger.IncreasePadding()

	// Default Braille spinner with success
	// spinner := logger.Spinner("downloading files")
	// time.Sleep(sleep3)
	// spinner.Success("downloaded 10 files")

	// Circle spinner with error
	spinner := logger.SpinnerCircle("connecting to database")
	// // spinner2 := logger.SpinnerCircle("connecting to database2")
	time.Sleep(sleep2)
	spinner.Error("connection failed")
	// spinner2.Success("connection OK")

	// Bounce spinner with custom completion
	// spinner = logger.SpinnerBounce("processing data")
	// time.Sleep(sleep2)
	// spinner.Replace("processed 1000 records")

	// Dots spinner (same as default)
	spinner = logger.SpinnerDots("installing packages")
	time.Sleep(sleep2)
	spinner.Success("packages installed")

	// Custom frames
	// spinner = logger.SpinnerWithFrames("compiling code", []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"})
	// time.Sleep(sleep2)
	// spinner.Success("compilation complete")

	// logger.DecreasePadding()

	logger.Success("all examples completed!")
}
