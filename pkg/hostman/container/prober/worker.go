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

package prober

import (
	"math/rand"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/runtime"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/container/prober/results"
)

// worker handles the periodic probing of its assigned container. Each worker has a go-routine
// associated with it which runs the probe loop until the container permanently terminates, or the
// stop channel is closed. The worker uses the probe Manager's statusManager to get up-to-date
// container IDs.
type worker struct {
	// Channel for stopping the probe.
	stopCh chan struct{}

	// The pod containing this probe (read-only)
	pod IPod

	// The container to probe (read-only)
	container *hostapi.ContainerDesc

	// Describes the probe configuration (read-only)
	spec *apis.ContainerProbe

	// The type of the worker.
	probeType apis.ContainerProbeType

	// The probe value during the initial delay.
	initialValue results.Result

	// Where to store this workers results.
	resultsManager results.Manager
	probeManager   *manager

	// The last known container ID for this worker.
	containerId string
	// The last probe result for this worker.
	lastResult results.Result
	// How many times in a row the probe has returned the same result.
	resultRun int

	// If set, skip probing
	onHold bool
}

// Creates and starts a new probe worker.
func newWorker(
	m *manager,
	probeType apis.ContainerProbeType,
	pod IPod,
	container *hostapi.ContainerDesc) *worker {
	w := &worker{
		stopCh:       make(chan struct{}, 1), // Buffer so stop() can be non-blocking.
		pod:          pod,
		container:    container,
		probeType:    probeType,
		probeManager: m,
		containerId:  container.Id,
	}

	switch probeType {
	//case apis.ContainerProbeTypeLiveness:
	//	w.spec = container.Spec.LivenessProbe
	//	w.resultsManager = m.livenessManager
	//	w.initialValue = results.Success
	case apis.ContainerProbeTypeStartup:
		w.spec = container.Spec.StartupProbe
		w.resultsManager = m.startupManager
		w.initialValue = results.Unknown
	}

	return w
}

// run periodically probes the container.
func (w *worker) run() {
	probeTickerPeriod := time.Duration(w.spec.PeriodSeconds) * time.Second

	// If host restarted the probes could be started in rapid succession.
	// Let the worker wait for a random portion of tickerPeriod before probing.
	time.Sleep(time.Duration(rand.Float64() * float64(probeTickerPeriod)))

	probeTicker := time.NewTicker(probeTickerPeriod)

	defer func() {
		// Clean up.
		probeTicker.Stop()
		if len(w.containerId) != 0 {
			w.resultsManager.Remove(w.containerId)
		}

		w.probeManager.removeWorker(w.pod.GetId(), w.container.Name, w.probeType)
	}()

probeLoop:
	for w.doProbe() {
		// Wait for next probe tick.
		select {
		case <-w.stopCh:
			break probeLoop
		case <-probeTicker.C:
			// continue
		}
	}
}

// stop stops the probe worker. The worker handles cleanup and removes itself from its manager.
// It is safe to call stop multiple times.
func (w *worker) stop() {
	select {
	case w.stopCh <- struct{}{}:
	default: // Non-blocking.
	}
}

// doProbe probes the container once and records the result.
// Returns whether the worker should continue.
func (w *worker) doProbe() (keepGoing bool) {
	// Actually eat panics (HandleCrash takes care of logging)
	defer func() { recover() }()
	defer runtime.HandleCrash(func(_ interface{}) {
		keepGoing = true
	})

	result, err := w.probeManager.prober.probe(w.probeType, w.pod, w.container)
	if err != nil {
		log.Errorf("probe: %s, pod: %s, container: %s, error: %v", w.probeType, w.pod.GetId(), w.container.Id, err)
		// prober error, throw away the result.
		return true
	}

	if w.lastResult == result.Result {
		w.resultRun++
	} else {
		w.lastResult = result.Result
		w.resultRun = 1
	}
	_, isContainerDirty := w.probeManager.dirtyContainers.Load(w.container.Id)

	if (result.Result == results.Failure && w.resultRun < int(w.spec.FailureThreshold)) ||
		(result.Result == results.Success && w.resultRun < int(w.spec.SuccessThreshold)) {
		return true
	}
	w.resultsManager.Set(w.containerId, result, w.pod, isContainerDirty)
	if isContainerDirty {
		log.Infof("clean dirty container %s of probe manager", w.container.Id)
		w.probeManager.cleanDirtyContainer(w.container.Id)
	}

	if (w.probeType == apis.ContainerProbeTypeLiveness || w.probeType == apis.ContainerProbeTypeStartup) && result.Result == results.Failure {
		// The container fails a liveness/startup check, it will need to be restarted.
		// Stop probing until we see a new container ID. This is to reduce the
		// chance of hitting #21751, where running `docker exec` when a
		// container is being stopped may lead to corrupted container state.
		w.onHold = true
		w.resultRun = 0
	}

	return true
}
