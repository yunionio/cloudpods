// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"strconv"
	"strings"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/globalconfig"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

// SQLCommentInjectionMode represents the mode of SQL comment injection.
//
// Deprecated: Use DBMPropagationMode instead.
type SQLCommentInjectionMode DBMPropagationMode

const (
	// SQLInjectionUndefined represents the comment injection mode is not set. This is the same as SQLInjectionDisabled.
	SQLInjectionUndefined SQLCommentInjectionMode = SQLCommentInjectionMode(DBMPropagationModeUndefined)
	// SQLInjectionDisabled represents the comment injection mode where all injection is disabled.
	SQLInjectionDisabled SQLCommentInjectionMode = SQLCommentInjectionMode(DBMPropagationModeDisabled)
	// SQLInjectionModeService represents the comment injection mode where only service tags (name, env, version) are injected.
	SQLInjectionModeService SQLCommentInjectionMode = SQLCommentInjectionMode(DBMPropagationModeService)
	// SQLInjectionModeFull represents the comment injection mode where both service tags and tracing tags. Tracing tags include span id, trace id and sampling priority.
	SQLInjectionModeFull SQLCommentInjectionMode = SQLCommentInjectionMode(DBMPropagationModeFull)
)

// DBMPropagationMode represents the mode of dbm propagation.
//
// Note that enabling sql comment propagation results in potentially confidential data (service names)
// being stored in the databases which can then be accessed by other 3rd parties that have been granted
// access to the database.
type DBMPropagationMode string

const (
	// DBMPropagationModeUndefined represents the dbm propagation mode not being set. This is the same as DBMPropagationModeDisabled.
	DBMPropagationModeUndefined DBMPropagationMode = ""
	// DBMPropagationModeDisabled represents the dbm propagation mode where all propagation is disabled.
	DBMPropagationModeDisabled DBMPropagationMode = "disabled"
	// DBMPropagationModeService represents the dbm propagation mode where only service tags (name, env, version) are propagated to dbm.
	DBMPropagationModeService DBMPropagationMode = "service"
	// DBMPropagationModeFull represents the dbm propagation mode where both service tags and tracing tags are propagated. Tracing tags include span id, trace id and the sampled flag.
	DBMPropagationModeFull DBMPropagationMode = "full"
)

// Key names for SQL comment tags.
const (
	sqlCommentTraceParent   = "traceparent"
	sqlCommentParentService = "ddps"
	sqlCommentDBService     = "dddbs"
	sqlCommentParentVersion = "ddpv"
	sqlCommentEnv           = "dde"
)

// Current trace context version (see https://www.w3.org/TR/trace-context/#version)
const w3cContextVersion = "00"

// SQLCommentCarrier is a carrier implementation that injects a span context in a SQL query in the form
// of a sqlcommenter formatted comment prepended to the original query text.
// See https://google.github.io/sqlcommenter/spec/ for more details.
type SQLCommentCarrier struct {
	Query         string
	Mode          DBMPropagationMode
	DBServiceName string
	SpanID        uint64
}

// Inject injects a span context in the carrier's Query field as a comment.
func (c *SQLCommentCarrier) Inject(spanCtx ddtrace.SpanContext) error {
	c.SpanID = generateSpanID(now())
	tags := make(map[string]string)
	switch c.Mode {
	case DBMPropagationModeUndefined:
		fallthrough
	case DBMPropagationModeDisabled:
		return nil
	case DBMPropagationModeFull:
		var (
			samplingPriority int
			traceID          uint64
		)
		if ctx, ok := spanCtx.(*spanContext); ok {
			if sp, ok := ctx.samplingPriority(); ok {
				samplingPriority = sp
			}
			traceID = ctx.TraceID()
		}
		if traceID == 0 {
			traceID = c.SpanID
		}
		sampled := int64(0)
		if samplingPriority > 0 {
			sampled = 1
		}
		tags[sqlCommentTraceParent] = encodeTraceParent(traceID, c.SpanID, sampled)
		fallthrough
	case DBMPropagationModeService:
		var env, version string
		if ctx, ok := spanCtx.(*spanContext); ok {
			if e, ok := ctx.meta(ext.Environment); ok {
				env = e
			}
			if v, ok := ctx.meta(ext.Version); ok {
				version = v
			}
		}
		if globalconfig.ServiceName() != "" {
			tags[sqlCommentParentService] = globalconfig.ServiceName()
		}
		if env != "" {
			tags[sqlCommentEnv] = env
		}
		if version != "" {
			tags[sqlCommentParentVersion] = version
		}
		tags[sqlCommentDBService] = c.DBServiceName
	}
	c.Query = commentQuery(c.Query, tags)
	return nil
}

// encodeTraceParent encodes trace parent as per the w3c trace context spec (https://www.w3.org/TR/trace-context/#version).
func encodeTraceParent(traceID uint64, spanID uint64, sampled int64) string {
	var b strings.Builder
	// traceparent has a fixed length of 55:
	// 2 bytes for the version, 32 for the trace id, 16 for the span id, 2 for the sampled flag and 3 for separators
	b.Grow(55)
	b.WriteString(w3cContextVersion)
	b.WriteRune('-')
	tid := strconv.FormatUint(traceID, 16)
	for i := 0; i < 32-len(tid); i++ {
		b.WriteRune('0')
	}
	b.WriteString(tid)
	b.WriteRune('-')
	sid := strconv.FormatUint(spanID, 16)
	for i := 0; i < 16-len(sid); i++ {
		b.WriteRune('0')
	}
	b.WriteString(sid)
	b.WriteRune('-')
	b.WriteRune('0')
	b.WriteString(strconv.FormatInt(sampled, 16))
	return b.String()
}

var (
	keyReplacer   = strings.NewReplacer(" ", "%20", "!", "%21", "#", "%23", "$", "%24", "%", "%25", "&", "%26", "'", "%27", "(", "%28", ")", "%29", "*", "%2A", "+", "%2B", ",", "%2C", "/", "%2F", ":", "%3A", ";", "%3B", "=", "%3D", "?", "%3F", "@", "%40", "[", "%5B", "]", "%5D")
	valueReplacer = strings.NewReplacer(" ", "%20", "!", "%21", "#", "%23", "$", "%24", "%", "%25", "&", "%26", "'", "%27", "(", "%28", ")", "%29", "*", "%2A", "+", "%2B", ",", "%2C", "/", "%2F", ":", "%3A", ";", "%3B", "=", "%3D", "?", "%3F", "@", "%40", "[", "%5B", "]", "%5D", "'", "\\'")
)

// commentQuery returns the given query with the tags from the SQLCommentCarrier applied to it as a
// prepended SQL comment. The format of the comment follows the sqlcommenter spec.
// See https://google.github.io/sqlcommenter/spec/ for more details.
func commentQuery(query string, tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	var b strings.Builder
	// the sqlcommenter specification dictates that tags should be sorted. Since we know all injected keys,
	// we skip a sorting operation by specifying the order of keys statically
	orderedKeys := []string{sqlCommentDBService, sqlCommentEnv, sqlCommentParentService, sqlCommentParentVersion, sqlCommentTraceParent}
	first := true
	for _, k := range orderedKeys {
		if v, ok := tags[k]; ok {
			// we need to URL-encode both keys and values and escape single quotes in values
			// https://google.github.io/sqlcommenter/spec/
			key := keyReplacer.Replace(k)
			val := valueReplacer.Replace(v)
			if first {
				b.WriteString("/*")
			} else {
				b.WriteRune(',')
			}
			b.WriteString(key)
			b.WriteRune('=')
			b.WriteRune('\'')
			b.WriteString(val)
			b.WriteRune('\'')
			first = false
		}
	}
	if b.Len() == 0 {
		return query
	}
	b.WriteString("*/")
	if query == "" {
		return b.String()
	}
	log.Debug("Injected sql comment: %s", b.String())
	b.WriteRune(' ')
	b.WriteString(query)
	return b.String()
}

// Extract is not implemented on SQLCommentCarrier
func (c *SQLCommentCarrier) Extract() (ddtrace.SpanContext, error) {
	return nil, nil
}
