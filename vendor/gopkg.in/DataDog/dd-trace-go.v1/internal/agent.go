// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package internal

import (
	"net/url"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

// AgentURLFromEnv determines the trace agent URL from environment variable
// DD_TRACE_AGENT_URL. If the determined value is valid and the scheme is
// supported (unix, http or https), it will return an *url.URL. Otherwise,
// it returns nil.
func AgentURLFromEnv() *url.URL {
	agentURL := os.Getenv("DD_TRACE_AGENT_URL")
	if agentURL == "" {
		return nil
	}
	u, err := url.Parse(agentURL)
	if err != nil {
		log.Warn("Failed to parse DD_TRACE_AGENT_URL: %v", err)
		return nil
	}
	switch u.Scheme {
	case "unix", "http", "https":
		return u
	default:
		log.Warn("Unsupported protocol %q in Agent URL %q. Must be one of: http, https, unix.", u.Scheme, agentURL)
		return nil
	}
}
