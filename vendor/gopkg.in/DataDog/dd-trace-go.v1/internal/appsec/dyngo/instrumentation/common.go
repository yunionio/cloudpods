// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

// Package instrumentation holds code commonly used between all instrumentation declinations (currently httpsec/grpcsec).
package instrumentation

import (
	"encoding/json"
	"fmt"
	"sync"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/samplernames"
)

// TagSetter is the interface needed to set a span tag.
type TagSetter interface {
	SetTag(string, interface{})
}

// TagsHolder wraps a map holding tags. The purpose of this struct is to be used by composition in an Operation
// to allow said operation to handle tags addition/retrieval. See httpsec/http.go and grpcsec/grpc.go.
type TagsHolder struct {
	tags map[string]interface{}
	mu   sync.Mutex
}

// NewTagsHolder returns a new instance of a TagsHolder struct.
func NewTagsHolder() TagsHolder {
	return TagsHolder{tags: map[string]interface{}{}}
}

// AddTag adds the key/value pair to the tags map
func (m *TagsHolder) AddTag(k string, v interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tags[k] = v
}

// Tags returns the tags map
func (m *TagsHolder) Tags() map[string]interface{} {
	return m.tags
}

// SecurityEventsHolder is a wrapper around a thread safe security events slice. The purpose of this struct is to be
// used by composition in an Operation to allow said operation to handle security events addition/retrieval.
// See httpsec/http.go and grpcsec/grpc.go.
type SecurityEventsHolder struct {
	events []json.RawMessage
	mu     sync.RWMutex
}

// AddSecurityEvents adds the security events to the collected events list.
// Thread safe.
func (s *SecurityEventsHolder) AddSecurityEvents(events ...json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
}

// Events returns the list of stored events.
func (s *SecurityEventsHolder) Events() []json.RawMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.events
}

// ClearEvents clears the list of stored events
func (s *SecurityEventsHolder) ClearEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = s.events[0:0]
}

// SetTags fills the span tags using the key/value pairs found in `tags`
func SetTags(span TagSetter, tags map[string]interface{}) {
	for k, v := range tags {
		span.SetTag(k, v)
	}
}

// SetStringTags fills the span tags using the key/value pairs of strings found
// in `tags`
func SetStringTags(span TagSetter, tags map[string]string) {
	for k, v := range tags {
		span.SetTag(k, v)
	}
}

// SetAppSecEnabledTags sets the AppSec-specific span tags that are expected to be in
// the web service entry span (span of type `web`) when AppSec is enabled.
func SetAppSecEnabledTags(span TagSetter) {
	span.SetTag("_dd.appsec.enabled", 1)
	span.SetTag("_dd.runtime_family", "go")
}

// SetEventSpanTags sets the security event span tags into the service entry span.
func SetEventSpanTags(span TagSetter, events []json.RawMessage) error {
	// Set the appsec event span tag
	val, err := makeEventTagValue(events)
	if err != nil {
		return err
	}
	span.SetTag("_dd.appsec.json", string(val))
	// Keep this span due to the security event
	//
	// This is a workaround to tell the tracer that the trace was kept by AppSec.
	// Passing any other value than `appsec.SamplerAppSec` has no effect.
	// Customers should use `span.SetTag(ext.ManualKeep, true)` pattern
	// to keep the trace, manually.
	span.SetTag(ext.ManualKeep, samplernames.AppSec)
	span.SetTag("_dd.origin", "appsec")
	// Set the appsec.event tag needed by the appsec backend
	span.SetTag("appsec.event", true)
	return nil
}

// Create the value of the security event tag.
// TODO(Julio-Guerra): a future libddwaf version should return something
//
//	avoiding us the following events concatenation logic which currently
//	involves unserializing the top-level JSON arrays to concatenate them
//	together.
//
// TODO(Julio-Guerra): avoid serializing the json in the request hot path
func makeEventTagValue(events []json.RawMessage) (json.RawMessage, error) {
	var v interface{}
	if l := len(events); l == 1 {
		// eventTag is the structure to use in the `_dd.appsec.json` span tag.
		// In this case of 1 event, it already is an array as expected.
		type eventTag struct {
			Triggers json.RawMessage `json:"triggers"`
		}
		v = eventTag{Triggers: events[0]}
	} else {
		// eventTag is the structure to use in the `_dd.appsec.json` span tag.
		// With more than one event, we need to concatenate the arrays together
		// (ie. convert [][]json.RawMessage into []json.RawMessage).
		type eventTag struct {
			Triggers []json.RawMessage `json:"triggers"`
		}
		concatenated := make([]json.RawMessage, 0, l) // at least len(events)
		for _, event := range events {
			// Unmarshal the top level array
			var tmp []json.RawMessage
			if err := json.Unmarshal(event, &tmp); err != nil {
				return nil, fmt.Errorf("unexpected error while unserializing the appsec event `%s`: %v", string(event), err)
			}
			concatenated = append(concatenated, tmp...)
		}
		v = eventTag{Triggers: concatenated}
	}

	tag, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("unexpected error while serializing the appsec event span tag: %v", err)
	}
	return tag, nil
}
