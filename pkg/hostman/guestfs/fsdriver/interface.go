package fsdriver

import (
	"os"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/sshkeys"
)

type IDiskPartition interface {
	GetLocalPath(sPath string, caseInsensitive bool) string
	FileGetContents(sPath string, caseInsensitive bool) ([]byte, error)
	FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error
	Exists(sPath string, caseInsensitive bool) bool
	Chown(sPath string, uid, gid int, caseInsensitive bool) error
	Chmod(sPath string, mode uint32, caseInsensitive bool) error
	UserAdd(user string, caseInsensitive bool) error
	Stat(sPath string, caseInsensitive bool) os.FileInfo

	Passwd(account, password string, caseInsensitive bool) error
	Mkdir(sPath string, mode int, caseInsensitive bool) error
	ListDir(sPath string, caseInsensitive bool) []string
	Remove(path string, caseInsensitive bool)
	Cleandir(dir string, keepdir, caseInsensitive bool) error
	Zerofiles(dir string, caseInsensitive bool) error
	SupportSerialPorts() bool

	Mount() bool
	Umount() bool
}

type IRootFsDriver interface {
	GetPartition() IDiskPartition
	GetName() string

	IsFsCaseInsensitive() bool
	RootSignatures() []string
	RootExcludeSignatures() []string
	GetReleaseInfo(IDiskPartition) *SReleaseInfo
	GetOs() string
	DeployHostname(part IDiskPartition, hn, domain string) error
	DeployHosts(part IDiskPartition, hn, domain string, ips []string) error
	DeployNetworkingScripts(IDiskPartition, []jsonutils.JSONObject) error
	DeployStandbyNetworkingScripts(part IDiskPartition, nics, nicsStandby []jsonutils.JSONObject) error
	DeployUdevSubsystemScripts(IDiskPartition) error
	DeployFstabScripts(IDiskPartition, []jsonutils.JSONObject) error
	GetLoginAccount(IDiskPartition, bool, bool) string
	DeployPublicKey(IDiskPartition, string, *sshkeys.SSHKeys) error
	ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error)
	DeployYunionroot(IDiskPartition, *sshkeys.SSHKeys) error
	EnableSerialConsole(IDiskPartition, *jsonutils.JSONDict) error
	DisableSerialConsole(IDiskPartition) error
	CommitChanges(IDiskPartition) error

	PrepareFsForTemplate(IDiskPartition) error
}

type IDebianRootFsDriver interface {
	IRootFsDriver

	DistroName() string
	VersionFilePath() string
}
