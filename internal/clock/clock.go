// Package clock provides the injectable time source every time-dependent
// behaviour in the playground flows through, so tests can freeze or advance
// time instead of sleeping.
package clock

import (
	"sync"
	"time"
)

// Clock reads the current time.
type Clock interface {
	Now() time.Time
}

// Real reads the operating system clock.
type Real struct{}

func (Real) Now() time.Time { return time.Now() }

// Fixed always reports the same instant. Useful as the base of a Controllable
// when byte-identical output is required across runs.
type Fixed time.Time

func (f Fixed) Now() time.Time { return time.Time(f) }

// Controllable layers freeze, offset and absolute overrides on top of a base
// clock. Every session owns one, so manipulating time in one session cannot be
// observed from another. Safe for concurrent use.
type Controllable struct {
	base Clock

	mu     sync.RWMutex
	offset time.Duration
	frozen *time.Time
}

// New returns a Controllable running at the same rate as base.
func New(base Clock) *Controllable {
	if base == nil {
		base = Real{}
	}
	return &Controllable{base: base}
}

func (c *Controllable) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.frozen != nil {
		return *c.frozen
	}
	return c.base.Now().Add(c.offset)
}

// Frozen reports whether time has stopped advancing.
func (c *Controllable) Frozen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frozen != nil
}

// Freeze stops the clock at the current instant. Advance and Set still apply.
func (c *Controllable) Freeze() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen == nil {
		now := c.base.Now().Add(c.offset)
		c.frozen = &now
	}
}

// Unfreeze resumes real time, preserving the instant the clock was reading.
func (c *Controllable) Unfreeze() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen == nil {
		return
	}
	c.offset = c.frozen.Sub(c.base.Now())
	c.frozen = nil
}

// Advance moves the clock forward by d. Negative durations move it backwards.
func (c *Controllable) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen != nil {
		moved := c.frozen.Add(d)
		c.frozen = &moved
		return
	}
	c.offset += d
}

// Set moves the clock to t, keeping its frozen state.
func (c *Controllable) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen != nil {
		c.frozen = &t
		return
	}
	c.offset = t.Sub(c.base.Now())
}

// Reset returns the clock to real time with no offset.
func (c *Controllable) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.offset = 0
	c.frozen = nil
}
