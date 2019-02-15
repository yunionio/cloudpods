package storageman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/storagetypes"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
)

type SRBDDisk struct {
	SBaseDisk
}

func NewRBDDisk(storage IStorage, id string) *SRBDDisk {
	var ret = new(SRBDDisk)
	ret.SBaseDisk = *NewBaseDisk(storage, id)
	return ret
}

func (d *SRBDDisk) GetType() string {
	return storagetypes.STORAGE_RBD
}

func (d *SRBDDisk) Probe() error {
	return nil
}

func (d *SRBDDisk) GetPath() string {
	return ""
}

func (d *SRBDDisk) GetSnapshotDir() string {
	return ""
}

func (d *SRBDDisk) GetDiskDesc() jsonutils.JSONObject {
	return nil
}

func (d *SRBDDisk) GetDiskSetupScripts(idx int) string {
	return ""
}

func (d *SRBDDisk) DeleteAllSnapshot() error {
	return nil
}

func (d *SRBDDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) PrepareMigrate(liveMigrate bool) (string, error) {
	return "", nil
}

func (d *SRBDDisk) CreateFromUrl(context.Context, string) error {
	return nil
}

func (d *SRBDDisk) CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) CreateFromImageFuse(context.Context, string) error {
	return nil
}

func (d *SRBDDisk) CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string, encryption bool, diskId string, back string) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (d *SRBDDisk) PostCreateFromImageFuse() {

}

func (d *SRBDDisk) CreateSnapshot(snapshotId string) error {
	return nil
}

func (d *SRBDDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	return nil
}

func (d *SRBDDisk) DeployGuestFs(diskPath string, guestDesc *jsonutils.JSONDict, deployInfo *guestfs.SDeployInfo) (jsonutils.JSONObject, error) {
	return nil, nil
}
