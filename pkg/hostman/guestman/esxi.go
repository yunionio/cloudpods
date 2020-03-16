package guestman

import (
	"context"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	hostutils "yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
)

func (m *SGuestManager) GuestCreateFromEsxi(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	createConfig, ok := params.(*SGuestCreateFromEsxi)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetServer(createConfig.Sid)
	if err := guest.SaveDesc(createConfig.GuestDesc); err != nil {
		return nil, err
	}
	esxiCli, err := esxi.NewESXiClientFromAccessInfo(ctx, &createConfig.EsxiAccessInfo.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "new esxi client")
	}
	host, err := esxiCli.FindHostByIp(createConfig.EsxiAccessInfo.HostIp)
	if err != nil {
		return nil, errors.Wrap(err, "esxi client find host by ip")
	}
	ivm, err := host.GetIVMById(createConfig.EsxiAccessInfo.GuestExtId)
	if err != nil {
		return nil, errors.Wrap(err, "get ivm by id")
	}
	vm := ivm.(*esxi.SVirtualMachine)
	disks, err := vm.GetIDisks()
	if err != nil {
		return nil, errors.Wrap(err, "vm get idisk")
	}
	if len(disks) == 0 {
		return nil, errors.Errorf("no such disks for vm %s", vm.GetId())
	}
	vmref := vm.GetMoid()
	var esxiDisks = new(deployapi.ConnectEsxiDisksParams)
	esxiDisks.VddkInfo = &deployapi.VDDKConInfo{
		Host:   createConfig.EsxiAccessInfo.Datastore.Host,
		Port:   int32(createConfig.EsxiAccessInfo.Datastore.Port),
		User:   createConfig.EsxiAccessInfo.Datastore.Account,
		Passwd: createConfig.EsxiAccessInfo.Datastore.Password,
		Vmref:  vmref,
	}
	esxiDisks.AccessInfo = make([]*deployapi.EsxiDiskInfo, len(disks))
	for i := 0; i < len(disks); i++ {
		esxiDisks.AccessInfo[i] = &deployapi.EsxiDiskInfo{
			DiskPath: disks[i].(*esxi.SVirtualDisk).GetFilename(),
		}
	}
	connections, err := deployclient.GetDeployClient().ConnectEsxiDisks(ctx, esxiDisks)
	if err != nil {
		return nil, errors.Wrap(err, "connect esxi disks")
	}
	log.Infof("Connection disks %v", connections.String())

	var ret = jsonutils.NewDict()
	disksDesc, _ := guest.Desc.GetArray("disks")
	for i := 0; i < len(disksDesc); i++ {
		storageId, _ := disksDesc[i].GetString("storage_id")
		if storage := storageman.GetManager().GetStorage(storageId); storage == nil {
			err = errors.Wrapf(err, "get storage %s", storageId)
			break
		} else {
			var diskInfo jsonutils.JSONObject
			diskId, _ := disksDesc[i].GetString("disk_id")
			iDisk := storage.CreateDisk(diskId)
			diskInfo, err = iDisk.CreateRaw(ctx, 0, "qcow2", "", false, "", connections.Disks[i].DiskPath)
			if err != nil {
				err = errors.Wrapf(err, "create disk %s failed", diskId)
				log.Errorf(err.Error())
				break
			}
			diskInfo.(*jsonutils.JSONDict).Set("esxi_flat_filepath",
				jsonutils.NewString(connections.Disks[i].DiskPath))
			ret.Set(diskId, diskInfo)
		}
	}
	if err != nil {
		_, e := deployclient.GetDeployClient().DisconnectEsxiDisks(ctx, connections)
		if e != nil {
			log.Errorf("disconnect esxi disks failed %s", e)
		}
		return nil, err
	}
	return ret, nil
}
