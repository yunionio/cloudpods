// Package sync extends basic synchronization primitives.
package sync

import (
	"sync"
)

// A WaitGroup waits for a collection of goroutines to finish.
// The main goroutine calls Add to set the number of
// goroutines to wait for. Then each of the goroutines
// runs and calls Done when finished. At the same time,
// Wait can be used to block until all goroutines have finished.
//
// WaitGroups in the sync package do not allow adding or
// subtracting from the counter while another goroutine is
// waiting, while this one does.
//
// A WaitGroup must not be copied after first use.
//
// In the terminology of the Go memory model, a call to Done
type WaitGroup struct {
	c     int64
	mutex sync.Mutex
	cond  *sync.Cond
}

// NewWaitGroup creates a new WaitGroup.
func NewWaitGroup() *WaitGroup {
	wg := &WaitGroup{}
	wg.cond = sync.NewCond(&wg.mutex)
	return wg
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
func (wg *WaitGroup) Add(delta int) {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	wg.c += int64(delta)
	if wg.c < 0 {
		panic("udp: negative WaitGroup counter") // nolint
	}
	wg.cond.Signal()
}

// Done decrements the WaitGroup counter by one.
func (wg *WaitGroup) Done() {
	wg.Add(-1)
}

// Wait blocks until the WaitGroup counter is zero.
func (wg *WaitGroup) Wait() {
	wg.mutex.Lock()
	defer wg.mutex.Unlock()
	for {
		c := wg.c
		switch {
		case c == 0:
			// wake another goroutine if there is one
			wg.cond.Signal()
			return
		case c < 0:
			panic("udp: negative WaitGroup counter") // nolint
		}
		wg.cond.Wait()
	}
}
