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
	"os"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/proxmox"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/agent/iagent"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"
)

type SProxmoxStorage struct {
	SLocalStorage
	agent iagent.IAgent
}

func NewProxmoxStorage(manager *SStorageManager, agent iagent.IAgent, path string) *SProxmoxStorage {
	s := &SProxmoxStorage{SLocalStorage: *NewLocalStorage(manager, path, 0)}
	s.agent = agent
	s.checkDirC(path)
	return s
}

func (ps *SProxmoxStorage) GetDiskById(diskId string) (IDisk, error) {
	return NewProxmoxDisk(ps, diskId), nil
}

func (ps *SProxmoxStorage) isExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (ps *SProxmoxStorage) checkDirC(dir string) error {
	if ps.isExist(dir) {
		return nil
	}
	return os.MkdirAll(dir, 0770)
}

func (ps *SProxmoxStorage) AgentDeployGuest(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	dataDict := data.(*jsonutils.JSONDict)
	log.Debugf("dataDict: %s", dataDict)
	deployInfo := SDeployInfo{}
	err := dataDict.Unmarshal(&deployInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal DeployInfo")
	}
	var vm cloudprovider.ICloudVM = nil
	var needDeploy = true
	switch deployInfo.Action {
	case "create":
		vm, needDeploy, err = ps.agentCreateGuest(ctx, dataDict)
		if err != nil {
			return nil, errors.Wrap(err, "agentCreateGuest")
		}
	case "rebuild":
		vm, err = ps.agentRebuildRoot(ctx, dataDict)
		if err != nil {
			return nil, errors.Wrap(err, "agentRebuildRoot")
		}
	case "deploy":
		vm, err = ps.agentGetVm(ctx, dataDict)
		if err != nil {
			return nil, errors.Wrap(err, "agentGetVm")
		}
	default:
		return nil, errors.Errorf("invalid action: %s", deployInfo.Action)
	}

	disks, err := vm.GetIDisks()
	if err != nil {
		return nil, errors.Wrap(err, "VM.GetIDisks")
	}
	if len(disks) == 0 {
		return nil, errors.Error(fmt.Sprintf("no such disks for vm %s", vm.GetId()))
	}

	var deploy *deployapi.DeployGuestFsResponse
	if needDeploy {
		accountPassword, _ := utils.DescryptAESBase64(deployInfo.Datastore.VcenterId, deployInfo.Datastore.Password)
		vddkInfo := deployapi.VDDKConInfo{
			Host:   deployInfo.HostIp,
			Port:   22,
			User:   strings.TrimSuffix(deployInfo.Datastore.Account, "@pam"),
			Passwd: accountPassword,
		}
		guestDesc := deployapi.GuestDesc{}
		err = dataDict.Unmarshal(&guestDesc, "desc")
		if err != nil {
			return nil, errors.Wrapf(err, "%s: unmarshal to guestDesc", hostutils.ParamsError.Error())
		}

		guestDesc.Hypervisor = api.HYPERVISOR_PROXMOX

		key := deployapi.SSHKeys{
			PublicKey:        deployInfo.PublicKey,
			DeletePublicKey:  deployInfo.DeletePublicKey,
			AdminPublicKey:   deployInfo.AdminPublicKey,
			ProjectPublicKey: deployInfo.ProjectPublicKey,
		}

		deployArray := make([]*deployapi.DeployContent, 0)
		for _, deploy := range deployInfo.Deploys {
			deployArray = append(deployArray, &deployapi.DeployContent{
				Path:    deploy.Path,
				Content: deploy.Content,
				Action:  deploy.Action,
			})
		}

		isRandomPassword := false
		passwd := deployInfo.Password
		resetPassword := deployInfo.ResetPassword
		if resetPassword && len(passwd) == 0 {
			passwd = seclib.RandomPassword(12)
			isRandomPassword = true
		}

		if deployInfo.DeployTelegraf && deployInfo.TelegrafConf == "" {
			return nil, errors.Errorf("missing telegraf_conf")
		}

		deployInformation := deployapi.NewDeployInfo(&key, deployArray, passwd, isRandomPassword, true, false,
			options.HostOptions.LinuxDefaultRootUser, options.HostOptions.WindowsDefaultAdminUser,
			deployInfo.EnableCloudInit, deployInfo.LoginAccount, deployInfo.DeployTelegraf, deployInfo.TelegrafConf,
			deployInfo.Desc.UserData,
		)
		rootPath := disks[0].GetAccessPath()
		log.Debugf("deployInfo: %s", jsonutils.Marshal(deployInfo))
		deploy, err = deployclient.GetDeployClient().DeployGuestFs(ctx, &deployapi.DeployParams{
			DiskInfo: &deployapi.DiskInfo{
				Path: rootPath,
			},
			GuestDesc:  &guestDesc,
			DeployInfo: deployInformation,
			VddkInfo:   &vddkInfo,
		})
		if err != nil {
			model := &esxiVm{}
			dataDict.Unmarshal(model, "desc")
			logclient.AddSimpleActionLog(model, logclient.ACT_VM_DEPLOY, errors.Wrapf(err, "DeployGuestFs"), hostutils.GetComputeSession(context.Background()).GetToken(), false)
			log.Errorf("unable to DeployGuestFs: %v", err)
		}
	}

	array := jsonutils.NewArray()
	for i := range disks {
		disk := disks[i]
		diskId := disk.GetId()
		diskDict := jsonutils.NewDict()
		diskDict.Add(jsonutils.NewString(diskId), "disk_id")
		diskDict.Add(jsonutils.NewString(disk.GetGlobalId()), "uuid")
		diskDict.Add(jsonutils.NewInt(int64(disk.GetDiskSizeMB())), "size")
		diskDict.Add(jsonutils.NewString(disk.GetAccessPath()), "path")
		diskDict.Add(jsonutils.NewString(disk.GetCacheMode()), "cache_mode")
		diskDict.Add(jsonutils.NewString(disk.GetDiskType()), "disk_type")
		diskDict.Add(jsonutils.NewString(disk.GetDriver()), "driver")
		array.Add(diskDict)
	}
	updated := jsonutils.NewDict()
	updated.Add(array, "disks")
	updated.Add(jsonutils.NewString(vm.GetGlobalId()), "uuid")
	var ret jsonutils.JSONObject
	if deploy != nil {
		ret = jsonutils.Marshal(deploy)
	} else {
		ret = jsonutils.NewDict()
	}
	ret.(*jsonutils.JSONDict).Update(updated)
	return ret, nil
}

func (ps *SProxmoxStorage) agentCreateGuest(ctx context.Context, data jsonutils.JSONObject) (cloudprovider.ICloudVM, bool, error) {
	hd := SHostDatastore{}
	err := data.Unmarshal(&hd)
	if err != nil {
		return nil, false, errors.Wrap(err, hostutils.ParamsError.Error())
	}

	needDeploy := true
	client, err := proxmox.NewProxmoxClientFromAccessInfo(&hd.Datastore)
	if err != nil {
		return nil, false, errors.Wrap(err, "proxmox.NewProxmoxClientFromAccessInfo")
	}
	host, err := client.GetHost(hd.HostId)
	if err != nil {
		return nil, false, errors.Wrap(err, "GetHost")
	}
	desc := api.GuestJsonDesc{}
	err = data.Unmarshal(&desc, "desc")
	if err != nil {
		return nil, false, errors.Wrap(err, hostutils.ParamsError.Error())
	}
	createParam := cloudprovider.SManagedVMCreateConfig{
		Name:            desc.Name,
		Description:     desc.Description,
		ExternalImageId: desc.ExternalImageId,
		Cpu:             desc.Cpu,
		MemoryMB:        desc.Mem,
		SysDisk:         cloudprovider.SDiskInfo{},
		DataDisks:       make([]cloudprovider.SDiskInfo, 0),
		KeypairName:     desc.Keypair,
		PublicKey:       desc.Pubkey,
	}
	if len(desc.Nics) > 0 {
		createParam.IpAddr = desc.Nics[0].Ip
	}
	for i := range desc.Disks {
		if i == 0 {
			createParam.SysDisk = cloudprovider.SDiskInfo{
				StorageExternalId: desc.Disks[i].StorageExternalId,
				StorageType:       desc.Disks[i].StorageType,
				SizeGB:            desc.Disks[i].Size / 1024,
			}
			if len(desc.Disks[i].ImageInfo.ImageExternalId) > 0 {
				createParam.ExternalImageId = desc.Disks[i].ImageInfo.ImageExternalId
			}
		} else {
			createParam.DataDisks = append(createParam.DataDisks, cloudprovider.SDiskInfo{
				StorageExternalId: desc.Disks[i].StorageExternalId,
				StorageType:       desc.Disks[i].StorageType,
				SizeGB:            desc.Disks[i].Size / 1024,
			})
		}
	}

	if strings.HasSuffix(createParam.ExternalImageId, ".iso") {
		needDeploy = false
	}

	vm, err := host.CreateVM(&createParam)
	if err != nil {
		return nil, false, errors.Wrap(err, "SHost.CreateVM2")
	}

	cloudprovider.Wait(3*time.Second, 2*time.Minute, func() (bool, error) {
		disks, err := vm.GetIDisks()
		if err != nil {
			return false, errors.Wrap(err, "VM.GetIDisks")
		}
		if len(disks) > 0 {
			return true, nil
		}
		log.Debugf("wait for vm %s disks ready", vm.GetId())
		return false, nil
	})

	return vm, needDeploy, nil
}

func (ps *SProxmoxStorage) agentRebuildRoot(ctx context.Context, data jsonutils.JSONObject) (cloudprovider.ICloudVM, error) {
	type sRebuildParam struct {
		SHostDatastore
		GuestExtId string
		Desc       struct {
			Disks []struct {
				SizeMb    int `json:"size"`
				ImageInfo struct {
					ImageExternalId string
				}
			}
		}
	}
	rp := sRebuildParam{}
	err := data.Unmarshal(&rp)
	if err != nil {
		return nil, errors.Wrap(err, hostutils.ParamsError.Error())
	}
	client, err := proxmox.NewProxmoxClientFromAccessInfo(&rp.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "as.getHostAndDatastore")
	}
	host, err := client.GetHost(rp.HostId)
	if err != nil {
		return nil, errors.Wrap(err, "GetHost")
	}
	vm, err := host.GetIVMById(rp.GuestExtId)
	if err != nil {
		return nil, errors.Wrapf(err, "host.GetIVMById of id '%s'", rp.GuestExtId)
	}
	sysSizeGB, imageId := 0, ""
	if len(rp.Desc.Disks) > 0 {
		sysSizeGB = rp.Desc.Disks[0].SizeMb / 1024
		imageId = rp.Desc.Disks[0].ImageInfo.ImageExternalId
	}
	_, err = vm.RebuildRoot(ctx, &cloudprovider.SManagedVMRebuildRootConfig{
		ImageId:   imageId,
		SysSizeGB: sysSizeGB,
	})
	if err != nil {
		return nil, errors.Wrap(err, "VM.RebuildRoot")
	}
	return vm, nil
}

func (ps *SProxmoxStorage) agentGetVm(ctx context.Context, data jsonutils.JSONObject) (cloudprovider.ICloudVM, error) {
	type sRebuildParam struct {
		SHostDatastore
		GuestExtId string
	}
	rp := sRebuildParam{}
	err := data.Unmarshal(&rp)
	if err != nil {
		return nil, errors.Wrap(err, hostutils.ParamsError.Error())
	}
	client, err := proxmox.NewProxmoxClientFromAccessInfo(&rp.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "proxmox.NewProxmoxClientFromAccessInfo")
	}
	host, err := client.GetHost(rp.HostId)
	if err != nil {
		return nil, errors.Wrap(err, "GetHost")
	}
	vm, err := host.GetIVMById(rp.GuestExtId)
	if err != nil {
		return nil, errors.Wrapf(err, "host.GetIVMById of id '%s'", rp.GuestExtId)
	}
	return vm, nil
}
