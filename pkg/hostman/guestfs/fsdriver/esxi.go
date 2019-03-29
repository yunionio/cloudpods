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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SEsxiRootFs struct {
	*sGuestRootFsDriver
}

func NewEsxiRootFs(part IDiskPartition) IRootFsDriver {
	return &SEsxiRootFs{sGuestRootFsDriver: newGuestRootFsDriver(part)}
}

func (m *SEsxiRootFs) IsFsCaseInsensitive() bool {
	return false
}

func (m *SEsxiRootFs) GetName() string {
	return "Esxi"
}

func (m *SEsxiRootFs) String() string {
	return "EsxiRootFs"
}

func (m *SEsxiRootFs) RootSignatures() []string {
	return []string{
		"/boot.cfg", "/imgdb.tgz",
	}
}

func (m *SEsxiRootFs) GetLoginAccount(rootFs IDiskPartition, defaultRootUser bool, windowsDefaultAdminUser bool) string {
	return "root"
}

func (m *SEsxiRootFs) GetOs() string {
	return "VMWare"
}

func (m *SEsxiRootFs) ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error) {
	return seclib2.EncryptBase64(gid, "(blank)")
}

func (m *SEsxiRootFs) DeployHostname(part IDiskPartition, hostname, domain string) error {
	return nil
}

func (m *SEsxiRootFs) DeployHosts(part IDiskPartition, hn, domain string, ips []string) error {
	return nil
}

func (m *SEsxiRootFs) GetReleaseInfo(IDiskPartition) *SReleaseInfo {
	spath := "/boot.cfg"
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	version, _ := prop["build"]
	return &SReleaseInfo{
		Distro:  "ESXi",
		Version: version,
		Arch:    "x86_64",
	}
}

func (m *SEsxiRootFs) DeployPublicKey(rootfs IDiskPartition, uname string, pubkeys *sshkeys.SSHKeys) error {
	return nil
}

func (m *SEsxiRootFs) PrepareFsForTemplate(IDiskPartition) error {
	return nil
}

func (m *SEsxiRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []jsonutils.JSONObject) error {
	return nil
}
