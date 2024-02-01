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

package storageman

import (
	"context"
	"fmt"
	"path"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	container_storage "yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type IDisk interface {
	GetType() string
	GetId() string
	Probe() error
	GetPath() string
	GetFormat() (string, error)
	GetDiskDesc() jsonutils.JSONObject
	GetDiskSetupScripts(idx int) string
	OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error

	GetSnapshotDir() string
	DoDeleteSnapshot(snapshotId string) error
	GetSnapshotLocation() string

	GetStorage() IStorage

	DeleteAllSnapshot(skipRecycle bool) error
	DiskSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	DiskDeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	PrepareMigrate(liveMigrate bool) ([]string, string, bool, error)
	CreateFromUrl(ctx context.Context, url string, size int64, callback func(progress, progressMbps float64, totalSizeMb int64)) error
	CreateFromTemplate(context.Context, string, string, int64, *apis.SEncryptInfo) (jsonutils.JSONObject, error)
	CreateFromSnapshotLocation(ctx context.Context, location string, size int64, encryptInfo *apis.SEncryptInfo) error
	CreateFromRbdSnapshot(ctx context.Context, snapshotId, srcDiskId, srcPool string) error
	CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error
	CreateRaw(ctx context.Context, sizeMb int, diskFormat string, fsFormat string,
		encryptInfo *apis.SEncryptInfo, diskId string, back string) (jsonutils.JSONObject, error)
	PostCreateFromImageFuse()
	CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error
	DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error
	DeployGuestFs(diskInfo *deployapi.DiskInfo, guestDesc *desc.SGuestDesc,
		deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error)

	// GetBackupDir() string
	DiskBackup(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	IsFile() bool

	GetContainerStorageDriver() (container_storage.IContainerStorage, error)
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

func (d *SBaseDisk) GetStorage() IStorage {
	return d.Storage
}

func (d *SBaseDisk) GetPath() string {
	return path.Join(d.Storage.GetPath(), d.Id)
}

func (d *SBaseDisk) GetFormat() (string, error) {
	return "", nil
}

func (d *SBaseDisk) OnRebuildRoot(ctx context.Context, params api.DiskAllocateInput) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CreateFromUrl(ctx context.Context, url string, size int64, callback func(progress, progressMbps float64, totalSizeMb int64)) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CreateFromTemplate(context.Context, string, string, int64, *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CreateFromSnapshotLocation(ctx context.Context, location string, size int64, encryptInfo *apis.SEncryptInfo) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CreateFromImageFuse(ctx context.Context, url string, size int64, encryptInfo *apis.SEncryptInfo) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) Resize(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) DeleteAllSnapshot(skipRecycle bool) error {
	return errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) PrepareMigrate(liveMigrate bool) ([]string, string, bool, error) {
	return nil, "", false, errors.Errorf("unsupported operation")
}

func (d *SBaseDisk) PostCreateFromImageFuse() {
}

func (d *SBaseDisk) GetZoneId() string {
	return d.Storage.GetZoneId()
}

func (d *SBaseDisk) DeployGuestFs(diskInfo *deployapi.DiskInfo, guestDesc *desc.SGuestDesc,
	deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	deployGuestDesc := deployapi.GuestStructDescToDeployDesc(guestDesc)
	deployGuestDesc.Hypervisor = api.HYPERVISOR_KVM
	ret, err := deployclient.GetDeployClient().DeployGuestFs(
		context.Background(), &deployapi.DeployParams{
			DiskInfo:   diskInfo,
			GuestDesc:  deployGuestDesc,
			DeployInfo: deployInfo,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "request deploy guest fs")
	}
	return jsonutils.Marshal(ret), nil
}

func (d *SBaseDisk) ResizeFs(diskInfo *deployapi.DiskInfo) error {
	_, err := deployclient.GetDeployClient().ResizeFs(
		context.Background(), &deployapi.ResizeFsParams{DiskInfo: diskInfo})
	return err
}

func (d *SBaseDisk) GetDiskSetupScripts(diskIndex int) string {
	return ""
}

func (d *SBaseDisk) GetSnapshotLocation() string {
	return ""
}

func (d *SBaseDisk) FormatFs(fsFormat, uuid string, diskInfo *deployapi.DiskInfo) {
	log.Infof("Make disk %s fs %s", uuid, fsFormat)
	_, err := deployclient.GetDeployClient().FormatFs(
		context.Background(),
		&deployapi.FormatFsParams{
			DiskInfo: diskInfo,
			FsFormat: fsFormat,
			Uuid:     uuid,
		},
	)
	if err != nil {
		log.Errorf("Format fs error : %s", err)
	}
}

func (d *SBaseDisk) DiskSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implement disk.DiskSnapshot")
}

func (d *SBaseDisk) DiskDeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implement disk.DiskDeleteSnapshot")
}

func (d *SBaseDisk) CreateFromRbdSnapshot(ctx context.Context, napshotUrl, srcDiskId, srcPool string) error {
	return fmt.Errorf("Not implement disk.CreateFromRbdSnapshot")
}

func (d *SBaseDisk) DoDeleteSnapshot(snapshotId string) error {
	return fmt.Errorf("Not implement disk.DoDeleteSnapshot")
}

func (d *SBaseDisk) GetBackupDir() string {
	return ""
}

func (d *SBaseDisk) GetContainerStorageDriver() (container_storage.IContainerStorage, error) {
	return nil, errors.Wrap(errors.ErrNotImplemented, "GetContainerStorageDriver")
}
