// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package results

import (
	"fmt"
	"strings"
	"sync"
)

func NewFailure(reason string) ProbeResult {
	return newProbeResult(Failure, reason)
}

func NewSuccess(reason string) ProbeResult {
	return newProbeResult(Success, reason)
}

func NewUnknown(reason string) ProbeResult {
	return newProbeResult(Unknown, reason)
}

func newProbeResult(r Result, reason string) ProbeResult {
	return ProbeResult{
		Result: r,
		Reason: reason,
	}
}

type ProbeResult struct {
	Result
	Reason string
}

func (pr ProbeResult) String() string {
	return fmt.Sprintf("%s: %s", pr.Result.String(), pr.Reason)
}

func (pr ProbeResult) IsNetFailedError() bool {
	if pr.Result != Failure {
		return false
	}
	netFailedMsg := []string{
		"no route to host",
		"i/o timeout",
	}
	for _, msg := range netFailedMsg {
		if strings.Contains(pr.Reason, msg) {
			return true
		}
	}
	return false
}

// Result is the type for probe results.
type Result int

const (
	// Unknown is encoded as -1 (type Result)
	Unknown Result = iota - 1

	// Success is encoded as 0 (type Result)
	Success

	// Failure is encoded as 1 (type Result)
	Failure
)

func (r Result) String() string {
	switch r {
	case Success:
		return "Success"
	case Failure:
		return "Failure"
	default:
		return "UNKNOWN"
	}
}

type IPod interface {
	GetId() string
	IsRunning() bool
}

// Update is an enum of the types of updates sent over the Updates channel.
type Update struct {
	ContainerID string
	Result      ProbeResult
	PodUID      string
	Pod         IPod
}

// Manager provides a probe results cache and channel of updates
type Manager interface {
	// Get returns the cached result for the container with the given ID.
	Get(containerId string) (ProbeResult, bool)
	// Set sets the cached result for the container with the given ID.
	// The pod is only included to be sent with the update.
	Set(containerId string, result ProbeResult, pod IPod, force bool)
	// Remove clears the cached result for the container with the given ID.
	Remove(containerId string)
	// Updates creates a channel that receives an Update whenever its result changes (but not
	// removed).
	// NOTE: The current implementation only supports a single updates channel.
	Updates() <-chan Update
}

var _ Manager = &manager{}

type manager struct {
	// guards the cache
	sync.RWMutex
	// map of container ID -> probe Result
	cache map[string]ProbeResult
	// channel of updates
	updates chan Update
}

func NewManager() Manager {
	return &manager{
		cache:   make(map[string]ProbeResult),
		updates: make(chan Update, 20),
	}
}

func (m *manager) Get(id string) (ProbeResult, bool) {
	m.RLock()
	defer m.RUnlock()
	result, found := m.cache[id]
	return result, found
}

func (m *manager) Set(id string, result ProbeResult, pod IPod, force bool) {
	if m.setInternal(id, result, force) {
		m.updates <- Update{
			ContainerID: id,
			Result:      result,
			PodUID:      pod.GetId(),
			Pod:         pod,
		}
	}
}

// Internal helper for locked portion of set. Returns whether an update should be sent.
func (m *manager) setInternal(id string, result ProbeResult, force bool) bool {
	m.Lock()
	defer m.Unlock()
	prev, exists := m.cache[id]
	if !exists || prev.Result != result.Result || prev.IsNetFailedError() != result.IsNetFailedError() || force {
		m.cache[id] = result
		return true
	}
	return false
}

func (m *manager) Remove(id string) {
	m.Lock()
	defer m.Unlock()
	delete(m.cache, id)
}

func (m *manager) Updates() <-chan Update {
	return m.updates
}
