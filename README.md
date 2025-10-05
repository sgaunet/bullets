# Bullets

A colorful terminal logger for Go with bullet-style output, inspired by [goreleaser](https://github.com/goreleaser/goreleaser)'s beautiful CLI output.

## Features

- 🎨 Colorful terminal output with ANSI colors
- 🔘 Configurable bullet symbols (default circles, optional special symbols, custom icons)
- 📊 Support for log levels (Debug, Info, Warn, Error, Fatal)
- 📝 Structured logging with fields
- ⏱️  Timing information for long-running operations
- 🔄 Indentation/padding support for nested operations
- ⏳ Animated spinners with multiple styles (Braille, Circle, Bounce)
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

// By default, all levels use colored bullets (•)
logger.Debug("debug message")    // ○ debug message (dim)
logger.Info("info message")      // • info message (cyan)
logger.Warn("warning message")   // • warning message (yellow)
logger.Error("error message")    // • error message (red)
logger.Success("success!")       // • success! (green)
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

### Spinners

Animated spinners for long-running operations:

```go
// Default Braille spinner (smooth dots)
spinner := logger.Spinner("downloading files")
time.Sleep(3 * time.Second)
spinner.Success("downloaded 10 files")

// Circle spinner (rotating circle)
spinner = logger.SpinnerCircle("connecting to database")
time.Sleep(2 * time.Second)
spinner.Error("connection failed")

// Bounce spinner (bouncing dots)
spinner = logger.SpinnerBounce("processing data")
time.Sleep(2 * time.Second)
spinner.Replace("processed 1000 records")

// Custom frames
spinner = logger.SpinnerWithFrames("compiling", []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"})
spinner.Stop() // or spinner.Success(), spinner.Error(), spinner.Replace()
```

### Customizing Bullets

```go
logger := bullets.New(os.Stdout)

// Enable special bullet symbols (✓, ✗, ⚠, ○)
logger.SetUseSpecialBullets(true)

// Set custom bullet for a specific level
logger.SetBullet(bullets.InfoLevel, "→")
logger.SetBullet(bullets.ErrorLevel, "💥")

// Set multiple custom bullets at once
logger.SetBullets(map[bullets.Level]string{
    bullets.WarnLevel:  "⚡",
    bullets.DebugLevel: "🔍",
})
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

**Default (bullets only):**
```
• building
  • binary=dist/app_linux_amd64
  • binary=dist/app_darwin_amd64
  • binary=dist/app_windows_amd64
• archiving
  • binary=app name=app_0.2.1_linux_amd64
  • binary=app name=app_0.2.1_darwin_amd64
• calculating checksums
• release succeeded
```

**With special bullets enabled:**
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

**With spinners:**
```
⠹ downloading files...    (animating)
• downloaded 10 files     (completed)
```

## Running the Example

```bash
cd examples/basic
go run main.go
```

## API Reference

### Logger Methods

**Core logging:**
- `Debug(msg)`, `Debugf(format, args...)`
- `Info(msg)`, `Infof(format, args...)`
- `Warn(msg)`, `Warnf(format, args...)`
- `Error(msg)`, `Errorf(format, args...)`
- `Fatal(msg)`, `Fatalf(format, args...)` - logs and exits
- `Success(msg)`, `Successf(format, args...)`

**Spinners:**
- `Spinner(msg)` - Default Braille dots spinner
- `SpinnerDots(msg)` - Braille dots (same as default)
- `SpinnerCircle(msg)` - Rotating circle
- `SpinnerBounce(msg)` - Bouncing dots
- `SpinnerWithFrames(msg, frames)` - Custom animation

**Spinner Control:**
- `spinner.Stop()` - Stop and clear
- `spinner.Success(msg)` - Complete with success
- `spinner.Error(msg)` / `spinner.Fail(msg)` - Complete with error
- `spinner.Replace(msg)` - Complete with custom message

**Configuration:**
- `SetLevel(level)`, `GetLevel()`
- `SetUseSpecialBullets(bool)` - Enable/disable special symbols
- `SetBullet(level, symbol)` - Set custom bullet for a level
- `SetBullets(map[Level]string)` - Set multiple custom bullets

**Structured logging:**
- `WithField(key, value)` - Add single field
- `WithFields(map[string]interface{})` - Add multiple fields
- `WithError(err)` - Add error field

**Indentation:**
- `IncreasePadding()`, `DecreasePadding()`, `ResetPadding()`

**Utilities:**
- `Step(msg)` - Returns cleanup function with timing

## Comparison with Other Loggers

This library is designed specifically for CLI applications that need beautiful, human-readable output. It's inspired by:

- [caarlos0/log](https://github.com/caarlos0/log) - The logger used by goreleaser
- [apex/log](https://github.com/apex/log) - The original apex logger

Unlike general-purpose loggers, `bullets` focuses on:
- Visual appeal for terminal output
- Animated spinners for long operations
- Customizable bullet symbols
- Simple API for CLI applications
- Zero configuration needed
- No external dependencies

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

Inspired by the beautiful CLI output of [goreleaser](https://github.com/goreleaser/goreleaser) and [caarlos0/log](https://github.com/caarlos0/log).
