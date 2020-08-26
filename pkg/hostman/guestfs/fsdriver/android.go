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
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

func ParsePropStr(lines string) map[string]string {
	ret := map[string]string{}
	for _, l := range strings.Split(lines, "\n") {
		if len(l) > 0 && l[0] != '#' {
			pos := strings.Index(l, "=")
			if pos > 0 {
				key := strings.TrimSpace(l[:pos])
				val := strings.TrimSpace(l[pos+1:])
				ret[key] = val
			}
		}
	}
	return ret
}

func BuildPropStr(prop map[string]string) string {
	keys := []string{}
	for k := range prop {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := ""
	for _, k := range keys {
		ret += fmt.Sprintf("%s=%s\n", k, prop[k])
	}
	return ret
}

type SAndroidRootFs struct {
	*sGuestRootFsDriver
}

func NewAndroidRootFs(part IDiskPartition) IRootFsDriver {
	return &SAndroidRootFs{sGuestRootFsDriver: newGuestRootFsDriver(part)}
}

func (m *SAndroidRootFs) IsFsCaseInsensitive() bool {
	return false
}

func (m *SAndroidRootFs) GetName() string {
	return "Android"
}

func (m *SAndroidRootFs) String() string {
	return "AndroidRootFs"
}

func (m *SAndroidRootFs) RootSignatures() []string {
	return []string{
		"/android-*", "/grub",
	}
}

func (m *SAndroidRootFs) GetLoginAccount(rootFs IDiskPartition, user string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	return "", nil
}

func (m *SAndroidRootFs) DeployPublicKey(rootfs IDiskPartition, uname string, pubkeys *deployapi.SSHKeys) error {
	return nil
}

func (m *SAndroidRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	return "", nil
}

func (m *SAndroidRootFs) DeployHostname(part IDiskPartition, hostname, domain string) error {
	return nil
}

func (m *SAndroidRootFs) DeployHosts(part IDiskPartition, hn, domain string, ips []string) error {
	return nil
}

func (m *SAndroidRootFs) GetOs() string {
	return "Android"
}

func (m *SAndroidRootFs) GetReleaseInfo(IDiskPartition) *deployapi.ReleaseInfo {
	spath := "/android-*/system/build.prop"
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	distro, _ := prop["ro.product.model"]
	version, _ := prop["ro.build.version.release"]
	arch, _ := prop["ro.product.cpu.abi"]
	return &deployapi.ReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}

func (m *SAndroidRootFs) PrepareFsForTemplate(IDiskPartition) error {
	return nil
}

func (m *SAndroidRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []*types.SServerNic) error {
	return nil
}

func (m *SAndroidRootFs) CommitChanges(part IDiskPartition) error {
	spath := "/android-*/system/build.prop"
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	prop["ro.setupwizard.mode"] = "DISABLED"
	prop["persist.sys.timezone"] = "Asia/Shanghai"
	return m.rootFs.FilePutContents(spath, BuildPropStr(prop), false, false)
}
