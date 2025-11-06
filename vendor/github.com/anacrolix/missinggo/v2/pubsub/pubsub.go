package pubsub

import (
	"sync"
)

type PubSub[T any] struct {
	mu     sync.Mutex
	next   chan item[T]
	closed bool
}

type item[T any] struct {
	value T
	next  chan item[T]
}

type Subscription[T any] struct {
	next   chan item[T]
	Values chan T
	mu     sync.Mutex
	closed chan struct{}
}

func (me *PubSub[T]) init() {
	me.next = make(chan item[T], 1)
}

func (me *PubSub[T]) lazyInit() {
	me.mu.Lock()
	defer me.mu.Unlock()
	if me.closed {
		return
	}
	if me.next == nil {
		me.init()
	}
}

func (me *PubSub[T]) Publish(v T) {
	me.lazyInit()
	next := make(chan item[T], 1)
	i := item[T]{v, next}
	me.mu.Lock()
	if !me.closed {
		me.next <- i
		me.next = next
	}
	me.mu.Unlock()
}

func (me *Subscription[T]) Close() {
	me.mu.Lock()
	defer me.mu.Unlock()
	select {
	case <-me.closed:
	default:
		close(me.closed)
	}
}

func (me *Subscription[T]) runner() {
	defer close(me.Values)
	for {
		select {
		case i, ok := <-me.next:
			if !ok {
				me.Close()
				return
			}
			// Send the value back into the channel for someone else. This
			// won't block because the channel has a capacity of 1, and this
			// is currently the only copy of this value being sent to this
			// channel.
			me.next <- i
			// The next value comes from the channel given to us by the value
			// we just got.
			me.next = i.next
			select {
			case me.Values <- i.value:
			case <-me.closed:
				return
			}
		case <-me.closed:
			return
		}
	}
}

func (me *PubSub[T]) Subscribe() (ret *Subscription[T]) {
	me.lazyInit()
	ret = &Subscription[T]{
		closed: make(chan struct{}),
		Values: make(chan T),
	}
	me.mu.Lock()
	ret.next = me.next
	me.mu.Unlock()
	go ret.runner()
	return
}

func (me *PubSub[T]) Close() {
	me.mu.Lock()
	defer me.mu.Unlock()
	if me.closed {
		return
	}
	if me.next != nil {
		close(me.next)
	}
	me.closed = true
}
