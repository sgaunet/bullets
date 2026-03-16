package bullets

import (
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"
)

// HandleState represents the state of a bullet handle.
type HandleState struct {
	Level   Level
	Message string
	Color   string
	Bullet  string
	Fields  map[string]any
}

// GetState returns the current state of the handle.
func (h *BulletHandle) GetState() HandleState {
	h.mu.Lock()
	defer h.mu.Unlock()

	fields := make(map[string]any)
	maps.Copy(fields, h.fields)

	return HandleState{
		Level:   h.level,
		Message: h.message,
		Color:   h.color,
		Bullet:  h.bullet,
		Fields:  fields,
	}
}

// SetState sets the complete state of the handle.
func (h *BulletHandle) SetState(state HandleState) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.level = state.Level
	if h.logger.sanitizeInput {
		state.Message = sanitizeMsg(state.Message)
	}
	h.message = state.Message
	if state.Color != "" {
		h.color = state.Color
	}
	if state.Bullet != "" {
		h.bullet = state.Bullet
	}

	// Update fields
	h.fields = make(map[string]any)
	maps.Copy(h.fields, state.Fields)

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// UpdateColor updates just the color of the bullet.
func (h *BulletHandle) UpdateColor(color string) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.color = color

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// UpdateBullet updates just the bullet symbol.
func (h *BulletHandle) UpdateBullet(bullet string) *BulletHandle {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.bullet = bullet

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// Pulse creates a pulsing effect by alternating between two states.
func (h *BulletHandle) Pulse(duration time.Duration, alternateMsg string) {
	if h.lineNum == -1 || !h.logger.isTTY {
		return
	}

	originalMsg := h.message

	go func() {
		const pulseInterval = 500 * time.Millisecond
		ticker := time.NewTicker(pulseInterval)
		timer := time.NewTimer(duration)
		defer ticker.Stop()
		defer timer.Stop()

		toggle := false
		for {
			select {
			case <-timer.C:
				h.UpdateMessage(originalMsg)
				return
			case <-ticker.C:
				if toggle {
					h.UpdateMessage(originalMsg)
				} else {
					h.UpdateMessage(alternateMsg)
				}
				toggle = !toggle
			}
		}
	}()
}

// Progress updates the bullet to show progress.
func (h *BulletHandle) Progress(current, total int) *BulletHandle {
	const percentMultiplier = 100
	percentage := (current * percentMultiplier) / total
	progressBar := renderProgressBar(percentage)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Store progress bar separately, keep message intact
	h.progressBar = progressBar

	if h.lineNum != -1 && h.logger.isTTY {
		h.redraw()
	}
	return h
}

// renderProgressBar creates a simple ASCII progress bar.
func renderProgressBar(percentage int) string {
	const (
		barWidth         = 20
		percentMultiplier = 100
	)
	filled := (percentage * barWidth) / percentMultiplier

	var bar strings.Builder
	bar.WriteString("[")
	for i := range barWidth {
		switch {
		case i < filled:
			bar.WriteString("=")
		case i == filled && percentage < 100:
			bar.WriteString(">")
		default:
			bar.WriteString(" ")
		}
	}
	bar.WriteString("]")

	return fmt.Sprintf("%s %d%%", bar.String(), percentage)
}

// HandleGroup manages a group of related handles.
type HandleGroup struct {
	handles []*BulletHandle
	mu      sync.RWMutex
}

// NewHandleGroup creates a new handle group.
func NewHandleGroup(handles ...*BulletHandle) *HandleGroup {
	return &HandleGroup{
		handles: handles,
	}
}

// Add adds a handle to the group.
func (hg *HandleGroup) Add(handle *BulletHandle) {
	hg.mu.Lock()
	defer hg.mu.Unlock()
	hg.handles = append(hg.handles, handle)
}

// UpdateAll updates all handles in the group.
func (hg *HandleGroup) UpdateAll(level Level, msg string) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	for _, h := range hg.handles {
		h.Update(level, msg)
	}
}

// SuccessAll marks all handles as success.
func (hg *HandleGroup) SuccessAll(msg string) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	for _, h := range hg.handles {
		h.Success(msg)
	}
}

// ErrorAll marks all handles as error.
func (hg *HandleGroup) ErrorAll(msg string) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	for _, h := range hg.handles {
		h.Error(msg)
	}
}

// UpdateEach updates each handle with a different message.
func (hg *HandleGroup) UpdateEach(updates map[int]struct {
	Level   Level
	Message string
}) {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	for idx, update := range updates {
		if idx < len(hg.handles) {
			hg.handles[idx].Update(update.Level, update.Message)
		}
	}
}

// Clear removes all handles from the group.
func (hg *HandleGroup) Clear() {
	hg.mu.Lock()
	defer hg.mu.Unlock()
	hg.handles = nil
}

// Size returns the number of handles in the group.
func (hg *HandleGroup) Size() int {
	hg.mu.RLock()
	defer hg.mu.RUnlock()
	return len(hg.handles)
}

// Get returns the handle at the specified index.
func (hg *HandleGroup) Get(index int) *BulletHandle {
	hg.mu.RLock()
	defer hg.mu.RUnlock()

	if index >= 0 && index < len(hg.handles) {
		return hg.handles[index]
	}
	return nil
}

// HandleChain allows chaining updates to multiple handles.
type HandleChain struct {
	handles []*BulletHandle
}

// Chain creates a new handle chain.
func Chain(handles ...*BulletHandle) *HandleChain {
	return &HandleChain{handles: handles}
}

// Update updates all handles in the chain.
func (hc *HandleChain) Update(level Level, msg string) *HandleChain {
	for _, h := range hc.handles {
		h.Update(level, msg)
	}
	return hc
}

// Success marks all handles in the chain as success.
func (hc *HandleChain) Success(msg string) *HandleChain {
	for _, h := range hc.handles {
		h.Success(msg)
	}
	return hc
}

// Error marks all handles in the chain as error.
func (hc *HandleChain) Error(msg string) *HandleChain {
	for _, h := range hc.handles {
		h.Error(msg)
	}
	return hc
}

// WithField adds a field to all handles in the chain.
func (hc *HandleChain) WithField(key string, value any) *HandleChain {
	for _, h := range hc.handles {
		h.WithField(key, value)
	}
	return hc
}