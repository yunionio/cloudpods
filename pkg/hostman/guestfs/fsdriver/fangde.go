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
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type SFangdeRootFs struct {
	*sRedhatLikeRootFs
}

func NewFangdeRootFs(part IDiskPartition) IRootFsDriver {
	return &SFangdeRootFs{sRedhatLikeRootFs: newRedhatLikeRootFs(part)}
}

func (d *SFangdeRootFs) GetName() string {
	return "Nfs"
}

func (d *SFangdeRootFs) String() string {
	return "NfsRootFs"
}

func (d *SFangdeRootFs) RootSignatures() []string {
	sig := d.sRedhatLikeRootFs.RootSignatures()
	return append([]string{"/etc/nfs-release", "/etc/centos-release"}, sig...)
}

func (d *SFangdeRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, _ := rootFs.FileGetContents("/etc/nfs-release", false)
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

func (d *SFangdeRootFs) DeployNetworkingScripts(rootFs IDiskPartition, nics []*types.SServerNic) error {
	relInfo := d.GetReleaseInfo(rootFs)
	if err := d.sRedhatLikeRootFs.deployNetworkingScripts(rootFs, nics, relInfo); err != nil {
		return err
	}
	return nil
}

func (c *SFangdeRootFs) EnableSerialConsole(rootFs IDiskPartition, sysInfo *jsonutils.JSONDict) error {
	return c.enableSerialConsole(c, rootFs, sysInfo)
}

func (c *SFangdeRootFs) DisableSerialConsole(rootFs IDiskPartition) error {
	return c.disableSerialConcole(c, rootFs)
}

type SFangdeDeskRootfs struct {
	*sDebianLikeRootFs
}

func NewFangdeDeskRootfs(part IDiskPartition) IRootFsDriver {
	driver := new(SFangdeDeskRootfs)
	driver.sDebianLikeRootFs = newDebianLikeRootFs(part)
	return driver
}

func (d *SFangdeDeskRootfs) GetName() string {
	return "Nfs"
}

func (d *SFangdeDeskRootfs) String() string {
	return "FangdeDeskRootfs"
}

func (d *SFangdeDeskRootfs) RootSignatures() []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{"/etc/lsb-release", "/etc/nfs/info"}, sig...)
}

func (d *SFangdeDeskRootfs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	rel, err := rootFs.FileGetContents("/etc/nfs/info", false)
	if err != nil {
		log.Errorf("SFangdeDeskRootfs open /etc/nfs/info fail %s", err)
		return nil
	}
	var distro, version string
	lines := strings.Split(string(rel), "\n")
	for _, l := range lines {
		parts := strings.Split(strings.TrimSpace(l), "=")
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if len(key) == 0 {
			continue
		}
		val := strings.TrimSpace(parts[1])
		if key == "NAME" {
			distro = strings.Trim(val, `'"`)
		} else if key == "VERSION" {
			version = strings.Trim(val, `'"`)
		}
	}
	return deployapi.NewReleaseInfo(distro, version, d.GetArch(rootFs))
}

func (d *SFangdeDeskRootfs) AllowAdminLogin() bool {
	return false
}
