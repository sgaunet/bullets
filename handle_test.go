package bullets

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestHandleGroup_SuccessAll(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	h1 := logger.InfoHandle("Task 1")
	h2 := logger.InfoHandle("Task 2")
	h3 := logger.InfoHandle("Task 3")

	group := NewHandleGroup(h1, h2, h3)
	group.SuccessAll("All completed")

	for i := 0; i < group.Size(); i++ {
		h := group.Get(i)
		state := h.GetState()
		if state.Message != "All completed" {
			t.Errorf("Handle %d: expected 'All completed', got %s", i, state.Message)
		}
	}
}

func TestHandleGroup_ErrorAll(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	h1 := logger.InfoHandle("Service 1")
	h2 := logger.InfoHandle("Service 2")
	h3 := logger.InfoHandle("Service 3")

	group := NewHandleGroup(h1, h2, h3)
	group.ErrorAll("Connection failed")

	for i := 0; i < group.Size(); i++ {
		h := group.Get(i)
		state := h.GetState()
		if state.Level != ErrorLevel {
			t.Errorf("Handle %d: expected ErrorLevel, got %v", i, state.Level)
		}
		if state.Message != "Connection failed" {
			t.Errorf("Handle %d: expected 'Connection failed', got %s", i, state.Message)
		}
	}
}

func TestHandleChain_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	h1 := logger.InfoHandle("Process 1")
	h2 := logger.InfoHandle("Process 2")

	chain := Chain(h1, h2)
	chain.Error("Critical error occurred")

	handles := []*BulletHandle{h1, h2}
	for i, h := range handles {
		state := h.GetState()
		if state.Level != ErrorLevel {
			t.Errorf("Handle %d: expected ErrorLevel, got %v", i, state.Level)
		}
		if state.Message != "Critical error occurred" {
			t.Errorf("Handle %d: expected error message, got %s", i, state.Message)
		}
	}
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		percentage int
		expected   string // partial match
	}{
		{0, "[>                   ] "},
		{25, "[====>               ] "},
		{50, "[==========>         ] "},
		{75, "[===============>    ] "},
		{100, "[====================] "},
	}

	for _, tt := range tests {
		result := renderProgressBar(tt.percentage)

		// Check for percentage in result
		expectedPercentage := fmt.Sprintf("%d%%", tt.percentage)
		if !strings.Contains(result, expectedPercentage) {
			t.Errorf("Progress bar for %d%% doesn't contain percentage: %s", tt.percentage, result)
		}

		// Check bar structure
		if !strings.HasPrefix(result, "[") {
			t.Errorf("Progress bar for %d%% doesn't start with '[': %s", tt.percentage, result)
		}
	}
}

func TestHandleGroup_GetBounds(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	group := NewHandleGroup()

	// Test getting from empty group
	if group.Get(0) != nil {
		t.Error("Get(0) on empty group should return nil")
	}

	// Add handles
	h1 := logger.InfoHandle("Test 1")
	h2 := logger.InfoHandle("Test 2")
	group.Add(h1)
	group.Add(h2)

	// Test valid indices
	if group.Get(0) != h1 {
		t.Error("Get(0) should return first handle")
	}
	if group.Get(1) != h2 {
		t.Error("Get(1) should return second handle")
	}

	// Test out of bounds
	if group.Get(-1) != nil {
		t.Error("Get(-1) should return nil")
	}
	if group.Get(2) != nil {
		t.Error("Get(2) should return nil when size is 2")
	}
}