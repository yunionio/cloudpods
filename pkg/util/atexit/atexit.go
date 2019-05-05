package atexit

import (
	"os"
	"runtime/debug"
	"sort"
	"sync"
)

// ExitHandlerFunc is the type of handler func
type ExitHandlerFunc func(ExitHandler)

// ExitHandler defines the spec of handler
//
// Reason and Func are mandatory and must not be empty or nil
//
// Handlers with smaller Prio will be executed earlier than those with bigger
// Prio at exit time.  Handler func will receive a copy of the ExitHandler
// struct previously registered
type ExitHandler struct {
	Prio   int
	Reason string
	Func   ExitHandlerFunc
	Value  interface{}
}

var (
	handlers     = map[int][]ExitHandler{}
	handlersLock = &sync.Mutex{}
	once         = &sync.Once{}
)

// Register registers ExitHandler
//
// Smaller prio number mean higher priority and exit handlers with higher
// priority will be executed first.  For handlers with equal priorities, those
// registered first will be executed earlier at exit time
func Register(eh ExitHandler) {
	if eh.Reason == "" {
		panic("handler reason must not be empty")
	}
	if eh.Func == nil {
		panic("handler func must not be nil")
	}

	handlersLock.Lock()
	defer handlersLock.Unlock()

	ehs, ok := handlers[eh.Prio]
	if ok {
		ehs = append(ehs, eh)
	} else {
		ehs = []ExitHandler{eh}
	}
	handlers[eh.Prio] = ehs
}

// Handle calls registered handlers sequentially according to priority and
// registration order
//
// Panics caused by handler func will be caught, recorded, then next func will
// be run
func Handle() {
	once.Do(func() {
		handlersLock.Lock()
		defer handlersLock.Unlock()

		prios := make([]int, 0, len(handlers))
		for prio := range handlers {
			prios = append(prios, prio)
		}
		sort.Ints(prios)
		for _, prio := range prios {
			ehs := handlers[prio]
			for _, eh := range ehs {
				print("atexit: prio=", prio, ", reason=", eh.Reason, "\n")
				func() {
					defer func() {
						val := recover()
						if val != nil {
							print("panic ", val, "\n")
							debug.PrintStack()
						}
					}()
					eh.Func(eh)
				}()
			}
		}
	})
}

// Exit calls handlers then does os.Exit(code)
func Exit(code int) {
	defer os.Exit(code)
	Handle()
}
