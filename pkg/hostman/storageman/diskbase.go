package storageman

import (
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
)

type IDisk interface {
	GetId() string
	Probe() bool
	DeployGuestFs(guestDesc *jsonutils.JSONDict,
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

func (d *SBaseDisk) getPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SBaseDisk) DeployGuestFs(
	guestDesc *jsonutils.JSONDict,
	deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	var kvmDisk = NewKVMGuestDisk(d.getPath())
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
