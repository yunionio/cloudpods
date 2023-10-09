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

type SNeoKylinRootFs struct {
	*sRedhatLikeRootFs
}

func NewNeoKylinRootFs(part IDiskPartition) IRootFsDriver {
	return &SNeoKylinRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SNeoKylinRootFs) GetName() string {
	return "NeoKylin"
}

func (d *SNeoKylinRootFs) String() string {
	return "NeoKylinRootFs"
}

func (d *SNeoKylinRootFs) RootSignatures() []string {
	sigs := []string{"/etc/neokylin-release", "/etc/.productinfo"}
	for _, sig := range d.sRedhatLikeRootFs.RootSignatures() {
		if sig != "/etc/redhat-release" {
			sigs = append(sigs, sig)
		}
	}
	return sigs
}

func (d *SNeoKylinRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/neokylin-release", false)
	var version string
	if len(rel) > 0 {
		relStr := string(rel)
		endPos := strings.IndexByte(relStr, '(')
		if endPos > 0 {
			relStr = strings.TrimSpace(relStr[:endPos])
		}
		dat := strings.Split(relStr, " ")
		version = dat[len(dat)-1]
	}
	return deployapi.NewReleaseInfo(d.GetName(), version, d.GetArch(rootFs))
}

func (d *SNeoKylinRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := d.GetReleaseInfo(rootFs)
	if err := d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

func (c *SNeoKylinRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsoleSystemd(rootFs)
}

func (c *SNeoKylinRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	c.disableSerialConsoleSystemd(rootFs)
	return nil
}

func (c *SNeoKylinRootFs) AllowAdminLogin() bool {
	return false
}
