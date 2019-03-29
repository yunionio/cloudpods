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

	defer kvmDisk.Disconnect()
	if !kvmDisk.Connect() {
		log.Infof("Failed to connect kvm disk")
		return nil, nil
	}

	root := kvmDisk.MountKvmRootfs()
	if root == nil {
		log.Infof("Failed mounting rootfs for kvm disk")
		return nil, nil
	}
	defer kvmDisk.UmountKvmRootfs(root)

	return guestfs.DeployGuestFs(root, guestDesc, deployInfo)
}

func (d *SBaseDisk) GetDiskSetupScripts(diskIndex int) string {
	return ""
}
