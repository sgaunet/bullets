package bullets

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// TestCoordinatorStateConsistency_SingleSpinner verifies state consistency with a single spinner.
func TestCoordinatorStateConsistency_SingleSpinner(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	spinner := logger.Spinner("Test")

	// Validate state immediately after creation
	inconsistencies := logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("State inconsistencies detected after spinner creation: %+v", inconsistencies)
	}

	// Verify invariants hold
	if !logger.coordinator.checkStateInvariants() {
		t.Error("State invariants violated after spinner creation")
	}

	// Verify spinner is registered in both coordinator and lineTracker
	lineNum := logger.coordinator.getSpinnerLineNumber(spinner)
	if lineNum == -1 {
		t.Error("Spinner not registered in lineTracker")
	}

	logger.coordinator.mu.Lock()
	if _, exists := logger.coordinator.spinners[spinner]; !exists {
		t.Error("Spinner not registered in coordinator")
	}
	logger.coordinator.mu.Unlock()

	// Complete spinner
	spinner.Success("Done")
	time.Sleep(50 * time.Millisecond)

	// Validate state after completion
	inconsistencies = logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("State inconsistencies detected after spinner completion: %+v", inconsistencies)
	}
}

// TestCoordinatorStateConsistency_ConcurrentCreation tests state consistency with concurrent spinner creation.
func TestCoordinatorStateConsistency_ConcurrentCreation(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	const numSpinners = 20
	spinners := make([]*Spinner, numSpinners)
	var wg sync.WaitGroup

	// Create spinners concurrently
	for i := 0; i < numSpinners; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			spinners[idx] = logger.Spinner(fmt.Sprintf("Spinner %d", idx))
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Validate state after all spinners created
	inconsistencies := logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("State inconsistencies detected after concurrent creation: %+v", inconsistencies)
		for _, inc := range inconsistencies {
			t.Logf("  - %s (severity: %s)", inc.description, inc.severity)
		}
	}

	// Verify all spinners have unique line numbers
	seenLines := make(map[int]bool)
	for i, spinner := range spinners {
		lineNum := logger.coordinator.getSpinnerLineNumber(spinner)
		if lineNum == -1 {
			t.Errorf("Spinner %d not registered in lineTracker", i)
			continue
		}
		if seenLines[lineNum] {
			t.Errorf("Duplicate line number %d detected for spinner %d", lineNum, i)
		}
		seenLines[lineNum] = true
	}

	// Complete all spinners
	for _, spinner := range spinners {
		spinner.Success("Done")
	}
	time.Sleep(100 * time.Millisecond)

	// Validate final state
	inconsistencies = logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("State inconsistencies after all completions: %+v", inconsistencies)
	}
}

// TestCoordinatorStateConsistency_RapidCreationCompletion tests state consistency under rapid state changes.
func TestCoordinatorStateConsistency_RapidCreationCompletion(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	// Rapidly create and complete spinners
	for i := 0; i < 50; i++ {
		spinner := logger.Spinner(fmt.Sprintf("Rapid %d", i))

		// Validate state during active spinner
		inconsistencies := logger.coordinator.validateCoordinatorState()
		if len(inconsistencies) > 0 {
			t.Errorf("Iteration %d: State inconsistencies during active spinner: %+v", i, inconsistencies)
		}

		// Complete immediately
		spinner.Success("Done")

		// Small delay to allow processing
		time.Sleep(5 * time.Millisecond)

		// Validate state after completion
		inconsistencies = logger.coordinator.validateCoordinatorState()
		if len(inconsistencies) > 0 {
			// Filter out warnings about stopped spinners (they're expected briefly)
			errors := 0
			for _, inc := range inconsistencies {
				if inc.severity == "error" {
					errors++
					t.Errorf("Iteration %d: %s: %s", i, inc.description, inc.severity)
				}
			}
			if errors > 0 {
				t.Errorf("Iteration %d: %d error-level inconsistencies detected", i, errors)
			}
		}

		// Verify invariants hold
		if !logger.coordinator.checkStateInvariants() {
			t.Errorf("Iteration %d: State invariants violated", i)
		}
	}
}

// TestCoordinatorStateConsistency_AlternatingCompletions tests consistency with alternating spinner completions.
func TestCoordinatorStateConsistency_AlternatingCompletions(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	const numSpinners = 10
	spinners := make([]*Spinner, numSpinners)

	// Create all spinners
	for i := 0; i < numSpinners; i++ {
		spinners[i] = logger.Spinner(fmt.Sprintf("Spinner %d", i))
	}
	time.Sleep(100 * time.Millisecond)

	// Validate initial state
	inconsistencies := logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("Initial state inconsistencies: %+v", inconsistencies)
	}

	// Complete spinners in alternating pattern: 0, 2, 4, 6, 8, 1, 3, 5, 7, 9
	for i := 0; i < numSpinners; i += 2 {
		spinners[i].Success(fmt.Sprintf("Done %d", i))
		time.Sleep(20 * time.Millisecond)

		// Validate after each completion
		inconsistencies := logger.coordinator.validateCoordinatorState()
		if len(inconsistencies) > 0 {
			t.Errorf("Inconsistencies after completing spinner %d: %+v", i, inconsistencies)
		}
	}

	for i := 1; i < numSpinners; i += 2 {
		spinners[i].Success(fmt.Sprintf("Done %d", i))
		time.Sleep(20 * time.Millisecond)

		// Validate after each completion
		inconsistencies := logger.coordinator.validateCoordinatorState()
		if len(inconsistencies) > 0 {
			t.Errorf("Inconsistencies after completing spinner %d: %+v", i, inconsistencies)
		}
	}

	// Final validation
	time.Sleep(50 * time.Millisecond)
	inconsistencies = logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("Final state inconsistencies: %+v", inconsistencies)
	}
}

// TestCoordinatorStateConsistency_StressTest performs continuous validation under heavy concurrent load.
func TestCoordinatorStateConsistency_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	const (
		numWorkers      = 10
		iterationsPerWorker = 20
		validationInterval = 50 * time.Millisecond
	)

	var wg sync.WaitGroup
	stopValidation := make(chan struct{})
	validationErrors := make(chan string, 100)

	// Background validator
	go func() {
		ticker := time.NewTicker(validationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stopValidation:
				return
			case <-ticker.C:
				inconsistencies := logger.coordinator.validateCoordinatorState()
				for _, inc := range inconsistencies {
					if inc.severity == "error" {
						validationErrors <- fmt.Sprintf("%s: %s", inc.description, inc.severity)
					}
				}

				if !logger.coordinator.checkStateInvariants() {
					validationErrors <- "State invariants violated"
				}
			}
		}
	}()

	// Workers creating and completing spinners
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerWorker; i++ {
				spinner := logger.Spinner(fmt.Sprintf("W%d-S%d", workerID, i))
				time.Sleep(10 * time.Millisecond)

				// Randomly complete with Success or Error
				if i%2 == 0 {
					spinner.Success(fmt.Sprintf("W%d done %d", workerID, i))
				} else {
					spinner.Error(fmt.Sprintf("W%d error %d", workerID, i))
				}
			}
		}(w)
	}

	wg.Wait()
	close(stopValidation)
	time.Sleep(100 * time.Millisecond)

	// Check for validation errors
	close(validationErrors)
	errorCount := 0
	for err := range validationErrors {
		t.Errorf("Validation error: %s", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total validation errors during stress test: %d", errorCount)
	}

	// Final validation
	inconsistencies := logger.coordinator.validateCoordinatorState()
	if len(inconsistencies) > 0 {
		t.Errorf("Final inconsistencies after stress test: %+v", inconsistencies)
	}
}

// TestCoordinatorStateConsistency_LineNumberUniqueness verifies line numbers are always unique.
func TestCoordinatorStateConsistency_LineNumberUniqueness(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	const numRounds = 5
	const spinnersPerRound = 10

	for round := 0; round < numRounds; round++ {
		spinners := make([]*Spinner, spinnersPerRound)
		lineNumbers := make(map[int]*Spinner)

		// Create spinners
		for i := 0; i < spinnersPerRound; i++ {
			spinners[i] = logger.Spinner(fmt.Sprintf("R%d-S%d", round, i))
			lineNum := logger.coordinator.getSpinnerLineNumber(spinners[i])

			// Check for duplicate line numbers
			if existing, found := lineNumbers[lineNum]; found {
				t.Errorf("Round %d: Duplicate line number %d assigned to spinners %v and %v",
					round, lineNum, existing, spinners[i])
			}
			lineNumbers[lineNum] = spinners[i]
		}

		time.Sleep(50 * time.Millisecond)

		// Complete all spinners
		for _, spinner := range spinners {
			spinner.Success("Done")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// TestCoordinatorStateConsistency_InvariantsAlwaysHold verifies that invariants hold throughout lifecycle.
func TestCoordinatorStateConsistency_InvariantsAlwaysHold(t *testing.T) {
	os.Setenv("BULLETS_FORCE_TTY", "1")
	defer os.Unsetenv("BULLETS_FORCE_TTY")

	var buf bytes.Buffer
	logger := New(&buf)

	// Helper to check invariants
	checkInvariants := func(phase string) {
		if !logger.coordinator.checkStateInvariants() {
			t.Errorf("Invariants violated during phase: %s", phase)
		}
	}

	checkInvariants("initial")

	s1 := logger.Spinner("S1")
	checkInvariants("after S1 creation")

	s2 := logger.Spinner("S2")
	checkInvariants("after S2 creation")

	time.Sleep(100 * time.Millisecond)
	checkInvariants("during animation")

	s1.Success("S1 done")
	checkInvariants("after S1 completion")

	s3 := logger.Spinner("S3")
	checkInvariants("after S3 creation")

	s2.Error("S2 failed")
	checkInvariants("after S2 completion")

	s3.Replace("S3 replaced")
	checkInvariants("after S3 completion")

	time.Sleep(100 * time.Millisecond)
	checkInvariants("final")
}
