// +build leak

package gp

import "time"

type Pool struct{}

// New returns a new *Pool object.
// When compile with leak flag, goroutine will not be reusing.
func New(idleTimeout time.Duration) *Pool {
	return &Pool{}
}

// Go run f() in a new goroutine.
func (pool *Pool) Go(f func()) {
	go f()
}
