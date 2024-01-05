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
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/hostman/storageman/lvmutils"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

// shared lvm
type SSLVMDisk struct {
	SCLVMDisk
}

func NewSLVMDisk(storage IStorage, id string) *SSLVMDisk {
	return &SSLVMDisk{
		SCLVMDisk: *NewCLVMDisk(storage, id),
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
	activated, err := lvmutils.LvIsActivated(lvPath)
	if err != nil {
		return errors.Wrap(err, "check lv is activated")
	}
	if !activated {
		if err := lvmutils.LVActive(lvPath); err != nil {
			return errors.Wrap(err, "lv active")
		}
	}

	qemuImg, err := qemuimg.NewQemuImage(d.GetPath())
	if err != nil {
		log.Errorln(err)
		return err
	}
	if qemuImg.BackFilePath != "" {
		originActivated, err := lvmutils.LvIsActivated(qemuImg.BackFilePath)
		if err != nil {
			return errors.Wrap(err, "check lv is activated")
		}
		if !originActivated {
			if err = lvmutils.LVActive(qemuImg.BackFilePath); err != nil {
				return errors.Wrap(err, "lv active origin")
			}
		}
	}

	if !fileutils2.Exists(d.GetPath()) {
		return errors.Wrapf(cloudprovider.ErrNotFound, "%s", d.GetPath())
	}
	return nil
}
