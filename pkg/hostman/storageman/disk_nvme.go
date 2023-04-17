package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/onecloud/pkg/apis"
)

type SNVMEDisk struct {
	SBaseDisk
}

func (d *SNVMEDisk) GetSnapshotDir() string {
	return ""
}

func (d *SNVMEDisk) GetDiskDesc() jsonutils.JSONObject {
	var desc = jsonutils.NewDict()

	desc.Set("disk_id", jsonutils.NewString(d.Id))
	desc.Set("disk_size", jsonutils.NewInt(int64(d.Storage.GetCapacity())))
	desc.Set("format", jsonutils.NewString(string(qemuimgfmt.RAW)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SNVMEDisk) CreateRaw(
	ctx context.Context, sizeMb int, diskFormat string, fsFormat string,
	encryptInfo *apis.SEncryptInfo, diskId string, back string,
) (jsonutils.JSONObject, error) {
	// do nothing
	return nil, nil
}

func (s *SNVMEDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SNVMEDisk) IsFile() bool {
	return false
}

func (d *SNVMEDisk) Probe() error {
	return nil
}

func NewNVMEDisk(storage IStorage, id string) *SNVMEDisk {
	return &SNVMEDisk{
		SBaseDisk: *NewBaseDisk(storage, id),
	}
}
