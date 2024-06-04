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

package hostdrivers

import (
	"context"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

type SCloudpodsBaremetalHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SCloudpodsBaremetalHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SCloudpodsBaremetalHostDriver) GetHostType() string {
	return api.HOST_TYPE_BAREMETAL
}

func (self *SCloudpodsBaremetalHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAREMETAL
}

func (self *SCloudpodsBaremetalHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsBaremetalHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return nil
}

func (driver *SCloudpodsBaremetalHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return -1
}

func (driver *SCloudpodsBaremetalHostDriver) RequestBaremetalUnmaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iHost, err := baremetal.GetIHost(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIHost")
		}
		h := iHost.(*cloudpods.SHost)
		err = h.Stop()
		if err != nil {
			return nil, err
		}
		err = cloudprovider.Wait(time.Second*10, time.Minute*10, func() (bool, error) {
			err = iHost.Refresh()
			if err != nil {
				return false, errors.Wrapf(err, "Refresh")
			}
			status := iHost.GetStatus()
			log.Debugf("expect baremetal host status %s current is: %s", api.HOST_STATUS_READY, status)
			if status != api.HOST_STATUS_READY {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Wait status")
		}
		return nil, nil
	})
	return nil
}

func (driver *SCloudpodsBaremetalHostDriver) RequestBaremetalMaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iHost, err := baremetal.GetIHost(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIHost")
		}
		h := iHost.(*cloudpods.SHost)
		err = h.Start()
		if err != nil {
			return nil, err
		}
		err = cloudprovider.Wait(time.Second*10, time.Minute*10, func() (bool, error) {
			err = iHost.Refresh()
			if err != nil {
				return false, errors.Wrapf(err, "Refresh")
			}
			status := iHost.GetStatus()
			log.Debugf("expect baremetal host status %s current is: %s", api.HOST_STATUS_RUNNING, status)
			if status != api.HOST_STATUS_RUNNING {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "Wait status")
		}
		return nil, nil
	})
	return nil
}

func (driver *SCloudpodsBaremetalHostDriver) RequestSyncBaremetalHostStatus(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		iHost, err := baremetal.GetIHost(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetIHost")
		}
		return nil, baremetal.SyncWithCloudHost(ctx, userCred, iHost)
	})
	return nil
}

func (driver *SCloudpodsBaremetalHostDriver) RequestSyncBaremetalHostConfig(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	return driver.RequestSyncBaremetalHostStatus(ctx, userCred, baremetal, task)
}
