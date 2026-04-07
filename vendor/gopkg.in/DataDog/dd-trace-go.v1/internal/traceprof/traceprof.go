// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

// Package traceprof contains shared logic for cross-cutting tracer/profiler features.
package traceprof

// pprof labels applied by the tracer to show up in the profiler's profiles.
const (
	SpanID          = "span id"
	LocalRootSpanID = "local root span id"
	TraceEndpoint   = "trace endpoint"
)

// env variables used to control cross-cutting tracer/profiling features.
const (
	CodeHotspotsEnvVar  = "DD_PROFILING_CODE_HOTSPOTS_COLLECTION_ENABLED" // aka code hotspots
	EndpointEnvVar      = "DD_PROFILING_ENDPOINT_COLLECTION_ENABLED"      // aka endpoint profiling
	EndpointCountEnvVar = "DD_PROFILING_ENDPOINT_COUNT_ENABLED"           // aka unit of work
)
