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

package containerd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/runtime/restart"
	"github.com/containerd/typeurl"
	"github.com/opencontainers/runtime-spec/specs-go"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func init() {
	const prefix = "types.containerd.io"
	// register TypeUrls for commonly marshaled external types
	major := strconv.Itoa(specs.VersionMajor)
	typeurl.Register(&specs.Spec{}, prefix, "opencontainers/runtime-spec", major, "Spec")
	typeurl.Register(&specs.Process{}, prefix, "opencontainers/runtime-spec", major, "Process")
	typeurl.Register(&specs.LinuxResources{}, prefix, "opencontainers/runtime-spec", major, "LinuxResources")
	typeurl.Register(&specs.WindowsResources{}, prefix, "opencontainers/runtime-spec", major, "WindowsResources")
}

// NewClient 创建 containerd 客户端
// 参考 nerdctl 的实现方式，使用 containerd.Client
func NewClient(ctx context.Context, address, namespace string) (*containerd.Client, error) {
	// 使用 containerd.New 创建客户端，参考 nerdctl 实现
	client, err := containerd.New(address,
		containerd.WithDefaultNamespace(namespace),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "create containerd client at %s", address)
	}

	return client, nil
}

type ProcessStatus string

const (
	ProcessStatusCreated    ProcessStatus = "created"
	ProcessStatusRunning    ProcessStatus = "running"
	ProcessStatusStopped    ProcessStatus = "stopped"
	ProcessStatusUnknown    ProcessStatus = "unknown"
	ProcessStatusRestarting ProcessStatus = "restarting"
)

func ContainerStatus(ctx context.Context, c containerd.Container) ProcessStatus {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	task, err := c.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ProcessStatusCreated
		}
		log.Warningf("failed to get task for container %s: %v", c.ID(), err)
		return ProcessStatusUnknown
	}

	status, err := task.Status(ctx)
	if err != nil {
		log.Warningf("failed to get status for task %s: %v", task.ID(), err)
		return ProcessStatusUnknown
	}
	labels, err := c.Labels(ctx)
	if err != nil {
		log.Warningf("failed to get labels for container %s: %v", c.ID(), err)
		return ProcessStatusUnknown
	}

	switch s := status.Status; s {
	case containerd.Stopped:
		if labels[restart.StatusLabel] == string(containerd.Running) && restart.Reconcile(status, labels) {
			s := fmt.Sprintf("Restarting (%d) %v", status.ExitStatus, status.ExitTime)
			log.Infof("container %s is restarting: %s", c.ID(), s)
			return ProcessStatusRestarting
		}
		return ProcessStatusStopped
	case containerd.Running:
		return ProcessStatusRunning
	}
	return ProcessStatus(status.Status)
}
