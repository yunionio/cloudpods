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
