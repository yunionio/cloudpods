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
	"encoding/json"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/typeurl"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

// UpdateContainerDevices 通过 containerd API 更新 container spec 中的 devices
// 用于 cgroup v2 场景
func UpdateContainerDevices(ctx context.Context, client *containerd.Client, containerID string, devices []*runtimespec.LinuxDeviceCgroup) error {
	if len(devices) == 0 {
		return nil
	}

	log.Infof("[UpdateContainerDevices] Updating container %s devices via containerd API: %v", containerID, devices)

	// 获取 container
	container, err := client.LoadContainer(ctx, containerID)
	if err != nil {
		return errors.Wrapf(err, "load container %s", containerID)
	}
	cStatus := ContainerStatus(ctx, container)
	if cStatus == ProcessStatusRestarting {
		return errors.Errorf("container %q is in restarting state", containerID)
	}
	// 注意：我们允许在 created、stopped 等状态下更新 spec
	spec, err := container.Spec(ctx)
	if err != nil {
		return errors.Wrapf(err, "get container %s spec", containerID)
	}
	oldSpec, err := copySpec(spec)
	if err != nil {
		return errors.Wrapf(err, "copy container %s spec", containerID)
	}
	log.Infof("======oldSpec: %s, try update devices: %s", jsonutils.Marshal(oldSpec).PrettyString(), jsonutils.Marshal(devices).PrettyString())

	// 确保 Linux.Resources 存在
	if spec.Linux == nil {
		spec.Linux = &runtimespec.Linux{}
	}
	if spec.Linux.Resources == nil {
		spec.Linux.Resources = &runtimespec.LinuxResources{}
	}
	if spec.Linux.Resources.BlockIO == nil {
		spec.Linux.Resources.BlockIO = &runtimespec.LinuxBlockIO{}
	}
	if spec.Linux.Resources.CPU == nil {
		spec.Linux.Resources.CPU = &runtimespec.LinuxCPU{}
	}
	if spec.Linux.Resources.Memory == nil {
		spec.Linux.Resources.Memory = &runtimespec.LinuxMemory{}
	}
	if spec.Linux.Resources.Pids == nil {
		spec.Linux.Resources.Pids = &runtimespec.LinuxPids{}
	}
	if spec.Linux.Resources.HugepageLimits == nil {
		spec.Linux.Resources.HugepageLimits = []runtimespec.LinuxHugepageLimit{}
	}
	if spec.Linux.Resources.Network == nil {
		spec.Linux.Resources.Network = &runtimespec.LinuxNetwork{}
	}
	if spec.Linux.Resources.Rdma == nil {
		spec.Linux.Resources.Rdma = map[string]runtimespec.LinuxRdma{}
	}
	if spec.Linux.Resources.Unified == nil {
		spec.Linux.Resources.Unified = map[string]string{}
	}
	if spec.Linux.Resources.Devices == nil {
		spec.Linux.Resources.Devices = []runtimespec.LinuxDeviceCgroup{}
	}

	// 更新 devices
	// spec.Linux.Resources.Devices = devices
	for i := range devices {
		dev := devices[i]
		spec.Linux.Resources.Devices = append(spec.Linux.Resources.Devices, *dev)
	}

	if err := updateContainerSpec(ctx, container, spec); err != nil {
		return errors.Wrapf(err, "update container %s spec", containerID)
	}

	log.Infof("[UpdateContainerDevices] Successfully updated container %s devices via containerd API", containerID)
	return nil
}

func updateContainerSpec(ctx context.Context, container containerd.Container, spec *runtimespec.Spec) error {
	if err := container.Update(ctx, func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		a, err := typeurl.MarshalAny(spec)
		if err != nil {
			return errors.Wrapf(err, "marshal container %s spec", container.ID())
		}
		c.Spec = a
		return nil
	}); err != nil {
		return errors.Wrapf(err, "update container %s spec", container.ID())
	}
	return nil
}

// DeepCopy makes a deep copy from src into dst.
func DeepCopy(dst interface{}, src interface{}) error {
	if dst == nil {
		return errors.Errorf("dst cannot be nil")
	}
	if src == nil {
		return errors.Errorf("src cannot be nil")
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("unable to marshal src: %w", err)
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return fmt.Errorf("unable to unmarshal into dst: %w", err)
	}
	return nil
}

func copySpec(spec *runtimespec.Spec) (*runtimespec.Spec, error) {
	var copySpec runtimespec.Spec
	if err := DeepCopy(&copySpec, spec); err != nil {
		return nil, fmt.Errorf("failed to deep copy:%w", err)
	}
	return &copySpec, nil
}
