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
	GetId() string
	Probe() bool

	GetDiskDesc() jsonutils.JSONObject

	// TODO
	// DeleteAllSnapshot() error
	Delete() error

	GetPath() string
	CreateFromUrl(context.Context, string) error
	// CreateFromSnapshot
	CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error)
	CreateFromImageFuse(context.Context, string) error
	CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string,
		encryption bool, diskId string, back string) (jsonutils.JSONObject, error)

	Resize(context.Context, int64) error

	// @params: diskPath, guestDesc, deployInfo
	DeployGuestFs(string, *jsonutils.JSONDict, *guestfs.SDeployInfo) (jsonutils.JSONObject, error)
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

func (d *SBaseDisk) Delete() error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromUrl(context.Context, string) error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) Resize(context.Context, int64) error {
	return fmt.Errorf("Not implemented")
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

		if root := kvmDisk.Mount(); root != nil {
			defer kvmDisk.Umount(root)
			return root.DeployGuestFs(root, guestDesc, deployInfo)
		}
	}
	return nil, fmt.Errorf("Kvm disk connect or mount error")
}
