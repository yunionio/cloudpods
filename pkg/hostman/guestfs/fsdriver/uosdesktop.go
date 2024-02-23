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

type SUOSDesktopRootFs struct {
	*SUbuntuRootFs
}

func NewUOSDesktopRootfs(part IDiskPartition) IRootFsDriver {
	return &SUOSDesktopRootFs{SUbuntuRootFs: NewUbuntuRootFs(part).(*SUbuntuRootFs)}
}

func (d *SUOSDesktopRootFs) GetName() string {
	return "UOSDesktop"
}

func (d *SUOSDesktopRootFs) String() string {
	return "UOSDesktopRootFs"
}

func (d *SUOSDesktopRootFs) RootSignatures() []string {
	sig := d.sDebianLikeRootFs.RootSignatures()
	return append([]string{"/etc/lsb-release", "/etc/deepin-version"}, sig...)
}

func (d *SUOSDesktopRootFs) GetReleaseInfo(rootFs IDiskPartition) *deployapi.ReleaseInfo {
	info := d.SUbuntuRootFs.GetReleaseInfo(rootFs)
	info.Distro = d.GetName()
	return info
}

func (d *SUOSDesktopRootFs) AllowAdminLogin() bool {
	return false
}
