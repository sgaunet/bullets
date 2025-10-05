# Bullets

A colorful terminal logger for Go with bullet-style output, inspired by [goreleaser](https://github.com/goreleaser/goreleaser)'s beautiful CLI output.

## Features

- 🎨 Colorful terminal output with ANSI colors
- 🔘 Bullet-style logging with different symbols for different levels
- 📊 Support for log levels (Debug, Info, Warn, Error, Fatal)
- 📝 Structured logging with fields
- ⏱️  Timing information for long-running operations
- 🔄 Indentation/padding support for nested operations
- 🧵 Thread-safe operations
- 🚀 Zero external dependencies (stdlib only)

## Installation

```bash
go get github.com/sgaunet/bullets
```

## Quick Start

```go
package main

import (
    "os"
    "github.com/sgaunet/bullets"
)

func main() {
    logger := bullets.New(os.Stdout)

    logger.Info("building")
    logger.IncreasePadding()
    logger.Info("binary=dist/app_linux_amd64")
    logger.Info("binary=dist/app_darwin_amd64")
    logger.DecreasePadding()

    logger.Success("build succeeded")
}
```

## Usage

### Basic Logging

```go
logger := bullets.New(os.Stdout)

logger.Debug("debug message")    // ◦ debug message (dim)
logger.Info("info message")      // • info message (cyan)
logger.Warn("warning message")   // ⚠ warning message (yellow)
logger.Error("error message")    // ✗ error message (red)
logger.Success("success!")       // ✓ success! (green)
```

### Formatted Messages

```go
logger.Infof("processing %d items", count)
logger.Warnf("retry %d/%d", current, total)
```

### Structured Logging

```go
// Single field
logger.WithField("user", "john").Info("logged in")

// Multiple fields
logger.WithFields(map[string]interface{}{
    "version": "1.2.3",
    "arch":    "amd64",
}).Info("building package")

// Error field
err := errors.New("connection timeout")
logger.WithError(err).Error("upload failed")
```

### Indentation

```go
logger.Info("main task")
logger.IncreasePadding()
    logger.Info("subtask 1")
    logger.Info("subtask 2")
    logger.IncreasePadding()
        logger.Info("nested subtask")
    logger.DecreasePadding()
logger.DecreasePadding()
```

### Step Function with Timing

The `Step` function is useful for tracking operations with automatic timing:

```go
done := logger.Step("running tests")
// ... do work ...
done() // Automatically logs completion with duration if > 10s
```

### Log Levels

```go
logger := bullets.New(os.Stdout)
logger.SetLevel(bullets.WarnLevel) // Only warn, error, and fatal will be logged

logger.Debug("not shown")
logger.Info("not shown")
logger.Warn("this is shown")
logger.Error("this is shown")
```

Available levels:
- `DebugLevel`
- `InfoLevel` (default)
- `WarnLevel`
- `ErrorLevel`
- `FatalLevel`

## Example Output

```
• building
  • binary=dist/app_linux_amd64
  • binary=dist/app_darwin_amd64
  • binary=dist/app_windows_amd64
• archiving
  • binary=app name=app_0.2.1_linux_amd64
  • binary=app name=app_0.2.1_darwin_amd64
• calculating checksums
✓ release succeeded
```

## Running the Example

```bash
cd examples/basic
go run main.go
```

## Comparison with Other Loggers

This library is designed specifically for CLI applications that need beautiful, human-readable output. It's inspired by:

- [caarlos0/log](https://github.com/caarlos0/log) - The logger used by goreleaser
- [apex/log](https://github.com/apex/log) - The original apex logger

Unlike general-purpose loggers, `bullets` focuses on:
- Visual appeal for terminal output
- Simple API for CLI applications
- Zero configuration needed
- No external dependencies

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

Inspired by the beautiful CLI output of [goreleaser](https://github.com/goreleaser/goreleaser) and [caarlos0/log](https://github.com/caarlos0/log).
