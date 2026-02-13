package timer

import (
	"sync"
	"time"
)

type Timer struct {
	mu       sync.RWMutex
	elapsed  time.Duration
	running  bool
	interval time.Duration
	stopChan chan struct{}
}

func New() *Timer {
	return &Timer{
		interval: time.Second,
		stopChan: make(chan struct{}),
	}
}

func (t *Timer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return
	}

	t.running = true
	t.stopChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()

		for {
			select {
			case <-t.stopChan:
				return
			case <-ticker.C:
				t.mu.Lock()
				if !t.running {
					t.mu.Unlock()
					return
				}
				t.elapsed += t.interval
				t.mu.Unlock()
			}
		}
	}()
}

func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return
	}

	t.running = false
	close(t.stopChan)
	t.stopChan = make(chan struct{})
}

func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.running = false
	t.elapsed = 0
	if t.stopChan != nil {
		close(t.stopChan)
	}
	t.stopChan = make(chan struct{})
}

func (t *Timer) SetElapsed(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.elapsed = d
}

func (t *Timer) Elapsed() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.elapsed
}

func (t *Timer) Running() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}
