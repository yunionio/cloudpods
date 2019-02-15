package fsdriver

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
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
	for k, _ := range prop {
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

func (m *SAndroidRootFs) GetLoginAccount(rootFs IDiskPartition, defaultRootUser bool, windowsDefaultAdminUser bool) string {
	return ""
}

func (m *SAndroidRootFs) DeployPublicKey(rootfs IDiskPartition, uname string, pubkeys *sshkeys.SSHKeys) error {
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

func (m *SAndroidRootFs) GetReleaseInfo(IDiskPartition) *SReleaseInfo {
	spath := "/android-*/system/build.prop"
	lines, _ := m.rootFs.FileGetContents(spath, false)
	prop := ParsePropStr(string(lines))
	distro, _ := prop["ro.product.model"]
	version, _ := prop["ro.build.version.release"]
	arch, _ := prop["ro.product.cpu.abi"]
	return &SReleaseInfo{
		Distro:  distro,
		Version: version,
		Arch:    arch,
	}
}

func (m *SAndroidRootFs) PrepareFsForTemplate(IDiskPartition) error {
	return nil
}

func (m *SAndroidRootFs) DeployNetworkingScripts(rootfs IDiskPartition, nics []jsonutils.JSONObject) error {
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
