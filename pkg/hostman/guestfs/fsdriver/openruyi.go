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

type SOpenRuyiRootFs struct {
	*sRedhatLikeRootFs
}

func NewOpenRuyiRootFs(part IDiskPartition) IRootFsDriver {
	return &SOpenRuyiRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SOpenRuyiRootFs) GetName() string {
	return "openRuyi"
}

func (d *SOpenRuyiRootFs) String() string {
	return "OpenRuyiRootFs"
}

func (d *SOpenRuyiRootFs) RootSignatures() []string {
	sig := d.sLinuxRootFs.RootSignatures()
	return append([]string{"/etc/os-release", "/usr/lib/rpm/openruyi"}, sig...)
}

func parseOpenRuyiVersion(osRelease string) string {
	for _, line := range strings.Split(osRelease, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VERSION_ID=") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "VERSION_ID=")), `"'`)
		}
	}
	return ""
}

func (d *SOpenRuyiRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/os-release", false)
	return deployapi.NewReleaseInfo(d.GetName(), parseOpenRuyiVersion(string(rel)), d.GetArch(rootFs))
}

func (d *SOpenRuyiRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	return d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, d.GetReleaseInfo(rootFs))
}

func (d *SOpenRuyiRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return d.enableSerialConsoleSystemd(rootFs)
}

func (d *SOpenRuyiRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	d.disableSerialConsoleSystemd(rootFs)
	return nil
}
