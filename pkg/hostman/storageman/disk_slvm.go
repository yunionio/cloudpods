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
	"path"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

// shared lvm
type SSLVMDisk struct {
	SLVMDisk
}

func NewSLVMDisk(storage IStorage, id string) *SSLVMDisk {
	return &SSLVMDisk{
		SLVMDisk: *NewLVMDisk(storage, id),
	}
}

func (d *SSLVMDisk) GetType() string {
	return api.STORAGE_SLVM
}

func (d *SSLVMDisk) GetPath() string {
	return path.Join("/dev", d.Storage.GetPath(), d.Id)
}

func (d *SSLVMDisk) Probe() error {
	if err := lvmutils.LvScan(); err != nil {
		return errors.Wrap(err, "lv scan")
	}

	var lvPath = d.GetPath()
	if err := lvmutils.LVActive(lvPath, d.Storage.Lvmlockd(), false); err != nil {
		return errors.Wrap(err, "lv active")
	}

	diskPath := d.GetPath()
	for diskPath != "" {
		qemuImg, err := qemuimg.NewQemuImage(diskPath)
		if err != nil {
			log.Errorln(err)
			return err
		}
		diskPath = qemuImg.BackFilePath
		if qemuImg.BackFilePath != "" {
			originActivated, err := lvmutils.LvIsActivated(qemuImg.BackFilePath)
			if err != nil {
				return errors.Wrap(err, "check lv is activated")
			}
			if !originActivated {
				if err = lvmutils.LVActive(qemuImg.BackFilePath, d.Storage.Lvmlockd(), false); err != nil {
					return errors.Wrap(err, "lv active origin")
				}
			}
		}
	}

	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}

func (d *SSLVMDisk) CreateRaw(ctx context.Context, sizeMb int, diskFormat string, fsFormat string, fsFeatures *api.DiskFsFeatures, encryptInfo *apis.SEncryptInfo, diskId string, back string) (jsonutils.JSONObject, error) {
	ret, err := d.SLVMDisk.CreateRaw(ctx, sizeMb, diskFormat, fsFormat, nil, encryptInfo, diskId, back)
	if err != nil {
		return ret, err
	}

	err = lvmutils.LVDeactivate(d.GetPath())
	if err != nil {
		return ret, errors.Wrap(err, "LVDeactivate")
	}
	return ret, nil
}

func (d *SSLVMDisk) CreateFromTemplate(
	ctx context.Context, imageId, format string, sizeMb int64, encryptInfo *apis.SEncryptInfo,
) (jsonutils.JSONObject, error) {
	ret, err := d.SLVMDisk.CreateFromTemplate(ctx, imageId, format, sizeMb, encryptInfo)
	if err != nil {
		return ret, err
	}
	err = lvmutils.LVDeactivate(d.GetPath())
	if err != nil {
		return ret, errors.Wrap(err, "LVDeactivate")
	}
	return ret, nil
}

func (d *SSLVMDisk) PreResize(ctx context.Context, sizeMb int64) error {
	if ok, err := lvmutils.LvIsActivated(d.GetPath()); err != nil {
		return err
	} else if ok && d.Storage.Lvmlockd() {
		err = lvmutils.LVActive(d.GetPath(), false, d.Storage.Lvmlockd())
		if err != nil {
			return errors.Wrap(err, "lvactive shared")
		}
	}
	err := d.SLVMDisk.PreResize(ctx, sizeMb)
	if err != nil {
		return err
	}
	err = lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
	if err != nil {
		return errors.Wrap(err, "lvactive shared")
	}
	return nil
}

func (d *SSLVMDisk) Resize(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	if ok, err := lvmutils.LvIsActivated(d.GetPath()); err != nil {
		return nil, err
	} else if ok && d.Storage.Lvmlockd() {
		err = lvmutils.LVActive(d.GetPath(), false, d.Storage.Lvmlockd())
		if err != nil {
			return nil, errors.Wrap(err, "lvactive shared")
		}
	}
	ret, err := d.SLVMDisk.Resize(ctx, params)
	if err != nil {
		return ret, err
	}
	err = lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
	if err != nil {
		return ret, errors.Wrap(err, "lvactive shared")
	}
	return ret, nil
}

func TryDeactivateBackingLvs(backingFile string) {
	if backingFile == "" {
		return
	}
	qemuImg, err := qemuimg.NewQemuImage(backingFile)
	if err != nil {
		log.Errorf("tryDeactivateBackingLvs NewQemuImage %s", err)
		return
	}
	err = lvmutils.LVDeactivate(backingFile)
	if err != nil {
		log.Errorf("tryDeactivateBackingLvs LVDeactivate %s", err)
		return
	}
	TryDeactivateBackingLvs(qemuImg.BackFilePath)
}

func (d *SSLVMDisk) Delete(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	var lvPath = d.GetPath()
	activated, err := lvmutils.LvIsActivated(lvPath)
	if err != nil {
		return nil, errors.Wrap(err, "check lv is activated")
	}
	if !activated {
		if err := lvmutils.LVActive(lvPath, d.Storage.Lvmlockd(), false); err != nil {
			return nil, errors.Wrap(err, "lv active")
		}
	}
	qemuImg, err := qemuimg.NewQemuImage(lvPath)
	if err != nil {
		return nil, errors.Wrap(err, "NewQemuImage")
	}
	TryDeactivateBackingLvs(qemuImg.BackFilePath)

	return d.SLVMDisk.Delete(ctx, params)
}

func (d *SSLVMDisk) CreateSnapshot(snapshotId string, encryptKey string, encFormat qemuimg.TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	err := lvmutils.LVActive(d.GetPath(), false, d.Storage.Lvmlockd())
	if err != nil {
		return errors.Wrap(err, "lvactive exclusive")
	}
	err = d.SLVMDisk.CreateSnapshot(snapshotId, encryptKey, encFormat, encAlg)
	if err != nil {
		e3 := lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
		if e3 != nil {
			log.Errorf("failed lvactive share %s", e3)
		}
		return err
	}

	// active disk share mode
	err = lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
	if err != nil {
		return errors.Wrap(err, "lvactive snapshot share")
	}

	// active snapshot active mode
	snapPath := d.GetSnapshotPath(snapshotId)
	err = lvmutils.LVActive(snapPath, d.Storage.Lvmlockd(), false)
	if err != nil {
		return errors.Wrap(err, "lvactive snapshot share")
	}
	return nil
}

func (d *SSLVMDisk) PostCreateFromImageFuse() {
	log.Infof("slvm post create from fuse do nothing")
}

func (d *SSLVMDisk) ResetFromSnapshot(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	err := lvmutils.LVActive(d.GetPath(), false, d.Storage.Lvmlockd())
	if err != nil {
		return nil, errors.Wrap(err, "lvactive exclusive")
	}
	ret, err := d.SLVMDisk.ResetFromSnapshot(ctx, params)
	if err != nil {
		err := lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
		if err != nil {
			log.Errorf("failed lvactive share %s", err)
		}
		return nil, err
	}
	return ret, nil
}

func (d *SSLVMDisk) GetDiskDesc() jsonutils.JSONObject {
	active, err := lvmutils.LvIsActivated(d.GetPath())
	if err != nil {
		log.Errorf("failed check active of %s: %s", d.GetPath(), err)
		return nil
	}
	if !active {
		if err := lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false); err != nil {
			log.Errorf("failed active lv %s: %s", d.GetPath(), err)
			return nil
		}
	}
	res := d.SLVMDisk.GetDiskDesc()
	if !active {
		if err := lvmutils.LVDeactivate(d.GetPath()); err != nil {
			log.Errorf("failed deactivate lv %s: %s", d.GetPath(), err)
		}
	}
	return res
}

func (d *SSLVMDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	active, err := lvmutils.LvIsActivated(d.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "LvIsActivated")
	}
	if !active {
		if err := lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false); err != nil {
			return nil, errors.Wrap(err, "LVActive")
		}
	}
	res, e := d.SLVMDisk.PrepareSaveToGlance(ctx, params)
	if !active {
		if err := lvmutils.LVDeactivate(d.GetPath()); err != nil {
			log.Errorf("failed deactivate lv %s: %s", d.GetPath(), err)
		}
	}
	return res, e
}

func (d *SSLVMDisk) CreateFromSnapshotLocation(ctx context.Context, snapshotLocation string, size int64, encryptInfo *apis.SEncryptInfo) (jsonutils.JSONObject, error) {
	ret, err := d.SLVMDisk.CreateRaw(ctx, int(size), "", "", nil, encryptInfo, d.Id, snapshotLocation)
	if err != nil {
		return nil, err
	}

	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "NewQemuImage")
	}
	TryDeactivateBackingLvs(qemuImg.BackFilePath)

	err = lvmutils.LVDeactivate(d.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "LVDeactivate")
	}
	return ret, nil
}

func (d *SSLVMDisk) DeleteSnapshot(snapshotId, convertSnapshot string, blockStream bool, encryptInfo apis.SEncryptInfo) error {
	err := lvmutils.LVActive(d.GetPath(), false, d.Storage.Lvmlockd())
	if err != nil {
		return errors.Wrap(err, "LVActive")
	}
	convertSnapshotPath := d.GetSnapshotPath(convertSnapshot)
	err = lvmutils.LVActive(convertSnapshotPath, false, d.Storage.Lvmlockd())
	if err != nil {
		return errors.Wrap(err, "LVActive convert snapshot")
	}

	err = d.SLVMDisk.DeleteSnapshot(snapshotId, convertSnapshot, blockStream, encryptInfo)
	// active disk share mode
	e := lvmutils.LVActive(d.GetPath(), d.Storage.Lvmlockd(), false)
	if e != nil {
		log.Errorf("failed active with share mode: %s", e)
	}
	e = lvmutils.LVActive(convertSnapshotPath, d.Storage.Lvmlockd(), false)
	if e != nil {
		log.Errorf("failed active convert snapshot %s with share mode: %s", convertSnapshotPath, e)
	}
	return err
}
