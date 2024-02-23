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
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type SOpenKylinRootFs struct {
	*SUbuntuRootFs
}

func NewOpenKylinRootfs(part IDiskPartition) IRootFsDriver {
	return &SOpenKylinRootFs{SUbuntuRootFs: NewUbuntuRootFs(part).(*SUbuntuRootFs)}
}

func (d *SOpenKylinRootFs) GetName() string {
	return "OpenKylin"
}

func (d *SOpenKylinRootFs) String() string {
	return "OpenKylinRootFs"
}

func (d *SOpenKylinRootFs) RootSignatures() []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{"/etc/lsb-release", "/etc/kylin-build", "/etc/ukui-tablet-desktop.conf"}, sig...)
}

func (d *SOpenKylinRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	info := d.SUbuntuRootFs.GetReleaseInfo(rootFs)
	info.Distro = d.GetName()
	return info
}

func (d *SOpenKylinRootFs) AllowAdminLogin() bool {
	return false
}
