package bullets

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewUpdatable(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	if logger == nil {
		t.Fatal("NewUpdatable returned nil")
	}

	if logger.Logger == nil {
		t.Fatal("NewUpdatable has nil Logger")
	}

	if logger.handles == nil {
		t.Fatal("NewUpdatable has nil handles")
	}
}

func TestBulletHandle_Update(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Create a handle
	handle := logger.InfoHandle("Initial message")

	// Since we're writing to a buffer (not TTY), updates won't work
	// but we can test that the state is updated
	handle.Update(ErrorLevel, "Updated message")

	state := handle.GetState()
	if state.Level != ErrorLevel {
		t.Errorf("Expected level ErrorLevel, got %v", state.Level)
	}

	if state.Message != "Updated message" {
		t.Errorf("Expected message 'Updated message', got %s", state.Message)
	}
}

func TestBulletHandle_Success(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Processing...")
	handle.Success("Completed successfully")

	state := handle.GetState()
	if state.Message != "Completed successfully" {
		t.Errorf("Expected message 'Completed successfully', got %s", state.Message)
	}
}

func TestBulletHandle_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Processing...")
	handle.Error("Failed with error")

	state := handle.GetState()
	if state.Level != ErrorLevel {
		t.Errorf("Expected level ErrorLevel, got %v", state.Level)
	}

	if state.Message != "Failed with error" {
		t.Errorf("Expected message 'Failed with error', got %s", state.Message)
	}
}

func TestBulletHandle_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Test message")
	handle.WithField("key1", "value1").WithField("key2", 42)

	state := handle.GetState()
	if len(state.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(state.Fields))
	}

	if state.Fields["key1"] != "value1" {
		t.Errorf("Expected field key1='value1', got %v", state.Fields["key1"])
	}

	if state.Fields["key2"] != 42 {
		t.Errorf("Expected field key2=42, got %v", state.Fields["key2"])
	}
}

func TestHandleGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Create multiple handles
	h1 := logger.InfoHandle("Task 1")
	h2 := logger.InfoHandle("Task 2")
	h3 := logger.InfoHandle("Task 3")

	// Create a group
	group := NewHandleGroup(h1, h2, h3)

	if group.Size() != 3 {
		t.Errorf("Expected group size 3, got %d", group.Size())
	}

	// Test adding a handle
	h4 := logger.InfoHandle("Task 4")
	group.Add(h4)

	if group.Size() != 4 {
		t.Errorf("Expected group size 4 after add, got %d", group.Size())
	}

	// Test getting a handle
	handle := group.Get(0)
	if handle == nil {
		t.Error("Get(0) returned nil")
	}

	// Test updating all
	group.UpdateAll(WarnLevel, "Warning for all")

	// Verify all handles were updated
	for i := 0; i < group.Size(); i++ {
		h := group.Get(i)
		state := h.GetState()
		if state.Level != WarnLevel {
			t.Errorf("Handle %d: expected WarnLevel, got %v", i, state.Level)
		}
		if state.Message != "Warning for all" {
			t.Errorf("Handle %d: expected 'Warning for all', got %s", i, state.Message)
		}
	}

	// Test clear
	group.Clear()
	if group.Size() != 0 {
		t.Errorf("Expected group size 0 after clear, got %d", group.Size())
	}
}

func TestHandleChain(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Create multiple handles
	h1 := logger.InfoHandle("Chain 1")
	h2 := logger.InfoHandle("Chain 2")
	h3 := logger.InfoHandle("Chain 3")

	// Create a chain and update all
	chain := Chain(h1, h2, h3)
	chain.Success("All successful")

	// Verify all handles were updated
	handles := []*BulletHandle{h1, h2, h3}
	for i, h := range handles {
		state := h.GetState()
		if state.Message != "All successful" {
			t.Errorf("Handle %d: expected 'All successful', got %s", i, state.Message)
		}
	}

	// Test chaining with fields
	chain.WithField("version", "1.0.0")

	for i, h := range handles {
		state := h.GetState()
		if state.Fields["version"] != "1.0.0" {
			t.Errorf("Handle %d: expected version='1.0.0', got %v", i, state.Fields["version"])
		}
	}
}

func TestUpdatableLogger_LineCount(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Log some messages
	logger.Info("Message 1")
	logger.Debug("Message 2")
	logger.Warn("Message 3")

	// Check line count
	logger.mu.RLock()
	lineCount := logger.lineCount
	logger.mu.RUnlock()

	// Debug messages might not be logged depending on level
	// but Info and Warn should be
	if lineCount < 2 {
		t.Errorf("Expected line count >= 2, got %d", lineCount)
	}
}

func TestBulletHandle_Progress(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Downloading...")

	// Test progress updates
	handle.Progress(25, 100)
	state := handle.GetState()

	// Check that message contains progress indicator
	if !strings.Contains(state.Message, "[") || !strings.Contains(state.Message, "]") {
		t.Errorf("Progress bar not found in message: %s", state.Message)
	}

	// Test 100% progress
	handle.Progress(100, 100)
	state = handle.GetState()

	if !strings.Contains(state.Message, "100%") {
		t.Errorf("100%% not found in message: %s", state.Message)
	}
}

func TestHandleState(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Original")
	handle.WithField("field1", "value1")

	// Get state
	state := handle.GetState()
	if state.Level != InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", state.Level)
	}
	if state.Message != "Original" {
		t.Errorf("Expected 'Original', got %s", state.Message)
	}
	if len(state.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(state.Fields))
	}

	// Modify and set state
	newState := HandleState{
		Level:   ErrorLevel,
		Message: "Modified",
		Fields: map[string]interface{}{
			"field2": "value2",
		},
	}

	handle.SetState(newState)

	// Verify state was set
	currentState := handle.GetState()
	if currentState.Level != ErrorLevel {
		t.Errorf("Expected ErrorLevel after SetState, got %v", currentState.Level)
	}
	if currentState.Message != "Modified" {
		t.Errorf("Expected 'Modified' after SetState, got %s", currentState.Message)
	}
	if len(currentState.Fields) != 1 || currentState.Fields["field2"] != "value2" {
		t.Errorf("Fields not properly set: %v", currentState.Fields)
	}
}

func TestBulletHandle_UpdateMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Test")

	// Test UpdateMessage
	handle.UpdateMessage("New message")
	state := handle.GetState()
	if state.Message != "New message" {
		t.Errorf("UpdateMessage failed: got %s", state.Message)
	}

	// Test UpdateLevel
	handle.UpdateLevel(WarnLevel)
	state = handle.GetState()
	if state.Level != WarnLevel {
		t.Errorf("UpdateLevel failed: got %v", state.Level)
	}

	// Test UpdateColor
	handle.UpdateColor(ColorRed)
	state = handle.GetState()
	if state.Color != ColorRed {
		t.Errorf("UpdateColor failed: got %s", state.Color)
	}

	// Test UpdateBullet
	handle.UpdateBullet("▶")
	state = handle.GetState()
	if state.Bullet != "▶" {
		t.Errorf("UpdateBullet failed: got %s", state.Bullet)
	}
}

func TestHandleGroup_UpdateEach(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Create handles
	h1 := logger.InfoHandle("Handle 1")
	h2 := logger.InfoHandle("Handle 2")
	h3 := logger.InfoHandle("Handle 3")

	group := NewHandleGroup(h1, h2, h3)

	// Update each with different messages
	updates := map[int]struct {
		Level   Level
		Message string
	}{
		0: {Level: DebugLevel, Message: "Updated 1"},
		1: {Level: WarnLevel, Message: "Updated 2"},
		2: {Level: ErrorLevel, Message: "Updated 3"},
	}

	group.UpdateEach(updates)

	// Verify updates
	expected := []struct {
		Level   Level
		Message string
	}{
		{DebugLevel, "Updated 1"},
		{WarnLevel, "Updated 2"},
		{ErrorLevel, "Updated 3"},
	}

	for i, exp := range expected {
		h := group.Get(i)
		state := h.GetState()
		if state.Level != exp.Level {
			t.Errorf("Handle %d: expected level %v, got %v", i, exp.Level, state.Level)
		}
		if state.Message != exp.Message {
			t.Errorf("Handle %d: expected message %s, got %s", i, exp.Message, state.Message)
		}
	}
}

func TestBatchUpdate(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	// Create handles
	h1 := logger.InfoHandle("Batch 1")
	h2 := logger.InfoHandle("Batch 2")
	h3 := logger.InfoHandle("Batch 3")

	// Prepare batch update
	updates := map[*BulletHandle]struct {
		Level   Level
		Message string
	}{
		h1: {Level: DebugLevel, Message: "Batch Updated 1"},
		h2: {Level: WarnLevel, Message: "Batch Updated 2"},
		h3: {Level: ErrorLevel, Message: "Batch Updated 3"},
	}

	// Perform batch update
	BatchUpdate([]*BulletHandle{h1, h2, h3}, updates)

	// Verify updates
	handles := []*BulletHandle{h1, h2, h3}
	expected := []struct {
		Level   Level
		Message string
	}{
		{DebugLevel, "Batch Updated 1"},
		{WarnLevel, "Batch Updated 2"},
		{ErrorLevel, "Batch Updated 3"},
	}

	for i, h := range handles {
		state := h.GetState()
		if state.Level != expected[i].Level {
			t.Errorf("Handle %d: expected level %v, got %v", i, expected[i].Level, state.Level)
		}
		if state.Message != expected[i].Message {
			t.Errorf("Handle %d: expected message %s, got %s", i, expected[i].Message, state.Message)
		}
	}
}

func TestPulse(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewUpdatable(buf)

	handle := logger.InfoHandle("Original message")

	// Start pulse (it runs in background)
	handle.Pulse(500*time.Millisecond, "Alternate message")

	// Wait a bit to ensure goroutine starts
	time.Sleep(100 * time.Millisecond)

	// After pulse duration, should return to original
	time.Sleep(500 * time.Millisecond)

	state := handle.GetState()
	if state.Message != "Original message" {
		t.Errorf("Message should return to original after pulse, got: %s", state.Message)
	}
}