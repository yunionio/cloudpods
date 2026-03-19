// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

type traceWriter interface {
	// add adds traces to be sent by the writer.
	add([]*span)

	// flush causes the writer to send any buffered traces.
	flush()

	// stop gracefully shuts down the writer.
	stop()
}

type agentTraceWriter struct {
	// config holds the tracer configuration
	config *config

	// payload encodes and buffers traces in msgpack format
	payload *payload

	// climit limits the number of concurrent outgoing connections
	climit chan struct{}

	// wg waits for all uploads to finish
	wg sync.WaitGroup

	// prioritySampling is the prioritySampler into which agentTraceWriter will
	// read sampling rates sent by the agent
	prioritySampling *prioritySampler

	// statsd is used to send metrics
	statsd statsdClient
}

func newAgentTraceWriter(c *config, s *prioritySampler, statsdClient statsdClient) *agentTraceWriter {
	return &agentTraceWriter{
		config:           c,
		payload:          newPayload(),
		climit:           make(chan struct{}, concurrentConnectionLimit),
		prioritySampling: s,
		statsd:           statsdClient,
	}
}

func (h *agentTraceWriter) add(trace []*span) {
	if err := h.payload.push(trace); err != nil {
		h.statsd.Incr("datadog.tracer.traces_dropped", []string{"reason:encoding_error"}, 1)
		log.Error("Error encoding msgpack: %v", err)
	}
	if h.payload.size() > payloadSizeLimit {
		h.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:size"}, 1)
		h.flush()
	}
}

func (h *agentTraceWriter) stop() {
	h.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:shutdown"}, 1)
	h.flush()
	h.wg.Wait()
}

// flush will push any currently buffered traces to the server.
func (h *agentTraceWriter) flush() {
	if h.payload.itemCount() == 0 {
		return
	}
	h.wg.Add(1)
	h.climit <- struct{}{}
	oldp := h.payload
	h.payload = newPayload()
	go func(p *payload) {
		defer func(start time.Time) {
			<-h.climit
			h.wg.Done()
			h.statsd.Timing("datadog.tracer.flush_duration", time.Since(start), nil, 1)
		}(time.Now())
		size, count := p.size(), p.itemCount()
		log.Debug("Sending payload: size: %d traces: %d\n", size, count)
		rc, err := h.config.transport.send(p)
		if err != nil {
			h.statsd.Count("datadog.tracer.traces_dropped", int64(count), []string{"reason:send_failed"}, 1)
			log.Error("lost %d traces: %v", count, err)
		} else {
			h.statsd.Count("datadog.tracer.flush_bytes", int64(size), nil, 1)
			h.statsd.Count("datadog.tracer.flush_traces", int64(count), nil, 1)
			if err := h.prioritySampling.readRatesJSON(rc); err != nil {
				h.statsd.Incr("datadog.tracer.decode_error", nil, 1)
			}
		}
	}(oldp)
}

// logWriter specifies the output target of the logTraceWriter; replaced in tests.
var logWriter io.Writer = os.Stdout

// logTraceWriter encodes traces into a format understood by the Datadog Forwarder
// (https://github.com/DataDog/datadog-serverless-functions/tree/master/aws/logs_monitoring)
// and writes them to os.Stdout. This is used to send traces from an AWS Lambda environment.
type logTraceWriter struct {
	config    *config
	buf       bytes.Buffer
	hasTraces bool
	w         io.Writer
	statsd    statsdClient
}

func newLogTraceWriter(c *config, statsdClient statsdClient) *logTraceWriter {
	w := &logTraceWriter{
		config: c,
		w:      logWriter,
		statsd: statsdClient,
	}
	w.resetBuffer()
	return w
}

const (
	// maxFloatLength is the maximum length that a string encoded by encodeFloat will be.
	maxFloatLength = 24

	// logBufferSuffix is the final string that the trace writer has to append to a buffer to close
	// the JSON.
	logBufferSuffix = "]}\n"

	// logBufferLimit is the maximum size log line allowed by cloudwatch
	logBufferLimit = 256 * 1024
)

func (h *logTraceWriter) resetBuffer() {
	h.buf.Reset()
	h.buf.WriteString(`{"traces": [`)
	h.hasTraces = false
}

// encodeFloat correctly encodes float64 into the JSON format followed by ES6.
// This code is reworked from Go's encoding/json package
// (https://github.com/golang/go/blob/go1.15/src/encoding/json/encode.go#L573)
//
// One important departure from encoding/json is that infinities and nans are encoded
// as null rather than signalling an error.
func encodeFloat(p []byte, f float64) []byte {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return append(p, "null"...)
	}
	abs := math.Abs(f)
	if abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		p = strconv.AppendFloat(p, f, 'e', -1, 64)
		// clean up e-09 to e-9
		n := len(p)
		if n >= 4 && p[n-4] == 'e' && p[n-3] == '-' && p[n-2] == '0' {
			p[n-2] = p[n-1]
			p = p[:n-1]
		}
	} else {
		p = strconv.AppendFloat(p, f, 'f', -1, 64)
	}
	return p
}

func (h *logTraceWriter) encodeSpan(s *span) {
	var scratch [maxFloatLength]byte
	h.buf.WriteString(`{"trace_id":"`)
	h.buf.Write(strconv.AppendUint(scratch[:0], uint64(s.TraceID), 16))
	h.buf.WriteString(`","span_id":"`)
	h.buf.Write(strconv.AppendUint(scratch[:0], uint64(s.SpanID), 16))
	h.buf.WriteString(`","parent_id":"`)
	h.buf.Write(strconv.AppendUint(scratch[:0], uint64(s.ParentID), 16))
	h.buf.WriteString(`","name":`)
	h.marshalString(s.Name)
	h.buf.WriteString(`,"resource":`)
	h.marshalString(s.Resource)
	h.buf.WriteString(`,"error":`)
	h.buf.Write(strconv.AppendInt(scratch[:0], int64(s.Error), 10))
	h.buf.WriteString(`,"meta":{`)
	first := true
	for k, v := range s.Meta {
		if first {
			first = false
		} else {
			h.buf.WriteString(`,`)
		}
		h.marshalString(k)
		h.buf.WriteString(":")
		h.marshalString(v)
	}
	h.buf.WriteString(`},"metrics":{`)
	first = true
	for k, v := range s.Metrics {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			// The trace forwarder does not support infinity or nan, so we do not send metrics with those values.
			continue
		}
		if first {
			first = false
		} else {
			h.buf.WriteString(`,`)
		}
		h.marshalString(k)
		h.buf.WriteString(`:`)
		h.buf.Write(encodeFloat(scratch[:0], v))
	}
	h.buf.WriteString(`},"start":`)
	h.buf.Write(strconv.AppendInt(scratch[:0], s.Start, 10))
	h.buf.WriteString(`,"duration":`)
	h.buf.Write(strconv.AppendInt(scratch[:0], s.Duration, 10))
	h.buf.WriteString(`,"service":`)
	h.marshalString(s.Service)
	h.buf.WriteString(`}`)
}

// marshalString marshals the string str as JSON into the writer's buffer.
// Should be used whenever writing non-constant string data to ensure correct sanitization.
func (h *logTraceWriter) marshalString(str string) {
	m, err := json.Marshal(str)
	if err != nil {
		log.Error("Error marshaling value %q: %v", str, err)
	} else {
		h.buf.Write(m)
	}
}

type encodingError struct {
	cause      error
	dropReason string
}

// writeTrace makes an effort to write the trace into the current buffer. It returns
// the number of spans (n) that it wrote and an error (err), if one occurred.
// n may be less than len(trace), meaning that only the first n spans of the trace
// fit into the current buffer. Once the buffer is flushed, the remaining spans
// from the trace can be retried.
// An error, if one is returned, indicates that a span in the trace is too large
// to fit in one buffer, and the trace cannot be written.
func (h *logTraceWriter) writeTrace(trace []*span) (n int, err *encodingError) {
	startn := h.buf.Len()
	if !h.hasTraces {
		h.buf.WriteByte('[')
	} else {
		h.buf.WriteString(", [")
	}
	written := 0
	for i, s := range trace {
		n := h.buf.Len()
		if i > 0 {
			h.buf.WriteByte(',')
		}
		h.encodeSpan(s)
		if h.buf.Len() > logBufferLimit-len(logBufferSuffix) {
			// This span is too big to fit in the current buffer.
			if i == 0 {
				// This was the first span in this trace. This means we should truncate
				// everything we wrote in writeTrace
				h.buf.Truncate(startn)
				if !h.hasTraces {
					// This is the first span of the first trace in the buffer and it's too big.
					// We will never be able to send this trace, so we will drop it.
					return 0, &encodingError{cause: errors.New("span too large for buffer"), dropReason: "trace_too_large"}
				}
				return 0, nil
			}
			// This span was too big, but it might fit in the next buffer.
			// We can finish this trace and try again with an empty buffer (see *logTaceWriter.add)
			h.buf.Truncate(n)
			break
		}
		written++
	}
	h.buf.WriteByte(']')
	h.hasTraces = true
	return written, nil
}

// add adds a trace to the writer's buffer.
func (h *logTraceWriter) add(trace []*span) {
	// Try adding traces to the buffer until we flush them all or encounter an error.
	for len(trace) > 0 {
		n, err := h.writeTrace(trace)
		if err != nil {
			log.Error("Lost a trace: %s", err.cause)
			h.statsd.Count("datadog.tracer.traces_dropped", 1, []string{"reason:" + err.dropReason}, 1)
			return
		}
		trace = trace[n:]
		// If there are traces left that didn't fit into the buffer, flush the buffer and loop to
		// write the remaining spans.
		if len(trace) > 0 {
			h.flush()
		}
	}
}

func (h *logTraceWriter) stop() {
	h.statsd.Incr("datadog.tracer.flush_triggered", []string{"reason:shutdown"}, 1)
	h.flush()
}

// flush will write any buffered traces to standard output.
func (h *logTraceWriter) flush() {
	if !h.hasTraces {
		return
	}
	h.buf.WriteString(logBufferSuffix)
	h.w.Write(h.buf.Bytes())
	h.resetBuffer()
}
