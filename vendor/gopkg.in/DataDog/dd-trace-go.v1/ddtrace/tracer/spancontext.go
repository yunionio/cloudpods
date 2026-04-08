// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"strconv"
	"sync"
	"sync/atomic"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/samplernames"
)

var _ ddtrace.SpanContext = (*spanContext)(nil)

// SpanContext represents a span state that can propagate to descendant spans
// and across process boundaries. It contains all the information needed to
// spawn a direct descendant of the span that it belongs to. It can be used
// to create distributed tracing by propagating it using the provided interfaces.
type spanContext struct {
	updated bool // updated is tracking changes for priority / origin / x-datadog-tags

	// the below group should propagate only locally

	trace  *trace // reference to the trace that this span belongs too
	span   *span  // reference to the span that hosts this context
	errors int32  // number of spans with errors in this trace

	// the below group should propagate cross-process

	traceID uint64
	spanID  uint64

	mu         sync.RWMutex // guards below fields
	baggage    map[string]string
	hasBaggage uint32 // atomic int for quick checking presence of baggage. 0 indicates no baggage, otherwise baggage exists.
	origin     string // e.g. "synthetics"
}

// newSpanContext creates a new SpanContext to serve as context for the given
// span. If the provided parent is not nil, the context will inherit the trace,
// baggage and other values from it. This method also pushes the span into the
// new context's trace and as a result, it should not be called multiple times
// for the same span.
func newSpanContext(span *span, parent *spanContext) *spanContext {
	context := &spanContext{
		traceID: span.TraceID,
		spanID:  span.SpanID,
		span:    span,
	}
	if parent != nil {
		context.trace = parent.trace
		context.origin = parent.origin
		context.errors = parent.errors
		parent.ForeachBaggageItem(func(k, v string) bool {
			context.setBaggageItem(k, v)
			return true
		})
	}
	if context.trace == nil {
		context.trace = newTrace()
	}
	if context.trace.root == nil {
		// first span in the trace can safely be assumed to be the root
		context.trace.root = span
	}
	// put span in context's trace
	context.trace.push(span)
	// setting context.updated to false here is necessary to distinguish
	// between initializing properties of the span (priority)
	// and updating them after extracting context through propagators
	context.updated = false
	return context
}

// SpanID implements ddtrace.SpanContext.
func (c *spanContext) SpanID() uint64 { return c.spanID }

// TraceID implements ddtrace.SpanContext.
func (c *spanContext) TraceID() uint64 { return c.traceID }

// ForeachBaggageItem implements ddtrace.SpanContext.
func (c *spanContext) ForeachBaggageItem(handler func(k, v string) bool) {
	if atomic.LoadUint32(&c.hasBaggage) == 0 {
		return
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.baggage {
		if !handler(k, v) {
			break
		}
	}
}

func (c *spanContext) setSamplingPriority(p int, sampler samplernames.SamplerName) {
	if c.trace == nil {
		c.trace = newTrace()
	}
	if c.trace.priority != nil && *c.trace.priority != float64(p) {
		c.updated = true
	}
	c.trace.setSamplingPriority(p, sampler)
}

func (c *spanContext) samplingPriority() (p int, ok bool) {
	if c.trace == nil {
		return 0, false
	}
	return c.trace.samplingPriority()
}

func (c *spanContext) setBaggageItem(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.baggage == nil {
		atomic.StoreUint32(&c.hasBaggage, 1)
		c.baggage = make(map[string]string, 1)
	}
	c.baggage[key] = val
}

func (c *spanContext) baggageItem(key string) string {
	if atomic.LoadUint32(&c.hasBaggage) == 0 {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baggage[key]
}

func (c *spanContext) meta(key string) (val string, ok bool) {
	c.span.RLock()
	defer c.span.RUnlock()
	val, ok = c.span.Meta[key]
	return val, ok
}

// finish marks this span as finished in the trace.
func (c *spanContext) finish() { c.trace.finishedOne(c.span) }

// samplingDecision is the decision to send a trace to the agent or not.
type samplingDecision uint32

const (
	// decisionNone is the default state of a trace.
	// If no decision is made about the trace, the trace won't be sent to the agent.
	decisionNone samplingDecision = iota
	// decisionDrop prevents the trace from being sent to the agent.
	decisionDrop
	// decisionKeep ensures the trace will be sent to the agent.
	decisionKeep
)

// trace contains shared context information about a trace, such as sampling
// priority, the root reference and a buffer of the spans which are part of the
// trace, if these exist.
type trace struct {
	mu               sync.RWMutex      // guards below fields
	spans            []*span           // all the spans that are part of this trace
	tags             map[string]string // trace level tags
	propagatingTags  map[string]string // trace level tags that will be propagated across service boundaries
	finished         int               // the number of finished spans
	full             bool              // signifies that the span buffer is full
	priority         *float64          // sampling priority
	locked           bool              // specifies if the sampling priority can be altered
	samplingDecision samplingDecision  // samplingDecision indicates whether to send the trace to the agent.

	// root specifies the root of the trace, if known; it is nil when a span
	// context is extracted from a carrier, at which point there are no spans in
	// the trace yet.
	root *span
}

var (
	// traceStartSize is the initial size of our trace buffer,
	// by default we allocate for a handful of spans within the trace,
	// reasonable as span is actually way bigger, and avoids re-allocating
	// over and over. Could be fine-tuned at runtime.
	traceStartSize = 10
	// traceMaxSize is the maximum number of spans we keep in memory for a
	// single trace. This is to avoid memory leaks. If more spans than this
	// are added to a trace, then the trace is dropped and the spans are
	// discarded. Adding additional spans after a trace is dropped does
	// nothing.
	traceMaxSize = int(1e5)
)

// newTrace creates a new trace using the given callback which will be called
// upon completion of the trace.
func newTrace() *trace {
	return &trace{spans: make([]*span, 0, traceStartSize)}
}

func (t *trace) samplingPriorityLocked() (p int, ok bool) {
	if t.priority == nil {
		return 0, false
	}
	return int(*t.priority), true
}

func (t *trace) samplingPriority() (p int, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.samplingPriorityLocked()
}

func (t *trace) setSamplingPriority(p int, sampler samplernames.SamplerName) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.setSamplingPriorityLocked(p, sampler)
}

func (t *trace) keep() {
	atomic.CompareAndSwapUint32((*uint32)(&t.samplingDecision), uint32(decisionNone), uint32(decisionKeep))
}

func (t *trace) drop() {
	atomic.CompareAndSwapUint32((*uint32)(&t.samplingDecision), uint32(decisionNone), uint32(decisionDrop))
}

func (t *trace) setTag(key, value string) {
	if t.tags == nil {
		t.tags = make(map[string]string, 1)
	}
	t.tags[key] = value
}

// setPropagatingTag sets the key/value pair as a trace propagating tag.
func (t *trace) setPropagatingTag(key, value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.setPropagatingTagLocked(key, value)
}

// setPropagatingTagLocked sets the key/value pair as a trace propagating tag.
// Not safe for concurrent use, setPropagatingTag should be used instead in that case.
func (t *trace) setPropagatingTagLocked(key, value string) {
	if t.propagatingTags == nil {
		t.propagatingTags = make(map[string]string, 1)
	}
	t.propagatingTags[key] = value
}

// unsetPropagatingTag deletes the key/value pair from the trace's propagated tags.
func (t *trace) unsetPropagatingTag(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.propagatingTags, key)
}

func (t *trace) setSamplingPriorityLocked(p int, sampler samplernames.SamplerName) {
	if t.locked {
		return
	}
	if t.priority == nil {
		t.priority = new(float64)
	}
	*t.priority = float64(p)
	_, ok := t.propagatingTags[keyDecisionMaker]
	if p > 0 && !ok && sampler != samplernames.Unknown {
		// We have a positive priority and the sampling mechanism isn't set.
		// Send nothing when sampler is `Unknown` for RFC compliance.
		t.setPropagatingTagLocked(keyDecisionMaker, "-"+strconv.Itoa(int(sampler)))
	}
	if p <= 0 && ok {
		delete(t.propagatingTags, keyDecisionMaker)
	}
}

// push pushes a new span into the trace. If the buffer is full, it returns
// a errBufferFull error.
func (t *trace) push(sp *span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.full {
		return
	}
	tr, haveTracer := internal.GetGlobalTracer().(*tracer)
	if len(t.spans) >= traceMaxSize {
		// capacity is reached, we will not be able to complete this trace.
		t.full = true
		t.spans = nil // GC
		log.Error("trace buffer full (%d), dropping trace", traceMaxSize)
		if haveTracer {
			atomic.AddUint32(&tr.tracesDropped, 1)
		}
		return
	}
	if v, ok := sp.Metrics[keySamplingPriority]; ok {
		t.setSamplingPriorityLocked(int(v), samplernames.Unknown)
	}
	t.spans = append(t.spans, sp)
	if haveTracer {
		atomic.AddUint32(&tr.spansStarted, 1)
	}
}

// finishedOne acknowledges that another span in the trace has finished, and checks
// if the trace is complete, in which case it calls the onFinish function. It uses
// the given priority, if non-nil, to mark the root span.
func (t *trace) finishedOne(s *span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.full {
		// capacity has been reached, the buffer is no longer tracking
		// all the spans in the trace, so the below conditions will not
		// be accurate and would trigger a pre-mature flush, exposing us
		// to a race condition where spans can be modified while flushing.
		return
	}
	t.finished++
	if s == t.root && t.priority != nil {
		// after the root has finished we lock down the priority;
		// we won't be able to make changes to a span after finishing
		// without causing a race condition.
		t.root.setMetric(keySamplingPriority, *t.priority)
		t.locked = true
	}
	if len(t.spans) > 0 && s == t.spans[0] {
		// first span in chunk finished, lock down the tags
		//
		// TODO(barbayar): make sure this doesn't happen in vain when switching to
		// the new wire format. We won't need to set the tags on the first span
		// in the chunk there.
		for k, v := range t.tags {
			s.setMeta(k, v)
		}
		for k, v := range t.propagatingTags {
			s.setMeta(k, v)
		}
	}
	if len(t.spans) != t.finished {
		return
	}
	defer func() {
		t.spans = nil
		t.finished = 0 // important, because a buffer can be used for several flushes
	}()
	tr, ok := internal.GetGlobalTracer().(*tracer)
	if !ok {
		return
	}
	// we have a tracer that can receive completed traces.
	atomic.AddUint32(&tr.spansFinished, uint32(len(t.spans)))
	tr.pushTrace(&finishedTrace{
		spans:    t.spans,
		willSend: decisionKeep == samplingDecision(atomic.LoadUint32((*uint32)(&t.samplingDecision))),
	})
}
