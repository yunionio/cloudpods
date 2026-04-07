// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	gocontext "context"
	"os"
	"runtime/pprof"
	rt "runtime/trace"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/remoteconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/traceprof"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
)

var _ ddtrace.Tracer = (*tracer)(nil)

// tracer creates, buffers and submits Spans which are used to time blocks of
// computation. They are accumulated and streamed into an internal payload,
// which is flushed to the agent whenever its size exceeds a specific threshold
// or when a certain interval of time has passed, whichever happens first.
//
// tracer operates based on a worker loop which responds to various request
// channels. It additionally holds two buffers which accumulates error and trace
// queues to be processed by the payload encoder.
type tracer struct {
	config *config

	// stats specifies the concentrator used to compute statistics, when client-side
	// stats are enabled.
	stats *concentrator

	// traceWriter is responsible for sending finished traces to their
	// destination, such as the Trace Agent or Datadog Forwarder.
	traceWriter traceWriter

	// out receives finishedTrace with spans  to be added to the payload.
	out chan *finishedTrace

	// flush receives a channel onto which it will confirm after a flush has been
	// triggered and completed.
	flush chan chan<- struct{}

	// stop causes the tracer to shut down when closed.
	stop chan struct{}

	// stopOnce ensures the tracer is stopped exactly once.
	stopOnce sync.Once

	// wg waits for all goroutines to exit when stopping.
	wg sync.WaitGroup

	// prioritySampling holds an instance of the priority sampler.
	prioritySampling *prioritySampler

	// pid of the process
	pid int

	// These integers track metrics about spans and traces as they are started,
	// finished, and dropped
	spansStarted, spansFinished, tracesDropped uint32

	// Records the number of dropped P0 traces and spans.
	droppedP0Traces, droppedP0Spans uint32

	// partialTrace the number of partially dropped traces.
	partialTraces uint32

	// rulesSampling holds an instance of the rules sampler used to apply either trace sampling,
	// or single span sampling rules on spans. These are user-defined
	// rules for applying a sampling rate to spans that match the designated service
	// or operation name.
	rulesSampling *rulesSampler

	// obfuscator holds the obfuscator used to obfuscate resources in aggregated stats.
	// obfuscator may be nil if disabled.
	obfuscator *obfuscate.Obfuscator

	// statsd is used for tracking metrics associated with the runtime and the tracer.
	statsd statsdClient
}

const (
	// flushInterval is the interval at which the payload contents will be flushed
	// to the transport.
	flushInterval = 2 * time.Second

	// payloadMaxLimit is the maximum payload size allowed and should indicate the
	// maximum size of the package that the agent can receive.
	payloadMaxLimit = 9.5 * 1024 * 1024 // 9.5 MB

	// payloadSizeLimit specifies the maximum allowed size of the payload before
	// it will trigger a flush to the transport.
	payloadSizeLimit = payloadMaxLimit / 2

	// concurrentConnectionLimit specifies the maximum number of concurrent outgoing
	// connections allowed.
	concurrentConnectionLimit = 100
)

// statsInterval is the interval at which health metrics will be sent with the
// statsd client; replaced in tests.
var statsInterval = 10 * time.Second

// Start starts the tracer with the given set of options. It will stop and replace
// any running tracer, meaning that calling it several times will result in a restart
// of the tracer by replacing the current instance with a new one.
func Start(opts ...StartOption) {
	if internal.Testing {
		return // mock tracer active
	}
	t := newTracer(opts...)
	if !t.config.enabled {
		return
	}
	internal.SetGlobalTracer(t)
	if t.config.logStartup {
		logStartup(t)
	}
	// Start AppSec with remote configuration
	cfg := remoteconfig.DefaultClientConfig()
	cfg.AgentURL = t.config.agentURL
	cfg.AppVersion = t.config.version
	cfg.Env = t.config.env
	cfg.HTTP = t.config.httpClient
	cfg.ServiceName = t.config.serviceName
	appsec.Start(appsec.WithRCConfig(cfg))
}

// Stop stops the started tracer. Subsequent calls are valid but become no-op.
func Stop() {
	internal.SetGlobalTracer(&internal.NoopTracer{})
	log.Flush()
}

// Span is an alias for ddtrace.Span. It is here to allow godoc to group methods returning
// ddtrace.Span. It is recommended and is considered more correct to refer to this type as
// ddtrace.Span instead.
type Span = ddtrace.Span

// StartSpan starts a new span with the given operation name and set of options.
// If the tracer is not started, calling this function is a no-op.
func StartSpan(operationName string, opts ...StartSpanOption) Span {
	return internal.GetGlobalTracer().StartSpan(operationName, opts...)
}

// Extract extracts a SpanContext from the carrier. The carrier is expected
// to implement TextMapReader, otherwise an error is returned.
// If the tracer is not started, calling this function is a no-op.
func Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	return internal.GetGlobalTracer().Extract(carrier)
}

// Inject injects the given SpanContext into the carrier. The carrier is
// expected to implement TextMapWriter, otherwise an error is returned.
// If the tracer is not started, calling this function is a no-op.
func Inject(ctx ddtrace.SpanContext, carrier interface{}) error {
	return internal.GetGlobalTracer().Inject(ctx, carrier)
}

// SetUser associates user information to the current trace which the
// provided span belongs to. The options can be used to tune which user
// bit of information gets monitored. In case of distributed traces,
// the user id can be propagated across traces using the WithPropagation() option.
// See https://docs.datadoghq.com/security_platform/application_security/setup_and_configure/?tab=set_user#add-user-information-to-traces
func SetUser(s Span, id string, opts ...UserMonitoringOption) {
	if s == nil {
		return
	}
	sp, ok := s.(interface {
		SetUser(string, ...UserMonitoringOption)
	})
	if !ok {
		return
	}
	sp.SetUser(id, opts...)
}

// payloadQueueSize is the buffer size of the trace channel.
const payloadQueueSize = 1000

func newUnstartedTracer(opts ...StartOption) *tracer {
	c := newConfig(opts...)
	sampler := newPrioritySampler()
	statsd, err := newStatsdClient(c)
	if err != nil {
		log.Warn("Runtime and health metrics disabled: %v", err)
	}
	var writer traceWriter
	if c.logToStdout {
		writer = newLogTraceWriter(c, statsd)
	} else {
		writer = newAgentTraceWriter(c, sampler, statsd)
	}
	traces, spans, err := samplingRulesFromEnv()
	if err != nil {
		log.Warn("DIAGNOSTICS Error(s) parsing sampling rules: found errors:%s", err)
	}
	if traces != nil {
		c.traceRules = traces
	}
	if spans != nil {
		c.spanRules = spans
	}
	t := &tracer{
		config:           c,
		traceWriter:      writer,
		out:              make(chan *finishedTrace, payloadQueueSize),
		stop:             make(chan struct{}),
		flush:            make(chan chan<- struct{}),
		rulesSampling:    newRulesSampler(c.traceRules, c.spanRules),
		prioritySampling: sampler,
		pid:              os.Getpid(),
		stats:            newConcentrator(c, defaultStatsBucketSize),
		obfuscator: obfuscate.NewObfuscator(obfuscate.Config{
			SQL: obfuscate.SQLConfig{
				TableNames:       c.agent.HasFlag("table_names"),
				ReplaceDigits:    c.agent.HasFlag("quantize_sql_tables") || c.agent.HasFlag("replace_sql_digits"),
				KeepSQLAlias:     c.agent.HasFlag("keep_sql_alias"),
				DollarQuotedFunc: c.agent.HasFlag("dollar_quoted_func"),
				Cache:            c.agent.HasFlag("sql_cache"),
			},
		}),
		statsd: statsd,
	}
	return t
}

func newTracer(opts ...StartOption) *tracer {
	t := newUnstartedTracer(opts...)
	c := t.config
	t.statsd.Incr("datadog.tracer.started", nil, 1)
	if c.runtimeMetrics {
		log.Debug("Runtime metrics enabled.")
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			t.reportRuntimeMetrics(defaultMetricsReportInterval)
		}()
	}
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		tick := t.config.tickChan
		if tick == nil {
			ticker := time.NewTicker(flushInterval)
			defer ticker.Stop()
			tick = ticker.C
		}
		t.worker(tick)
	}()
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.reportHealthMetrics(statsInterval)
	}()
	t.stats.Start()
	return t
}

// Flush flushes any buffered traces. Flush is in effect only if a tracer
// is started. Users do not have to call Flush in order to ensure that
// traces reach Datadog. It is a convenience method dedicated to a specific
// use case described below.
//
// Flush is of use in Lambda environments, where starting and stopping
// the tracer on each invokation may create too much latency. In this
// scenario, a tracer may be started and stopped by the parent process
// whereas the invokation can make use of Flush to ensure any created spans
// reach the agent.
func Flush() {
	if t, ok := internal.GetGlobalTracer().(*tracer); ok {
		t.flushSync()
	}
}

// flushSync triggers a flush and waits for it to complete.
func (t *tracer) flushSync() {
	done := make(chan struct{})
	t.flush <- done
	<-done
}

// worker receives finished traces to be added into the payload, as well
// as periodically flushes traces to the transport.
func (t *tracer) worker(tick <-chan time.Time) {
	for {
		select {
		case trace := <-t.out:
			t.sampleFinishedTrace(trace)
			if len(trace.spans) != 0 {
				t.traceWriter.add(trace.spans)
			}
		case <-tick:
			t.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:scheduled"}, 1)
			t.traceWriter.flush()

		case done := <-t.flush:
			t.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:invoked"}, 1)
			t.traceWriter.flush()
			t.statsd.Flush()
			t.stats.flushAndSend(time.Now(), withCurrentBucket)
			// TODO(x): In reality, the traceWriter.flush() call is not synchronous
			// when using the agent traceWriter. However, this functionnality is used
			// in Lambda so for that purpose this mechanism should suffice.
			done <- struct{}{}

		case <-t.stop:
		loop:
			// the loop ensures that the payload channel is fully drained
			// before the final flush to ensure no traces are lost (see #526)
			for {
				select {
				case trace := <-t.out:
					t.sampleFinishedTrace(trace)
					if len(trace.spans) != 0 {
						t.traceWriter.add(trace.spans)
					}
				default:
					break loop
				}
			}
			return
		}
	}
}

// finishedTrace holds information about a trace that has finished, including its spans.
type finishedTrace struct {
	spans    []*span
	willSend bool // willSend indicates whether the trace will be sent to the agent.
}

// sampleFinishedTrace applies single-span sampling to the provided trace, which is considered to be finished.
func (t *tracer) sampleFinishedTrace(info *finishedTrace) {
	if len(info.spans) > 0 {
		if p, ok := info.spans[0].context.samplingPriority(); ok && p > 0 {
			// The trace is kept, no need to run single span sampling rules.
			return
		}
	}
	var kept []*span
	if t.rulesSampling.HasSpanRules() {
		// Apply sampling rules to individual spans in the trace.
		for _, span := range info.spans {
			if t.rulesSampling.SampleSpan(span) {
				kept = append(kept, span)
			}
		}
		if len(kept) > 0 && len(kept) < len(info.spans) {
			// Some spans in the trace were kept, so a partial trace will be sent.
			atomic.AddUint32(&t.partialTraces, 1)
		}
	}
	if len(kept) == 0 {
		atomic.AddUint32(&t.droppedP0Traces, 1)
	}
	atomic.AddUint32(&t.droppedP0Spans, uint32(len(info.spans)-len(kept)))
	if !info.willSend {
		info.spans = kept
	}
}

func (t *tracer) pushTrace(trace *finishedTrace) {
	select {
	case <-t.stop:
		return
	default:
	}
	select {
	case t.out <- trace:
	default:
		log.Error("payload queue full, dropping %d traces", len(trace.spans))
	}
}

// StartSpan creates, starts, and returns a new Span with the given `operationName`.
func (t *tracer) StartSpan(operationName string, options ...ddtrace.StartSpanOption) ddtrace.Span {
	var opts ddtrace.StartSpanConfig
	for _, fn := range options {
		fn(&opts)
	}
	var startTime int64
	if opts.StartTime.IsZero() {
		startTime = now()
	} else {
		startTime = opts.StartTime.UnixNano()
	}
	var context *spanContext
	// The default pprof context is taken from the start options and is
	// not nil when using StartSpanFromContext()
	pprofContext := opts.Context
	if opts.Parent != nil {
		if ctx, ok := opts.Parent.(*spanContext); ok {
			context = ctx
			if pprofContext == nil && ctx.span != nil {
				// Inherit the context.Context from parent span if it was propagated
				// using ChildOf() rather than StartSpanFromContext(), see
				// applyPPROFLabels() below.
				pprofContext = ctx.span.pprofCtxActive
			}
		}
	}
	if pprofContext == nil {
		// For root span's without context, there is no pprofContext, but we need
		// one to avoid a panic() in pprof.WithLabels(). Using context.Background()
		// is not ideal here, as it will cause us to remove all labels from the
		// goroutine when the span finishes. However, the alternatives of not
		// applying labels for such spans or to leave the endpoint/hotspot labels
		// on the goroutine after it finishes are even less appealing. We'll have
		// to properly document this for users.
		pprofContext = gocontext.Background()
	}
	id := opts.SpanID
	if id == 0 {
		id = generateSpanID(startTime)
	}
	// span defaults
	span := &span{
		Name:         operationName,
		Service:      t.config.serviceName,
		Resource:     operationName,
		SpanID:       id,
		TraceID:      id,
		Start:        startTime,
		noDebugStack: t.config.noDebugStack,
	}
	if t.config.hostname != "" {
		span.setMeta(keyHostname, t.config.hostname)
	}
	if context != nil {
		// this is a child span
		span.TraceID = context.traceID
		span.ParentID = context.spanID
		if p, ok := context.samplingPriority(); ok {
			span.setMetric(keySamplingPriority, float64(p))
		}
		if context.span != nil {
			// local parent, inherit service
			context.span.RLock()
			span.Service = context.span.Service
			context.span.RUnlock()
		} else {
			// remote parent
			if context.origin != "" {
				// mark origin
				span.setMeta(keyOrigin, context.origin)
			}
		}
	}
	span.context = newSpanContext(span, context)
	span.setMetric(ext.Pid, float64(t.pid))
	span.setMeta("language", "go")

	// add tags from options
	for k, v := range opts.Tags {
		span.SetTag(k, v)
	}
	// add global tags
	for k, v := range t.config.globalTags {
		span.SetTag(k, v)
	}
	if t.config.serviceMappings != nil {
		if newSvc, ok := t.config.serviceMappings[span.Service]; ok {
			span.Service = newSvc
		}
	}
	if context == nil || context.span == nil || context.span.Service != span.Service {
		span.setMetric(keyTopLevel, 1)
		// all top level spans are measured. So the measured tag is redundant.
		delete(span.Metrics, keyMeasured)
	}
	if t.config.version != "" {
		if t.config.universalVersion || (!t.config.universalVersion && span.Service == t.config.serviceName) {
			span.setMeta(ext.Version, t.config.version)
		}
	}
	if t.config.env != "" {
		span.setMeta(ext.Environment, t.config.env)
	}
	if _, ok := span.context.samplingPriority(); !ok {
		// if not already sampled or a brand new trace, sample it
		t.sample(span)
	}
	pprofContext, span.taskEnd = startExecutionTracerTask(pprofContext, span)
	if t.config.profilerHotspots || t.config.profilerEndpoints {
		t.applyPPROFLabels(pprofContext, span)
	}
	if t.config.serviceMappings != nil {
		if newSvc, ok := t.config.serviceMappings[span.Service]; ok {
			span.Service = newSvc
		}
	}
	if log.DebugEnabled() {
		// avoid allocating the ...interface{} argument if debug logging is disabled
		log.Debug("Started Span: %v, Operation: %s, Resource: %s, Tags: %v, %v",
			span, span.Name, span.Resource, span.Meta, span.Metrics)
	}
	return span
}

// generateSpanID returns a random uint64 that has been XORd with the startTime.
// This is done to get around the 32-bit random seed limitation that may create collisions if there is a large number
// of go services all generating spans.
func generateSpanID(startTime int64) uint64 {
	return random.Uint64() ^ uint64(startTime)
}

// applyPPROFLabels applies pprof labels for the profiler's code hotspots and
// endpoint filtering feature to span. When span finishes, any pprof labels
// found in ctx are restored. Additionally this func informs the profiler how
// many times each endpoint is called.
func (t *tracer) applyPPROFLabels(ctx gocontext.Context, span *span) {
	var labels []string
	if t.config.profilerHotspots {
		// allocate the max-length slice to avoid growing it later
		labels = make([]string, 0, 6)
		labels = append(labels, traceprof.SpanID, strconv.FormatUint(span.SpanID, 10))
	}
	// nil checks might not be needed, but better be safe than sorry
	if localRootSpan := span.root(); localRootSpan != nil {
		if t.config.profilerHotspots {
			labels = append(labels, traceprof.LocalRootSpanID, strconv.FormatUint(localRootSpan.SpanID, 10))
		}
		if t.config.profilerEndpoints && spanResourcePIISafe(localRootSpan) {
			labels = append(labels, traceprof.TraceEndpoint, localRootSpan.Resource)
			if span == localRootSpan {
				// Inform the profiler of endpoint hits. This is used for the unit of
				// work feature. We can't use APM stats for this since the stats don't
				// have enough cardinality (e.g. runtime-id tags are missing).
				traceprof.GlobalEndpointCounter().Inc(localRootSpan.Resource)
			}
		}
	}
	if len(labels) > 0 {
		span.pprofCtxRestore = ctx
		span.pprofCtxActive = pprof.WithLabels(ctx, pprof.Labels(labels...))
		pprof.SetGoroutineLabels(span.pprofCtxActive)
	}
}

// spanResourcePIISafe returns true if s.Resource can be considered to not
// include PII with reasonable confidence. E.g. SQL queries may contain PII,
// but http, rpc or custom (s.Type == "") span resource names generally do not.
func spanResourcePIISafe(s *span) bool {
	return s.Type == ext.SpanTypeWeb || s.Type == ext.AppTypeRPC || s.Type == ""
}

// Stop stops the tracer.
func (t *tracer) Stop() {
	t.stopOnce.Do(func() {
		close(t.stop)
		t.statsd.Incr("datadog.tracer.stopped", nil, 1)
	})
	t.stats.Stop()
	t.wg.Wait()
	t.traceWriter.stop()
	t.statsd.Close()
	appsec.Stop()
}

// Inject uses the configured or default TextMap Propagator.
func (t *tracer) Inject(ctx ddtrace.SpanContext, carrier interface{}) error {
	return t.config.propagator.Inject(ctx, carrier)
}

// Extract uses the configured or default TextMap Propagator.
func (t *tracer) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	return t.config.propagator.Extract(carrier)
}

// sampleRateMetricKey is the metric key holding the applied sample rate. Has to be the same as the Agent.
const sampleRateMetricKey = "_sample_rate"

// Sample samples a span with the internal sampler.
func (t *tracer) sample(span *span) {
	if _, ok := span.context.samplingPriority(); ok {
		// sampling decision was already made
		return
	}
	sampler := t.config.sampler
	if !sampler.Sample(span) {
		span.context.trace.drop()
		return
	}
	if rs, ok := sampler.(RateSampler); ok && rs.Rate() < 1 {
		span.setMetric(sampleRateMetricKey, rs.Rate())
	}
	if t.rulesSampling.SampleTrace(span) {
		return
	}
	t.prioritySampling.apply(span)
}

func startExecutionTracerTask(ctx gocontext.Context, span *span) (gocontext.Context, func()) {
	if !rt.IsEnabled() {
		return ctx, func() {}
	}
	// Task name is the resource (operationName) of the span, e.g.
	// "POST /foo/bar" (http) or "/foo/pkg.Method" (grpc).
	taskName := span.Resource
	// If the resource could contain PII (e.g. SQL query that's not using bind
	// arguments), play it safe and just use the span type as the taskName,
	// e.g. "sql".
	if !spanResourcePIISafe(span) {
		taskName = span.Type
	}
	ctx, task := rt.NewTask(ctx, taskName)
	rt.Log(ctx, "span id", strconv.FormatUint(span.SpanID, 10))
	return ctx, task.End
}
