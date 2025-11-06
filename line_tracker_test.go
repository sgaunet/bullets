package bullets

import (
	"testing"
	"time"
)

func TestLineTrackerAllocation(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)

	// Create mock spinners
	s1 := &Spinner{}
	s2 := &Spinner{}
	s3 := &Spinner{}

	// Allocate lines
	line1 := tracker.allocateLine(s1)
	line2 := tracker.allocateLine(s2)
	line3 := tracker.allocateLine(s3)

	if line1 != 0 {
		t.Errorf("Expected s1 to get line 0, got %d", line1)
	}
	if line2 != 1 {
		t.Errorf("Expected s2 to get line 1, got %d", line2)
	}
	if line3 != 2 {
		t.Errorf("Expected s3 to get line 2, got %d", line3)
	}

	// Verify active count
	if count := tracker.getActiveLineCount(); count != 3 {
		t.Errorf("Expected 3 active lines, got %d", count)
	}
}

func TestLineTrackerDeallocation(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)

	s1 := &Spinner{}
	s2 := &Spinner{}
	s3 := &Spinner{}

	line1 := tracker.allocateLine(s1)
	_ = tracker.allocateLine(s2) // line2
	line3 := tracker.allocateLine(s3)

	// Deallocate s2 (middle spinner)
	tracker.deallocateLine(s2)

	// In TTY mode, line should be reserved, not available
	if count := tracker.getActiveLineCount(); count != 2 {
		t.Errorf("Expected 2 active lines after deallocation, got %d", count)
	}

	if count := tracker.getReservedLineCount(); count != 1 {
		t.Errorf("Expected 1 reserved line, got %d", count)
	}

	// Verify s1 and s3 positions after deallocation
	// In TTY mode, spinners do NOT reallocate to preserve completion message lines
	// s1 should remain at line 0
	if num := tracker.getLineNumber(s1); num != line1 {
		t.Errorf("Expected s1 to still be at line %d, got %d", line1, num)
	}
	// s3 should REMAIN at line 2 (NOT shift to line 1, to preserve s2's completion message)
	if num := tracker.getLineNumber(s3); num != line3 {
		t.Errorf("Expected s3 to remain at line %d, got %d", line3, num)
	}

	// s2 should no longer be registered
	if num := tracker.getLineNumber(s2); num != -1 {
		t.Errorf("Expected s2 to return -1, got %d", num)
	}
}

func TestLineTrackerReuseAfterReclaim(t *testing.T) {
	tracker := newLineTracker(true, 100*time.Millisecond) // Short timeout for testing

	s1 := &Spinner{}
	s2 := &Spinner{}

	_ = tracker.allocateLine(s1) // line1
	line2 := tracker.allocateLine(s2)

	// Deallocate s2
	tracker.deallocateLine(s2)

	// Reserved lines should not be reused immediately
	s3 := &Spinner{}
	line3 := tracker.allocateLine(s3)

	if line3 == line2 {
		t.Error("Line should not be reused immediately after deallocation")
	}

	// Wait for reserved line to be reclaimable
	time.Sleep(150 * time.Millisecond)

	// Reclaim reserved lines
	reclaimed := tracker.reclaimReservedLines()
	if reclaimed != 1 {
		t.Errorf("Expected to reclaim 1 line, reclaimed %d", reclaimed)
	}

	// Now new allocations should reuse the available line
	s4 := &Spinner{}
	line4 := tracker.allocateLine(s4)

	if line4 != line2 {
		t.Errorf("Expected to reuse line %d, got %d", line2, line4)
	}

	tracker.deallocateLine(s1)
	tracker.deallocateLine(s3)
	tracker.deallocateLine(s4)
}

func TestLineTrackerNonTTYMode(t *testing.T) {
	tracker := newLineTracker(false, 3*time.Second)

	s1 := &Spinner{}
	s2 := &Spinner{}

	_ = tracker.allocateLine(s1) // line1
	_ = tracker.allocateLine(s2) // line2

	// Deallocate s2
	tracker.deallocateLine(s2)

	// In non-TTY mode, lines should be available immediately
	if count := tracker.getReservedLineCount(); count != 0 {
		t.Errorf("Expected 0 reserved lines in non-TTY mode, got %d", count)
	}

	// New allocation can reuse the line immediately
	s3 := &Spinner{}
	line3 := tracker.allocateLine(s3)

	// In non-TTY mode, we just keep incrementing, so line3 should be the next number
	// (we don't reuse in non-TTY mode because there's no visual constraint)
	if line3 != 2 {
		t.Errorf("Expected line 2 in non-TTY mode, got %d", line3)
	}

	tracker.deallocateLine(s1)
	tracker.deallocateLine(s3)
}

func TestLineTrackerValidation(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)

	s1 := &Spinner{}
	s2 := &Spinner{}

	tracker.allocateLine(s1)
	tracker.allocateLine(s2)

	// Validation should find no issues
	invalid := tracker.validateLinePositions()
	if len(invalid) != 0 {
		t.Errorf("Expected no invalid positions, found %d", len(invalid))
	}

	// Deallocate s2
	tracker.deallocateLine(s2)

	// Validation should still find no issues (s2 is no longer tracked)
	invalid = tracker.validateLinePositions()
	if len(invalid) != 0 {
		t.Errorf("Expected no invalid positions after deallocation, found %d", len(invalid))
	}

	tracker.deallocateLine(s1)
}

func TestLinePositionLedger(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)
	ledger := newLinePositionLedger(tracker, 100)

	s1 := &Spinner{}
	line1 := tracker.allocateLine(s1)
	ledger.recordAllocation(s1, line1)

	tracker.deallocateLine(s1)
	ledger.recordDeallocation(s1, line1)

	reclaimed := tracker.reclaimReservedLines()
	ledger.recordReclaim(reclaimed)

	// Check history
	history := ledger.getHistory()
	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}

	if history[0].operation != "allocate" {
		t.Errorf("Expected first operation to be 'allocate', got '%s'", history[0].operation)
	}
	if history[1].operation != "deallocate" {
		t.Errorf("Expected second operation to be 'deallocate', got '%s'", history[1].operation)
	}
	if history[2].operation != "reclaim" {
		t.Errorf("Expected third operation to be 'reclaim', got '%s'", history[2].operation)
	}
}

func TestLinePositionLedgerReconcile(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)
	ledger := newLinePositionLedger(tracker, 100)

	s1 := &Spinner{}
	line1 := tracker.allocateLine(s1)
	ledger.recordAllocation(s1, line1)

	// Reconcile should succeed with valid state
	if !ledger.reconcile() {
		t.Error("Expected reconcile to succeed with valid state")
	}

	tracker.deallocateLine(s1)
}

func TestLineTrackerHistoryLimit(t *testing.T) {
	tracker := newLineTracker(true, 3*time.Second)
	ledger := newLinePositionLedger(tracker, 5) // Small limit for testing

	// Add more entries than the limit
	for i := 0; i < 10; i++ {
		s := &Spinner{}
		line := tracker.allocateLine(s)
		ledger.recordAllocation(s, line)
		tracker.deallocateLine(s)
	}

	history := ledger.getHistory()
	if len(history) > 5 {
		t.Errorf("Expected history to be limited to 5 entries, got %d", len(history))
	}
}

func BenchmarkLineTrackerAllocation(b *testing.B) {
	tracker := newLineTracker(true, 3*time.Second)
	spinners := make([]*Spinner, b.N)

	for i := range spinners {
		spinners[i] = &Spinner{}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker.allocateLine(spinners[i])
	}
}

func BenchmarkLineTrackerDeallocation(b *testing.B) {
	tracker := newLineTracker(true, 3*time.Second)
	spinners := make([]*Spinner, b.N)

	for i := range spinners {
		spinners[i] = &Spinner{}
		tracker.allocateLine(spinners[i])
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker.deallocateLine(spinners[i])
	}
}

func BenchmarkLineTrackerValidation(b *testing.B) {
	tracker := newLineTracker(true, 3*time.Second)

	// Setup some spinners
	for i := 0; i < 100; i++ {
		s := &Spinner{}
		tracker.allocateLine(s)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker.validateLinePositions()
	}
}
