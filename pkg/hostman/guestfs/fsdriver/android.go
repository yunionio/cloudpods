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

type sBaseAndroidRootFs struct {
	*sGuestRootFsDriver

	rootDir    string
	distroKey  string
	versionKey string
}

func (m *sBaseAndroidRootFs) IsFsCaseInsensitive() bool {
	return false
}

func (m *sBaseAndroidRootFs) RootSignatures() []string {
	return []string{
		m.rootDir, "/grub",
	}
}

func (m *sBaseAndroidRootFs) GetLoginAccount(rootFs IDiskPartition, user string, defaultRootUser bool, windowsDefaultAdminUser bool) (string, error) {
	return "", nil
}

func (m *sBaseAndroidRootFs) DeployPublicKey(rootfs IDiskPartition, uname string, pubkeys *deployapi.SSHKeys) error {
	return nil
}

func (m *sBaseAndroidRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	return "", nil
}

func (m *sBaseAndroidRootFs) DeployHostname(part IDiskPartition, hostname, domain string) error {
	return nil
}

func (m *sBaseAndroidRootFs) DeployHosts(part IDiskPartition, hn, domain string, ips []string) error {
	return nil
}

func (m *sBaseAndroidRootFs) DeployQgaBlackList(part IDiskPartition) error {
	return nil
}

func (m *sBaseAndroidRootFs) GetOs() string {
	return "Android"
}

func (m *sBaseAndroidRootFs) GetReleaseInfo(IDiskPartition) *deployapi.ReleaseInfo {
	spath := fmt.Sprintf("/%s/system/build.prop", m.rootDir)
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	distro, _ := prop[m.distroKey]
	version, _ := prop[m.versionKey]
	arch, _ := prop["ro.product.cpu.abi"]
	return &deployapi.ReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}

func (m *sBaseAndroidRootFs) PrepareFsForTemplate(IDiskPartition) error {
	return nil
}

func (m *sBaseAndroidRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []*types.SServerNic) error {
	return nil
}

func (m *sBaseAndroidRootFs) CommitChanges(part IDiskPartition) error {
	spath := fmt.Sprintf("/%s/system/build.prop", m.rootDir)
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	prop["ro.setupwizard.mode"] = "DISABLED"
	prop["persist.sys.timezone"] = "Asia/Shanghai"
	return m.rootFs.FilePutContents(spath, BuildPropStr(prop), false, false)
}

type SAndroidRootFs struct {
	*sBaseAndroidRootFs
}

func NewAndroidRootFs(part IDiskPartition) IRootFsDriver {
	return &SAndroidRootFs{
		&sBaseAndroidRootFs{
			sGuestRootFsDriver: newGuestRootFsDriver(part),
			rootDir:            "android-*",
			distroKey:          "ro.product.model",
			versionKey:         "ro.build.version.release",
		},
	}
}

func (m *SAndroidRootFs) GetName() string {
	return "Android"
}

func (m *SAndroidRootFs) String() string {
	return "AndroidRootFs"
}

type SPhoenixOSRootFs struct {
	*sBaseAndroidRootFs
}

func NewPhoenixOSRootFs(part IDiskPartition) IRootFsDriver {
	return &SPhoenixOSRootFs{
		&sBaseAndroidRootFs{
			sGuestRootFsDriver: newGuestRootFsDriver(part),
			rootDir:            "PhoenixOS",
			distroKey:          "ro.phoenix.os.branch",
			versionKey:         "ro.phoenix.version.codename",
		},
	}
}

func (m *SPhoenixOSRootFs) GetName() string {
	return "PhoenixOS"
}

func (m *SPhoenixOSRootFs) String() string {
	return "PhoenixOSRootFs"
}
