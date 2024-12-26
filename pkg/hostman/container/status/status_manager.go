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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/container/prober/results"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
)

type Manager interface {
	// SetContainerStartup updates the container status with the given startup
	// and triggers a status update.
	SetContainerStartup(podId string, containerId string, started bool, result results.ProbeResult, pod results.IPod) error
}

type manager struct{}

func NewManager() Manager {
	return &manager{}
}

func (m *manager) SetContainerStartup(podId string, containerId string, started bool, result results.ProbeResult, pod results.IPod) error {
	status := computeapi.CONTAINER_STATUS_PROBE_FAILED
	if started {
		status = computeapi.CONTAINER_STATUS_RUNNING
	} else {
		if result.IsNetFailedError() && pod.IsRunning() {
			status = computeapi.CONTAINER_STATUS_NET_FAILED
		}
	}
	input := &computeapi.ContainerPerformStatusInput{
		PerformStatusInput: apis.PerformStatusInput{
			Status: status,
			Reason: result.Reason,
		},
	}
	if _, err := hostutils.UpdateContainerStatus(context.Background(), containerId, input); err != nil {
		err = errors.Wrapf(err, "set container(%s/%s) status failed, input: %s", podId, containerId, jsonutils.Marshal(input))
		log.Warningf(err.Error())
		errMsg := []string{
			"service is abnormal",
			"connection refused",
		}
		for _, msg := range errMsg {
			if strings.Contains(err.Error(), msg) {
				return errors.Wrap(err, "update container status")
			}
		}
	} else {
		log.Infof("set container(%s/%s) status to %s", podId, containerId, jsonutils.Marshal(input).String())
	}
	return nil
}
