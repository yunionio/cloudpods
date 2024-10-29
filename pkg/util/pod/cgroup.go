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

package pod

import (
	"fmt"
	"path/filepath"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CGROUP_PATH_SYSFS = "/sys/fs/cgroup"
)

type CgroupUtil interface {
	SetMemoryLimitBytes(ctrId string, bytes int64) error
	SetCPUCfs(ctrId string, quota int64, period int64) error
	SetDevicesAllow(ctrId string, allows []string) error
	SetPidsMax(ctrId string, max int) error
	SetCpusetCloneChildren(ctrId string) error
}

type podCgroupV1Util struct {
	parentPath string
}

func NewPodCgroupV1Util(parentPath string) CgroupUtil {
	return &podCgroupV1Util{
		parentPath: parentPath,
	}
}

func (p podCgroupV1Util) getContainerControllerPath(controller string, ctrId string) string {
	return filepath.Join(CGROUP_PATH_SYSFS, controller, p.parentPath, ctrId)
}

func (p podCgroupV1Util) getContainerCGFilePath(controller string, ctrId string, filename string) string {
	return filepath.Join(p.getContainerControllerPath(controller, ctrId), filename)
}

func (p podCgroupV1Util) write(fp string, content string) error {
	cmd := fmt.Sprintf("echo %q > %s", content, fp)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "%s: %s", cmd, out)
	}
	return nil
}

func (p podCgroupV1Util) SetMemoryLimitBytes(ctrId string, bytes int64) error {
	memFp := p.getContainerCGFilePath("memory", ctrId, "memory.limit_in_bytes")
	return p.write(memFp, fmt.Sprintf("%d", bytes))
}

func (p podCgroupV1Util) SetCPUCfs(ctrId string, quota int64, period int64) error {
	quotaFp := p.getContainerCGFilePath("cpu,cpuacct", ctrId, "cpu.cfs_quota_us")
	periodFp := p.getContainerCGFilePath("cpu,cpuacct", ctrId, "cpu.cfs_period_us")
	if err := p.write(quotaFp, fmt.Sprintf("%d", quota)); err != nil {
		return errors.Wrapf(err, "write quota: %d", quota)
	}
	if err := p.write(periodFp, fmt.Sprintf("%d", period)); err != nil {
		return errors.Wrapf(err, "write period: %d", period)
	}
	return nil
}

func (p podCgroupV1Util) SetDevicesAllow(ctrId string, allows []string) error {
	devicesFp := p.getContainerCGFilePath("devices", ctrId, "devices.allow")
	for _, allowStr := range allows {
		if err := p.write(devicesFp, allowStr); err != nil {
			return errors.Wrapf(err, "write: %s", allowStr)
		}
	}
	return nil
}

func (p podCgroupV1Util) SetPidsMax(ctrId string, max int) error {
	pidFp := p.getContainerCGFilePath("pids", ctrId, "pids.max")
	return p.write(pidFp, fmt.Sprintf("%d", max))
}

func (p podCgroupV1Util) SetCpusetCloneChildren(ctrId string) error {
	ccFp := p.getContainerCGFilePath("cpuset", ctrId, "cgroup.clone_children")
	return p.write(ccFp, "1")
}
