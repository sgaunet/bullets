package bullets

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestUpdatableConcurrentWritesAndHandle exercises B1/B2/B3: concurrent
// Info/Success calls on UpdatableLogger plus a goroutine repeatedly updating
// a BulletHandle. Before the fix, writes could race on lineCount and
// UpdatableLogger.Success bypassed writeMu — the race detector flagged both.
// In non-TTY mode handle.Update falls back to a plain log line, but the
// internal mutex acquisition pattern still exercises ul.mu vs. the embedded
// Logger.writeMu plumbing — the value of this test is the -race signal.
func TestUpdatableConcurrentWritesAndHandle(t *testing.T) {
	var buf bytes.Buffer
	ul := NewUpdatable(&buf)

	handle := ul.InfoHandle("seed")

	const writers = 8
	const perWriter = 200

	var wg sync.WaitGroup
	wg.Add(writers + 1)

	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				ul.Info("info")
				ul.Success("done")
			}
		}()
	}

	go func() {
		defer wg.Done()
		for j := 0; j < perWriter*writers; j++ {
			handle.Update(InfoLevel, "tick")
		}
	}()

	wg.Wait()

	// Output line accounting in non-TTY mode:
	//   - InfoHandle("seed") writes 1 line via ul.log (logHandle does not
	//     bump lineCount in non-TTY).
	//   - writers*perWriter*2 lines from Info+Success.
	//   - writers*perWriter lines from handle.Update's non-TTY fallback,
	//     which also goes through the base Logger.log (no lineCount bump).
	wantLines := 1 + writers*perWriter*2 + writers*perWriter
	if got := strings.Count(buf.String(), "\n"); got != wantLines {
		t.Errorf("expected %d output lines, got %d", wantLines, got)
	}

	// lineCount only counts what UpdatableLogger.Info/Success bumped:
	// writers*perWriter*2.
	ul.mu.RLock()
	gotCount := ul.lineCount
	ul.mu.RUnlock()
	if want := writers * perWriter * 2; gotCount != want {
		t.Errorf("expected lineCount %d, got %d", want, gotCount)
	}
}

// TestSpinnerCtxRaceWithSuccess exercises B4: race a context timeout against a
// manual Success() call. With the completion gate, exactly one of the two
// paths renders, and Success() never returns before the spinner is fully torn
// down. Before the fix, both paths queued an updateComplete and the visible
// outcome was non-deterministic; this also tripped the race detector
// occasionally on the spinner's stop/done bookkeeping.
func TestSpinnerCtxRaceWithSuccess(t *testing.T) {
	t.Setenv("BULLETS_FORCE_TTY", "0") // exercise the non-TTY path; deterministic output

	var ctxWon, manualWon atomic.Int64

	const iters = 200
	for i := 0; i < iters; i++ {
		var buf bytes.Buffer
		logger := New(&buf)

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		spinner := logger.Spinner(ctx, "work")

		// Race: ctx will fire ~1ms; Success fires after a tiny variable delay.
		var done sync.WaitGroup
		done.Add(1)
		go func() {
			defer done.Done()
			// Vary the delay to interleave both orderings.
			time.Sleep(time.Duration(i%3) * 500 * time.Microsecond)
			spinner.Success("ok")
		}()
		done.Wait()
		cancel()

		out := buf.String()
		switch {
		case strings.Contains(out, "ok"):
			manualWon.Add(1)
		case strings.Contains(out, "context") || strings.Contains(out, "deadline"):
			ctxWon.Add(1)
		default:
			t.Fatalf("iteration %d produced no completion line: %q", i, out)
		}

		// Crucially: each iteration must produce exactly ONE completion line.
		// The first line is the spinner's static non-TTY frame ("work"),
		// the second is the completion. No more.
		nLines := strings.Count(out, "\n")
		if nLines != 2 {
			t.Fatalf("iteration %d expected 2 lines (frame + completion), got %d: %q",
				i, nLines, out)
		}
	}

	// Sanity check: across 200 randomized iterations we should see both
	// outcomes — otherwise the test is degenerate (always taking one branch).
	if ctxWon.Load() == 0 || manualWon.Load() == 0 {
		t.Logf("ctx wins: %d, manual wins: %d (expected at least 1 of each)",
			ctxWon.Load(), manualWon.Load())
	}
}

// TestPulseNoRace exercises B5: Pulse used to read h.message without h.mu,
// racing concurrent UpdateMessage calls. With the fix the read is locked.
func TestPulseNoRace(t *testing.T) {
	t.Setenv("BULLETS_FORCE_TTY", "1")

	var buf bytes.Buffer
	ul := NewUpdatable(&buf)
	handle := ul.InfoHandle("seed")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handle.Pulse(ctx, 10*time.Millisecond, "alt")

	// Hammer UpdateMessage while Pulse is running — race detector must stay quiet.
	for i := 0; i < 1000; i++ {
		handle.UpdateMessage("update")
	}
}
