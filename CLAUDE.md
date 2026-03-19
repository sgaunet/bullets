# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**bullets** is a colorful terminal logger for Go with bullet-style output, inspired by goreleaser's CLI aesthetics. The library provides:
- Basic logger with colored bullets and log levels
- Animated spinners for long-running operations
- **Updatable bullets** - the ability to update previously rendered terminal lines in-place using ANSI escape codes
- Thread-safe operations with minimal dependencies (only golang.org/x/term for TTY detection)

## Development Commands

### Task Management
The project uses [Task](https://taskfile.dev) for common operations:

```bash
# List all available tasks
task

# Run linter (golangci-lint)
task linter

# Run tests with race detector
task tests

# Generate coverage report (outputs total coverage percentage)
task coverage

# Create snapshot release (for testing)
task snapshot

# Create release
task release
```

### Running Examples

**Basic example:**
```bash
go run examples/basic/main.go
```

**Updatable bullets example (requires TTY):**
```bash
# MUST set this environment variable for in-place updates to work
export BULLETS_FORCE_TTY=1
go run examples/updatable/main.go
```

**Spinner examples (demonstrates concurrent usage):**
```bash
# Shows multiple spinners with edge cases: rapid creation, out-of-order completion, mixed outcomes
export BULLETS_FORCE_TTY=1
go run examples/spinner/main.go
```

**Context examples (demonstrates cancellation and timeouts):**
```bash
# Shows context-driven spinner lifecycle: timeouts, manual cancel, shared context
export BULLETS_FORCE_TTY=1
go run examples/context/main.go
```

**Debug mode:**
```bash
# Basic debug output (registration, completion, errors) - production-safe, no panics
export BULLETS_DEBUG=1
export BULLETS_FORCE_TTY=1
go run examples/spinner/main.go

# Verbose debug output (includes frame updates, ANSI sequences, periodic state dumps)
# WARNING: Panics on validation failures - development only
export BULLETS_DEBUG=2
export BULLETS_FORCE_TTY=1
go run examples/spinner/main.go
```

### Testing

```bash
# Run all tests with race detector
go test -count=1 -race ./...

# Run specific test file
go test -v -run TestName ./...

# Run benchmarks
go test -bench=. ./...
```

## Architecture

### Core Components

The package has a layered architecture with distinct responsibilities:

1. **Logger (`bullets.go`)** - Base logger implementation
   - Thread-safe with `sync.Mutex`
   - Manages log levels, padding (indentation), structured fields
   - Configurable bullets (special symbols or custom icons)
   - Creates spinners and provides step timing functionality

2. **UpdatableLogger (`updatable.go`)** - Extends Logger with in-place update capability
   - Wraps base Logger and tracks line positions
   - TTY detection via `golang.org/x/term` (with `BULLETS_FORCE_TTY=1` override)
   - Returns `BulletHandle` objects that can update their original lines
   - Maintains handle registry and line count for cursor positioning

3. **BulletHandle (`handle.go`)** - Handle to an updatable bullet line
   - Stores state: level, message, fields, line number
   - Update operations: `Update()`, `Success()`, `Error()`, `Progress()`
   - Thread-safe rendering with ANSI cursor manipulation
   - `HandleGroup` and `HandleChain` for batch operations

4. **Spinner (`spinner.go`)** - Animated spinner implementation
   - Runs animation in goroutine with configurable frames and intervals
   - Can be stopped with `Stop()`, `Success()`, `Error()`, or `Replace()`
   - Delegates to SpinnerCoordinator for centralized management

5. **SpinnerCoordinator (`coordinator.go`)** - Centralized spinner management (v2.0)
   - **Coordinator Pattern**: All spinners managed by single coordinator instance
   - **Channel-based Communication**: Thread-safe updates via buffered channels
   - **Central Animation Loop**: Single ticker goroutine handles all animations (80ms interval)
   - **Automatic Line Management**: Dynamically assigns and recalculates line numbers
   - **TTY Detection**: Unified TTY handling for consistent behavior
   - **No Timing Hacks**: Clean architecture without sleep/wait workarounds
   - **LineTracker as Single Source of Truth**: All line position queries go through LineTracker

6. **Debug System (`debug.go`)** - Comprehensive debugging infrastructure (optional, disabled by default)
   - **Debug Levels**: Off (0), Basic (1), Verbose (2) via `BULLETS_DEBUG` environment variable
   - **Level 1 (Basic)**: Logging only, no panics — production-safe diagnostics
   - **Level 2 (Verbose)**: Logging + panics on validation failures — development only
   - **Timestamped Logging**: All debug output includes elapsed time since initialization
   - **Performance Profiling**: Debug timers for measuring operation duration
   - **State Visualization**: Visual spinner position map showing line allocations
   - **Zero Cost When Disabled**: Debug code has minimal overhead when BULLETS_DEBUG is unset

7. **Level & Colors (`level.go`, `colors.go`)** - Log level definitions and ANSI color codes
   - Levels: Debug, Info, Warn, Error, Fatal
   - Color utilities with fallback for non-TTY environments

### Critical Implementation Details

**TTY Detection and Updatable Features:**
- Updatable bullets require TTY support to use ANSI escape codes
- `NewUpdatable()` checks `BULLETS_FORCE_TTY=1` env var or uses `term.IsTerminal()`
- When not a TTY, `BulletHandle` operations fall back to printing new lines
- This ensures code works in all environments (terminals, pipes, CI/CD logs)

**ANSI Escape Sequences (updatable.go:39-46):**
```go
const (
    ansiMoveUp        = "\033[%dA"  // Move cursor up N lines
    ansiMoveDown      = "\033[%dB"  // Move cursor down N lines
    ansiClearLine     = "\033[2K"   // Clear entire line
    ansiMoveToCol     = "\033[0G"   // Move to column 0
)
```

**Thread Safety:**
- `Logger.mu` protects logger state (level, padding, fields, bullets)
- `UpdatableLogger.mu` protects line count and handle registry
- `UpdatableLogger.writeMu` serializes terminal write operations to prevent interleaved output
- `BulletHandle.mu` protects individual handle state
- `HandleGroup.mu` protects group operations

**Handle Redraw Logic (updatable.go:298-341):**
1. Calculate lines to move up from current cursor position
2. Lock `writeMu` to prevent other writes
3. Move cursor up to target line
4. Clear line and rewrite content
5. Move cursor back to original position

This is critical - understand this before modifying updatable behavior.

### Spinner Architecture (v2.0)

The spinner system uses a **centralized coordinator pattern** for robust concurrent spinner management:

**Architecture Overview:**
```
┌─────────────────────────────────────────────────────────────────┐
│                         Logger                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              SpinnerCoordinator                           │  │
│  │  ┌──────────────────────────────────────────────────┐    │  │
│  │  │  Central Animation Ticker (80ms)                │    │  │
│  │  │  • Updates all active spinners                   │    │  │
│  │  │  • Single goroutine for all animations          │    │  │
│  │  └──────────────────────────────────────────────────┘    │  │
│  │                                                            │  │
│  │  Channel (buffered: 100)                                  │  │
│  │    ↑         ↑         ↑                                  │  │
│  │    │         │         │                                  │  │
│  └────┼─────────┼─────────┼──────────────────────────────────┘  │
│       │         │         │                                      │
│  ┌────┴───┐ ┌──┴────┐ ┌──┴────┐                               │
│  │Spinner1│ │Spinner2│ │Spinner3│                              │
│  │Line: 0 │ │Line: 1 │ │Line: 2 │                              │
│  └────────┘ └────────┘ └────────┘                              │
└─────────────────────────────────────────────────────────────────┘
```

**Key Design Principles:**

1. **Single Source of Truth**: SpinnerCoordinator owns all spinner state
   - **LineTracker** maintains authoritative line number mappings
   - All line position queries go through LineTracker (no cached positions in spinner state)
   - Prevents line position drift and wrong-line updates

2. **Channel-Based Updates**: Spinners communicate via buffered channels (non-blocking)
   - Spinners send updates via `updateCh` (100 buffer)
   - Completion updates block via `doneCh` for proper synchronization
   - Frame updates can be dropped if channel is full (graceful degradation)

3. **Centralized Animation**: One ticker updates all spinners (prevents timing issues)
   - Single 80ms ticker advances all spinner frames together
   - Eliminates per-spinner goroutines and timing hacks
   - Consistent animation speed across all spinners

4. **Automatic Line Management**: Coordinator handles line number allocation/reallocation
   - LineTracker assigns sequential line numbers on registration (0, 1, 2, ...)
   - On completion, remaining spinners automatically reflow to close gaps
   - Maintains cursor at bottom, updates spinners by moving up

5. **Thread-Safe by Design**: Minimal locking, channel-based synchronization
   - `coordinator.mu` protects spinner registry
   - `writeMu` serializes terminal writes
   - Channel communication reduces lock contention

6. **Context-Aware Lifecycle**: All spinner creation methods accept `context.Context`
   - When context is cancelled/times out, spinner auto-stops with error message from `ctx.Err()`
   - The `animate()` goroutine selects on both `stopCh` and `ctx.Done()`
   - `handleContextCancellation()` sends completion directly to coordinator (avoids deadlock with `stopAnimation()`)
   - Race between manual `Success()`/`Error()` and context cancellation is safe: coordinator ignores duplicate completions
   - Context's `Done()` channel and `Err()` function are stored (not the context itself) to satisfy `containedctx` lint rule

**TTY Mode Behavior:**
- Each spinner gets a dedicated terminal line
- ANSI escape codes move cursor to update specific lines
- Central ticker advances animation frames (80ms interval)
- Completion messages overwrite spinner lines
- Line numbers dynamically recalculated when spinners finish

**Non-TTY Mode Behavior:**
- Static output with no animation
- Each update prints a new line (safe for logs/CI)
- No ANSI codes used
- Coordinator still manages registration but skips animation

**Thread Safety Guarantees:**
- `coordinator.mu` protects spinner registry and state
- `logger.writeMu` serializes all terminal writes
- Completion updates are blocking (via `doneCh`) to ensure proper ordering
- Frame updates are non-blocking and can be dropped if channel is full

**Environment Variables:**
- `BULLETS_FORCE_TTY=1`: Force TTY mode (useful for `go run` and IDEs)
- `BULLETS_DEBUG=1`: Enable basic debug output to stderr (production-safe, no panics)
- `BULLETS_DEBUG=2`: Enable verbose debug output with panics on validation failures (development only)

**Recent Bug Fixes:**

The spinner coordination system has been hardened to eliminate several edge cases:

1. **Blank Line Artifacts (Fixed in v2.0)**
   - **Problem**: Spinner completions could leave blank lines in output
   - **Root Cause**: Cursor positioning assumed spinners were at bottom, but completions moved cursor
   - **Solution**: LineTracker now maintains authoritative cursor position; all updates query LineTracker

2. **Line Position Drift (Fixed in v2.0)**
   - **Problem**: Out-of-order completions caused spinners to update wrong lines
   - **Root Cause**: Spinner state cached line numbers that became stale after reallocations
   - **Solution**: Removed cached line numbers; all position queries go through LineTracker

3. **Concurrent Completion Race (Fixed in v2.0)**
   - **Problem**: Rapid concurrent completions could cause visual corruption
   - **Root Cause**: Completion updates were non-blocking, allowing interleaved writes
   - **Solution**: Completion updates now block via `doneCh` until fully rendered

4. **Cross-Session Line Reuse (Fixed in v2.0)**
   - **Problem**: Sequential spinner groups reused line numbers, causing completions to appear in wrong sections
   - **Root Cause**: LineTracker persisted line numbers across spinner mode sessions
   - **Solution**: Added `spinnerModeBaseLine` to track session start; reset LineTracker on new session; translate relative line numbers to absolute terminal positions

5. **Data Race in Tests (Fixed)**
   - **Problem**: `go test -race` flagged mutex copying in test setup
   - **Root Cause**: SpinnerCoordinator was copied by value in test fixtures
   - **Solution**: Changed test fixtures to use pointers to coordinator

**Best Practices for Concurrent Spinner Usage:**

1. **Create Spinners Upfront**: Allocate all spinners before starting concurrent work
   ```go
   ctx := context.Background()
   s1, s2, s3 := logger.Spinner(ctx, "Task 1"), logger.Spinner(ctx, "Task 2"), logger.Spinner(ctx, "Task 3")
   // Now start goroutines that complete the spinners
   ```

2. **Use WaitGroups for Coordination**: Ensure goroutines complete before exiting
   ```go
   var wg sync.WaitGroup
   wg.Add(3)
   go func() { defer wg.Done(); s1.Success("Done") }()
   // ...
   wg.Wait()
   ```

3. **Out-of-Order Completion is Safe**: Spinners can complete in any order
   - Coordinator automatically reallocates line numbers
   - No manual line tracking needed

4. **Mixed Completion Types Supported**: Mix Success(), Error(), Replace(), Stop()
   - All completion types handled consistently
   - No special ordering requirements

5. **Debug Mode for Troubleshooting**: Use `BULLETS_DEBUG=1` or `BULLETS_DEBUG=2`
   - Debug output goes to stderr (doesn't interfere with main output)
   - Verbose mode shows ANSI sequences and state maps

### Troubleshooting

**Problem: Spinners not animating, just printing static lines**

- **Cause**: Not running in TTY mode
- **Solution**: Set `BULLETS_FORCE_TTY=1` environment variable
- **Verification**: Check that `go run` or your IDE supports ANSI codes

**Problem: Blank lines appearing in output**

- **Cause**: Bug in older versions (< v2.0)
- **Solution**: Ensure you're using the latest version with LineTracker fixes
- **Debug**: Enable `BULLETS_DEBUG=1` to see completion rendering

**Problem: Spinners updating wrong lines**

- **Cause**: Line position drift bug (fixed in v2.0)
- **Solution**: Update to v2.0+ which queries LineTracker for authoritative positions
- **Debug**: Enable `BULLETS_DEBUG=2` to see periodic spinner position map

**Problem: Visual corruption with rapid concurrent completions**

- **Cause**: Race condition in completion rendering (fixed in v2.0)
- **Solution**: Update to v2.0+ which uses blocking completion updates via `doneCh`
- **Debug**: Run with `-race` flag and check for data races

**Problem: go test -race reports mutex copying**

- **Cause**: SpinnerCoordinator copied by value in test code
- **Solution**: Use pointers when passing coordinator to helper functions
- **Example**:
  ```go
  // Wrong (copies mutex)
  func helper(c SpinnerCoordinator) { }

  // Correct (uses pointer)
  func helper(c *SpinnerCoordinator) { }
  ```

**Problem: Spinners not visible in CI/CD logs**

- **Cause**: CI environments often don't support ANSI codes
- **Expected**: In non-TTY mode, spinners print static messages (no animation)
- **Verification**: This is correct behavior; each update prints a new line

**Problem: Debug output not appearing**

- **Cause**: Debug output goes to stderr, not stdout
- **Solution**: Check stderr stream or redirect: `go run main.go 2>&1 | less`
- **Verification**: Debug output format: `[HH:MM:SS.mmm] [COMPONENT] message`

## Code Organization

```
bullets/
├── bullets.go         # Core Logger implementation
├── updatable.go       # UpdatableLogger with ANSI cursor control
├── handle.go          # BulletHandle, HandleGroup, HandleChain
├── spinner.go         # Animated spinner implementation
├── coordinator.go     # SpinnerCoordinator for centralized spinner management
├── debug.go           # Debug system (optional, controlled by BULLETS_DEBUG)
├── level.go           # Log level definitions
├── colors.go          # ANSI color utilities
├── examples/
│   ├── basic/         # Basic logger examples
│   ├── context/       # Context cancellation and timeout examples
│   ├── updatable/     # Updatable bullets examples
│   └── spinner/       # Spinner examples (edge cases, concurrent usage)
└── *_test.go          # Test files (edge cases, integration, benchmarks)
```

## Testing Approach

- `*_test.go` - Unit tests for core functionality
- `*_edge_test.go` - Edge cases and boundary conditions
- `integration_test.go` - End-to-end scenarios
- `*_bench_test.go` - Performance benchmarks

When testing updatable features, tests check both TTY and non-TTY modes.

## Important Patterns

**Creating loggers:**
```go
logger := bullets.New(os.Stdout)              // Basic logger
updatableLogger := bullets.NewUpdatable(os.Stdout)  // With update capability
```

**Structured logging pattern:**
```go
logger.WithField("key", "value").
       WithError(err).
       Error("operation failed")
```

**Spinner lifecycle:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
spinner := logger.Spinner(ctx, "processing")
// ... work ...
spinner.Success("completed")  // or Error(), Replace(), Stop(), or context auto-cancels
```

**Updatable pattern:**
```go
handle := logger.InfoHandle("Starting task")
// ... work ...
handle.Progress(50, 100)
// ... more work ...
handle.Success("Task completed")
```

## Module Information

- Module: `github.com/sgaunet/bullets`
- Go version: 1.25.0
- Only external dependency: `golang.org/x/term` (for TTY detection)

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md
