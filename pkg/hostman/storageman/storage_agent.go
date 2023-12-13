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
	"path"
	"path/filepath"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/util/timeutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/agent/iagent"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SAgentStorage struct {
	SLocalStorage
	agent iagent.IAgent
}

func NewAgentStorage(manager *SStorageManager, agent iagent.IAgent, path string) *SAgentStorage {
	s := &SAgentStorage{SLocalStorage: *NewLocalStorage(manager, path, 0)}
	s.agent = agent
	s.checkDirC(path)
	return s
}

func (as *SAgentStorage) GetDiskById(diskId string) (IDisk, error) {
	return NewAgentDisk(as, diskId), nil
}

func (as *SAgentStorage) CreateDiskByDiskInfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input, ok := params.(*SDiskCreateByDiskinfo)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "CreateDiskByDiskInfo params format error")
	}

	diskMeta, err := as.SLocalStorage.CreateDiskByDiskinfo(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "as.SLocalStorage.CreateDiskByDiskinfo")
	}
	disk, err := as.GetDiskById(input.DiskId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDiskById(%s)", input.DiskId)
	}

	_, ds, err := as.getHostAndDatastore(ctx, SHostDatastore{HostIp: input.DiskInfo.HostIp, Datastore: input.DiskInfo.Datastore})
	if err != nil {
		return nil, errors.Wrap(err, "as.getHostAndDatastore")
	}
	remotePath := "disks/" + input.DiskId
	file, err := os.Open(disk.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "fail to open disk path")
	}
	defer file.Close()
	err = ds.Upload(ctx, remotePath, file)
	if err != nil {
		return nil, errors.Wrap(err, "dataStore.Upload")
	}
	as.RemoveDisk(disk)
	return diskMeta, nil
}

func (as *SAgentStorage) agentRebuildRoot(ctx context.Context, data jsonutils.JSONObject) error {
	type sRebuildParam struct {
		GuestExtId string
		SHostDatastore
		Desc struct {
			Disks []struct {
				DiskId    string
				ImagePath string
			}
		}
	}
	rp := sRebuildParam{}
	err := data.Unmarshal(&rp)
	if err != nil {
		return errors.Wrap(err, hostutils.ParamsError.Error())
	}
	host, _, err := as.getHostAndDatastore(ctx, rp.SHostDatastore)
	if err != nil {
		return errors.Wrap(err, "as.getHostAndDatastore")
	}
	ivm, err := host.GetIVMById(rp.GuestExtId)
	if err != nil {
		return errors.Wrapf(err, "host.GetIVMById of id '%s'", rp.GuestExtId)
	}
	if len(rp.Desc.Disks) == 0 {
		return errors.Wrap(hostutils.ParamsError, "agentRebuildRoot data.desc.disks is empty")
	}
	imagePath := rp.Desc.Disks[0].ImagePath
	diskId := rp.Desc.Disks[0].DiskId
	newPath, err := host.FileUrlPathToDsPath(imagePath)
	if err != nil {
		return err
	}
	vm := ivm.(*esxi.SVirtualMachine)
	return vm.DoRebuildRoot(ctx, newPath, diskId)
}

func (as *SAgentStorage) agentCreateGuest(ctx context.Context, data *jsonutils.JSONDict) (bool, error) {
	hd := SHostDatastore{}
	err := data.Unmarshal(&hd)
	if err != nil {
		return false, errors.Wrap(err, hostutils.ParamsError.Error())
	}
	host, ds, err := as.getHostAndDatastore(ctx, hd)
	if err != nil {
		return false, err
	}
	desc, _ := data.Get("desc")
	descDict, ok := desc.(*jsonutils.JSONDict)
	if !ok {
		return false, errors.Wrap(hostutils.ParamsError, "agentCreateGuest data format error")
	}
	createParam := esxi.SCreateVMParam{}
	err = descDict.Unmarshal(&createParam)
	if err != nil {
		return false, errors.Wrapf(err, "%s: fail to unmarshal to esxi.SCreateVMParam", hostutils.ParamsError)
	}
	needDeploy, vm, err := host.CreateVM2(ctx, ds, createParam)
	if err != nil {
		return false, errors.Wrap(err, "SHost.CreateVM2")
	}
	name, _ := descDict.GetString("name")
	err = as.tryRenameVm(ctx, vm, name)
	if err != nil {
		return false, errors.Wrapf(err, "RenameVm name '%s'", name)
	}
	return needDeploy, nil
}

func (as *SAgentStorage) tryRenameVm(ctx context.Context, vm *esxi.SVirtualMachine, name string) error {
	var (
		tried = 0
		err   error
	)
	alterName := fmt.Sprintf("%s-%s", name, timeutils.ShortDate(time.Now()))
	cands := []string{name, alterName}

	for tried < 10 {
		var n string
		if tried < len(cands) {
			n = cands[tried]
		} else {
			n = fmt.Sprintf("%s-%d", alterName, tried-len(cands)+1)
		}
		tried += 1
		err = vm.DoRename(ctx, n)
		if err == nil {
			return nil
		}
	}
	return err
}

type esxiVm struct {
	UUID string
	Name string
}

func (self esxiVm) Keyword() string {
	return "server"
}

func (self esxiVm) GetId() string {
	return self.UUID
}

func (self esxiVm) GetName() string {
	return self.Name
}

func (as *SAgentStorage) AgentDeployGuest(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	init := false
	dataDict := data.(*jsonutils.JSONDict)
	action, _ := dataDict.GetString("action")
	var (
		needDeploy = true
		err        error
	)
	if action == "create" {
		needDeploy, err = as.agentCreateGuest(ctx, dataDict)
		if err != nil {
			return nil, errors.Wrap(err, "agentCreateGuest")
		}
		init = true
	} else if action == "rebuild" {
		err := as.agentRebuildRoot(ctx, dataDict)
		if err != nil {
			return nil, errors.Wrap(err, "agentRebuildRoot")
		}
		init = true
	}

	var (
		hostIp, _ = dataDict.GetString("host_ip")
		dsInfo, _ = dataDict.Get("datastore")
	)
	dc, info, err := esxi.NewESXiClientFromJson(ctx, dsInfo)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	host, err := dc.FindHostByIp(hostIp)
	if err != nil {
		return nil, errors.Wrap(err, "SDatacenter.FindHostByIp")
	}
	vmId, _ := dataDict.GetString("guest_ext_id")
	realHost := host
	ivm, err := host.GetIVMById(vmId)
	if err == cloudprovider.ErrNotFound {
		// reschedule by DRS, migrate to other host
		siblingHosts, err := host.GetSiblingHosts()
		if err != nil {
			return nil, errors.Wrap(err, "SHost.GetSiblingHosts")
		}
		for _, sh := range siblingHosts {
			ivm, err = host.GetIVMById(vmId)
			if err == nil {
				realHost = sh
				break
			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "SHost.GetIVMById")
	}
	if ivm == nil {
		return nil, errors.Error(fmt.Sprintf("no such vm '%s'", ivm.GetId()))
	}
	vm := ivm.(*esxi.SVirtualMachine)
	disks, err := vm.GetIDisks()
	if err != nil {
		return nil, errors.Wrap(err, "VM.GetIDisks")
	}
	if len(disks) == 0 {
		return nil, errors.Error(fmt.Sprintf("no such disks for vm %s", vm.GetId()))
	}
	vmref := vm.GetMoid()
	rootPath := disks[0].(*esxi.SVirtualDisk).GetFilename()

	key := deployapi.SSHKeys{}
	err = dataDict.Unmarshal(&key)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: unmarshal to deployapi.SSHKeys", hostutils.ParamsError.Error())
	}

	deployArray := make([]*deployapi.DeployContent, 0)
	if dataDict.Contains("deploys") {
		err = dataDict.Unmarshal(&deployArray, "deploys")
		if err != nil {
			return nil, errors.Wrapf(err, "%s: unmarshal to array of deployapi.DeployContent", hostutils.ParamsError.Error())
		}
	}

	resetPassword := jsonutils.QueryBoolean(dataDict, "reset_password", false)
	passwd, _ := dataDict.GetString("password")
	if resetPassword && len(passwd) == 0 {
		passwd = seclib.RandomPassword(12)
	}

	log.Debugf("host: %s, port: %d, user: %s, passwd: %s", info.Host, info.Port, info.Account, info.Password)
	var deploy *deployapi.DeployGuestFsResponse
	if needDeploy {
		vddkInfo := deployapi.VDDKConInfo{
			Host:   info.Host,
			Port:   int32(info.Port),
			User:   info.Account,
			Passwd: info.Password,
			Vmref:  vmref,
		}
		guestDesc := deployapi.GuestDesc{}
		err = dataDict.Unmarshal(&guestDesc, "desc")
		if err != nil {
			return nil, errors.Wrapf(err, "%s: unmarshal to guestDesc", hostutils.ParamsError.Error())
		}

		desc, _ := dataDict.Get("desc")
		guestDesc.Hypervisor = api.HYPERVISOR_ESXI
		deploy, err = deployclient.GetDeployClient().DeployGuestFs(ctx, &deployapi.DeployParams{
			DiskInfo: &deployapi.DiskInfo{
				Path: rootPath,
			},
			GuestDesc: &guestDesc,
			DeployInfo: &deployapi.DeployInfo{
				PublicKey:               &key,
				Deploys:                 deployArray,
				Password:                passwd,
				IsInit:                  init,
				WindowsDefaultAdminUser: true,
			},
			VddkInfo: &vddkInfo,
		})
		customize := false
		if err != nil {
			model := &esxiVm{}
			dataDict.Unmarshal(model, "desc")
			logclient.AddSimpleActionLog(model, logclient.ACT_VM_DEPLOY, errors.Wrapf(err, "DeployGuestFs"), hostutils.GetComputeSession(context.Background()).GetToken(), false)
			log.Errorf("unable to DeployGuestFs: %v", err)
			customize = true
		} else if deploy == nil {
			log.Errorf("unable to DeployGuestFs: deploy is nil")
			customize = true
		} else if len(deploy.Os) == 0 {
			log.Errorf("unable to DeployGuestFs: os is empty")
			customize = true
		}
		if customize == true {
			as.waitVmToolsVersion(ctx, vm)
			err = vm.DoCustomize(ctx, desc)
			if err != nil {
				log.Errorf("unable to DoCustomize for vm %s: %v", vm.GetId(), err)
			}
		}
	}

	array := jsonutils.NewArray()
	for _, d := range disks {
		disk := d.(*esxi.SVirtualDisk)
		diskId := disk.GetId()
		diskDict := jsonutils.NewDict()
		diskDict.Add(jsonutils.NewString(diskId), "disk_id")
		diskDict.Add(jsonutils.NewString(disk.GetGlobalId()), "uuid")
		diskDict.Add(jsonutils.NewInt(int64(disk.GetDiskSizeMB())), "size")
		diskDict.Add(jsonutils.NewString(disk.GetFilename()), "path")
		diskDict.Add(jsonutils.NewString(disk.GetCacheMode()), "cache_mode")
		diskDict.Add(jsonutils.NewString(disk.GetDiskType()), "disk_type")
		diskDict.Add(jsonutils.NewString(disk.GetDriver()), "driver")
		array.Add(diskDict)
	}
	updated := jsonutils.NewDict()
	updated.Add(array, "disks")
	updated.Add(jsonutils.NewString(vm.GetGlobalId()), "uuid")
	updated.Add(jsonutils.NewString(realHost.GetAccessIp()), "host_ip")
	var ret jsonutils.JSONObject
	if deploy != nil {
		ret = jsonutils.Marshal(deploy)
	} else {
		ret = jsonutils.NewDict()
	}
	ret.(*jsonutils.JSONDict).Update(updated)
	return ret, nil
}

func (as *SAgentStorage) waitVmToolsVersion(ctx context.Context, vm *esxi.SVirtualMachine) {
	timeout := 90 * time.Second

	timeUpper := time.Now().Add(timeout)
	for len(vm.GetToolsVersion()) == 0 && time.Now().Before(timeUpper) {
		if vm.GetStatus() != api.VM_RUNNING {
			vm.StartVM(ctx)
			time.Sleep(5 * time.Second)
		}
	}
	timeUpper = time.Now().Add(timeout)
	for vm.GetStatus() == api.VM_RUNNING && time.Now().Before(timeUpper) {
		opts := &cloudprovider.ServerStopOptions{
			IsForce: true,
		}
		vm.StopVM(ctx, opts)
		time.Sleep(5 * time.Second)
	}
	return
}

type SHostDatastore struct {
	HostIp    string
	Datastore vcenter.SVCenterAccessInfo
}

func (as *SAgentStorage) getHostAndDatastore(ctx context.Context, data SHostDatastore) (*esxi.SHost, *esxi.SDatastore, error) {
	client, err := esxi.NewESXiClientFromAccessInfo(ctx, &data.Datastore)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fail to generate client")
	}
	host, err := client.FindHostByIp(data.HostIp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fail to find host")
	}
	ds, err := host.FindDataStoreById(data.Datastore.PrivateId)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "fail to find datastore")
	}
	return host, ds, nil
}

func (as *SAgentStorage) isExist(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func (as *SAgentStorage) checkDirC(dir string) error {
	if as.isExist(dir) {
		return nil
	}
	return os.MkdirAll(dir, 0770)
}

func (as *SAgentStorage) checkFileR(file string) error {
	if !as.isExist(file) {
		return nil
	}
	return os.Remove(file)
}

func (as *SAgentStorage) PrepareSaveToGlance(ctx context.Context, taskId string, diskInfo jsonutils.JSONObject) (
	ret jsonutils.JSONObject, err error) {

	type specStruct struct {
		Vm      vcenter.SVCenterAccessInfo
		Disk    vcenter.SVCenterAccessInfo
		HostIp  string
		ImageId string
	}

	spec := specStruct{}
	err = diskInfo.Unmarshal(&spec)
	if err != nil {
		return nil, errors.Wrap(hostutils.ParamsError, err.Error())
	}

	destDir := as.GetImgsaveBackupPath()
	as.checkDirC(destDir)
	backupPath := filepath.Join(destDir, fmt.Sprintf("%s.%s", spec.Vm.PrivateId, taskId))

	client, err := esxi.NewESXiClientFromAccessInfo(ctx, &spec.Vm)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	host, err := client.FindHostByIp(spec.HostIp)
	if err != nil {
		return nil, errors.Wrapf(err, "ESXiClient.FindHostByIp of ip '%s'", spec.HostIp)
	}
	ivm, err := host.GetIVMById(spec.Vm.PrivateId)
	if err != nil {
		return nil, errors.Wrapf(err, "esxi.SHost.GetIVMById for '%s'", spec.Vm.PrivateId)
	}
	vm := ivm.(*esxi.SVirtualMachine)
	idisk, err := vm.GetIDiskById(spec.Disk.PrivateId)
	if err != nil {
		return nil, errors.Wrapf(err, "esxi.SVirtualMachine for '%s'", spec.Disk.PrivateId)
	}
	disk := idisk.(*esxi.SVirtualDisk)
	log.Infof("Export VM image to %s", backupPath)
	err = vm.ExportTemplate(ctx, disk.GetIndex(), backupPath)
	if err != nil {
		return nil, errors.Wrap(err, "VM.ExportTemplate")
	}
	dict := jsonutils.NewDict()
	dict.Add(jsonutils.NewString(backupPath), "backup")
	return dict, nil
}

func (as *SAgentStorage) SaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	data, ok := params.(jsonutils.JSONObject)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var (
		imageId, _   = data.GetString("image_id")
		imagePath, _ = data.GetString("image_path")
		compress     = jsonutils.QueryBoolean(data, "compress", true)
		format, _    = data.GetString("format")
	)
	log.Debugf("image path: %s", imagePath)

	if err := as.saveToGlance(ctx, imageId, imagePath, compress, format); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		as.onSaveToGlanceFailed(ctx, imageId, err.Error())
		return nil, err
	}
	// delete the backup image
	as.checkFileR(imagePath)

	imagecacheManager := as.Manager.LocalStorageImagecacheManager
	if len(imagecacheManager.GetId()) > 0 {
		err := procutils.NewCommand("rm", "-f", imagePath).Run()
		return nil, err
	} else {
		dstPath := path.Join(imagecacheManager.GetPath(), imageId)
		if err := procutils.NewCommand("mv", imagePath, dstPath).Run(); err != nil {
			log.Errorf("Fail to move saved image to cache: %s", err)
		}
		imagecacheManager.LoadImageCache(imageId)
		_, err := hostutils.RemoteStoragecacheCacheImage(ctx,
			imagecacheManager.GetId(), imageId, "active", dstPath)
		if err != nil {
			log.Errorf("Fail to remote cache image: %s", err)
		}
	}
	return nil, nil
}

func (as *SAgentStorage) saveToGlance(ctx context.Context, imageId, imagePath string,
	compress bool, format string) error {
	diskInfo := &deployapi.DiskInfo{
		Path: imagePath,
	}
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskInfo: diskInfo, Compress: compress})
	if err != nil {
		return errors.Wrap(err, "DeployClient.SaveToGlance")
	}

	if compress {
		origin, err := qemuimg.NewQemuImage(imagePath)
		if err != nil {
			log.Errorln(err)
			return errors.Wrap(err, "qemuimg.NewQemuImage")
		}
		if len(format) == 0 {
			format = options.HostOptions.DefaultImageSaveFormat
		}
		if format == string(qemuimg.QCOW2) {
			// may be encrypted
			if err := origin.Convert2Qcow2(true, "", "", ""); err != nil {
				log.Errorln(err)
				return err
			}
		} else {
			if err := origin.Convert2Vmdk(true); err != nil {
				log.Errorln(err)
				return err
			}
		}
	}

	f, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer f.Close()
	finfo, err := f.Stat()
	if err != nil {
		return err
	}
	size := finfo.Size()

	var params = jsonutils.NewDict()
	if len(ret.OsInfo) > 0 {
		params.Set("os_type", jsonutils.NewString(ret.OsInfo))
	}
	relInfo := ret.ReleaseInfo
	if relInfo != nil {
		params.Set("os_distribution", jsonutils.NewString(relInfo.Distro))
		if len(relInfo.Version) > 0 {
			params.Set("os_version", jsonutils.NewString(relInfo.Version))
		}
		if len(relInfo.Arch) > 0 {
			params.Set("os_arch", jsonutils.NewString(relInfo.Arch))
		}
		if len(relInfo.Version) > 0 {
			params.Set("os_language", jsonutils.NewString(relInfo.Language))
		}
	}
	params.Set("image_id", jsonutils.NewString(imageId))

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx),
		params, f, size)
	if err != nil {
		return errors.Wrap(err, "Images.Upload")
	}
	return nil
}
