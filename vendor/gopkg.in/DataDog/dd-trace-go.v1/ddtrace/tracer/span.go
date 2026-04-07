// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:generate msgp -unexported -marshal=false -o=span_msgp.go -tests=false

package tracer

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/internal"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/globalconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/samplernames"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/traceprof"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
	"github.com/tinylib/msgp/msgp"
	"golang.org/x/xerrors"
)

type (
	// spanList implements msgp.Encodable on top of a slice of spans.
	spanList []*span

	// spanLists implements msgp.Decodable on top of a slice of spanList.
	// This type is only used in tests.
	spanLists []spanList
)

var (
	_ ddtrace.Span   = (*span)(nil)
	_ msgp.Encodable = (*spanList)(nil)
	_ msgp.Decodable = (*spanLists)(nil)
)

// errorConfig holds customization options for setting error tags.
type errorConfig struct {
	noDebugStack bool
	stackFrames  uint
	stackSkip    uint
}

// span represents a computation. Callers must call Finish when a span is
// complete to ensure it's submitted.
type span struct {
	sync.RWMutex `msg:"-"` // all fields are protected by this RWMutex

	Name     string             `msg:"name"`              // operation name
	Service  string             `msg:"service"`           // service name (i.e. "grpc.server", "http.request")
	Resource string             `msg:"resource"`          // resource name (i.e. "/user?id=123", "SELECT * FROM users")
	Type     string             `msg:"type"`              // protocol associated with the span (i.e. "web", "db", "cache")
	Start    int64              `msg:"start"`             // span start time expressed in nanoseconds since epoch
	Duration int64              `msg:"duration"`          // duration of the span expressed in nanoseconds
	Meta     map[string]string  `msg:"meta,omitempty"`    // arbitrary map of metadata
	Metrics  map[string]float64 `msg:"metrics,omitempty"` // arbitrary map of numeric metrics
	SpanID   uint64             `msg:"span_id"`           // identifier of this span
	TraceID  uint64             `msg:"trace_id"`          // identifier of the root span
	ParentID uint64             `msg:"parent_id"`         // identifier of the span's direct parent
	Error    int32              `msg:"error"`             // error status of the span; 0 means no errors

	noDebugStack bool         `msg:"-"` // disables debug stack traces
	finished     bool         `msg:"-"` // true if the span has been submitted to a tracer.
	context      *spanContext `msg:"-"` // span propagation context

	pprofCtxActive  context.Context `msg:"-"` // contains pprof.WithLabel labels to tell the profiler more about this span
	pprofCtxRestore context.Context `msg:"-"` // contains pprof.WithLabel labels of the parent span (if any) that need to be restored when this span finishes

	taskEnd func() // ends execution tracer (runtime/trace) task, if started
}

// Context yields the SpanContext for this Span. Note that the return
// value of Context() is still valid after a call to Finish(). This is
// called the span context and it is different from Go's context.
func (s *span) Context() ddtrace.SpanContext { return s.context }

// SetBaggageItem sets a key/value pair as baggage on the span. Baggage items
// are propagated down to descendant spans and injected cross-process. Use with
// care as it adds extra load onto your tracing layer.
func (s *span) SetBaggageItem(key, val string) {
	s.context.setBaggageItem(key, val)
}

// BaggageItem gets the value for a baggage item given its key. Returns the
// empty string if the value isn't found in this Span.
func (s *span) BaggageItem(key string) string {
	return s.context.baggageItem(key)
}

// SetTag adds a set of key/value metadata to the span.
func (s *span) SetTag(key string, value interface{}) {
	s.Lock()
	defer s.Unlock()
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		return
	}
	switch key {
	case ext.Error:
		s.setTagError(value, errorConfig{
			noDebugStack: s.noDebugStack,
		})
		return
	}
	if v, ok := value.(bool); ok {
		s.setTagBool(key, v)
		return
	}
	if v, ok := value.(string); ok {
		if key == ext.ResourceName && s.pprofCtxActive != nil && spanResourcePIISafe(s) {
			// If the user overrides the resource name for the span,
			// update the endpoint label for the runtime profilers.
			//
			// We don't change s.pprofCtxRestore since that should
			// stay as the original parent span context regardless
			// of what we change at a lower level.
			s.pprofCtxActive = pprof.WithLabels(s.pprofCtxActive, pprof.Labels(traceprof.TraceEndpoint, v))
			pprof.SetGoroutineLabels(s.pprofCtxActive)
		}
		s.setMeta(key, v)
		return
	}
	if v, ok := toFloat64(value); ok {
		s.setMetric(key, v)
		return
	}
	if v, ok := value.(fmt.Stringer); ok {
		defer func() {
			if e := recover(); e != nil {
				if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
					// If .String() panics due to a nil receiver, we want to catch this
					// and replace the string value with "<nil>", just as Sprintf does.
					// Other panics should not be handled.
					s.setMeta(key, "<nil>")
					return
				}
				panic(e)
			}
		}()
		s.setMeta(key, v.String())
		return
	}
	// not numeric, not a string, not a fmt.Stringer, not a bool, and not an error
	s.setMeta(key, fmt.Sprint(value))
}

// setSamplingPriority locks then span, then updates the sampling priority.
// It also updates the trace's sampling priority.
func (s *span) setSamplingPriority(priority int, sampler samplernames.SamplerName) {
	s.Lock()
	defer s.Unlock()
	s.setSamplingPriorityLocked(priority, sampler)
}

// Root returns the root span of the span's trace. The return value shouldn't be
// nil as long as the root span is valid and not finished.
func (s *span) Root() Span {
	return s.root()
}

// root returns the root span of the span's trace. The return value shouldn't be
// nil as long as the root span is valid and not finished.
// As opposed to the public Root method, this one returns the actual span type
// when internal usage requires it (to avoid type assertions from Root's return
// value).
func (s *span) root() *span {
	if s == nil || s.context == nil {
		return nil
	}
	if s.context.trace == nil {
		return nil
	}
	return s.context.trace.root
}

// SetUser associates user information to the current trace which the
// provided span belongs to. The options can be used to tune which user
// bit of information gets monitored. In case of distributed traces,
// the user id can be propagated across traces using the WithPropagation() option.
// See https://docs.datadoghq.com/security_platform/application_security/setup_and_configure/?tab=set_user#add-user-information-to-traces
func (s *span) SetUser(id string, opts ...UserMonitoringOption) {
	var cfg UserMonitoringConfig
	for _, fn := range opts {
		fn(&cfg)
	}
	root := s.root()
	trace := root.context.trace
	root.Lock()
	defer root.Unlock()
	if cfg.PropagateID {
		// Delete usr.id from the tags since _dd.p.usr.id takes precedence
		delete(root.Meta, keyUserID)
		idenc := base64.StdEncoding.EncodeToString([]byte(id))
		trace.setPropagatingTag(keyPropagatedUserID, idenc)
		s.context.updated = true
	} else {
		// Unset the propagated user ID so that a propagated user ID coming from upstream won't be propagated anymore.
		trace.unsetPropagatingTag(keyPropagatedUserID)
		if _, ok := trace.propagatingTags[keyPropagatedUserID]; ok {
			s.context.updated = true
		}
		delete(root.Meta, keyPropagatedUserID)
		// setMeta is used since the span is already locked
		root.setMeta(keyUserID, id)
	}
	for k, v := range map[string]string{
		keyUserEmail:     cfg.Email,
		keyUserName:      cfg.Name,
		keyUserScope:     cfg.Scope,
		keyUserRole:      cfg.Role,
		keyUserSessionID: cfg.SessionID,
	} {
		if v != "" {
			root.setMeta(k, v)
		}
	}
}

// setSamplingPriorityLocked updates the sampling priority.
// It also updates the trace's sampling priority.
func (s *span) setSamplingPriorityLocked(priority int, sampler samplernames.SamplerName) {
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		return
	}
	s.setMetric(keySamplingPriority, float64(priority))
	s.context.setSamplingPriority(priority, sampler)
}

// setTagError sets the error tag. It accounts for various valid scenarios.
// This method is not safe for concurrent use.
func (s *span) setTagError(value interface{}, cfg errorConfig) {
	setError := func(yes bool) {
		if yes {
			if s.Error == 0 {
				// new error
				atomic.AddInt32(&s.context.errors, 1)
			}
			s.Error = 1
		} else {
			if s.Error > 0 {
				// flip from active to inactive
				atomic.AddInt32(&s.context.errors, -1)
			}
			s.Error = 0
		}
	}
	if s.finished {
		return
	}
	switch v := value.(type) {
	case bool:
		// bool value as per Opentracing spec.
		setError(v)
	case error:
		// if anyone sets an error value as the tag, be nice here
		// and provide all the benefits.
		setError(true)
		s.setMeta(ext.ErrorMsg, v.Error())
		s.setMeta(ext.ErrorType, reflect.TypeOf(v).String())
		if !cfg.noDebugStack {
			s.setMeta(ext.ErrorStack, takeStacktrace(cfg.stackFrames, cfg.stackSkip))
		}
		switch v.(type) {
		case xerrors.Formatter:
			s.setMeta(ext.ErrorDetails, fmt.Sprintf("%+v", v))
		case fmt.Formatter:
			// pkg/errors approach
			s.setMeta(ext.ErrorDetails, fmt.Sprintf("%+v", v))
		}
	case nil:
		// no error
		setError(false)
	default:
		// in all other cases, let's assume that setting this tag
		// is the result of an error.
		setError(true)
	}
}

// defaultStackLength specifies the default maximum size of a stack trace.
const defaultStackLength = 32

// takeStacktrace takes a stack trace of maximum n entries, skipping the first skip entries.
// If n is 0, up to 20 entries are retrieved.
func takeStacktrace(n, skip uint) string {
	if n == 0 {
		n = defaultStackLength
	}
	var builder strings.Builder
	pcs := make([]uintptr, n)

	// +2 to exclude runtime.Callers and takeStacktrace
	numFrames := runtime.Callers(2+int(skip), pcs)
	if numFrames == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:numFrames])
	for i := 0; ; i++ {
		frame, more := frames.Next()
		if i != 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(frame.Function)
		builder.WriteByte('\n')
		builder.WriteByte('\t')
		builder.WriteString(frame.File)
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(frame.Line))
		if !more {
			break
		}
	}
	return builder.String()
}

// setMeta sets a string tag. This method is not safe for concurrent use.
func (s *span) setMeta(key, v string) {
	if s.Meta == nil {
		s.Meta = make(map[string]string, 1)
	}
	delete(s.Metrics, key)
	switch key {
	case ext.SpanName:
		s.Name = v
	case ext.ServiceName:
		s.Service = v
	case ext.ResourceName:
		s.Resource = v
	case ext.SpanType:
		s.Type = v
	default:
		s.Meta[key] = v
	}
}

// setTagBool sets a boolean tag on the span.
func (s *span) setTagBool(key string, v bool) {
	switch key {
	case ext.AnalyticsEvent:
		if v {
			s.setMetric(ext.EventSampleRate, 1.0)
		} else {
			s.setMetric(ext.EventSampleRate, 0.0)
		}
	case ext.ManualDrop:
		if v {
			s.setSamplingPriorityLocked(ext.PriorityUserReject, samplernames.Manual)
		}
	case ext.ManualKeep:
		if v {
			s.setSamplingPriorityLocked(ext.PriorityUserKeep, samplernames.Manual)
		}
	default:
		if v {
			s.setMeta(key, "true")
		} else {
			s.setMeta(key, "false")
		}
	}
}

// setMetric sets a numeric tag, in our case called a metric. This method
// is not safe for concurrent use.
func (s *span) setMetric(key string, v float64) {
	if s.Metrics == nil {
		s.Metrics = make(map[string]float64, 1)
	}
	delete(s.Meta, key)
	switch key {
	case ext.ManualKeep:
		if v == float64(samplernames.AppSec) {
			s.setSamplingPriorityLocked(ext.PriorityUserKeep, samplernames.AppSec)
		}
	case ext.SamplingPriority:
		// ext.SamplingPriority is deprecated in favor of ext.ManualKeep and ext.ManualDrop.
		// We have it here for backward compatibility.
		s.setSamplingPriorityLocked(int(v), samplernames.Manual)
	default:
		s.Metrics[key] = v
	}
}

// Finish closes this Span (but not its children) providing the duration
// of its part of the tracing session.
func (s *span) Finish(opts ...ddtrace.FinishOption) {
	t := now()
	if len(opts) > 0 {
		cfg := ddtrace.FinishConfig{
			NoDebugStack: s.noDebugStack,
		}
		for _, fn := range opts {
			fn(&cfg)
		}
		if !cfg.FinishTime.IsZero() {
			t = cfg.FinishTime.UnixNano()
		}
		if cfg.Error != nil {
			s.Lock()
			s.setTagError(cfg.Error, errorConfig{
				noDebugStack: cfg.NoDebugStack,
				stackFrames:  cfg.StackFrames,
				stackSkip:    cfg.SkipStackFrames,
			})
			s.Unlock()
		}
	}
	if s.taskEnd != nil {
		s.taskEnd()
	}
	s.finish(t)

	if s.pprofCtxRestore != nil {
		// Restore the labels of the parent span so any CPU samples after this
		// point are attributed correctly.
		pprof.SetGoroutineLabels(s.pprofCtxRestore)
	}
}

// SetOperationName sets or changes the operation name.
func (s *span) SetOperationName(operationName string) {
	s.Lock()
	defer s.Unlock()
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		// already finished
		return
	}
	s.Name = operationName
}

func (s *span) finish(finishTime int64) {
	s.Lock()
	defer s.Unlock()
	// We don't lock spans when flushing, so we could have a data race when
	// modifying a span as it's being flushed. This protects us against that
	// race, since spans are marked `finished` before we flush them.
	if s.finished {
		// already finished
		return
	}
	if s.Duration == 0 {
		s.Duration = finishTime - s.Start
	}
	if s.Duration < 0 {
		s.Duration = 0
	}
	s.finished = true

	keep := true
	if t, ok := internal.GetGlobalTracer().(*tracer); ok {
		// we have an active tracer
		if t.config.canComputeStats() && shouldComputeStats(s) {
			// the agent supports computed stats
			select {
			case t.stats.In <- newAggregableSpan(s, t.obfuscator):
				// ok
			default:
				log.Error("Stats channel full, disregarding span.")
			}
		}
		if t.config.canDropP0s() {
			// the agent supports dropping p0's in the client
			keep = shouldKeep(s)
		}
	}
	if keep {
		// a single kept span keeps the whole trace.
		s.context.trace.keep()
	}
	s.context.finish()
}

// newAggregableSpan creates a new summary for the span s, within an application
// version version.
func newAggregableSpan(s *span, obfuscator *obfuscate.Obfuscator) *aggregableSpan {
	var statusCode uint32
	if sc, ok := s.Meta["http.status_code"]; ok && sc != "" {
		if c, err := strconv.Atoi(sc); err == nil && c > 0 && c <= math.MaxInt32 {
			statusCode = uint32(c)
		}
	}
	key := aggregation{
		Name:       s.Name,
		Resource:   obfuscatedResource(obfuscator, s.Type, s.Resource),
		Service:    s.Service,
		Type:       s.Type,
		Synthetics: strings.HasPrefix(s.Meta[keyOrigin], "synthetics"),
		StatusCode: statusCode,
	}
	return &aggregableSpan{
		key:      key,
		Start:    s.Start,
		Duration: s.Duration,
		TopLevel: s.Metrics[keyTopLevel] == 1,
		Error:    s.Error,
	}
}

// textNonParsable specifies the text that will be assigned to resources for which the resource
// can not be parsed due to an obfuscation error.
const textNonParsable = "Non-parsable SQL query"

// obfuscatedResource returns the obfuscated version of the given resource. It is
// obfuscated using the given obfuscator for the given span type typ.
func obfuscatedResource(o *obfuscate.Obfuscator, typ, resource string) string {
	if o == nil {
		return resource
	}
	switch typ {
	case "sql", "cassandra":
		oq, err := o.ObfuscateSQLString(resource)
		if err != nil {
			log.Error("Error obfuscating stats group resource %q: %v", resource, err)
			return textNonParsable
		}
		return oq.Query
	case "redis":
		return o.QuantizeRedisString(resource)
	default:
		return resource
	}
}

// shouldKeep reports whether the trace should be kept.
// a single span being kept implies the whole trace being kept.
func shouldKeep(s *span) bool {
	if p, ok := s.context.samplingPriority(); ok && p > 0 {
		// positive sampling priorities stay
		return true
	}
	if atomic.LoadInt32(&s.context.errors) > 0 {
		// traces with any span containing an error get kept
		return true
	}
	if v, ok := s.Metrics[ext.EventSampleRate]; ok {
		return sampledByRate(s.TraceID, v)
	}
	return false
}

// shouldComputeStats mentions whether this span needs to have stats computed for.
// Warning: callers must guard!
func shouldComputeStats(s *span) bool {
	if v, ok := s.Metrics[keyMeasured]; ok && v == 1 {
		return true
	}
	if v, ok := s.Metrics[keyTopLevel]; ok && v == 1 {
		return true
	}
	return false
}

// String returns a human readable representation of the span. Not for
// production, just debugging.
func (s *span) String() string {
	s.RLock()
	defer s.RUnlock()
	lines := []string{
		fmt.Sprintf("Name: %s", s.Name),
		fmt.Sprintf("Service: %s", s.Service),
		fmt.Sprintf("Resource: %s", s.Resource),
		fmt.Sprintf("TraceID: %d", s.TraceID),
		fmt.Sprintf("SpanID: %d", s.SpanID),
		fmt.Sprintf("ParentID: %d", s.ParentID),
		fmt.Sprintf("Start: %s", time.Unix(0, s.Start)),
		fmt.Sprintf("Duration: %s", time.Duration(s.Duration)),
		fmt.Sprintf("Error: %d", s.Error),
		fmt.Sprintf("Type: %s", s.Type),
		"Tags:",
	}
	for key, val := range s.Meta {
		lines = append(lines, fmt.Sprintf("\t%s:%s", key, val))
	}
	for key, val := range s.Metrics {
		lines = append(lines, fmt.Sprintf("\t%s:%f", key, val))
	}
	return strings.Join(lines, "\n")
}

// Format implements fmt.Formatter.
func (s *span) Format(f fmt.State, c rune) {
	switch c {
	case 's':
		fmt.Fprint(f, s.String())
	case 'v':
		if svc := globalconfig.ServiceName(); svc != "" {
			fmt.Fprintf(f, "dd.service=%s ", svc)
		}
		if tr, ok := internal.GetGlobalTracer().(*tracer); ok {
			if tr.config.env != "" {
				fmt.Fprintf(f, "dd.env=%s ", tr.config.env)
			}
			if tr.config.version != "" {
				fmt.Fprintf(f, "dd.version=%s ", tr.config.version)
			}
		} else {
			if env := os.Getenv("DD_ENV"); env != "" {
				fmt.Fprintf(f, "dd.env=%s ", env)
			}
			if v := os.Getenv("DD_VERSION"); v != "" {
				fmt.Fprintf(f, "dd.version=%s ", v)
			}
		}
		fmt.Fprintf(f, `dd.trace_id="%d" dd.span_id="%d"`, s.TraceID, s.SpanID)
	default:
		fmt.Fprintf(f, "%%!%c(ddtrace.Span=%v)", c, s)
	}
}

const (
	keySamplingPriority        = "_sampling_priority_v1"
	keySamplingPriorityRate    = "_dd.agent_psr"
	keyDecisionMaker           = "_dd.p.dm"
	keyServiceHash             = "_dd.dm.service_hash"
	keyOrigin                  = "_dd.origin"
	keyHostname                = "_dd.hostname"
	keyRulesSamplerAppliedRate = "_dd.rule_psr"
	keyRulesSamplerLimiterRate = "_dd.limit_psr"
	keyMeasured                = "_dd.measured"
	// keyTopLevel is the key of top level metric indicating if a span is top level.
	// A top level span is a local root (parent span of the local trace) or the first span of each service.
	keyTopLevel = "_dd.top_level"
	// keyPropagationError holds any error from propagated trace tags (if any)
	keyPropagationError = "_dd.propagation_error"
	// keySpanSamplingMechanism specifies the sampling mechanism by which an individual span was sampled
	keySpanSamplingMechanism = "_dd.span_sampling.mechanism"
	// keySingleSpanSamplingRuleRate specifies the configured sampling probability for the single span sampling rule.
	keySingleSpanSamplingRuleRate = "_dd.span_sampling.rule_rate"
	// keySingleSpanSamplingMPS specifies the configured limit for the single span sampling rule
	// that the span matched. If there is no configured limit, then this tag is omitted.
	keySingleSpanSamplingMPS = "_dd.span_sampling.max_per_second"
	// keyPropagatedUserID holds the propagated user identifier, if user id propagation is enabled.
	keyPropagatedUserID = "_dd.p.usr.id"
)

// The following set of tags is used for user monitoring and set through calls to span.SetUser().
const (
	keyUserID        = "usr.id"
	keyUserEmail     = "usr.email"
	keyUserName      = "usr.name"
	keyUserRole      = "usr.role"
	keyUserScope     = "usr.scope"
	keyUserSessionID = "usr.session_id"
)

const (
	// samplingMechanismSingleSpan specifies value reserved to indicate that a span was kept
	// on account of a single span sampling rule.
	samplingMechanismSingleSpan = 8
)
