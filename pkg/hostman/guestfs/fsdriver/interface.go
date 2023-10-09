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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/object"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

type IDiskPartition interface {
	// GetLocalPath join mountpoint as full path
	GetLocalPath(sPath string, caseInsensitive bool) string
	// FileGetContents will get file contents by join mountpoint as full path
	FileGetContents(sPath string, caseInsensitive bool) ([]byte, error)
	// FileGetContentsByPath get file contents directly
	FileGetContentsByPath(sPath string) ([]byte, error)
	FilePutContents(sPath, content string, modAppend, caseInsensitive bool) error
	Exists(sPath string, caseInsensitive bool) bool
	Chown(sPath string, uid, gid int, caseInsensitive bool) error
	Chmod(sPath string, mode uint32, caseInsensitive bool) error
	CheckOrAddUser(user, homeDir string, isSys bool) (realHomeDir string, err error)
	Stat(sPath string, caseInsensitive bool) os.FileInfo
	Symlink(src, dst string, caseInsensitive bool) error

	Passwd(account, password string, caseInsensitive bool) error
	Mkdir(sPath string, mode int, caseInsensitive bool) error
	ListDir(sPath string, caseInsensitive bool) []string
	Remove(path string, caseInsensitive bool)
	Cleandir(dir string, keepdir, caseInsensitive bool) error
	Zerofiles(dir string, caseInsensitive bool) error
	SupportSerialPorts() bool
	//Copy(src, dest string) error

	GetPartDev() string
	IsMounted() bool
	Mount() bool
	MountPartReadOnly() bool
	Umount() error
	GetMountPath() string
	IsReadonly() bool
	GetPhysicalPartitionType() string
	Zerofree()

	GenerateSshHostKeys() error
}

type IRootFsDriver interface {
	object.IObject

	GetPartition() IDiskPartition
	GetName() string
	String() string

	IsFsCaseInsensitive() bool
	RootSignatures() []string
	RootExcludeSignatures() []string
	GetReleaseInfo(IDiskPartition) *deployapi.ReleaseInfo
	GetOs() string
	DeployHostname(part IDiskPartition, hn, domain string) error
	DeployHosts(part IDiskPartition, hn, domain string, ips []string) error
	DeployQgaBlackList(part IDiskPartition) error
	DeployNetworkingScripts(IDiskPartition, []*types.SServerNic) error
	DeployStandbyNetworkingScripts(part IDiskPartition, nics, nicsStandby []*types.SServerNic) error
	DeployUdevSubsystemScripts(IDiskPartition) error
	DeployFstabScripts(IDiskPartition, []*deployapi.Disk) error
	GetLoginAccount(IDiskPartition, string, bool, bool) (string, error)
	DeployPublicKey(IDiskPartition, string, *deployapi.SSHKeys) error
	ChangeUserPasswd(part IDiskPartition, account, gid, publicKey, password string) (string, error)
	DeployYunionroot(rootFs IDiskPartition, pubkeys *deployapi.SSHKeys, isInit bool, enableCloudInit bool) error
	EnableSerialConsole(IDiskPartition, *jsonutils.JSONDict) error
	DisableSerialConsole(IDiskPartition) error
	CommitChanges(IDiskPartition) error
	DeployFiles(deploys []*deployapi.DeployContent) error
	DeployUserData(userData string) error
	DeployTelegraf(config string) (bool, error)
	DetectIsUEFISupport(IDiskPartition) bool
	IsCloudinitInstall() bool
	IsResizeFsPartitionSupport() bool

	PrepareFsForTemplate(IDiskPartition) error
	CleanNetworkScripts(rootFs IDiskPartition) error

	AllowAdminLogin() bool
}

type IDebianRootFsDriver interface {
	IRootFsDriver

	DistroName() string
	VersionFilePath() string
}
