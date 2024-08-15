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

package status

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/container/prober/results"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type Manager interface {
	// SetContainerStartup updates the container status with the given startup
	// and triggers a status update.
	SetContainerStartup(podId string, containerId string, started bool, result results.ProbeResult)
}

type manager struct{}

func NewManager() Manager {
	return &manager{}
}

func (m *manager) SetContainerStartup(podId string, containerId string, started bool, result results.ProbeResult) {
	status := computeapi.CONTAINER_STATUS_PROBE_FAILED
	if started {
		status = computeapi.CONTAINER_STATUS_RUNNING
	}
	input := &apis.PerformStatusInput{
		Status: status,
		Reason: result.Reason,
	}
	if _, err := hostutils.UpdateContainerStatus(context.Background(), containerId, input); err != nil {
		log.Errorf("set container(%s/%s) status failed: %s", podId, containerId, err)
	}
}
