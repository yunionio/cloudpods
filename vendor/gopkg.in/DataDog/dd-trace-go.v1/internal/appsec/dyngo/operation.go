// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package dyngo is the Go implementation of Datadog's Instrumentation Gateway
// which provides an event-based instrumentation API based on a stack
// representation of instrumented functions along with nested event listeners.
// It allows to both correlate passed and future function calls in order to
// react and monitor specific function call scenarios, while keeping the
// monitoring state local to the monitoring logic thanks to nested Go function
// closures.
// dyngo is not intended to be directly used and should be instead wrapped
// behind statically and strongly typed wrapper types. Indeed, dyngo is a
// generic implementation relying on empty interface values (values of type
// `interface{}`) and using it directly can be error-prone due to the lack of
// compile-time type-checking. For example, AppSec provides the package
// `httpsec`, built on top of dyngo, as its HTTP instrumentation API and which
// defines the abstract HTTP operation representation expected by the AppSec
// monitoring.
package dyngo

import (
	"reflect"
	"sort"
	"sync"
	"sync/atomic"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

// Operation interface type allowing to register event listeners to the
// operation. The event listeners will be automatically removed from the
// operation once it finishes so that it no longer can be called on finished
// operations.
type Operation interface {
	// On allows to register an event listener to the operation. The event
	// listener will be removed from the operation once it finishes.
	On(EventListener)

	// Parent return the parent operation. It returns nil for the root
	// operation.
	Parent() Operation

	// emitEvent emits the event to listeners of the given argsType and calls
	// them with the given op and v values.
	// emitEvent is a private method implemented by the operation struct type so
	// that no other package can define it.
	emitEvent(argsType reflect.Type, op Operation, v interface{})

	// register the given event listeners and return the unregistration
	// function allowing to remove the event listener from the operation.
	// register is a private method implemented by the operation struct type so
	// that no other package can define it.
	register(...EventListener) UnregisterFunc

	// finish the operation. This method allows to pass the operation value to
	// use to emit the finish event.
	// finish is a private method implemented by the operation struct type so
	// that no other package can define it.
	finish(op Operation, results interface{})
}

// EventListener interface allowing to identify the Go type listened to and
// dispatch calls to the underlying event listener function.
type EventListener interface {
	// ListenedType returns the Go type the event listener listens to.
	ListenedType() reflect.Type
	// Call the underlying event listener function. The type of the value v
	// is the type the event listener listens to, according to the type
	// returned by ListenedType().
	Call(op Operation, v interface{})
}

// UnregisterFunc is a function allowing to unregister from an operation the
// previously registered event listeners.
type UnregisterFunc func()

var rootOperation = newOperation(nil)

// Register global operation event listeners to listen to.
func Register(listeners ...EventListener) UnregisterFunc {
	return rootOperation.register(listeners...)
}

// operation structure allowing to subscribe to operation events and to
// navigate in the operation stack. Events
// bubble-up the operation stack, which allows listening to future events that
// might happen in the operation lifetime.
type operation struct {
	parent Operation
	eventRegister

	disabled bool
	mu       sync.RWMutex
}

// NewOperation creates and returns a new operationIt must be started by calling
// StartOperation, and finished by calling FinishOperation. The returned
// operation should be used in wrapper types to provide statically typed start
// and finish functions. The following example shows how to wrap an operation
// so that its functions are statically typed (instead of dyngo's interface{}
// values):
//   package mypackage
//   import "dyngo"
//   type (
//     MyOperation struct {
//       dyngo.Operation
//     }
//     MyOperationArgs { /* ... */ }
//     MyOperationRes { /* ... */ }
//   )
//   func StartOperation(args MyOperationArgs, parent dyngo.Operation) MyOperation {
//     op := MyOperation{Operation: dyngo.NewOperation(parent)}
//     dyngo.StartOperation(op, args)
//     return op
//   }
//   func (op MyOperation) Finish(res MyOperationRes) {
//       dyngo.FinishOperation(op, res)
//     }
func NewOperation(parent Operation) Operation {
	if parent == nil {
		parent = rootOperation
	}
	return newOperation(parent)
}

// StartOperation starts a new operation along with its arguments and emits a
// start event with the operation arguments.
func StartOperation(op Operation, args interface{}) {
	argsType := reflect.TypeOf(args)
	// Bubble-up the start event starting from the parent operation as you can't
	// listen for your own start event
	for current := op.Parent(); current != nil; current = current.Parent() {
		current.emitEvent(argsType, op, args)
	}
}

func newOperation(parent Operation) *operation {
	return &operation{parent: parent}
}

// Parent return the parent operation. It returns nil for the root operation.
func (o *operation) Parent() Operation {
	return o.parent
}

// FinishOperation finishes the operation along with its results and emits a
// finish event with the operation results.
// The operation is then disabled and its event listeners removed.
func FinishOperation(op Operation, results interface{}) {
	op.finish(op, results)
}

func (o *operation) finish(op Operation, results interface{}) {
	// Defer the call to o.disable() first so that the RWMutex gets unlocked first
	defer o.disable()
	o.mu.RLock()
	defer o.mu.RUnlock() // Deferred and stacked on top of the previously deferred call to o.disable()
	if o.disabled {
		return
	}
	resType := reflect.TypeOf(results)
	for current := op; current != nil; current = current.Parent() {
		current.emitEvent(resType, op, results)
	}
}

// Disable the operation and remove all its event listeners.
func (o *operation) disable() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.disabled {
		return
	}
	o.disabled = true
	o.eventRegister.clear()
}

// Register allows to register the given event listeners to the operation. An
// unregistration function is returned allowing to unregister the event
// listeners from the operation.
func (o *operation) register(l ...EventListener) UnregisterFunc {
	// eventRegisterIndex allows to lookup for the event listener in the event register.
	type eventRegisterIndex struct {
		key reflect.Type
		id  eventListenerID
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.disabled {
		return func() {}
	}
	indices := make([]eventRegisterIndex, len(l))
	for i, l := range l {
		if l == nil {
			continue
		}
		key := l.ListenedType()
		id := o.eventRegister.add(key, l)
		indices[i] = eventRegisterIndex{
			key: key,
			id:  id,
		}
	}
	return func() {
		for _, ix := range indices {
			o.eventRegister.remove(ix.key, ix.id)
		}
	}
}

// On registers the event listener. The difference with the Register() is that
// it doesn't return a function closure, which avoids unnecessary allocations
// For example:
//     op.On(MyOperationStart(func (op MyOperation, args MyOperationArgs) {
//         // ...
//     }))
func (o *operation) On(l EventListener) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.disabled {
		return
	}
	o.eventRegister.add(l.ListenedType(), l)
}

type (
	// eventRegister implements a thread-safe list of event listeners.
	eventRegister struct {
		mu        sync.RWMutex
		listeners eventListenerMap
	}

	// eventListenerMap is the map of event listeners. The list of listeners are
	// indexed by the operation argument or result type the event listener
	// expects.
	eventListenerMap      map[reflect.Type][]eventListenerMapEntry
	eventListenerMapEntry struct {
		id       eventListenerID
		listener EventListener
	}

	// eventListenerID is the unique ID of an event when registering it. It
	// allows to find it back and remove it from the list of event listeners
	// when unregistering it.
	eventListenerID uint32
)

// lastID is the last event listener ID that was given to the latest event
// listener.
var lastID eventListenerID

// nextID atomically increments lastID and returns the new event listener ID to
// use.
func nextID() eventListenerID {
	return eventListenerID(atomic.AddUint32((*uint32)(&lastID), 1))
}

func (r *eventRegister) add(key reflect.Type, l EventListener) eventListenerID {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listeners == nil {
		r.listeners = make(eventListenerMap)
	}
	// id is computed when the lock is exclusively taken so that we know
	// listeners are added in incremental id order.
	// This allows to use the optimized sort.Search() function to remove the
	// entry.
	id := nextID()
	r.listeners[key] = append(r.listeners[key], eventListenerMapEntry{
		id:       id,
		listener: l,
	})
	return id
}

func (r *eventRegister) remove(key reflect.Type, id eventListenerID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listeners == nil {
		return
	}
	listeners := r.listeners[key]
	length := len(listeners)
	i := sort.Search(length, func(i int) bool {
		return listeners[i].id >= id
	})
	if i < length && listeners[i].id == id {
		r.listeners[key] = append(listeners[:i], listeners[i+1:]...)
	}
}

func (r *eventRegister) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = nil
}

func (r *eventRegister) emitEvent(key reflect.Type, op Operation, v interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("appsec: recovered from an unexpected panic from an event listener: %+v", r)
		}
	}()
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.listeners[key] {
		e.listener.Call(op, v)
	}
}
