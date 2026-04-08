// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package traceprof

import (
	"sync"
	"sync/atomic"
)

// globalEndpointCounter is shared between the profiler and the tracer.
var globalEndpointCounter = (func() *EndpointCounter {
	// Create endpoint counter with arbitrary limit.
	// The pathological edge case would be a service with a high rate (10k/s) of
	// short (100ms) spans with unique endpoints (resource names). Over a 60s
	// period this would grow the map to 600k items which may cause noticable
	// memory, GC overhead and lock contention overhead. The pprof endpoint
	// labels are less problematic since there will only be 1000 spans in-flight
	// on average. Using a limit of 1000 will result in a similar overhead of
	// this features compared to the pprof labels. It also seems like a
	// reasonable upper bound for the number of endpoints a normal application
	// may service in a 60s period.
	ec := NewEndpointCounter(1000)
	// Disabled by default ensures almost-zero overhead for tracing users that
	// don't have the profiler turned on.
	ec.SetEnabled(false)
	return ec
})()

// GlobalEndpointCounter returns the endpoint counter that is shared between
// tracing and profiling to support the unit of work feature.
func GlobalEndpointCounter() *EndpointCounter {
	return globalEndpointCounter
}

// NewEndpointCounter returns a new NewEndpointCounter that will track hit
// counts for up to limit endpoints. A limit of <= 0 indicates no limit.
func NewEndpointCounter(limit int) *EndpointCounter {
	return &EndpointCounter{enabled: 1, limit: limit, counts: map[string]uint64{}}
}

// EndpointCounter counts hits per endpoint.
//
// TODO: This is a naive implementation with poor performance, e.g. 125ns/op in
// BenchmarkEndpointCounter on M1. We can do 10-20x better with something more
// complicated [1]. This will be done in a follow-up PR.
// [1] https://github.com/felixge/countermap/blob/main/xsync_map_counter_map.go
type EndpointCounter struct {
	enabled uint64
	mu      sync.Mutex
	counts  map[string]uint64
	limit   int
}

// SetEnabled changes if endpoint counting is enabled or not. The previous
// value is returned.
func (e *EndpointCounter) SetEnabled(enabled bool) bool {
	oldVal := atomic.SwapUint64(&e.enabled, boolToUint64(enabled))
	return oldVal == 1
}

// Inc increments the hit counter for the given endpoint by 1. If endpoint
// counting is disabled, this method does nothing and is almost zero-cost.
func (e *EndpointCounter) Inc(endpoint string) {
	// Fast-path return if endpoint counter is disabled.
	if atomic.LoadUint64(&e.enabled) == 0 {
		return
	}

	// Acquire lock until func returns
	e.mu.Lock()
	defer e.mu.Unlock()

	// Don't add another endpoint to the map if the limit is reached. See
	// globalEndpointCounter comment.
	count, ok := e.counts[endpoint]
	if !ok && e.limit > 0 && len(e.counts) >= e.limit {
		return
	}
	// Increment the endpoint count
	e.counts[endpoint] = count + 1
}

// GetAndReset returns the hit counts for all endpoints and resets their counts
// back to 0.
func (e *EndpointCounter) GetAndReset() map[string]uint64 {
	// Acquire lock until func returns
	e.mu.Lock()
	defer e.mu.Unlock()

	// Return current counts and reset internal map.
	counts := e.counts
	e.counts = make(map[string]uint64)
	return counts
}

// boolToUint64 converts b to 0 if false or 1 if true.
func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
