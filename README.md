# Bullets

[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/bullets)](https://goreportcard.com/report/github.com/sgaunet/bullets)
[![GitHub release](https://img.shields.io/github/release/sgaunet/bullets.svg)](https://github.com/sgaunet/bullets/releases/latest)
[![linter](https://github.com/sgaunet/bullets/actions/workflows/linter.yml/badge.svg)](https://github.com/sgaunet/bullets/actions/workflows/linter.yml)
[![coverage](https://github.com/sgaunet/bullets/actions/workflows/coverage.yml/badge.svg)](https://github.com/sgaunet/bullets/actions/workflows/coverage.yml)
[![tests](https://github.com/sgaunet/bullets/actions/workflows/tests.yml/badge.svg)](https://github.com/sgaunet/bullets/actions/workflows/tests.yml)
[![vulnerability-scan](https://github.com/sgaunet/bullets/actions/workflows/vulnerability-scan.yml/badge.svg)](https://github.com/sgaunet/bullets/actions/workflows/vulnerability-scan.yml)
[![Release Build](https://github.com/sgaunet/bullets/actions/workflows/release.yml/badge.svg)](https://github.com/sgaunet/bullets/actions/workflows/release.yml)
![License](https://img.shields.io/github/license/sgaunet/bullets.svg)

A colorful terminal logger for Go with bullet-style output, inspired by [goreleaser](https://github.com/goreleaser/goreleaser)'s beautiful CLI output.

## Features

- 🎨 Colorful terminal output with ANSI colors
- 🔘 Configurable bullet symbols (default circles, optional special symbols, custom icons)
- 📊 Support for log levels (Debug, Info, Warn, Error, Fatal)
- 📝 Structured logging with fields
- ⏱️  Timing information for long-running operations
- 🔄 Indentation/padding support for nested operations
- ⏳ Animated spinners with multiple styles (Braille, Circle, Bounce)
- 🔄 **Updatable bullets** - Update previously rendered bullets in real-time
- 📊 **Progress indicators** - Show progress bars within bullets
- 🎯 **Batch operations** - Update multiple bullets simultaneously
- 🧵 Thread-safe operations
- 🚀 Zero external dependencies (stdlib only)

## Demo

![demo](docs/demo.gif)

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

### Updatable Bullets

Create bullets that can be updated after rendering - perfect for showing progress, updating status, and creating dynamic terminal UIs.

**⚠️ Important Terminal Requirements:**

The updatable feature requires ANSI escape code support and proper TTY detection. If bullets are not updating in-place (appearing as new lines instead):

1. **Force TTY mode** by setting an environment variable:
   ```bash
   export BULLETS_FORCE_TTY=1
   go run your-program.go
   ```

2. **Why this is needed:**
   - `go run` often doesn't properly detect terminal capabilities
   - Some terminal emulators don't report as TTY correctly
   - IDE integrated terminals may not support ANSI codes

3. **Fallback behavior:**
   - When TTY is not detected, updates print as new lines (safe fallback)
   - This ensures your program works in all environments (logs, CI/CD, etc.)

```go
// Create an updatable logger
logger := bullets.NewUpdatable(os.Stdout)

// Create bullets that return handles
handle1 := logger.InfoHandle("Downloading package...")
handle2 := logger.InfoHandle("Installing dependencies...")
handle3 := logger.InfoHandle("Running tests...")

// Update them later
handle1.Success("Package downloaded ✓")
handle2.Error("Dependencies failed ✗")
handle3.Warning("Tests completed with warnings ⚠")
```

**Progress indicators:**
```go
download := logger.InfoHandle("Downloading file...")

// Show progress (updates message with progress bar)
for i := 0; i <= 100; i += 10 {
    download.Progress(i, 100)
    time.Sleep(100 * time.Millisecond)
}
download.Success("Download complete!")
```

**Batch operations:**
```go
// Group handles for batch updates
h1 := logger.InfoHandle("Service 1")
h2 := logger.InfoHandle("Service 2")
h3 := logger.InfoHandle("Service 3")

group := bullets.NewHandleGroup(h1, h2, h3)
group.SuccessAll("All services running")

// Or use chains
bullets.Chain(h1, h2, h3).
    WithField("status", "active").
    Success("All systems operational")
```

**Adding fields dynamically:**
```go
handle := logger.InfoHandle("Building project")

// Add fields as the operation progresses
handle.WithField("version", "1.2.3")
handle.WithFields(map[string]interface{}{
    "arch": "amd64",
    "os": "linux",
})
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

## Running the Examples

**Basic example:**
```bash
cd examples/basic
go run main.go
```

**Updatable bullets example:**
```bash
# REQUIRED: Set this environment variable for the updates to work properly
export BULLETS_FORCE_TTY=1
go run examples/updatable/main.go
```

**Note:** The updatable feature uses ANSI escape codes to update lines in place. You MUST:
1. Run in a terminal that supports ANSI codes (most modern terminals)
2. Set `BULLETS_FORCE_TTY=1` environment variable
3. Run directly in the terminal (not through pipes or output redirection)

This demonstrates all updatable features including status updates, progress tracking, batch operations, and parallel operations.

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

### UpdatableLogger Methods

**Create updatable logger:**
- `NewUpdatable(w io.Writer)` - Create new updatable logger

**Create handle-returning bullets:**
- `InfoHandle(msg string) *BulletHandle` - Log info and return handle
- `DebugHandle(msg string) *BulletHandle` - Log debug and return handle
- `WarnHandle(msg string) *BulletHandle` - Log warning and return handle
- `ErrorHandle(msg string) *BulletHandle` - Log error and return handle

### BulletHandle Methods

**Update operations:**
- `Update(level Level, msg string)` - Update level and message
- `UpdateMessage(msg string)` - Update message only
- `UpdateLevel(level Level)` - Update level only
- `UpdateColor(color string)` - Update color only
- `UpdateBullet(bullet string)` - Update bullet symbol only

**State transitions:**
- `Success(msg string)` - Mark as success with message
- `Error(msg string)` - Mark as error with message
- `Warning(msg string)` - Mark as warning with message

**Fields and metadata:**
- `WithField(key, value)` - Add a field
- `WithFields(fields)` - Add multiple fields

**Progress tracking:**
- `Progress(current, total int)` - Show progress bar

**State management:**
- `GetState() HandleState` - Get current state
- `SetState(state HandleState)` - Set complete state

### HandleGroup Methods

- `NewHandleGroup(handles...)` - Create handle group
- `Add(handle)` - Add handle to group
- `UpdateAll(level, msg)` - Update all handles
- `SuccessAll(msg)` - Mark all as success
- `ErrorAll(msg)` - Mark all as error

### HandleChain Methods

- `Chain(handles...)` - Create handle chain
- `Update(level, msg)` - Chain update operation
- `Success(msg)` - Chain success operation
- `Error(msg)` - Chain error operation
- `WithField(key, value)` - Chain field addition

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
