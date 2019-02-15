package storageman

import (
	"context"
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
)

type IDisk interface {
	GetType() string
	GetId() string
	Probe() error
	GetPath() string
	GetSnapshotDir() string
	GetDiskDesc() jsonutils.JSONObject
	GetDiskSetupScripts(idx int) string

	DeleteAllSnapshot() error
	Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	PrepareMigrate(liveMigrate bool) (string, error)
	CreateFromUrl(context.Context, string) error
	CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error)
	CreateFromImageFuse(context.Context, string) error
	CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string,
		encryption bool, diskId string, back string) (jsonutils.JSONObject, error)
	PostCreateFromImageFuse()
	CreateSnapshot(snapshotId string) error
	DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error
	DeployGuestFs(diskPath string, guestDesc *jsonutils.JSONDict,
		deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error)
}

type SBaseDisk struct {
	Id      string
	Storage IStorage
}

func NewBaseDisk(storage IStorage, id string) *SBaseDisk {
	var ret = new(SBaseDisk)
	ret.Storage = storage
	ret.Id = id
	return ret
}

func (d *SBaseDisk) GetId() string {
	return d.Id
}

func (d *SBaseDisk) GetPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SBaseDisk) Probe() error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromUrl(context.Context, string) error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) Resize(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) GetZone() string {
	return d.Storage.GetZone()
}

func (d *SBaseDisk) DeployGuestFs(diskPath string, guestDesc *jsonutils.JSONDict,
	deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	var kvmDisk = NewKVMGuestDisk(diskPath)
	if kvmDisk.Connect() {
		defer kvmDisk.Disconnect()
		log.Infof("Kvm Disk Connect Success !!")

		if root := kvmDisk.MountKvmRootfs(); root != nil {
			defer kvmDisk.UmountKvmRootfs(root)
			return guestfs.DeployGuestFs(root, guestDesc, deployInfo)
		} else {
			return nil, fmt.Errorf("Kvm Disk Mount error")
		}
	} else {
		return nil, fmt.Errorf("Kvm disk connecterror")
	}
}

func (d *SBaseDisk) GetDiskSetupScripts(diskIndex int) string {
	return ""
}
