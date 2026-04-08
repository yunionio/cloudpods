// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package tracer contains Datadog's core tracing client. It is used to trace
// requests as they flow across web servers, databases and microservices, giving
// developers visibility into bottlenecks and troublesome requests. To start the
// tracer, simply call the start method along with an optional set of options.
// By default, the trace agent is considered to be found at "localhost:8126". In a
// setup where this would be different (let's say 127.0.0.1:1234), we could do:
//
//	tracer.Start(tracer.WithAgentAddr("127.0.0.1:1234"))
//	defer tracer.Stop()
//
// The tracing client can perform trace sampling. While the trace agent
// already samples traces to reduce bandwidth usage, client sampling reduces
// performance overhead. To make use of it, the package comes with a ready-to-use
// rate sampler that can be passed to the tracer. To use it and keep only 30% of the
// requests, one would do:
//
//	s := tracer.NewRateSampler(0.3)
//	tracer.Start(tracer.WithSampler(s))
//
// More precise control of sampling rates can be configured using sampling rules.
// This can be applied based on span name, service or both, and is used to determine
// the sampling rate to apply. MaxPerSecond specifies max number of spans per second
// that can be sampled per the rule and applies only to sampling rules of type
// tracer.SamplingRuleSpan. If MaxPerSecond is not specified, the default is no limit.
//
//	rules := []tracer.SamplingRule{
//	      // sample 10% of traces with the span name "web.request"
//	      tracer.NameRule("web.request", 0.1),
//	      // sample 20% of traces for the service "test-service"
//	      tracer.ServiceRule("test-service", 0.2),
//	      // sample 30% of traces when the span name is "db.query" and the service
//	      // is "postgres.db"
//	      tracer.NameServiceRule("db.query", "postgres.db", 0.3),
//	      // sample 100% of traces when service and name match these regular expressions
//	      {Service: regexp.MustCompile("^test-"), Name: regexp.MustCompile("http\\..*"), Rate: 1.0},
//	      // sample 50% of spans when service and name match these glob patterns with no limit on the number of spans
//	      tracer.SpanNameServiceRule("^test-", "http\\..*", 0.5),
//	      // sample 50% of spans when service and name match these glob patterns up to 100 spans per second
//	      tracer.SpanNameServiceMPSRule("^test-", "http\\..*", 0.5, 100),
//	}
//	tracer.Start(tracer.WithSamplingRules(rules))
//	defer tracer.Stop()
//
// Sampling rules can also be configured at runtime using the DD_TRACE_SAMPLING_RULES and
// DD_SPAN_SAMPLING_RULES environment variables. When set, it overrides rules set by tracer.WithSamplingRules.
// The value is a JSON array of objects.
// For trace sampling rules, the "sample_rate" field is required, the "name" and "service" fields are optional.
// For span sampling rules, the "name" and "service", if specified, must be a valid glob pattern,
// i.e. a string where "*" matches any contiguous substring, even an empty string,
// and "?" character matches exactly one of any character.
// The "sample_rate" field is optional, and if not specified, defaults to "1.0", sampling 100% of the spans.
// The "max_per_second" field is optional, and if not specified, defaults to 0, keeping all the previously sampled spans.
//
//	export DD_TRACE_SAMPLING_RULES='[{"name": "web.request", "sample_rate": 1.0}]'
//	export DD_SPAN_SAMPLING_RULES='[{"service":"test.?","name": "web.*", "sample_rate": 1.0, "max_per_second":100}]'
//
// To create spans, use the functions StartSpan and StartSpanFromContext. Both accept
// StartSpanOptions that can be used to configure the span. A span that is started
// with no parent will begin a new trace. See the function documentation for details
// on specific usage. Each trace has a hard limit of 100,000 spans, after which the
// trace will be dropped and give a diagnostic log message. In practice users should
// not approach this limit as traces of this size are not useful and impossible to
// visualize.
//
// See the contrib package ( https://pkg.go.dev/gopkg.in/DataDog/dd-trace-go.v1/contrib )
// for integrating datadog with various libraries, frameworks and clients.
//
// All spans created by the tracer contain a context hereby referred to as the span
// context. Note that this is different from Go's context. The span context is used
// to package essential information from a span, which is needed when creating child
// spans that inherit from it. Thus, a child span is created from a span's span context.
// The span context can originate from within the same process, but also a
// different process or even a different machine in the case of distributed tracing.
//
// To make use of distributed tracing, a span's context may be injected via a carrier
// into a transport (HTTP, RPC, etc.) to be extracted on the other end and used to
// create spans that are direct descendants of it. A couple of carrier interfaces
// which should cover most of the use-case scenarios are readily provided, such as
// HTTPCarrier and TextMapCarrier. Users are free to create their own, which will work
// with our propagation algorithm as long as they implement the TextMapReader and TextMapWriter
// interfaces. An example alternate implementation is the MDCarrier in our gRPC integration.
//
// As an example, injecting a span's context into an HTTP request would look like this:
//
//	req, err := http.NewRequest("GET", "http://example.com", nil)
//	// ...
//	err := tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header))
//	// ...
//	http.DefaultClient.Do(req)
//
// Then, on the server side, to continue the trace one would do:
//
//	sctx, err := tracer.Extract(tracer.HTTPHeadersCarrier(req.Header))
//	// ...
//	span := tracer.StartSpan("child.span", tracer.ChildOf(sctx))
//
// In the same manner, any means can be used as a carrier to inject a context into a transport. Go's
// context can also be used as a means to transport spans within the same process. The methods
// StartSpanFromContext, ContextWithSpan and SpanFromContext exist for this reason.
//
// Some libraries and frameworks are supported out-of-the-box by using one
// of our integrations. You can see a list of supported integrations here:
// https://godoc.org/gopkg.in/DataDog/dd-trace-go.v1/contrib
package tracer // import "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
