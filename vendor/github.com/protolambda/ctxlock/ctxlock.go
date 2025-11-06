package ctxlock

import (
	"context"
	"sync"
)

// Lock is a channels based sync.Locker, as well as a read-write locker,
// and supporting context-timeouts on locking with LockCtx and RLockCtx.
//
// This lock allows lock-attempts to abort on context cancellation.
// It is used like a standard sync.RWMutex.
//
// The Lock must not be copied after use.
// Run `go vet -copylocks .` to detect copies.
//
// The implementation is heavily inspired by the channel based RWMutex described
// in [Roberto Clapis's series on advanced concurrency patterns].
// This very blog is [referenced in the Go 1.22 standard-library].
// But adapted to support context-cancellation and detect invalid usage.
//
// [referenced in the Go 1.22 standard-library]: https://github.com/golang/go/blob/a10e42f219abb9c5bc4e7d86d9464700a42c7d57/src/sync/cond.go#L34
// [Roberto Clapis's series on advanced concurrency patterns]: https://blogtitle.github.io/go-advanced-concurrency-patterns-part-3-channels/
type Lock struct {
	// enables us to just embed Lock anywhere without calling a constructor, alike to sync.Mutex usage.
	initOnce sync.Once

	// write and global lock. Empty if no lock is held.
	// Holds true if a write-purpose lock is held.
	// Holds false if a read-purpose lock is held.
	write chan bool

	// readers lock. Only non-empty if write lock is held.
	// Holds a counter of the number of active readers.
	// May be temporarily empty while readers enter or leave.
	readers chan int
}

func (l *Lock) init() {
	l.write = make(chan bool, 1)
	l.readers = make(chan int, 1)
}

// Lock acquires the write-lock, blocking until acquired.
func (l *Lock) Lock() {
	l.initOnce.Do(l.init)

	l.write <- true
}

// LockCtx tries to get the write-lock, but may abort with error if the provided ctx is canceled first.
func (l *Lock) LockCtx(ctx context.Context) error {
	if ctx == nil {
		panic("nil context argument")
	}
	l.initOnce.Do(l.init)

	select {
	case l.write <- true:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Unlock releases the write-lock. Unlock panics if the state was not write-locked, or held for reading purposes.
func (l *Lock) Unlock() {
	l.initOnce.Do(l.init)

	select {
	case v := <-l.write:
		if !v {
			panic("Unlock complete, but lock was held for reading")
		}
	default:
		panic("cannot Unlock: no write lock was held")
	}
}

// RLock acquires a read-lock, blocking until acquired.
func (l *Lock) RLock() {
	_ = l.rlock(nil)
}

// RLockCtx tries to get a read lock, but may abort with error if the provided ctx is canceled first.
func (l *Lock) RLockCtx(ctx context.Context) error {
	if ctx == nil {
		panic("nil context argument")
	}
	if l.rlock(ctx.Done()) {
		return ctx.Err()
	}
	return nil
}

// rlock is an internal helper, implementing read-locking.
// The read-lock may be aborted by signaling through a non-nil abort channel.
// If nil, the read-lock cannot be aborted.
func (l *Lock) rlock(abort <-chan struct{}) (aborted bool) {
	l.initOnce.Do(l.init)

	// Count current readers. Default to 0.
	var rs int
	// Select on the channels without default.
	// One and only one case will be selected and this
	// will block until one case becomes available.
	select {
	case l.write <- false: // One sending case for write.
		// If the write lock is available we have no readers.
		// We grab the write lock to prevent concurrent
		// read-writes.
	case rs = <-l.readers: // One receiving case for read.
		// There already are readers, let's grab and update the
		// readers count.
	case <-abort: // if abort == nil: the abort case is effectively ignored
		return true
	}
	// If we grabbed a write lock this is 0.
	rs++
	// Updated the readers count. If there are none this
	// just adds an item to the empty readers channel.
	l.readers <- rs
	return false
}

// RUnlock releases a read-lock.
// RUnlock panics if the state was not read-locked.
// RUnlock will block if there is a write-lock, and panic once the write lock
// is released and if a read-lock isn't acquired first.
func (l *Lock) RUnlock() {
	l.initOnce.Do(l.init)

	var rs int
	select {
	case l.write <- false:
		<-l.write
		panic("cannot RUnlock; no readers left, as there was no shared write lock held")
	case rs = <-l.readers:
	}
	// Take the value of readers and decrement it.
	rs--

	// If zero, make the write lock available again and return.
	if rs == 0 {
		<-l.write
		return
	}
	// If not zero just update the readers count.
	// 0 will never be written to the readers channel,
	// at most one of the two channels will have a value
	// at any given time.
	l.readers <- rs
}
