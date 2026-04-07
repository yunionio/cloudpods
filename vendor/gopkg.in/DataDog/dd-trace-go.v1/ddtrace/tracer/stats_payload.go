// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:generate msgp -unexported -marshal=false -o=stats_payload_msgp.go -tests=false

package tracer

// statsPayload specifies information about client computed stats and is encoded
// to be sent to the agent.
type statsPayload struct {
	// Hostname specifies the hostname of the application.
	Hostname string

	// Env specifies the env. of the application, as defined by the user.
	Env string

	// Version specifies the application version.
	Version string

	// Stats holds all stats buckets computed within this payload.
	Stats []statsBucket
}

// statsBucket specifies a set of stats computed over a duration.
type statsBucket struct {
	// Start specifies the beginning of this bucket.
	Start uint64

	// Duration specifies the duration of this bucket.
	Duration uint64

	// Stats contains a set of statistics computed for the duration of this bucket.
	Stats []groupedStats
}

// groupedStats contains a set of statistics grouped under various aggregation keys.
type groupedStats struct {
	// These fields indicate the properties under which the stats were aggregated.
	Service        string `json:"service,omitempty"`
	Name           string `json:"name,omitempty"`
	Resource       string `json:"resource,omitempty"`
	HTTPStatusCode uint32 `json:"HTTP_status_code,omitempty"`
	Type           string `json:"type,omitempty"`
	DBType         string `json:"DB_type,omitempty"`

	// These fields specify the stats for the above aggregation.
	Hits         uint64 `json:"hits,omitempty"`
	Errors       uint64 `json:"errors,omitempty"`
	Duration     uint64 `json:"duration,omitempty"`
	OkSummary    []byte `json:"okSummary,omitempty"`
	ErrorSummary []byte `json:"errorSummary,omitempty"`
	Synthetics   bool   `json:"synthetics,omitempty"`
	TopLevelHits uint64 `json:"topLevelHits,omitempty"`
}
