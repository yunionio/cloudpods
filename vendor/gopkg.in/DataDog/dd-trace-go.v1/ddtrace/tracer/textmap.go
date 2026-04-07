// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/samplernames"
)

// HTTPHeadersCarrier wraps an http.Header as a TextMapWriter and TextMapReader, allowing
// it to be used using the provided Propagator implementation.
type HTTPHeadersCarrier http.Header

var _ TextMapWriter = (*HTTPHeadersCarrier)(nil)
var _ TextMapReader = (*HTTPHeadersCarrier)(nil)

// Set implements TextMapWriter.
func (c HTTPHeadersCarrier) Set(key, val string) {
	http.Header(c).Set(key, val)
}

// ForeachKey implements TextMapReader.
func (c HTTPHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range c {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// TextMapCarrier allows the use of a regular map[string]string as both TextMapWriter
// and TextMapReader, making it compatible with the provided Propagator.
type TextMapCarrier map[string]string

var _ TextMapWriter = (*TextMapCarrier)(nil)
var _ TextMapReader = (*TextMapCarrier)(nil)

// Set implements TextMapWriter.
func (c TextMapCarrier) Set(key, val string) {
	c[key] = val
}

// ForeachKey conforms to the TextMapReader interface.
func (c TextMapCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, v := range c {
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

const (
	headerPropagationStyleInject  = "DD_TRACE_PROPAGATION_STYLE_INJECT"
	headerPropagationStyleExtract = "DD_TRACE_PROPAGATION_STYLE_EXTRACT"
	headerPropagationStyle        = "DD_TRACE_PROPAGATION_STYLE"

	headerPropagationStyleInjectDeprecated  = "DD_PROPAGATION_STYLE_INJECT"  // deprecated
	headerPropagationStyleExtractDeprecated = "DD_PROPAGATION_STYLE_EXTRACT" // deprecated
)

const (
	// DefaultBaggageHeaderPrefix specifies the prefix that will be used in
	// HTTP headers or text maps to prefix baggage keys.
	DefaultBaggageHeaderPrefix = "ot-baggage-"

	// DefaultTraceIDHeader specifies the key that will be used in HTTP headers
	// or text maps to store the trace ID.
	DefaultTraceIDHeader = "x-datadog-trace-id"

	// DefaultParentIDHeader specifies the key that will be used in HTTP headers
	// or text maps to store the parent ID.
	DefaultParentIDHeader = "x-datadog-parent-id"

	// DefaultPriorityHeader specifies the key that will be used in HTTP headers
	// or text maps to store the sampling priority value.
	DefaultPriorityHeader = "x-datadog-sampling-priority"
)

// originHeader specifies the name of the header indicating the origin of the trace.
// It is used with the Synthetics product and usually has the value "synthetics".
const originHeader = "x-datadog-origin"

// traceTagsHeader holds the propagated trace tags
const traceTagsHeader = "x-datadog-tags"

// propagationExtractMaxSize limits the total size of incoming propagated tags to parse
const propagationExtractMaxSize = 512

// PropagatorConfig defines the configuration for initializing a propagator.
type PropagatorConfig struct {
	// BaggagePrefix specifies the prefix that will be used to store baggage
	// items in a map. It defaults to DefaultBaggageHeaderPrefix.
	BaggagePrefix string

	// TraceHeader specifies the map key that will be used to store the trace ID.
	// It defaults to DefaultTraceIDHeader.
	TraceHeader string

	// ParentHeader specifies the map key that will be used to store the parent ID.
	// It defaults to DefaultParentIDHeader.
	ParentHeader string

	// PriorityHeader specifies the map key that will be used to store the sampling priority.
	// It defaults to DefaultPriorityHeader.
	PriorityHeader string

	// MaxTagsHeaderLen specifies the maximum length of trace tags header value.
	// It defaults to defaultMaxTagsHeaderLen, a value of 0 disables propagation of tags.
	MaxTagsHeaderLen int

	// B3 specifies if B3 headers should be added for trace propagation.
	// See https://github.com/openzipkin/b3-propagation
	B3 bool
}

// NewPropagator returns a new propagator which uses TextMap to inject
// and extract values. It propagates trace and span IDs and baggage.
// To use the defaults, nil may be provided in place of the config.
//
// The inject and extract propagators are determined using environment variables
// with the following order of precedence:
//  1. DD_TRACE_PROPAGATION_STYLE_INJECT
//  2. DD_PROPAGATION_STYLE_INJECT (deprecated)
//  3. DD_TRACE_PROPAGATION_STYLE (applies to both inject and extract)
//  4. If none of the above, use default values
func NewPropagator(cfg *PropagatorConfig, propagators ...Propagator) Propagator {
	if cfg == nil {
		cfg = new(PropagatorConfig)
	}
	if cfg.BaggagePrefix == "" {
		cfg.BaggagePrefix = DefaultBaggageHeaderPrefix
	}
	if cfg.TraceHeader == "" {
		cfg.TraceHeader = DefaultTraceIDHeader
	}
	if cfg.ParentHeader == "" {
		cfg.ParentHeader = DefaultParentIDHeader
	}
	if cfg.PriorityHeader == "" {
		cfg.PriorityHeader = DefaultPriorityHeader
	}
	if len(propagators) > 0 {
		return &chainedPropagator{
			injectors:  propagators,
			extractors: propagators,
		}
	}
	injectorsPs := os.Getenv(headerPropagationStyleInject)
	if injectorsPs == "" {
		if injectorsPs = os.Getenv(headerPropagationStyleInjectDeprecated); injectorsPs != "" {
			log.Warn("%v is deprecated. Please use %v or %v instead.\n", headerPropagationStyleInjectDeprecated, headerPropagationStyleInject, headerPropagationStyle)
		}
	}
	extractorsPs := os.Getenv(headerPropagationStyleExtract)
	if extractorsPs == "" {
		if extractorsPs = os.Getenv(headerPropagationStyleExtractDeprecated); extractorsPs != "" {
			log.Warn("%v is deprecated. Please use %v or %v instead.\n", headerPropagationStyleExtractDeprecated, headerPropagationStyleExtract, headerPropagationStyle)
		}
	}
	return &chainedPropagator{
		injectors:  getPropagators(cfg, injectorsPs),
		extractors: getPropagators(cfg, extractorsPs),
	}
}

// chainedPropagator implements Propagator and applies a list of injectors and extractors.
// When injecting, all injectors are called to propagate the span context.
// When extracting, it tries each extractor, selecting the first successful one.
type chainedPropagator struct {
	injectors  []Propagator
	extractors []Propagator
}

// getPropagators returns a list of propagators based on ps, which is a comma seperated
// list of propagators. If the list doesn't contain any valid values, the
// default propagator will be returned. Any invalid values in the list will log
// a warning and be ignored.
func getPropagators(cfg *PropagatorConfig, ps string) []Propagator {
	dd := &propagator{cfg}
	defaultPs := []Propagator{&propagatorW3c{}, dd}
	if cfg.B3 {
		defaultPs = append(defaultPs, &propagatorB3{})
	}
	if ps == "" {
		if prop := os.Getenv(headerPropagationStyle); prop != "" {
			ps = prop // use the generic DD_TRACE_PROPAGATION_STYLE if set
		} else {
			return defaultPs // no env set, so use default from configuration
		}
	}
	ps = strings.ToLower(ps)
	if ps == "none" {
		return nil
	}
	var list []Propagator
	if cfg.B3 {
		list = append(list, &propagatorB3{})
	}
	for _, v := range strings.Split(ps, ",") {
		switch strings.ToLower(v) {
		case "datadog":
			list = append(list, dd)
		case "tracecontext":
			list = append([]Propagator{&propagatorW3c{}}, list...)
		case "b3", "b3multi":
			if !cfg.B3 {
				// propagatorB3 hasn't already been added, add a new one.
				list = append(list, &propagatorB3{})
			}
		case "b3 single header":
			list = append(list, &propagatorB3SingleHeader{})
		case "none":
			log.Warn("Propagator \"none\" has no effect when combined with other propagators. " +
				"To disable the propagator, set to `none`")
		default:
			log.Warn("unrecognized propagator: %s\n", v)
		}
	}
	if len(list) == 0 {
		return defaultPs // no valid propagators, so return default
	}
	return list
}

// Inject defines the Propagator to propagate SpanContext data
// out of the current process. The implementation propagates the
// TraceID and the current active SpanID, as well as the Span baggage.
func (p *chainedPropagator) Inject(spanCtx ddtrace.SpanContext, carrier interface{}) error {
	for _, v := range p.injectors {
		err := v.Inject(spanCtx, carrier)
		if err != nil {
			return err
		}
	}
	return nil
}

// Extract implements Propagator.
func (p *chainedPropagator) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	for _, v := range p.extractors {
		ctx, err := v.Extract(carrier)
		if ctx != nil {
			// first extractor returns
			log.Debug("Extracted span context: %#v", ctx)
			return ctx, nil
		}
		if err == ErrSpanContextNotFound {
			continue
		}
		return nil, err
	}
	return nil, ErrSpanContextNotFound
}

// propagator implements Propagator and injects/extracts span contexts
// using datadog headers. Only TextMap carriers are supported.
type propagator struct {
	cfg *PropagatorConfig
}

func (p *propagator) Inject(spanCtx ddtrace.SpanContext, carrier interface{}) error {
	switch c := carrier.(type) {
	case TextMapWriter:
		return p.injectTextMap(spanCtx, c)
	default:
		return ErrInvalidCarrier
	}
}

func (p *propagator) injectTextMap(spanCtx ddtrace.SpanContext, writer TextMapWriter) error {
	ctx, ok := spanCtx.(*spanContext)
	if !ok || ctx.traceID == 0 || ctx.spanID == 0 {
		return ErrInvalidSpanContext
	}
	// propagate the TraceID and the current active SpanID
	writer.Set(p.cfg.TraceHeader, strconv.FormatUint(ctx.traceID, 10))
	writer.Set(p.cfg.ParentHeader, strconv.FormatUint(ctx.spanID, 10))
	if sp, ok := ctx.samplingPriority(); ok {
		writer.Set(p.cfg.PriorityHeader, strconv.Itoa(sp))
	}
	if ctx.origin != "" {
		writer.Set(originHeader, ctx.origin)
	}
	// propagate OpenTracing baggage
	for k, v := range ctx.baggage {
		writer.Set(p.cfg.BaggagePrefix+k, v)
	}
	if p.cfg.MaxTagsHeaderLen <= 0 {
		return nil
	}
	if s := p.marshalPropagatingTags(ctx); len(s) > 0 {
		writer.Set(traceTagsHeader, s)
	}
	return nil
}

// marshalPropagatingTags marshals all propagating tags included in ctx to a comma separated string
func (p *propagator) marshalPropagatingTags(ctx *spanContext) string {
	var sb strings.Builder
	if ctx.trace == nil {
		return ""
	}
	ctx.trace.mu.Lock()
	defer ctx.trace.mu.Unlock()
	for k, v := range ctx.trace.propagatingTags {
		if err := isValidPropagatableTag(k, v); err != nil {
			log.Warn("Won't propagate tag '%s': %v", k, err.Error())
			ctx.trace.setTag(keyPropagationError, "encoding_error")
			continue
		}
		if sb.Len()+len(k)+len(v) > p.cfg.MaxTagsHeaderLen {
			sb.Reset()
			log.Warn("Won't propagate tag: maximum trace tags header len (%d) reached.", p.cfg.MaxTagsHeaderLen)
			ctx.trace.setTag(keyPropagationError, "inject_max_size")
			break
		}
		if sb.Len() > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(v)
	}
	return sb.String()
}

func (p *propagator) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	switch c := carrier.(type) {
	case TextMapReader:
		return p.extractTextMap(c)
	default:
		return nil, ErrInvalidCarrier
	}
}

func (p *propagator) extractTextMap(reader TextMapReader) (ddtrace.SpanContext, error) {
	var ctx spanContext
	err := reader.ForeachKey(func(k, v string) error {
		var err error
		key := strings.ToLower(k)
		switch key {
		case p.cfg.TraceHeader:
			ctx.traceID, err = parseUint64(v)
			if err != nil {
				return ErrSpanContextCorrupted
			}
		case p.cfg.ParentHeader:
			ctx.spanID, err = parseUint64(v)
			if err != nil {
				return ErrSpanContextCorrupted
			}
		case p.cfg.PriorityHeader:
			priority, err := strconv.Atoi(v)
			if err != nil {
				return ErrSpanContextCorrupted
			}
			ctx.setSamplingPriority(priority, samplernames.Unknown)
		case originHeader:
			ctx.origin = v
		case traceTagsHeader:
			unmarshalPropagatingTags(&ctx, v)
		default:
			if strings.HasPrefix(key, p.cfg.BaggagePrefix) {
				ctx.setBaggageItem(strings.TrimPrefix(key, p.cfg.BaggagePrefix), v)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.traceID == 0 || (ctx.spanID == 0 && ctx.origin != "synthetics") {
		return nil, ErrSpanContextNotFound
	}
	return &ctx, nil
}

// unmarshalPropagatingTags unmarshals tags from v into ctx
func unmarshalPropagatingTags(ctx *spanContext, v string) {
	if ctx.trace == nil {
		ctx.trace = newTrace()
	}
	ctx.trace.mu.Lock()
	defer ctx.trace.mu.Unlock()
	if len(v) > propagationExtractMaxSize {
		log.Warn("Did not extract %s, size limit exceeded: %d. Incoming tags will not be propagated further.", traceTagsHeader, propagationExtractMaxSize)
		ctx.trace.setTag(keyPropagationError, "extract_max_size")
		return
	}
	var err error
	ctx.trace.propagatingTags, err = parsePropagatableTraceTags(v)
	if err != nil {
		log.Warn("Did not extract %s: %v. Incoming tags will not be propagated further.", traceTagsHeader, err.Error())
		ctx.trace.setTag(keyPropagationError, "decoding_error")
	}
}

// setPropagatingTag adds the key value pair to the map of propagating tags on the trace,
// creating the map if one is not initialized.
func setPropagatingTag(ctx *spanContext, k, v string) {
	if ctx.trace == nil {
		// extractors initialize a new spanContext, so the trace might be nil
		ctx.trace = newTrace()
	}
	ctx.trace.setPropagatingTag(k, v)
}

const (
	b3TraceIDHeader = "x-b3-traceid"
	b3SpanIDHeader  = "x-b3-spanid"
	b3SampledHeader = "x-b3-sampled"
	b3SingleHeader  = "b3"
)

// propagatorB3 implements Propagator and injects/extracts span contexts
// using B3 headers. Only TextMap carriers are supported.
type propagatorB3 struct{}

func (p *propagatorB3) Inject(spanCtx ddtrace.SpanContext, carrier interface{}) error {
	switch c := carrier.(type) {
	case TextMapWriter:
		return p.injectTextMap(spanCtx, c)
	default:
		return ErrInvalidCarrier
	}
}

func (*propagatorB3) injectTextMap(spanCtx ddtrace.SpanContext, writer TextMapWriter) error {
	ctx, ok := spanCtx.(*spanContext)
	if !ok || ctx.traceID == 0 || ctx.spanID == 0 {
		return ErrInvalidSpanContext
	}
	writer.Set(b3TraceIDHeader, fmt.Sprintf("%016x", ctx.traceID))
	writer.Set(b3SpanIDHeader, fmt.Sprintf("%016x", ctx.spanID))
	if p, ok := ctx.samplingPriority(); ok {
		if p >= ext.PriorityAutoKeep {
			writer.Set(b3SampledHeader, "1")
		} else {
			writer.Set(b3SampledHeader, "0")
		}
	}
	return nil
}

func (p *propagatorB3) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	switch c := carrier.(type) {
	case TextMapReader:
		return p.extractTextMap(c)
	default:
		return nil, ErrInvalidCarrier
	}
}

func (*propagatorB3) extractTextMap(reader TextMapReader) (ddtrace.SpanContext, error) {
	var ctx spanContext
	err := reader.ForeachKey(func(k, v string) error {
		var err error
		key := strings.ToLower(k)
		switch key {
		case b3TraceIDHeader:
			if len(v) > 16 {
				v = v[len(v)-16:]
			}
			ctx.traceID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return ErrSpanContextCorrupted
			}
		case b3SpanIDHeader:
			ctx.spanID, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return ErrSpanContextCorrupted
			}
		case b3SampledHeader:
			priority, err := strconv.Atoi(v)
			if err != nil {
				return ErrSpanContextCorrupted
			}
			ctx.setSamplingPriority(priority, samplernames.Unknown)
		default:
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.traceID == 0 || ctx.spanID == 0 {
		return nil, ErrSpanContextNotFound
	}
	return &ctx, nil
}

// propagatorB3 implements Propagator and injects/extracts span contexts
// using B3 headers. Only TextMap carriers are supported.
type propagatorB3SingleHeader struct{}

func (p *propagatorB3SingleHeader) Inject(spanCtx ddtrace.SpanContext, carrier interface{}) error {
	switch c := carrier.(type) {
	case TextMapWriter:
		return p.injectTextMap(spanCtx, c)
	default:
		return ErrInvalidCarrier
	}
}

func (*propagatorB3SingleHeader) injectTextMap(spanCtx ddtrace.SpanContext, writer TextMapWriter) error {
	ctx, ok := spanCtx.(*spanContext)
	if !ok || ctx.traceID == 0 || ctx.spanID == 0 {
		return ErrInvalidSpanContext
	}
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%016x-%016x", ctx.traceID, ctx.spanID))
	if p, ok := ctx.samplingPriority(); ok {
		if p >= ext.PriorityAutoKeep {
			sb.WriteString("-1")
		} else {
			sb.WriteString("-0")
		}
	}
	writer.Set(b3SingleHeader, sb.String())
	return nil
}

func (p *propagatorB3SingleHeader) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	switch c := carrier.(type) {
	case TextMapReader:
		return p.extractTextMap(c)
	default:
		return nil, ErrInvalidCarrier
	}
}

func (*propagatorB3SingleHeader) extractTextMap(reader TextMapReader) (ddtrace.SpanContext, error) {
	var ctx spanContext
	err := reader.ForeachKey(func(k, v string) error {
		var err error
		key := strings.ToLower(k)
		switch key {
		case b3SingleHeader:
			b3Parts := strings.Split(v, "-")
			if len(b3Parts) >= 2 {
				if len(b3Parts[0]) > 16 {
					b3Parts[0] = b3Parts[0][len(b3Parts[0])-16:]
				}
				ctx.traceID, err = strconv.ParseUint(b3Parts[0], 16, 64)
				if err != nil {
					return ErrSpanContextCorrupted
				}
				ctx.spanID, err = strconv.ParseUint(b3Parts[1], 16, 64)
				if err != nil {
					return ErrSpanContextCorrupted
				}
				if len(b3Parts) >= 3 {
					switch b3Parts[2] {
					case "":
						break
					case "1", "d": // Treat 'debug' traces as priority 1
						ctx.setSamplingPriority(1, samplernames.Unknown)
					case "0":
						ctx.setSamplingPriority(0, samplernames.Unknown)
					default:
						return ErrSpanContextCorrupted
					}
				}
			} else {
				return ErrSpanContextCorrupted
			}
		default:
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.traceID == 0 || ctx.spanID == 0 {
		return nil, ErrSpanContextNotFound
	}
	return &ctx, nil
}

const (
	traceparentHeader = "traceparent"
	tracestateHeader  = "tracestate"
	w3cTraceIDTag     = "w3cTraceID"
)

// propagatorW3c implements Propagator and injects/extracts span contexts
// using W3C tracecontext/traceparent headers. Only TextMap carriers are supported.
type propagatorW3c struct{}

func (p *propagatorW3c) Inject(spanCtx ddtrace.SpanContext, carrier interface{}) error {
	switch c := carrier.(type) {
	case TextMapWriter:
		return p.injectTextMap(spanCtx, c)
	default:
		return ErrInvalidCarrier
	}
}

// injectTextMap propagates span context attributes into the writer,
// in the format of the traceparentHeader and tracestateHeader.
// traceparentHeader encodes W3C Trace Propagation version, 128-bit traceID,
// spanID, and a flags field, which supports 8 unique flags.
// The current specification only supports a single flag called sampled,
// which is equal to 00000001 when no other flag is present.
// tracestateHeader is a comma-separated list of list-members with a <key>=<value> format,
// where each list-member is managed by a vendor or instrumentation library.
func (*propagatorW3c) injectTextMap(spanCtx ddtrace.SpanContext, writer TextMapWriter) error {
	ctx, ok := spanCtx.(*spanContext)
	if !ok || ctx.traceID == 0 || ctx.spanID == 0 {
		return ErrInvalidSpanContext
	}
	flags := ""
	p, ok := ctx.samplingPriority()
	if ok && p >= ext.PriorityAutoKeep {
		flags = "01"
	} else {
		flags = "00"
	}

	var traceID string
	// if previous traceparent is valid, do NOT update the trace ID
	if ctx.trace != nil && ctx.trace.propagatingTags != nil {
		tag := ctx.trace.propagatingTags[w3cTraceIDTag]
		if len(tag) == 32 {
			id, err := strconv.ParseUint(tag[16:], 16, 64)
			if err == nil && id != 0 {
				traceID = tag
			}
		}
	}
	if len(traceID) == 0 {
		traceID = fmt.Sprintf("%032x", ctx.traceID)
	}
	writer.Set(traceparentHeader, fmt.Sprintf("00-%s-%016x-%v", traceID, ctx.spanID, flags))
	// if context priority / origin / tags were updated after extraction,
	// or the tracestateHeader doesn't start with `dd=`
	// we need to recreate tracestate
	if ctx.updated ||
		(ctx.trace != nil && ctx.trace.propagatingTags != nil && !strings.HasPrefix(ctx.trace.propagatingTags[tracestateHeader], "dd=")) ||
		len(ctx.trace.propagatingTags[tracestateHeader]) == 0 {
		writer.Set(tracestateHeader, composeTracestate(ctx, p, ctx.trace.propagatingTags[tracestateHeader]))
	} else {
		writer.Set(tracestateHeader, ctx.trace.propagatingTags[tracestateHeader])
	}
	return nil
}

var (
	// keyRgx is used to sanitize the keys of the datadog propagating tags.
	// Disallowed characters are comma (reserved as a list-member separator),
	// equals (reserved for list-member key-value separator),
	// space and characters outside the ASCII range 0x20 to 0x7E.
	// Disallowed characters must be replaced with the underscore.
	keyRgx = regexp.MustCompile(",|=|[^\\x20-\\x7E]+")

	// valueRgx is used to sanitize the values of the datadog propagating tags.
	// Disallowed characters are comma (reserved as a list-member separator),
	// semi-colon (reserved for separator between entries in the dd list-member),
	// tilde (reserved, will represent 0x3D (equals) in the encoded tag value,
	// and characters outside the ASCII range 0x20 to 0x7E.
	// Equals character must be encoded with a tilde.
	// Other disallowed characters must be replaced with the underscore.
	valueRgx = regexp.MustCompile(",|;|~|[^\\x20-\\x7E]+")

	// originRgx is used to sanitize the value of the datadog origin tag.
	// Disallowed characters are comma (reserved as a list-member separator),
	// semi-colon (reserved for separator between entries in the dd list-member),
	// equals (reserved for list-member key-value separator),
	// and characters outside the ASCII range 0x21 to 0x7E.
	// Disallowed characters must be replaced with the underscore.
	originRgx = regexp.MustCompile(",|=|;|[^\\x21-\\x7E]+")
)

// composeTracestate creates a tracestateHeader from the spancontext.
// The Datadog tracing library is only responsible for managing the list member with key dd,
// which holds the values of the sampling decision(`s:<value>`), origin(`o:<origin>`),
// and propagated tags prefixed with `t.`(e.g. _dd.p.usr.id:usr_id tag will become `t.usr.id:usr_id`).
func composeTracestate(ctx *spanContext, priority int, oldState string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("dd=s:%d", priority))
	listLength := 1

	if ctx.origin != "" {
		b.WriteString(fmt.Sprintf(";o:%s",
			originRgx.ReplaceAllString(ctx.origin, "_")))
	}

	for k, v := range ctx.trace.propagatingTags {
		if !strings.HasPrefix(k, "_dd.p.") {
			continue
		}
		// Datadog propagating tags must be appended to the tracestateHeader
		// with the `t.` prefix. Tag value must have all `=` signs replaced with a tilde (`~`).
		tag := fmt.Sprintf("t.%s:%s",
			keyRgx.ReplaceAllString(k[len("_dd.p."):], "_"),
			strings.ReplaceAll(valueRgx.ReplaceAllString(v, "_"), "=", "~"))
		if b.Len()+len(tag) > 256 {
			break
		}
		b.WriteString(";")
		b.WriteString(tag)
	}
	// the old state is split by vendors, must be concatenated with a `,`
	if len(oldState) == 0 {
		return b.String()
	}
	for _, s := range strings.Split(strings.Trim(oldState, " \t"), ",") {
		if strings.HasPrefix(s, "dd=") {
			continue
		}
		listLength++
		// if the resulting tracestateHeader exceeds 32 list-members,
		// remove the rightmost list-member(s)
		if listLength > 32 {
			break
		}
		b.WriteString("," + strings.Trim(s, " \t"))
	}
	return b.String()
}

func (p *propagatorW3c) Extract(carrier interface{}) (ddtrace.SpanContext, error) {
	switch c := carrier.(type) {
	case TextMapReader:
		return p.extractTextMap(c)
	default:
		return nil, ErrInvalidCarrier
	}
}

func (*propagatorW3c) extractTextMap(reader TextMapReader) (ddtrace.SpanContext, error) {
	var parentHeader string
	var stateHeader string
	// to avoid parsing tracestate header(s) if traceparent is invalid
	if err := reader.ForeachKey(func(k, v string) error {
		key := strings.ToLower(k)
		switch key {
		case traceparentHeader:
			if parentHeader != "" {
				return ErrSpanContextCorrupted
			}
			parentHeader = v
		case tracestateHeader:
			stateHeader = v
		}
		return nil
	}); err != nil {
		return nil, err
	}
	var ctx spanContext
	if err := parseTraceparent(&ctx, parentHeader); err != nil {
		return nil, err
	}
	if err := parseTracestate(&ctx, stateHeader); err != nil {
		return nil, err
	}
	return &ctx, nil
}

// parseTraceparent attempts to parse traceparentHeader which describes the position
// of the incoming request in its trace graph in a portable, fixed-length format.
// The format of the traceparentHeader is `-` separated string with in the
// following format: `version-traceId-spanID-flags`,
// where:
// - version - represents the version of the W3C Tracecontext Propagation format in hex format.
// - traceId - represents the propagated traceID in the format of 32 hex-encoded digits.
// - spanID - represents the propagated spanID (parentID) in the format of 16 hex-encoded digits.
// - flags - represents the propagated flags in the format of 2 hex-encoded digits, and supports 8 unique flags.
// Example value of HTTP `traceparent` header: `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`,
// Currently, Go tracer doesn't support 128-bit traceIDs, so the full traceID (32 hex-encoded digits) must be
// stored into a field that is accessible from the span’s context. TraceId will be parsed from the least significant 16
// hex-encoded digits into a 64-bit number.
func parseTraceparent(ctx *spanContext, header string) error {
	nonWordCutset := "_-\t \n"
	header = strings.ToLower(strings.Trim(header, "\t -"))
	if len(header) == 0 {
		return ErrSpanContextNotFound
	}
	if len(header) != 55 {
		return ErrSpanContextCorrupted
	}
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return ErrSpanContextCorrupted
	}
	version := strings.Trim(parts[0], nonWordCutset)
	if len(version) != 2 {
		return ErrSpanContextCorrupted
	}
	if v, err := strconv.ParseUint(version, 16, 64); err != nil || v == 255 {
		return ErrSpanContextCorrupted
	}
	// parsing traceID
	fullTraceID := strings.Trim(parts[1], nonWordCutset)
	if len(fullTraceID) != 32 {
		return ErrSpanContextCorrupted
	}
	// checking that the entire TraceID is a valid hex string
	if ok, err := regexp.MatchString("^[a-f0-9]+$", fullTraceID); !ok || err != nil {
		return ErrSpanContextCorrupted
	}
	var err error
	if ctx.traceID, err = strconv.ParseUint(fullTraceID[16:], 16, 64); err != nil {
		return ErrSpanContextCorrupted
	}
	if ctx.traceID == 0 {
		if strings.Trim(fullTraceID[:16], "0") == "" {
			return ErrSpanContextNotFound
		}
	}
	// setting trace-id to be used for span context propagation
	setPropagatingTag(ctx, w3cTraceIDTag, fullTraceID)
	// parsing spanID
	spanID := strings.Trim(parts[2], nonWordCutset)
	if len(spanID) != 16 {
		return ErrSpanContextCorrupted
	}
	if ok, err := regexp.MatchString("[a-f0-9]+", spanID); !ok || err != nil {
		return ErrSpanContextCorrupted
	}
	if ctx.spanID, err = strconv.ParseUint(spanID, 16, 64); err != nil {
		return ErrSpanContextCorrupted
	}
	if ctx.spanID == 0 {
		return ErrSpanContextNotFound
	}
	// parsing flags
	flags := parts[3]
	f, err := strconv.ParseInt(flags, 16, 8)
	if err != nil {
		return ErrSpanContextCorrupted
	}
	ctx.setSamplingPriority(int(f)&0x1, samplernames.Unknown)
	return nil
}

// parseTracestate attempts to parse tracestateHeader which is a list
// with up to 32 comma-separated (,) list-members.
// An example value would be: `vendorname1=opaqueValue1,vendorname2=opaqueValue2,dd=s:1;o:synthetics`,
// Where `dd` list contains values that would be in x-datadog-tags as well as those needed for propagation information.
// The keys to the “dd“ values have been shortened as follows to save space:
// `sampling_priority` = `s`
// `origin` = `o`
// `_dd.p.` prefix = `t.`
func parseTracestate(ctx *spanContext, header string) error {
	// if multiple headers are present, they must be combined and stored
	setPropagatingTag(ctx, tracestateHeader, header)
	list := strings.Split(strings.Trim(header, "\t "), ",")
	for _, s := range list {
		if !strings.HasPrefix(s, "dd=") {
			continue
		}
		dd := strings.Split(s[len("dd="):], ";")
		for _, val := range dd {
			x := strings.SplitN(val, ":", 2)
			if len(x) != 2 {
				continue
			}
			k, v := x[0], x[1]
			if k == "o" {
				ctx.origin = v
			} else if k == "s" {
				p, err := strconv.Atoi(v)
				if err != nil {
					// if the tracestate priority is absent, relying on traceparent value
					continue
				}
				flagPriority, _ := ctx.samplingPriority()
				if (flagPriority == 1 && p > 0) || (flagPriority == 0 && p <= 0) {
					ctx.setSamplingPriority(p, samplernames.Unknown)
				}
			} else if strings.HasPrefix(k, "t.") {
				k = k[len("t."):]
				v = strings.ReplaceAll(v, "~", "=")
				setPropagatingTag(ctx, "_dd.p."+k, v)
			}
		}
	}
	return nil
}
