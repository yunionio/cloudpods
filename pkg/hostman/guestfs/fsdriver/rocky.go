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

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type SRockyRootFs struct {
	*sRedhatLikeRootFs
}

func NewRockyRootFs(part IDiskPartition) IRootFsDriver {
	return &SRockyRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SRockyRootFs) GetName() string {
	return "Rocky"
}

func (d *SRockyRootFs) String() string {
	return "RockyRootFs"
}

func (d *SRockyRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/sysconfig/network", "/etc/rocky-release"}, sig...)
}

func (d *SRockyRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/rocky-release", false)
	var version string
	if len(rel) > 0 {
		dat := strings.Split(string(rel), " ")
		if len(dat) > 3 {
			version = dat[3]
		}
	}
	return deployapi.NewReleaseInfo(d.GetName(), version, d.GetArch(rootFs))
}

func (d *SRockyRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := d.GetReleaseInfo(rootFs)
	if err := d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

func (c *SRockyRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsoleSystemd(rootFs)
}

func (c *SRockyRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	c.disableSerialConsoleSystemd(rootFs)
	return nil
}
