// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package ext

// span_kind values are set per span following the opentelemetry standard
// falls under the values of client, server, producer, consumer, and internal
const (

	// SpanKindServer indicates that the span covers server-side handling of a synchronous RPC or other remote request
	// This span should not have any local parents but can have other distributed parents
	SpanKindServer = "server"

	// SpanKindClient indicates that the span describes a request to some remote service.
	// This span should not have any local children but can have other distributed children
	SpanKindClient = "client"

	// SpanKindConsumer indicates that the span describes the initiators of an asynchronous request.
	// This span should not have any local parents but can have other distributed parents
	SpanKindConsumer = "consumer"

	// SpanKindProducer indicates that the span describes a child of an asynchronous producer request.
	// This span should not have any local children but can have other distributed children
	SpanKindProducer = "producer"

	// SpanKindInternal indicates that the span represents an internal operation within an application,
	// as opposed to an operations with remote parents or children.
	// This is the default value and not explicitly set to save memory
	SpanKindInternal = "internal"
)
