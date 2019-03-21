package guestman

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

type SBaseParms struct {
	Sid  string
	Body jsonutils.JSONObject
}

type SGuestDeploy struct {
	Sid    string
	Body   jsonutils.JSONObject
	IsInit bool
}

type SSrcPrepareMigrate struct {
	Sid         string
	LiveMigrate bool
}

type SDestPrepareMigrate struct {
	Sid             string
	ServerUrl       string
	QemuVersion     string
	SnapshotsUri    string
	DisksUri        string
	TargetStorageId string
	LiveMigrate     bool

	Desc             jsonutils.JSONObject
	DisksBackingFile jsonutils.JSONObject
	SrcSnapshots     jsonutils.JSONObject
}

type SLiveMigrate struct {
	Sid      string
	DestPort int
	DestIp   string
	IsLocal  bool
}

type SDriverMirror struct {
	Sid          string
	NbdServerUri string
	Desc         jsonutils.JSONObject
}

type SGuestHotplugCpuMem struct {
	Sid         string
	AddCpuCount int64
	AddMemSize  int64
}

type SReloadDisk struct {
	Sid  string
	Disk storageman.IDisk
}

type SDiskSnapshot struct {
	Sid        string
	SnapshotId string
	Disk       storageman.IDisk
}

type SDeleteDiskSnapshot struct {
	Sid             string
	DeleteSnapshot  string
	Disk            storageman.IDisk
	ConvertSnapshot string
	PendingDelete   bool
}

type SLibvirtServer struct {
	Uuid  string
	MacIp map[string]string
}

type SLibvirtDomainImportConfig struct {
	LibvritDomainXmlDir string
	Servers             []SLibvirtServer
}

type SGuestCreateFromLibvirt struct {
	Sid       string
	GuestDesc *jsonutils.JSONDict
	DisksPath *jsonutils.JSONDict
}

// type SGuestImportConfig struct {
// 	Uuid     string
// 	Name     string
// 	CpuCount int
// 	MemSize  int //MB
// 	Disks    []SImportDiskConfig
// 	Nics     []SImportNicConfig
// }

// type SImportDiskConfig struct {
// 	Index  int
// 	Id     string
// 	Format string
// 	Path   string

// 	// virtio or scsi
// 	Driver string
// 	Size   int // MB
// }

// type SImportNicConfig struct {
// 	Ip  string
// 	Mac string
// }
