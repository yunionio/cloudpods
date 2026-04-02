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

package fsdriver

import (
	"strings"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type CTyunOSRootFs struct {
	*SOpenEulerRootFs
}

func NewCTyunOSRootFs(part IDiskPartition) IRootFsDriver {
	return &CTyunOSRootFs{
		SOpenEulerRootFs: NewOpenEulerRootFs(part).(*SOpenEulerRootFs),
	}
}

func (c *CTyunOSRootFs) String() string {
	return "CTyunOSRootFs"
}

func (c *CTyunOSRootFs) GetName() string {
	return "CTyunOS"
}

func (c *CTyunOSRootFs) RootSignatures() []string {
	sig := c.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/ctyunos-release"}, sig...)
}

func (c *CTyunOSRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/os-release", false)
	var version string
	if len(rel) > 0 {
		for _, line := range strings.Split(string(rel), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "VERSION=") {
				version = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "VERSION=")), `"`)
				break
			}
		}
	}
	version = strings.TrimSpace(version)
	return deployapi.NewReleaseInfo(c.GetName(), version, c.GetArch(rootFs))
}

func (c *CTyunOSRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := c.GetReleaseInfo(rootFs)
	if err := c.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}
