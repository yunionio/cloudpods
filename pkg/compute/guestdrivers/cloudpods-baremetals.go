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

package guestdrivers

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/cloudpods"
)

type SCloudpodsBaremetalGuestDriver struct {
	SManagedVirtualizedGuestDriver
}

func init() {
	driver := SCloudpodsBaremetalGuestDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SCloudpodsBaremetalGuestDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAREMETAL
}

func (self *SCloudpodsBaremetalGuestDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CLOUDPODS
}

func (self *SCloudpodsBaremetalGuestDriver) GetInstanceCapability() cloudprovider.SInstanceCapability {
	return cloudprovider.SInstanceCapability{
		Hypervisor: self.GetHypervisor(),
		Provider:   self.GetProvider(),
	}
}

func (self *SCloudpodsBaremetalGuestDriver) GetComputeQuotaKeys(scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, brand string) models.SComputeResourceKeys {
	keys := models.SComputeResourceKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(scope, ownerId)
	keys.CloudEnv = api.CLOUD_ENV_PRIVATE_CLOUD
	keys.Provider = api.CLOUD_PROVIDER_CLOUDPODS
	keys.Brand = api.CLOUD_PROVIDER_CLOUDPODS
	keys.Hypervisor = api.HYPERVISOR_BAREMETAL
	return keys
}

func (self *SCloudpodsBaremetalGuestDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SCloudpodsBaremetalGuestDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SCloudpodsBaremetalGuestDriver) GetMaxSecurityGroupCount() int {
	//暂不支持绑定安全组
	return 0
}

func (self *SCloudpodsBaremetalGuestDriver) GetMaxVCpuCount() int {
	return 1024
}

func (self *SCloudpodsBaremetalGuestDriver) GetMaxVMemSizeGB() int {
	return 4096
}

func (self *SCloudpodsBaremetalGuestDriver) PrepareDiskRaidConfig(userCred mcclient.TokenCredential, host *models.SHost, confs []*api.BaremetalDiskConfig, disks []*api.DiskConfig) ([]*api.DiskConfig, error) {
	baremetalStorage := models.ConvertStorageInfo2BaremetalStorages(host.StorageInfo)
	if baremetalStorage == nil {
		return nil, fmt.Errorf("Convert storage info error")
	}
	if len(confs) == 0 {
		parsedConf, _ := baremetal.ParseDiskConfig("")
		confs = []*api.BaremetalDiskConfig{&parsedConf}
	}
	layouts, err := baremetal.CalculateLayout(confs, baremetalStorage)
	if err != nil {
		return nil, err
	}
	err = host.UpdateDiskConfig(userCred, layouts)
	if err != nil {
		return nil, err
	}
	allocable, extra := baremetal.CheckDisksAllocable(layouts, disks)
	if !allocable {
		return nil, fmt.Errorf("baremetal.CheckDisksAllocable not allocable")
	}
	return extra, nil
}

func (self *SCloudpodsBaremetalGuestDriver) GetRebuildRootStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SCloudpodsBaremetalGuestDriver) GetChangeConfigStatus(guest *models.SGuest) ([]string, error) {
	return nil, httperrors.NewUnsupportOperationError("Cannot change config for baremtal")
}

func (self *SCloudpodsBaremetalGuestDriver) GetDeployStatus() ([]string, error) {
	return []string{api.VM_READY, api.VM_ADMIN}, nil
}

func (self *SCloudpodsBaremetalGuestDriver) ValidateResizeDisk(guest *models.SGuest, disk *models.SDisk, storage *models.SStorage) error {
	return httperrors.NewUnsupportOperationError("Cannot resize disk for baremtal")
}

func (self *SCloudpodsBaremetalGuestDriver) GetNamedNetworkConfiguration(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig) (*models.SNetwork, []models.SNicConfig, api.IPAllocationDirection, bool, error) {
	netifs, net, err := host.GetNetinterfacesWithIdAndCredential(netConfig.Network, userCred, netConfig.Reserved)
	if err != nil {
		return nil, nil, "", false, errors.Wrap(err, "get host netinterfaces")
	}
	if netifs != nil {
		nicCnt := 1
		if netConfig.RequireTeaming || netConfig.TryTeaming {
			nicCnt = 2
		}
		if len(netifs) < nicCnt {
			if netConfig.RequireTeaming {
				return net, nil, "", false, errors.Errorf("not enough network interfaces, want %d got %d", nicCnt, len(netifs))
			}
			nicCnt = len(netifs)
		}
		nicConfs := make([]models.SNicConfig, 0)
		for i := 0; i < nicCnt; i += 1 {
			nicConf := models.SNicConfig{
				Mac:    netifs[i].Mac,
				Index:  netifs[i].Index,
				Ifname: "",
			}
			nicConfs = append(nicConfs, nicConf)
		}
		reuseAddr := false
		hn := host.GetAttach2Network(netConfig.Network)
		if hn != nil && options.Options.BaremetalServerReuseHostIp {
			if netConfig.Address == "" || netConfig.Address == hn.IpAddr {
				// try to reuse host network IP address
				netConfig.Address = hn.IpAddr
				reuseAddr = true
			}
		}

		return net, nicConfs, api.IPAllocationStepup, reuseAddr, nil
	}
	return net, nil, "", false, nil
}

func (self *SCloudpodsBaremetalGuestDriver) GetRandomNetworkTypes() []api.TNetworkType {
	return []api.TNetworkType{api.NETWORK_TYPE_BAREMETAL, api.NETWORK_TYPE_GUEST}
}

func (self *SCloudpodsBaremetalGuestDriver) Attach2RandomNetwork(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, netConfig *api.NetworkConfig, pendingUsage quotas.IQuota) ([]models.SGuestnetwork, error) {
	netifs := host.GetHostNetInterfaces()
	netsAvaiable := make([]models.SNetwork, 0)
	netifIndexs := make(map[string][]models.SNetInterface, 0)

	drv, err := guest.GetDriver()
	if err != nil {
		return nil, err
	}

	netTypes := drv.GetRandomNetworkTypes()
	if len(netConfig.NetType) > 0 {
		netTypes = []api.TNetworkType{netConfig.NetType}
	}
	var wirePattern *regexp.Regexp
	if len(netConfig.Wire) > 0 {
		wirePattern = regexp.MustCompile(netConfig.Wire)
	}
	for idx, netif := range netifs {
		if !netif.IsUsableServernic() {
			continue
		}
		wire := netif.GetWire()
		if wire == nil {
			continue
		}
		if wirePattern != nil && !wirePattern.MatchString(wire.Id) && !wirePattern.MatchString(wire.Name) {
			continue
		}
		var net *models.SNetwork
		if netConfig.Private {
			net, _ = wire.GetCandidatePrivateNetwork(ctx, userCred, userCred, models.NetworkManager.AllowScope(userCred), netConfig.Exit, netTypes)
		} else {
			net, _ = wire.GetCandidateAutoAllocNetwork(ctx, userCred, userCred, models.NetworkManager.AllowScope(userCred), netConfig.Exit, netTypes)
		}
		if net != nil {
			netsAvaiable = append(netsAvaiable, *net)
			if _, exist := netifIndexs[net.WireId]; !exist {
				netifIndexs[net.WireId] = make([]models.SNetInterface, 0)
			}
			netifIndexs[net.WireId] = append(netifIndexs[net.WireId], netifs[idx])
		}
	}
	if len(netsAvaiable) == 0 {
		return nil, fmt.Errorf("No appropriate host virtual network...")
	}
	net := models.ChooseCandidateNetworks(netsAvaiable, netConfig.Exit, netTypes)
	if net != nil {
		netifs := netifIndexs[net.WireId]
		nicConfs := make([]models.SNicConfig, 0)
		nicCnt := 1
		if netConfig.RequireTeaming || netConfig.TryTeaming {
			nicCnt = 2
		}
		if len(netifs) < nicCnt {
			if netConfig.RequireTeaming {
				return nil, fmt.Errorf("not enough network interfaces, want %d got %d", nicCnt, len(netifs))
			}
			nicCnt = len(netifs)
		}
		for i := 0; i < nicCnt; i += 1 {
			nicConf := models.SNicConfig{
				Mac:    netifs[i].Mac,
				Index:  netifs[i].Index,
				Ifname: "",
			}
			nicConfs = append(nicConfs, nicConf)
		}
		address := ""
		reuseAddr := false
		hn := host.GetAttach2Network(net.Id)
		if hn != nil && options.Options.BaremetalServerReuseHostIp {
			// try to reuse host network IP address
			address = hn.IpAddr
			reuseAddr = true
		}
		return guest.Attach2Network(ctx, userCred, models.Attach2NetworkArgs{
			Network:             net,
			PendingUsage:        pendingUsage,
			IpAddr:              address,
			NicDriver:           netConfig.Driver,
			BwLimit:             netConfig.BwLimit,
			Virtual:             netConfig.Vip,
			TryReserved:         false,
			AllocDir:            api.IPAllocationStepup,
			RequireDesignatedIP: false,
			UseDesignatedIP:     reuseAddr,
			NicConfs:            nicConfs,

			IsDefault: netConfig.IsDefault,
		})
	}
	return nil, fmt.Errorf("No appropriate host virtual network...")
}

func (self *SCloudpodsBaremetalGuestDriver) GetStorageTypes() []string {
	return []string{
		api.STORAGE_BAREMETAL,
	}
}

func (self *SCloudpodsBaremetalGuestDriver) ChooseHostStorage(host *models.SHost, guest *models.SGuest, diskConfig *api.DiskConfig, storageIds []string) (*models.SStorage, error) {
	if len(storageIds) != 0 {
		return models.StorageManager.FetchStorageById(storageIds[0]), nil
	}
	bs := host.GetBaremetalstorage()
	if bs == nil {
		return nil, nil
	}
	return bs.GetStorage(), nil
}

func (self *SCloudpodsBaremetalGuestDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	diskCat := guest.CategorizeDisks()
	var imageId string
	if diskCat.Root != nil {
		imageId = diskCat.Root.GetTemplateId()
	}
	if len(imageId) == 0 {
		task.ScheduleRun(nil)
		return nil
	}
	storage, _ := diskCat.Root.GetStorage()
	if storage == nil {
		return fmt.Errorf("no valid storage")
	}
	storageCache := storage.GetStoragecache()
	if storageCache == nil {
		return fmt.Errorf("no valid storage cache")
	}
	input := api.CacheImageInput{
		ImageId:      imageId,
		Format:       "qcow2",
		ParentTaskId: task.GetTaskId(),
	}
	return storageCache.StartImageCacheTask(ctx, task.GetUserCred(), input)
}

func (self *SCloudpodsBaremetalGuestDriver) NeedRequestGuestHotAddIso(ctx context.Context, guest *models.SGuest) bool {
	return true
}

func (self *SCloudpodsBaremetalGuestDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	host, _ := guest.GetHost()
	return host.StartInsertIsoTask(ctx, task.GetUserCred(), filepath.Base(path), boot, task.GetTaskId())
}

func (self *SCloudpodsBaremetalGuestDriver) RequestGuestHotRemoveIso(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	host, _ := guest.GetHost()
	return host.StartEjectIsoTask(ctx, task.GetUserCred(), task.GetTaskId())
}

func (self *SCloudpodsBaremetalGuestDriver) RequestGuestCreateInsertIso(ctx context.Context, imageId string, bootIndex *int8, task taskman.ITask, guest *models.SGuest) error {
	return guest.StartInsertIsoTask(ctx, 0, imageId, true, nil, guest.HostId, task.GetUserCred(), task.GetTaskId())
}

func (self *SCloudpodsBaremetalGuestDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if len(input.BaremetalDiskConfigs) != 0 {
		if err := baremetal.ValidateDiskConfigs(input.BaremetalDiskConfigs); err != nil {
			return nil, httperrors.NewInputParameterError("Invalid raid config: %v", err)
		}
	}
	return input, nil
}

func (self *SCloudpodsBaremetalGuestDriver) ValidateCreateDataOnHost(ctx context.Context, userCred mcclient.TokenCredential, bmName string, host *models.SHost, input *api.ServerCreateInput) (*api.ServerCreateInput, error) {
	if host.HostType != api.HOST_TYPE_BAREMETAL || !host.IsBaremetal {
		return nil, httperrors.NewInputParameterError("Host %s is not a baremetal", bmName)
	}
	if !utils.IsInStringArray(host.Status, []string{api.BAREMETAL_READY, api.BAREMETAL_RUNNING, api.BAREMETAL_START_CONVERT}) {
		return nil, httperrors.NewInvalidStatusError("CloudpodsBaremetal %s is not ready", bmName)
	}
	if host.GetBaremetalServer() != nil {
		return nil, httperrors.NewInsufficientResourceError("CloudpodsBaremetal %s is occupied", bmName)
	}
	input.VmemSize = host.MemSize
	input.VcpuCount = int(host.CpuCount)
	return input, nil
}

func (self *SCloudpodsBaremetalGuestDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost, input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	ret := &cloudprovider.ServerVncOutput{}
	ret.HostId = host.Id
	zone, _ := host.GetZone()
	ret.Zone = zone.Name
	return ret, nil
}

func (self *SCloudpodsBaremetalGuestDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	val, _ := config.GetString("action")
	if len(val) == 0 {
		val = "deploy"
	}

	desc := cloudprovider.SManagedVMCreateConfig{}
	desc.Description = guest.Description
	// 账号必须在desc.GetConfig()之前设置，避免默认用户不能正常注入
	osInfo := struct {
		OsType         string
		OsDistribution string
		ImageType      string
	}{}
	config.Unmarshal(&osInfo, "desc")

	driver, err := guest.GetDriver()
	if err != nil {
		return errors.Wrapf(err, "GetDriver")
	}

	desc.Account = driver.GetDefaultAccount(osInfo.OsType, osInfo.OsDistribution, osInfo.ImageType)
	err = desc.GetConfig(config)
	if err != nil {
		return errors.Wrapf(err, "desc.GetConfig")
	}

	log.Debugf("%s baremetal config: %s", val, jsonutils.Marshal(desc).String())

	iHost, err := host.GetIHost(ctx)
	if err != nil {
		return err
	}

	switch val {
	case "create":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			h := iHost.(*cloudpods.SHost)
			opts := &api.ServerCreateInput{
				ServerConfigs: &api.ServerConfigs{},
			}
			task.GetParams().Unmarshal(&opts.BaremetalDiskConfigs, "baremetal_disk_configs")
			opts.Name = guest.Name
			opts.Hostname = guest.Hostname
			opts.Description = guest.Description
			opts.InstanceType = guest.InstanceType
			opts.VcpuCount = guest.VcpuCount
			opts.VmemSize = guest.VmemSize
			opts.Password = desc.Password
			opts.Metadata, _ = guest.GetAllUserMetadata()
			opts.UserData, _ = desc.GetUserData()
			opts.Hypervisor = api.HYPERVISOR_BAREMETAL
			networks := []*api.NetworkConfig{}
			if len(desc.ExternalNetworkId) > 0 {
				networks = append(networks, &api.NetworkConfig{
					Network: desc.ExternalNetworkId,
					Address: desc.IpAddr,
				})
			}
			disks := []*api.DiskConfig{}
			disks = append(disks, &api.DiskConfig{
				Index:    0,
				ImageId:  desc.ExternalImageId,
				DiskType: api.DISK_TYPE_SYS,
				SizeMb:   desc.SysDisk.SizeGB * 1024,
				Backend:  desc.SysDisk.StorageType,
				Storage:  desc.SysDisk.StorageExternalId,
			})
			for idx, disk := range desc.DataDisks {
				info := &api.DiskConfig{
					Index:    idx + 1,
					DiskType: api.DISK_TYPE_DATA,
					SizeMb:   -1,
					Backend:  disk.StorageType,
					Storage:  disk.StorageExternalId,
				}
				if disk.SizeGB > 0 {
					info.SizeMb = disk.SizeGB * 1024
				}
				disks = append(disks, info)
			}
			opts.Disks = disks
			opts.Networks = networks
			if len(desc.ProjectId) > 0 {
				opts.ProjectId = desc.ProjectId
			}

			log.Debugf("create baremetal params: %s", jsonutils.Marshal(opts))

			iVM, err := h.CreateBaremetalServer(opts)
			if err != nil {
				return nil, errors.Wrapf(err, "CreateBaremetalServer")
			}
			db.SetExternalId(guest, task.GetUserCred(), iVM.GetGlobalId())

			vmId := iVM.GetGlobalId()
			initialState := driver.GetGuestInitialStateAfterCreate()
			log.Debugf("VMcreated %s, wait status %s ...", vmId, initialState)
			err = cloudprovider.WaitStatusWithInstanceErrorCheck(iVM, initialState, time.Second*5, time.Second*1800, func() error {
				return iVM.GetError()
			})
			if err != nil {
				return nil, err
			}
			log.Debugf("VMcreated %s, and status is running", vmId)

			iVM, err = iHost.GetIVMById(vmId)
			if err != nil {
				return nil, errors.Wrapf(err, "GetIVMById(%s)", vmId)
			}

			data := fetchIVMinfo(desc, iVM, guest.Id, desc.Account, desc.Password, desc.PublicKey, "create")
			return data, nil
		})
	case "deploy":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			return self.SManagedVirtualizedGuestDriver.RemoteDeployGuestForDeploy(ctx, guest, iHost, task, desc)
		})
	case "rebuild":
		taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
			iVm, err := guest.GetIVM(ctx)
			if err != nil {
				return nil, errors.Wrapf(err, "GetIVM")
			}

			opts := &cloudprovider.SManagedVMRebuildRootConfig{
				Account:   desc.Account,
				ImageId:   desc.ExternalImageId,
				Password:  desc.Password,
				PublicKey: desc.PublicKey,
			}

			_, err = iVm.RebuildRoot(ctx, opts)
			if err != nil {
				return nil, err
			}
			initialState := driver.GetGuestInitialStateAfterRebuild()
			err = cloudprovider.WaitStatus(iVm, initialState, time.Second*5, time.Hour*1)
			if err != nil {
				return nil, err
			}
			data := fetchIVMinfo(desc, iVm, guest.Id, desc.Account, desc.Password, desc.PublicKey, "rebuild")
			return data, nil
		})
		return nil
	default:
		return fmt.Errorf("Action %s not supported", val)
	}
	return nil
}

func (self *SCloudpodsBaremetalGuestDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SCloudpodsBaremetalGuestDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return task.ScheduleRun(nil)
}

func (self *SCloudpodsBaremetalGuestDriver) StartGuestDetachdiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot detach disk from a baremetal server")
}

func (self *SCloudpodsBaremetalGuestDriver) StartGuestAttachDiskTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot attach disk to a baremetal server")
}

func (self *SCloudpodsBaremetalGuestDriver) StartSuspendTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot suspend a baremetal server")
}

func (self *SCloudpodsBaremetalGuestDriver) StartResumeTask(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return fmt.Errorf("Cannot resume a baremetal server")
}

func (self *SCloudpodsBaremetalGuestDriver) StartGuestSaveImage(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {
	return httperrors.NewUnsupportOperationError("Cannot save image for baremtal")
}

func (self *SCloudpodsBaremetalGuestDriver) StartGuestSaveGuestImage(ctx context.Context, userCred mcclient.TokenCredential,
	guest *models.SGuest, params *jsonutils.JSONDict, parentTaskId string) error {

	return httperrors.NewUnsupportOperationError("Cannot save image for baremtal")
}

func (self *SCloudpodsBaremetalGuestDriver) StartGuestResetTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isHard bool, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerResetTask", guest, userCred, nil, "", parentTaskId, nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SCloudpodsBaremetalGuestDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	err := guest.DeleteAllDisksInDB(ctx, userCred)
	if err != nil {
		return err
	}
	baremetal, _ := guest.GetHost()
	if baremetal != nil {
		return baremetal.UpdateDiskConfig(userCred, nil)
	}
	return nil
}

func (self *SCloudpodsBaremetalGuestDriver) IsSupportGuestClone() bool {
	return false
}

func (self *SCloudpodsBaremetalGuestDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	host, _ := guest.GetHost()
	if host == nil {
		return false, errors.Wrap(httperrors.ErrNotFound, "no host")
	}
	ipmiInfo, err := host.GetIpmiInfo()
	if err != nil {
		return false, errors.Wrap(err, "host.GetIpmiInfo")
	}
	return ipmiInfo.CdromBoot, nil
}

func (self *SCloudpodsBaremetalGuestDriver) IsSupportFloppy(guest *models.SGuest) (bool, error) {
	return false, nil
}
