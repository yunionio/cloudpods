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
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/hostman/container/prober/results"
	"yunion.io/x/onecloud/pkg/hostman/container/status"
	"yunion.io/x/onecloud/pkg/hostman/guestman/container"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

// Key uniquely identifying container probes
type probeKey struct {
	podUid        string
	containerName string
	probeType     apis.ContainerProbeType
}

// Manager manages pod probing. It creates a probe "worker" for every container that specifies a
// probe (AddPod). The worker periodically probes its assigned container and caches the results. The
// manager use the cached probe results to set the appropriate Ready state in the PodStatus when
// requested (UpdatePodStatus). Updating probe parameters is not currently supported.
// TODO: Move liveness probing out of the runtime, to here.
type Manager interface {
	// AddPod creates new probe workers for every container probe. This should be called for every
	// pod created.
	AddPod(pod *desc.SGuestDesc)

	// RemovePod handles cleaning up the removed pod state, including terminating probe workers and
	// deleting cached results.
	RemovePod(pod *desc.SGuestDesc)

	// CleanupPods handles cleaning up pods which should no longer be running.
	// It takes a map of "desired pods" which should not be cleaned up.
	CleanupPods(desiredPods map[string]sets.Empty)

	// UpdatePodStatus modifies the given PodStatus with the appropriate Ready state for each
	// container based on container running status, cached probe results and worker states.
	UpdatePodStatus(podId string)

	// Start starts the Manager sync loops.
	Start()
}

type manager struct {
	// Map of active workers for probes
	workers map[probeKey]*worker
	// Lock for accessing & mutating workers
	workerLock sync.RWMutex

	statusManager status.Manager

	// readinessManager manages the results of readiness probes
	// readinessManager results.Manager

	// livenessManager manages the results of liveness probes
	livenessManager results.Manager

	// startupManager manages the results of startup probes
	startupManager results.Manager

	// prober executes the probe actions
	prober *prober
}

func NewManager(
	statusManager status.Manager,
	livenessManager results.Manager,
	startupManager results.Manager,
	runner container.CommandRunner) Manager {
	prober := newProber(runner)
	return &manager{
		statusManager:   statusManager,
		prober:          prober,
		livenessManager: livenessManager,
		startupManager:  startupManager,
		workers:         make(map[probeKey]*worker),
		workerLock:      sync.RWMutex{},
	}
}

// Start syncing probe status. This should only be called once.
func (m *manager) Start() {
	// start syncing readiness.
	//go wait.Forever(m.updateReadiness, 0)
	// start syncing startup.
	go wait.Forever(m.updateStartup, 0)
}

func (m *manager) AddPod(pod *desc.SGuestDesc) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()

	key := probeKey{podUid: pod.Uuid}
	for _, c := range pod.Containers {
		key.containerName = c.Name
		if c.Spec.StartupProbe != nil {
			key.probeType = apis.ContainerProbeTypeStartup
			if _, ok := m.workers[key]; ok {
				log.Errorf("Startup probe already exists: %s:%s", pod.Name, c.Name)
				return
			}
			w := newWorker(m, key.probeType, pod, c)
			m.workers[key] = w
			go w.run()
		}

		/*if c.Spec.LivenessProbe != nil {
			key.probeType = apis.ContainerProbeTypeLiveness
			if _, ok := m.workers[key]; ok {
				log.Errorf("Liveness probe already exists: %s:%s", pod.Name, c.Name)
				return
			}
			w := newWorker(m, key.probeType, pod, c)
			m.workers[key] = w
			go w.run()
		}*/
	}
}

func (m *manager) RemovePod(pod *desc.SGuestDesc) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()

	key := probeKey{podUid: pod.Uuid}
	for _, c := range pod.Containers {
		key.containerName = c.Name
		for _, probeType := range []apis.ContainerProbeType{apis.ContainerProbeTypeLiveness, apis.ContainerProbeTypeReadiness, apis.ContainerProbeTypeStartup} {
			key.probeType = probeType
			if worker, ok := m.workers[key]; ok {
				worker.stop()
			}
		}
	}
}

func (m *manager) CleanupPods(desiredPods map[string]sets.Empty) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()

	for key, worker := range m.workers {
		if _, ok := desiredPods[key.podUid]; !ok {
			worker.stop()
		}
	}
}

func (m *manager) UpdatePodStatus(status string) {}

func (m *manager) getWorker(podId string, containerName string, probeType apis.ContainerProbeType) (*worker, bool) {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	worker, ok := m.workers[probeKey{podId, containerName, probeType}]
	return worker, ok
}

// Called by the worker after exiting
func (m *manager) removeWorker(podId string, containerName string, probeType apis.ContainerProbeType) {
	m.workerLock.Lock()
	defer m.workerLock.Unlock()
	delete(m.workers, probeKey{podUid: podId, containerName: containerName, probeType: probeType})
}

func (m *manager) workerCount() int {
	m.workerLock.RLock()
	defer m.workerLock.RUnlock()
	return len(m.workers)
}

/*func (m *manager) updateReadiness() {
	update := <-m.readinessManager.Updates()

	ready := update.Result == results.Success
	m.statusManager.SetContainerReadiness(update.PodUID, update.ContainerID, ready)
}*/

func (m *manager) updateStartup() {
	update := <-m.startupManager.Updates()

	started := update.Result.Result == results.Success
	m.statusManager.SetContainerStartup(update.PodUID, update.ContainerID, started, update.Result)
}
