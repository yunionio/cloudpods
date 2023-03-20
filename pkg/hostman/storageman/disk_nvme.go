package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/qemuimgfmt"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
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
	//desc.Set("disk_size", jsonutils.NewInt(qemuImg.SizeBytes/1024/1024))
	desc.Set("format", jsonutils.NewString(string(qemuimgfmt.RAW)))
	desc.Set("disk_path", jsonutils.NewString(d.Storage.GetPath()))
	return desc
}

func (d *SNVMEDisk) DeleteAllSnapshot(skipRecycle bool) error {
	return errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) PrepareMigrate(liveMigrate bool) (string, error) {
	return "", errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	return errors.Errorf("unsupported operation")
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

func (d *SNVMEDisk) PostCreateFromImageFuse() {
}

func (d *SNVMEDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	return errors.Errorf("unsupported operation")
}

func (d *SNVMEDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	return errors.Errorf("unsupported operation")
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
