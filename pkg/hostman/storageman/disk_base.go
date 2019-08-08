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

	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
)

type IDisk interface {
	GetType() string
	GetId() string
	Probe() error
	GetPath() string
	GetSnapshotDir() string
	GetDiskDesc() jsonutils.JSONObject
	GetDiskSetupScripts(idx int) string
	GetSnapshotLocation() string

	DeleteAllSnapshot() error
	DiskSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	DiskDeleteSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)
	CleanupSnapshots(ctx context.Context, params interface{}) (jsonutils.JSONObject, error)

	PrepareMigrate(liveMigrate bool) (string, error)
	CreateFromUrl(ctx context.Context, url string, size int64) error
	CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error)
	CreateFromSnapshotLocation(ctx context.Context, location string, size int64) error
	CreateFromRbdSnapshot(ctx context.Context, snapshotId, srcDiskId, srcPool string) error
	CreateFromImageFuse(ctx context.Context, url string, size int64) error
	CreateRaw(ctx context.Context, sizeMb int, diskFromat string, fsFormat string,
		encryption bool, diskId string, back string) (jsonutils.JSONObject, error)
	PostCreateFromImageFuse()
	CreateSnapshot(snapshotId string) error
	DeleteSnapshot(snapshotId, convertSnapshot string, pendingDelete bool) error
	DeployGuestFs(diskPath string, guestDesc *jsonutils.JSONDict,
		deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error)
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

func (d *SBaseDisk) CreateFromUrl(ctx context.Context, url string, size int64) error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromTemplate(context.Context, string, string, int64) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) CreateFromSnapshotLocation(ctx context.Context, location string, size int64) error {
	return fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) Resize(context.Context, interface{}) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("Not implemented")
}

func (d *SBaseDisk) GetZone() string {
	return d.Storage.GetZone()
}

func (d *SBaseDisk) DeployGuestFs(diskPath string, guestDesc *jsonutils.JSONDict,
	deployInfo *deployapi.DeployInfo) (jsonutils.JSONObject, error) {
	deployGuestDesc, err := deployapi.GuestDescToDeployDesc(guestDesc)
	if err != nil {
		return nil, errors.Wrap(err, "guest desc to deploy desc")
	}
	ret, err := deployclient.GetDeployClient().DeployGuestFs(
		context.Background(), &deployapi.DeployParams{
			DiskPath:   diskPath,
			GuestDesc:  deployGuestDesc,
			DeployInfo: deployInfo,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "request deploy guest fs")
	}
	return jsonutils.Marshal(ret), nil
}

func (d *SBaseDisk) ResizeFs(diskPath string) error {
	_, err := deployclient.GetDeployClient().ResizeFs(
		context.Background(), &deployapi.ResizeFsParams{DiskPath: diskPath})
	return err
}

func (d *SBaseDisk) GetDiskSetupScripts(diskIndex int) string {
	return ""
}

func (d *SBaseDisk) GetSnapshotLocation() string {
	return ""
}

func (d *SBaseDisk) FormatFs(fsFormat, uuid, diskPath string) {
	log.Infof("Make disk %s fs %s", uuid, fsFormat)
	_, err := deployclient.GetDeployClient().FormatFs(
		context.Background(),
		&deployapi.FormatFsParams{
			DiskPath: diskPath,
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
