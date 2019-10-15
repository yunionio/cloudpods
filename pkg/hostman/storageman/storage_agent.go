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
	"yunion.io/x/onecloud/pkg/cloudcommon/agent"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SAgentStorage struct {
	SLocalStorage
	agent agent.IAgent
}

func NewAgentStorage(manager *SStorageManager, agent agent.IAgent, path string) *SAgentStorage {
	s := &SAgentStorage{SLocalStorage: *NewLocalStorage(manager, path, 0)}
	s.agent = agent
	s.checkDirC(path)
	return s
}

func (as *SAgentStorage) GetDiskById(diskId string) IDisk {
	return NewAgentDisk(as, diskId)
}

func (as *SAgentStorage) CreateDiskByDiskInfo(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	createParams, ok := params.(*SDiskCreateByDiskinfo)
	if !ok {
		return nil, hostutils.ParamsError
	}
	err := as.checkParams(createParams.DiskInfo, "host_ip", "datastrore")
	if err != nil {
		return nil, err
	}

	diskMeta, err := as.SLocalStorage.CreateDiskByDiskinfo(ctx, params)
	if err != nil {
		return nil, err
	}
	disk := as.GetDiskById(createParams.DiskId)

	_, ds, err := as.getHostAndDatastore(ctx, createParams.DiskInfo)
	if err != nil {
		return nil, err
	}
	remotePath := "disks/" + createParams.DiskId
	file, err := os.Open(disk.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "fail to open disk path")
	}
	defer file.Close()
	err = ds.Upload(ctx, remotePath, file)
	if err != nil {
		return nil, err
	}
	as.RemoveDisk(disk)
	return diskMeta, nil
}

func (as *SAgentStorage) agentRebuildRoot(ctx context.Context, data jsonutils.JSONObject) error {
	//This function do not need to check params
	host, _, err := as.getHostAndDatastore(ctx, data)
	if err != nil {
		return err
	}
	guestExtId, _ := data.GetString("guest_ext_id")
	ivm, err := host.GetIVMById(guestExtId)
	if err != nil {
		return err
	}
	desc, _ := data.GetMap("desc")
	disks, _ := desc["disks"].(*jsonutils.JSONArray).GetArray()
	if len(disks) == 0 {
		return hostutils.ParamsError
	}
	imagePath, _ := disks[0].GetString("image_path")
	diskId, _ := disks[0].GetString("disk_id")
	if len(imagePath) == 0 || len(diskId) == 0 {
		return hostutils.ParamsError
	}
	newPath, err := host.FileUrlPathToDsPath(imagePath)
	if err != nil {
		return err
	}
	vm := ivm.(*esxi.SVirtualMachine)
	return vm.DoRebuildRoot(ctx, newPath, diskId)
}

func (as *SAgentStorage) agentCreateGuest(ctx context.Context, data *jsonutils.JSONDict) error {
	host, ds, err := as.getHostAndDatastore(ctx, data)
	if err != nil {
		return err
	}
	desc, _ := data.Get("desc")
	descDict, ok := desc.(*jsonutils.JSONDict)
	if !ok {
		return hostutils.ParamsError
	}
	_, err = host.DoCreateVM(ctx, ds, descDict)
	if err != nil {
		return errors.Wrap(err, "SHost.DoCreateVM")
	}
	id, _ := data.GetString("guest_ext_id")
	ivm, err := host.GetIVMById(id)
	if err != nil {
		return errors.Wrap(err, "SHost.GetIVMById")
	}
	vm := ivm.(*esxi.SVirtualMachine)
	name, _ := descDict.GetString("name")
	err = as.tryRenameVm(ctx, vm, name)
	if err != nil {
		return errors.Wrapf(err, "RenameVm name '%s'", name)
	}
	return nil
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
			n = fmt.Sprint("%s-%d", alterName, tried-len(cands)+1)
		}
		tried += 1
		err = vm.DoRename(ctx, n)
		if err == nil {
			return nil
		}
	}
	return err
}

func (as *SAgentStorage) AgentDeployGuest(ctx context.Context, data interface{}) (jsonutils.JSONObject, error) {
	init := false
	dataDict := data.(*jsonutils.JSONDict)
	action, _ := dataDict.GetString("action")
	if action == "create" {
		as.agentCreateGuest(ctx, dataDict)
		init = true
	} else if action == "rebuild" {
		as.agentRebuildRoot(ctx, dataDict)
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

	desc, _ := dataDict.Get("desc")
	var (
		publicKey, _        = desc.GetString("public_key")
		deletePublicKey, _  = desc.GetString("delete_public_key")
		adminPublicKey, _   = desc.GetString("admin_public_key")
		projectPublicKey, _ = desc.GetString("project_public_key")
	)

	key := deployapi.SSHKeys{PublicKey: publicKey, DeletePublicKey: deletePublicKey,
		AdminPublicKey: adminPublicKey, ProjectPublicKey: projectPublicKey}
	deploys, _ := dataDict.GetArray("deploys")

	deployArray := make([]*deployapi.DeployContent, len(deploys))
	for i := range deploys {
		var deploy deployapi.DeployContent
		deploys[i].Unmarshal(&deploy)
		deployArray[i] = &deploy
	}
	resetPassword := jsonutils.QueryBoolean(dataDict, "reset_password", false)
	passwd, _ := dataDict.GetString("password")
	if resetPassword && len(passwd) == 0 {
		passwd = seclib.RandomPassword(12)
	}
	var deploy jsonutils.JSONObject
	MountVDDKRootfs(MoutVdDDKRootfsParam{
		Vmref:    vmref,
		DiskPath: rootPath,
		Host:     info.Host,
		Port:     info.Port,
		User:     info.Account,
		Passwd:   info.Password,
	}, func(d fsdriver.IRootFsDriver) {
		if d != nil {
			deploy, _ = guestfs.DeployGuestFs(d, desc.(*jsonutils.JSONDict), &deployapi.DeployInfo{
				PublicKey: &key,
				Deploys:   deployArray,
				Password:  passwd,
				IsInit:    init,
			})
		}
	})

	// if deploy fail, try customization
	if deploy == nil {
		as.waitVmToolsVersion(ctx, vm)
		err = vm.DoCustomize(ctx, desc)
		if err != nil {
			return nil, errors.Wrap(err, "VM.DoCustomize")
		}
	}

	ret := jsonutils.NewArray()
	diskArray, _ := desc.GetArray("disks")
	for idx, d := range disks {
		disk := d.(*esxi.SVirtualDisk)
		diskId, _ := diskArray[idx].GetString("disk_id")
		diskDict := jsonutils.NewDict()
		diskDict.Add(jsonutils.NewString(diskId), "disk_id")
		diskDict.Add(jsonutils.NewString(disk.GetGlobalId()), "uuid")
		diskDict.Add(jsonutils.NewInt(int64(disk.GetDiskSizeMB())), "size")
		diskDict.Add(jsonutils.NewString(disk.GetFilename()), "path")
		diskDict.Add(jsonutils.NewString(disk.GetCacheMode()), "cache_mode")
		diskDict.Add(jsonutils.NewString(disk.GetDiskType()), "disk_type")
		diskDict.Add(jsonutils.NewString(disk.GetDriver()), "driver")
		ret.Add(diskDict)
	}
	updated := jsonutils.NewDict()
	updated.Add(ret, "disks")
	updated.Add(jsonutils.NewString(vm.GetGlobalId()), "uuid")
	updated.Add(jsonutils.NewString(realHost.GetAccessIp()), "host_ip")
	deploy.(*jsonutils.JSONDict).Update(updated)
	return deploy, nil
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
		vm.StopVM(ctx, true)
		time.Sleep(5 * time.Second)
	}
	return
}

func (as *SAgentStorage) getHostAndDatastore(ctx context.Context, data jsonutils.JSONObject) (*esxi.SHost, *esxi.SDatastore, error) {
	hostIp, _ := data.GetString("host_ip")
	dsInfo, _ := data.Get("datastore")
	client, accessInfo, err := esxi.NewESXiClientFromJson(ctx, dsInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fail to generate client")
	}
	host, err := client.FindHostByIp(hostIp)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fail to find host")
	}
	ds, err := host.FindDataStoreById(accessInfo.PrivateId)
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

func (as *SAgentStorage) checkParams(data jsonutils.JSONObject, params ...string) error {
	for _, param := range params {
		if !data.Contains(param) {
			return hostutils.ParamsError
		}
	}
	return nil
}

func (as *SAgentStorage) PrePareSaveToGlance(ctx context.Context, taskId string, diskInfo jsonutils.JSONObject) (
	ret jsonutils.JSONObject, err error) {

	var (
		vmInfo, _  = diskInfo.Get("vm")
		vmDisk, _  = diskInfo.Get("disk")
		hostIp, _  = diskInfo.GetString("host_ip")
		imageId, _ = diskInfo.GetString("image_id")
		vmId, _    = vmInfo.GetString("private_id")
		diskId, _  = vmDisk.GetString("private_id")
	)

	destDir := as.GetImgsaveBackupPath()
	as.checkDirC(destDir)
	backupPath := filepath.Join(destDir, fmt.Sprintf("%s.%s", imageId, taskId))
	defer func() {
		if err == nil {
			return
		}
		as.checkFileR(backupPath)
	}()

	client, _, err := esxi.NewESXiClientFromJson(ctx, vmInfo)
	if err != nil {
		return nil, errors.Wrap(err, "esxi.NewESXiClientFromJson")
	}
	host, err := client.FindHostByIp(hostIp)
	if err != nil {
		return nil, errors.Wrapf(err, "ESXiClient.FindHostByIp of ip '%s'", hostIp)
	}
	ivm, err := host.GetIVMById(vmId)
	if err != nil {
		return nil, errors.Wrapf(err, "esxi.SHost.GetIVMById for '%s'", vmId)
	}
	vm := ivm.(*esxi.SVirtualMachine)
	idisk, err := vm.GetIDiskById(diskId)
	if err != nil {
		return nil, errors.Wrapf(err, "esxi.SVirtualMachine for '%s'", diskId)
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
	data, ok := params.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}

	var (
		imageId, _   = data.GetString("image_id")
		imagePath, _ = data.GetString("image_path")
		compress     = jsonutils.QueryBoolean(data, "compress", true)
		format, _    = data.GetString("format")
	)

	if err := as.saveToGlance(ctx, imageId, imagePath, compress, format); err != nil {
		log.Errorf("Save to glance failed: %s", err)
		as.onSaveToGlanceFailed(ctx, imageId)
	}

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
			imagecacheManager.GetId(), imageId, "ready", dstPath)
		if err != nil {
			log.Errorf("Fail to remote cache image: %s", err)
		}
	}
	return nil, nil
}

func (as *SAgentStorage) saveToGlance(ctx context.Context, imageId, imagePath string,
	compress bool, format string) error {
	ret, err := deployclient.GetDeployClient().SaveToGlance(context.Background(),
		&deployapi.SaveToGlanceParams{DiskPath: imagePath, Compress: compress})
	if err != nil {
		return err
	}

	if compress {
		origin, err := qemuimg.NewQemuImage(imagePath)
		if err != nil {
			log.Errorln(err)
			return err
		}
		if len(format) == 0 {
			format = options.HostOptions.DefaultImageSaveFormat
		}
		if format == "qcow2" {
			if err := origin.Convert2Qcow2(true); err != nil {
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

	_, err = modules.Images.Upload(hostutils.GetImageSession(ctx, as.agent.GetZoneName()),
		params, f, size)
	return err
}

func (as *SAgentStorage) onSaveToGlanceFailed(ctx context.Context, imageId string) {
	params := jsonutils.NewDict()
	params.Set("status", jsonutils.NewString("killed"))
	_, err := modules.Images.Update(hostutils.GetImageSession(ctx, as.agent.GetZoneName()),
		imageId, params)
	if err != nil {
		log.Errorln(err)
	}
}
