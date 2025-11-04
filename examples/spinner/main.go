// Package main demonstrates the basic usage of the bullets logger.
package main

import (
	"os"
	"time"

	"github.com/sgaunet/bullets"
)

const (
	sleep2 = 2 * time.Second
)

func main() {
	// Create a new logger
	logger := bullets.New(os.Stdout)

	// Basic logging
	logger.Info("starting build process")
	logger.IncreasePadding()

	// Circle spinner with error
	spinner := logger.SpinnerCircle("connecting to database")
	spinner2 := logger.SpinnerCircle("connecting to database2")
	time.Sleep(sleep2)
	spinner2.Error("connection to database2")
	spinner.Success("connection to database: OK")
}
