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
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseHostDriver struct {
}

func (self *SBaseHostDriver) RequestBaremetalUnmaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "RequestBaremetalUnmaintence")
}

func (self *SBaseHostDriver) RequestBaremetalMaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "RequestBaremetalMaintence")
}

func (self *SBaseHostDriver) RequestSyncBaremetalHostStatus(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "RequestSyncBaremetalHostStatus")
}

func (self *SBaseHostDriver) RequestSyncBaremetalHostConfig(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "RequestSyncBaremetalHostConfig")
}

func (self *SBaseHostDriver) ValidateUpdateDisk(ctx context.Context, userCred mcclient.TokenCredential, input *api.DiskUpdateInput) (*api.DiskUpdateInput, error) {
	return input, nil
}

func (self *SBaseHostDriver) ValidateResetDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *models.SDisk, snapshot *models.SSnapshot, guests []models.SGuest, input *api.DiskResetInput) (*api.DiskResetInput, error) {
	return nil, httperrors.NewNotImplementedError("Not Implement ValidateResetDisk")
}

func (self *SBaseHostDriver) ValidateAttachStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, input api.HostStorageCreateInput) (api.HostStorageCreateInput, error) {
	return input, httperrors.NewNotImplementedError("Not Implement ValidateAttachStorage")
}

func (self *SBaseHostDriver) RequestAttachStorage(ctx context.Context, hoststorage *models.SHoststorage, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("Not Implement RequestAttachStorage")
}

func (self *SBaseHostDriver) RequestDetachStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, task taskman.ITask) error {
	return httperrors.NewNotImplementedError("Not Implement RequestDetachStorage")
}

func (self *SBaseHostDriver) ValidateDiskSize(storage *models.SStorage, sizeGb int) error {
	return fmt.Errorf("Not Implement ValidateDiskSize")
}

func (self *SBaseHostDriver) RequestDeleteSnapshotsWithStorage(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, task taskman.ITask, snapshotIds []string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) RequestDeleteSnapshotWithoutGuest(ctx context.Context, host *models.SHost, snapshot *models.SSnapshot, params *jsonutils.JSONDict, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) RequestResetDisk(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) RequestCleanUpDiskSnapshots(ctx context.Context, host *models.SHost, disk *models.SDisk, params *jsonutils.JSONDict, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (self *SBaseHostDriver) PrepareConvert(host *models.SHost, image, raid string, data jsonutils.JSONObject) (*api.ServerCreateInput, error) {
	params := &api.ServerCreateInput{
		ServerConfigs: &api.ServerConfigs{
			PreferHost: host.Id,
			Hypervisor: api.HYPERVISOR_BAREMETAL,
		},
		VcpuCount: int(host.CpuCount),
		VmemSize:  host.MemSize,
		AutoStart: true,
	}
	params.Description = "Baremetal convered Hypervisor"
	isSystem := true
	params.IsSystem = &isSystem
	name, err := data.GetString("name")
	if err == nil {
		params.Name = name
	} else {
		params.Name = host.Name
	}
	return params, nil
}

func (self *SBaseHostDriver) PrepareUnconvert(host *models.SHost) error {
	return nil
}

func (self *SBaseHostDriver) FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost) error {
	bss := host.GetBaremetalstorage()
	if bss != nil {
		bs := bss.GetStorage()
		if bs != nil {
			bs.SetStatus(ctx, userCred, api.STORAGE_ONLINE, "")
		} else {
			log.Errorf("ERROR: baremetal storage is None???")
		}
	} else {
		log.Errorf("ERROR: baremetal has no valid baremetalstorage????")
	}
	adminNetifs := host.GetAdminNetInterfaces()
	if len(adminNetifs) != 1 {
		return fmt.Errorf("admin netif is nil or multiple %d", len(adminNetifs))
	}
	adminNetif := &adminNetifs[0]
	adminNic := adminNetif.GetHostNetwork()
	if adminNic == nil {
		return fmt.Errorf("admin nic is nil")
	}
	db.Update(host, func() error {
		host.AccessIp = adminNic.IpAddr
		host.SetEnabled(true)
		host.HostType = api.HOST_TYPE_BAREMETAL
		host.HostStatus = api.HOST_OFFLINE
		host.ManagerUri = ""
		host.Version = ""
		host.MemReserved = 0
		return nil
	})
	log.Infof("Do finish_unconvert!!!!!!!!")
	self.CleanSchedCache(host)
	return nil
}

func (self *SBaseHostDriver) CleanSchedCache(host *models.SHost) error {
	return host.ClearSchedDescCache()
}
func (self *SBaseHostDriver) FinishConvert(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, guest *models.SGuest, hostType string) error {
	_, err := db.Update(guest, func() error {
		guest.VmemSize = 0
		guest.VcpuCount = 0
		return nil
	})
	if err != nil {
		return err
	}
	disks, err := guest.GetDisks()
	if err != nil {
		return errors.Wrapf(err, "GetDisks")
	}
	for i := range disks {
		disk := &disks[i]
		db.Update(disk, func() error {
			disk.DiskSize = 0
			return nil
		})
	}
	bs := host.GetBaremetalstorage().GetStorage()
	bs.SetStatus(ctx, userCred, api.STORAGE_OFFLINE, "")
	db.Update(host, func() error {
		host.Name = guest.GetName()
		host.CpuReserved = 0
		host.MemReserved = 0
		host.AccessIp = guest.GetRealIPs()[0]
		host.SetEnabled(false)
		host.HostStatus = api.HOST_OFFLINE
		host.HostType = hostType
		host.IsBaremetal = true
		return nil
	})
	self.CleanSchedCache(host)
	return nil
}

func (self *SBaseHostDriver) ConvertFailed(host *models.SHost) error {
	return self.CleanSchedCache(host)
}

func (self *SBaseHostDriver) checkSameDiskSpec(host *models.SHost) error {
	diskSpec := models.GetDiskSpecV2(host.StorageInfo)
	if len(diskSpec) == 0 {
		return fmt.Errorf("No raid driver")
	}
	if len(diskSpec) > 1 {
		return fmt.Errorf("Raid driver is not same")
	}
	var driverName string
	var adapterSpec api.DiskAdapterSpec
	for key, as := range diskSpec {
		driverName = key
		adapterSpec = as
	}
	if len(adapterSpec) > 1 {
		return fmt.Errorf("Raid driver %s adapter not same", driverName)
	}
	var diskSpecs []*api.DiskSpec
	var adapterKey string
	for key, ds := range adapterSpec {
		adapterKey = key
		diskSpecs = ds
	}
	if len(diskSpecs) > 1 {
		return fmt.Errorf("Raid driver %s adapter %s disk not same", driverName, adapterKey)
	}
	return nil
}

func (self *SBaseHostDriver) GetRaidScheme(host *models.SHost, raid string) (string, error) {
	var candidates []string
	if err := self.checkSameDiskSpec(host); err != nil {
		return "", fmt.Errorf("check same disk spec: %v", err)
	}
	if len(raid) == 0 {
		candidates = []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1, baremetal.DISK_CONF_RAID5, baremetal.DISK_CONF_RAID0, baremetal.DISK_CONF_NONE}
	} else {
		if utils.IsInStringArray(raid, []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1}) {
			candidates = []string{baremetal.DISK_CONF_RAID10, baremetal.DISK_CONF_RAID1}
		} else {
			candidates = []string{raid}
		}
	}
	var conf []*api.BaremetalDiskConfig
	for i := 0; i < len(candidates); i++ {
		if candidates[i] == baremetal.DISK_CONF_NONE {
			conf = []*api.BaremetalDiskConfig{}
		} else {
			parsedConf, err := baremetal.ParseDiskConfig(candidates[i])
			if err != nil {
				log.Errorf("try raid %s failed: %s", candidates[i], err.Error())
				return "", err
			}
			conf = []*api.BaremetalDiskConfig{&parsedConf}
		}
		baremetalStorage := models.ConvertStorageInfo2BaremetalStorages(host.StorageInfo)
		if baremetalStorage == nil {
			return "", fmt.Errorf("Convert storage info error")
		}
		layout, err := baremetal.CalculateLayout(conf, baremetalStorage)
		if err != nil {
			log.Errorf("try raid %s failed: %s", candidates[i], err.Error())
			continue
		}
		log.Infof("convert layout %v", layout)
		raid = candidates[i]
		break
	}
	if len(raid) == 0 {
		return "", fmt.Errorf("Disk misconfiguration")
	}
	return raid, nil
}

func (driver *SBaseHostDriver) IsReachStoragecacheCapacityLimit(host *models.SHost, cachedImages []models.SCachedimage) bool {
	return false
}

func (driver *SBaseHostDriver) GetStoragecacheQuota(host *models.SHost) int {
	return -1
}

func (driver *SBaseHostDriver) RequestDeallocateBackupDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask) error {
	return fmt.Errorf("Not Implement")
}

func (driver *SBaseHostDriver) RequestSyncOnHost(ctx context.Context, host *models.SHost, task taskman.ITask) error {
	return nil
}

func (driver *SBaseHostDriver) RequestProbeIsolatedDevices(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, input jsonutils.JSONObject) (*jsonutils.JSONArray, error) {
	return nil, nil
}

func (driver *SBaseHostDriver) RequestDiskSrcMigratePrepare(ctx context.Context, host *models.SHost, disk *models.SDisk, task taskman.ITask) (jsonutils.JSONObject, error) {
	return nil, fmt.Errorf("not supported")
}

func (driver *SBaseHostDriver) RequestDiskMigrate(ctx context.Context, targetHost *models.SHost, targetStorage *models.SStorage, disk *models.SDisk, task taskman.ITask, body *jsonutils.JSONDict) error {
	return fmt.Errorf("not supported")
}
