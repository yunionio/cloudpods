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

package esxi

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/imagetools"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var (
	vmSummaryProps = []string{"summary.runtime.powerState", "summary.config.uuid", "summary.config.memorySizeMB", "summary.config.numCpu", "summary.customValue"}
	// vmConfigProps   = []string{"config.template", "config.alternateGuestName", "config.hardware", "config.guestId", "config.guestFullName", "config.firmware", "config.version", "config.createDate"}
	vmGuestProps    = []string{"guest.net", "guest.guestState", "guest.toolsStatus", "guest.toolsRunningStatus", "guest.toolsVersion"}
	vmLayoutExProps = []string{"layoutEx.file"}
)

var VIRTUAL_MACHINE_PROPS = []string{"name", "parent", "resourcePool", "snapshot", "config", "availableField", "datastore"}

func init() {
	VIRTUAL_MACHINE_PROPS = append(VIRTUAL_MACHINE_PROPS, vmSummaryProps...)
	// VIRTUAL_MACHINE_PROPS = append(VIRTUAL_MACHINE_PROPS, vmConfigProps...)
	VIRTUAL_MACHINE_PROPS = append(VIRTUAL_MACHINE_PROPS, vmGuestProps...)
}

type SVirtualMachine struct {
	multicloud.SInstanceBase
	multicloud.STagBase
	SManagedObject

	vnics     []SVirtualNIC
	vdisks    []SVirtualDisk
	vga       SVirtualVGA
	cdroms    []SVirtualCdrom
	devs      map[int32]SVirtualDevice
	ihost     cloudprovider.ICloudHost
	snapshots []SVirtualMachineSnapshot

	guestIps map[string]string

	osInfo *imagetools.ImageInfo
}

type VMFetcher interface {
	FetchNoTemplateVMs() ([]*SVirtualMachine, error)
	FetchTemplateVMs() ([]*SVirtualMachine, error)
	FetchFakeTempateVMs(string) ([]*SVirtualMachine, error)
}

type byDiskType []SVirtualDisk

func (d byDiskType) Len() int      { return len(d) }
func (d byDiskType) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d byDiskType) Less(i, j int) bool {
	if d[i].GetDiskType() == api.DISK_TYPE_SYS {
		return true
	}
	return false
}

func NewVirtualMachine(manager *SESXiClient, vm *mo.VirtualMachine, dc *SDatacenter) *SVirtualMachine {
	svm := &SVirtualMachine{SManagedObject: newManagedObject(manager, vm, dc)}
	err := svm.fetchHardwareInfo()
	if err != nil {
		log.Errorf("NewVirtualMachine: %v", err)
		return nil
	}
	return svm
}

func (svm *SVirtualMachine) GetSecurityGroupIds() ([]string, error) {
	return []string{}, cloudprovider.ErrNotSupported
}

func (svm *SVirtualMachine) GetTags() (map[string]string, error) {
	// not support tags
	if gotypes.IsNil(svm.manager.client.ServiceContent.CustomFieldsManager) {
		return nil, cloudprovider.ErrNotSupported
	}
	ret := map[int32]string{}
	for _, val := range svm.object.Entity().ExtensibleManagedObject.AvailableField {
		ret[val.Key] = val.Name
	}
	result := map[string]string{}
	vm := svm.getVirtualMachine()
	for _, val := range vm.Summary.CustomValue {
		value := struct {
			Key   int32
			Value string
		}{}
		jsonutils.Update(&value, val)
		_, ok := ret[value.Key]
		if ok && len(value.Value) > 0 {
			result[ret[value.Key]] = value.Value
		}
	}
	for _, key := range ret {
		_, ok := result[key]
		if !ok {
			delete(result, key)
		}
	}
	return result, nil
}

func (svm *SVirtualMachine) SetTags(tags map[string]string, replace bool) error {
	// not support tags
	if gotypes.IsNil(svm.manager.client.ServiceContent.CustomFieldsManager) {
		return cloudprovider.ErrNotSupported
	}
	oldTags, err := svm.GetTags()
	if err != nil {
		return errors.Wrapf(err, "GetTags")
	}

	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}

	cfm := object.NewCustomFieldsManager(svm.manager.client.Client)
	ctx := context.Background()

	for k := range removed {
		id, err := cfm.FindKey(ctx, k)
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return errors.Wrapf(err, "FindKey %s", k)
			}
			continue
		}
		err = cfm.Set(ctx, svm.object.Reference(), id, "")
		if err != nil {
			return errors.Wrapf(err, "Set")
		}
	}
	for k, v := range added {
		id, err := cfm.FindKey(ctx, k)
		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return errors.Wrapf(err, "FindKey %s", k)
			}
			ref, err := cfm.Add(ctx, k, "VirtualMachine", nil, nil)
			if err != nil {
				return errors.Wrapf(err, "Add %s", k)
			}
			id = ref.Key
		}
		err = cfm.Set(ctx, svm.object.Reference(), id, v)
		if err != nil {
			return errors.Wrapf(err, "Set")
		}
	}
	return nil
}

func (svm *SVirtualMachine) getVirtualMachine() *mo.VirtualMachine {
	return svm.object.(*mo.VirtualMachine)
}

func (svm *SVirtualMachine) GetGlobalId() string {
	return svm.getUuid()
}

func (svm *SVirtualMachine) GetHostname() string {
	return svm.GetName()
}

func (svm *SVirtualMachine) GetStatus() string {
	// err := svm.CheckFileInfo(context.Background())
	// if err != nil {
	// 	return api.VM_UNKNOWN
	// }
	vm := object.NewVirtualMachine(svm.manager.client.Client, svm.getVirtualMachine().Self)
	state, err := vm.PowerState(svm.manager.context)
	if err != nil {
		return api.VM_UNKNOWN
	}
	switch state {
	case types.VirtualMachinePowerStatePoweredOff:
		return api.VM_READY
	case types.VirtualMachinePowerStatePoweredOn:
		return api.VM_RUNNING
	case types.VirtualMachinePowerStateSuspended:
		return api.VM_SUSPEND
	default:
		return api.VM_UNKNOWN
	}
}

func (svm *SVirtualMachine) Refresh() error {
	base := svm.SManagedObject
	var moObj mo.VirtualMachine
	err := svm.manager.reference2Object(svm.object.Reference(), VIRTUAL_MACHINE_PROPS, &moObj)
	if err != nil {
		if e := errors.Cause(err); soap.IsSoapFault(e) {
			_, ok := soap.ToSoapFault(e).VimFault().(types.ManagedObjectNotFound)
			if ok {
				return cloudprovider.ErrNotFound
			}
		}
		return err
	}
	base.object = &moObj
	*svm = SVirtualMachine{}
	svm.SManagedObject = base
	svm.fetchHardwareInfo()
	return nil
}

func (svm *SVirtualMachine) IsEmulated() bool {
	return false
}

func (svm *SVirtualMachine) GetInstanceType() string {
	return ""
}

func (svm *SVirtualMachine) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (svm *SVirtualMachine) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (svm *SVirtualMachine) DoRebuildRoot(ctx context.Context, imagePath string, uuid string) error {
	if len(svm.vdisks) == 0 {
		return errors.Wrapf(errors.ErrNotFound, "empty vdisks")
	}
	return svm.rebuildDisk(ctx, &svm.vdisks[0], imagePath)
}

func (svm *SVirtualMachine) rebuildDisk(ctx context.Context, disk *SVirtualDisk, imagePath string) error {
	uuid := disk.GetId()
	sizeMb := disk.GetDiskSizeMB()
	diskKey := disk.getKey()
	ctlKey := disk.getControllerKey()
	unitNumber := *disk.dev.GetVirtualDevice().UnitNumber

	err := svm.doDetachAndDeleteDisk(ctx, disk)
	if err != nil {
		return err
	}
	return svm.createDiskInternal(ctx, SDiskConfig{
		SizeMb:        int64(sizeMb),
		Uuid:          uuid,
		ControllerKey: ctlKey,
		UnitNumber:    unitNumber,
		Key:           diskKey,
		ImagePath:     imagePath,
		IsRoot:        len(imagePath) > 0,
	}, false)
}

func (svm *SVirtualMachine) UpdateVM(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	err := svm.SetConfig(ctx, input)
	if err != nil {
		return errors.Wrap(err, "set description")
	}
	return nil
}

// TODO: detach disk to a separate directory, so as to keep disk independent of VM

func (svm *SVirtualMachine) DetachDisk(ctx context.Context, diskId string) error {
	vdisk, err := svm.GetIDiskById(diskId)
	if err != nil {
		return err
	}
	return svm.doDetachDisk(ctx, vdisk.(*SVirtualDisk), false)
}

func (svm *SVirtualMachine) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (svm *SVirtualMachine) getUuid() string {
	return svm.getVirtualMachine().Summary.Config.Uuid
}

func (svm *SVirtualMachine) GetIHost() cloudprovider.ICloudHost {
	if svm.ihost == nil {
		svm.ihost = svm.getIHost()
	}
	return svm.ihost
}

func (svm *SVirtualMachine) getIHost() cloudprovider.ICloudHost {
	vm := svm.getVmObj()

	hostsys, err := vm.HostSystem(svm.manager.context)
	if err != nil {
		log.Errorf("fail to find host system for vm %s", err)
		return nil
	}
	ihost, err := svm.manager.FindHostByMoId(moRefId(hostsys.Reference()))
	if err != nil {
		log.Errorf("fail to find host %s for vm %s???", hostsys.Name(), svm.GetName())
		return nil
	}
	return ihost
}

func (svm *SVirtualMachine) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	idisks := make([]cloudprovider.ICloudDisk, len(svm.vdisks))
	for i := 0; i < len(svm.vdisks); i += 1 {
		idisks[i] = &(svm.vdisks[i])
	}
	return idisks, nil
}

func (svm *SVirtualMachine) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	for i := 0; i < len(svm.vdisks); i += 1 {
		if svm.vdisks[i].MatchId(idStr) {
			return &svm.vdisks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (svm *SVirtualMachine) GetINics() ([]cloudprovider.ICloudNic, error) {
	inics := make([]cloudprovider.ICloudNic, len(svm.vnics))
	for i := 0; i < len(svm.vnics); i += 1 {
		inics[i] = &(svm.vnics[i])
	}
	return inics, nil
}

func (svm *SVirtualMachine) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, nil
}

func (svm *SVirtualMachine) GetCpuSockets() int {
	vm := svm.getVirtualMachine()
	if vm.Config != nil {
		return int(vm.Config.Hardware.NumCoresPerSocket)
	}
	return 1
}

func (svm *SVirtualMachine) GetVcpuCount() int {
	return int(svm.getVirtualMachine().Summary.Config.NumCpu)
}

func (svm *SVirtualMachine) GetVmemSizeMB() int {
	return int(svm.getVirtualMachine().Summary.Config.MemorySizeMB)
}

func (svm *SVirtualMachine) GetBootOrder() string {
	return "cdn"
}

func (svm *SVirtualMachine) GetVga() string {
	return "vga"
}

func (svm *SVirtualMachine) GetVdi() string {
	return "vmrc"
}

func (svm *SVirtualMachine) GetGuestFamily() string {
	moVM := svm.getVirtualMachine()
	return moVM.Config.AlternateGuestName
}

func (svm *SVirtualMachine) GetGuestId() string {
	moVM := svm.getVirtualMachine()
	return moVM.Config.GuestId
}

func (svm *SVirtualMachine) GetGuestFullName() string {
	moVM := svm.getVirtualMachine()
	return moVM.Config.GuestFullName
}

func (svm *SVirtualMachine) GetGuestState() string {
	moVM := svm.getVirtualMachine()
	return moVM.Guest.GuestState
}

func (svm *SVirtualMachine) GetGuestToolsStatus() string {
	moVM := svm.getVirtualMachine()
	return string(moVM.Guest.ToolsStatus)
}

func (svm *SVirtualMachine) isToolsOk() bool {
	switch svm.getVirtualMachine().Guest.ToolsStatus {
	case types.VirtualMachineToolsStatusToolsNotInstalled:
		return false
	case types.VirtualMachineToolsStatusToolsNotRunning:
		return false
	}
	return true
}

func (svm *SVirtualMachine) GetGuestToolsRunningStatus() string {
	moVM := svm.getVirtualMachine()
	return string(moVM.Guest.ToolsRunningStatus)
}

func (vm *SVirtualMachine) getNormalizedOsInfo() *imagetools.ImageInfo {
	if vm.osInfo == nil {
		if osInfo, ok := GuestOsInfo[vm.GetGuestId()]; ok {
			osInfo := imagetools.NormalizeImageInfo("", string(osInfo.OsArch), string(osInfo.OsType), osInfo.OsDistribution, osInfo.OsVersion)
			vm.osInfo = &osInfo
		} else {
			osInfo := imagetools.NormalizeImageInfo("", "", "", "", "")
			vm.osInfo = &osInfo
		}
	}
	return vm.osInfo
}

func (vm *SVirtualMachine) GetOsType() cloudprovider.TOsType {
	return cloudprovider.TOsType(vm.getNormalizedOsInfo().OsType)
}

func (vm *SVirtualMachine) GetFullOsName() string {
	return vm.getNormalizedOsInfo().GetFullOsName()
}

func (vm *SVirtualMachine) GetOsDist() string {
	return vm.getNormalizedOsInfo().OsDistro
}

func (vm *SVirtualMachine) GetOsVersion() string {
	return vm.getNormalizedOsInfo().OsVersion
}

func (vm *SVirtualMachine) GetOsLang() string {
	return vm.getNormalizedOsInfo().OsLang
}

func (vm *SVirtualMachine) GetOsArch() string {
	return vm.getNormalizedOsInfo().OsArch
}

func (vm *SVirtualMachine) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(vm.getBios())
}

func (vm *SVirtualMachine) getBios() string {
	// svm.obj.config.firmware
	switch vm.getVirtualMachine().Config.Firmware {
	case "efi":
		return "UEFI"
	case "bios":
		return "BIOS"
	default:
		return "BIOS"
	}
}

func (svm *SVirtualMachine) GetMachine() string {
	return "pc"
}

func (svm *SVirtualMachine) GetHypervisor() string {
	return api.HYPERVISOR_ESXI
}

func (svm *SVirtualMachine) getVmObj() *object.VirtualMachine {
	return object.NewVirtualMachine(svm.manager.client.Client, svm.getVirtualMachine().Self)
}

// ideopotent start
func (svm *SVirtualMachine) StartVM(ctx context.Context) error {
	if svm.GetStatus() == api.VM_RUNNING {
		return nil
	}
	return svm.startVM(ctx)
}

func (svm *SVirtualMachine) startVM(ctx context.Context) error {
	ihost := svm.GetIHost()
	if ihost == nil {
		return errors.Wrap(cloudprovider.ErrInvalidStatus, "no valid host")
	}

	err := svm.makeNicsStartConnected(ctx)
	if err != nil {
		return errors.Wrapf(err, "makeNicStartConnected")
	}

	vm := svm.getVmObj()

	task, err := vm.PowerOn(ctx)
	if err != nil {
		return errors.Wrapf(err, "PowerOn")
	}
	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrapf(err, "task.Wait")
	}
	return nil
}

func (svm *SVirtualMachine) makeNicsStartConnected(ctx context.Context) error {
	spec := types.VirtualMachineConfigSpec{}
	spec.CpuHotAddEnabled = &True
	spec.CpuHotRemoveEnabled = &True
	spec.MemoryHotAddEnabled = &True
	spec.DeviceChange = make([]types.BaseVirtualDeviceConfigSpec, len(svm.vnics))
	for i := 0; i < len(svm.vnics); i += 1 {
		spec.DeviceChange[i] = makeNicStartConnected(&svm.vnics[i])
	}

	vm := svm.getVmObj()

	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func makeNicStartConnected(nic *SVirtualNIC) *types.VirtualDeviceConfigSpec {
	editSpec := types.VirtualDeviceConfigSpec{}
	editSpec.Operation = types.VirtualDeviceConfigSpecOperationEdit
	editSpec.FileOperation = ""
	editSpec.Device = nic.dev
	editSpec.Device.GetVirtualDevice().Connectable.StartConnected = true
	return &editSpec
}

func (svm *SVirtualMachine) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if svm.GetStatus() == api.VM_READY {
		return nil
	}
	if !opts.IsForce && svm.isToolsOk() {
		return svm.shutdownVM(ctx)
	} else {
		return svm.poweroffVM(ctx)
	}
}

func (svm *SVirtualMachine) SuspendVM(ctx context.Context) error {
	vm := svm.getVmObj()
	task, err := vm.Suspend(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (svm *SVirtualMachine) ResumeVM(ctx context.Context) error {
	if svm.GetStatus() == api.VM_RUNNING {
		return nil
	}
	vm := svm.getVmObj()

	task, err := vm.PowerOn(ctx)
	if err != nil {
		return errors.Wrap(err, "VM.PowerOn")
	}
	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "Task.Wait after VM.PowerOn")
	}
	return nil
}

func (svm *SVirtualMachine) poweroffVM(ctx context.Context) error {
	vm := svm.getVmObj()

	task, err := vm.PowerOff(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (svm *SVirtualMachine) shutdownVM(ctx context.Context) error {
	vm := svm.getVmObj()

	err := vm.ShutdownGuest(ctx)
	if err != nil {
		return err
	}
	return err
}

func (svm *SVirtualMachine) doDestroy(ctx context.Context) error {
	vm := svm.getVmObj()
	task, err := vm.Destroy(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to destroy vm")
	}
	return task.Wait(ctx)
}

func (svm *SVirtualMachine) doDelete(ctx context.Context) error {
	// detach all disks first
	for i := range svm.vdisks {
		err := svm.doDetachAndDeleteDisk(ctx, &svm.vdisks[i])
		if err != nil {
			return errors.Wrap(err, "doDetachAndDeteteDisk")
		}
	}

	return svm.doDestroy(ctx)
}

func (svm *SVirtualMachine) doUnregister(ctx context.Context) error {
	vm := svm.getVmObj()

	err := vm.Unregister(ctx)
	if err != nil {
		return errors.Wrapf(err, "Unregister")
	}
	return nil
}

func (svm *SVirtualMachine) DeleteVM(ctx context.Context) error {
	err := svm.CheckFileInfo(ctx)
	if err != nil {
		return svm.doUnregister(ctx)
	}
	return svm.doDestroy(ctx)
}

func (svm *SVirtualMachine) doDetachAndDeleteDisk(ctx context.Context, vdisk *SVirtualDisk) error {
	return svm.doDetachDisk(ctx, vdisk, true)
}

func (svm *SVirtualMachine) doDetachDisk(ctx context.Context, vdisk *SVirtualDisk, remove bool) error {
	removeSpec := types.VirtualDeviceConfigSpec{}
	removeSpec.Operation = types.VirtualDeviceConfigSpecOperationRemove
	removeSpec.Device = vdisk.dev

	spec := types.VirtualMachineConfigSpec{}
	spec.DeviceChange = []types.BaseVirtualDeviceConfigSpec{&removeSpec}

	vm := svm.getVmObj()

	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		return errors.Wrapf(err, "Reconfigure remove disk %s", vdisk.GetName())
	}

	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrapf(err, "wait remove disk %s task", vdisk.GetName())
	}

	if !remove {
		return nil
	}
	return vdisk.Delete(ctx)
}

func (svm *SVirtualMachine) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	hostVer := svm.GetIHost().GetVersion()
	if version.GE(hostVer, "6.5") {
		info, err := svm.acquireWebmksTicket("webmks")
		if err == nil {
			return info, nil
		}
	}
	return svm.acquireVmrcUrl()
}

func (svm *SVirtualMachine) GetVmrcInfo() (*cloudprovider.ServerVncOutput, error) {
	return svm.acquireVmrcUrl()
}

func (svm *SVirtualMachine) GetWebmksInfo() (*cloudprovider.ServerVncOutput, error) {
	return svm.acquireWebmksTicket("webmks")
}

func (svm *SVirtualMachine) acquireWebmksTicket(ticketType string) (*cloudprovider.ServerVncOutput, error) {
	vm := object.NewVirtualMachine(svm.manager.client.Client, svm.getVirtualMachine().Self)
	ticket, err := vm.AcquireTicket(svm.manager.context, ticketType)
	if err != nil {
		return nil, err
	}

	host := ticket.Host
	if len(host) == 0 {
		host = svm.manager.host
	}
	port := ticket.Port
	if port == 0 {
		port = int32(svm.manager.port)
	}
	if port == 0 {
		port = 443
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:        fmt.Sprintf("wss://%s:%d/ticket/%s", host, port, ticket.Ticket),
		Protocol:   "wmks",
		Hypervisor: api.HYPERVISOR_ESXI,
	}
	return ret, nil
}

func (svm *SVirtualMachine) acquireVmrcUrl() (*cloudprovider.ServerVncOutput, error) {
	ticket, err := svm.manager.acquireCloneTicket()
	if err != nil {
		return nil, err
	}
	port := svm.manager.port
	if port == 0 {
		port = 443
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:      fmt.Sprintf("vmrc://clone:%s@%s:%d/?moid=%s", ticket, svm.manager.host, port, svm.GetId()),
		Protocol: "vmrc",
	}
	return ret, nil
}

func (svm *SVirtualMachine) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return svm.doChangeConfig(ctx, int32(config.Cpu), int32(config.CpuSocket), int64(config.MemoryMB), "", "")
}

func (svm *SVirtualMachine) GetVersion() string {
	return svm.getVirtualMachine().Config.Version
}

func (svm *SVirtualMachine) doChangeConfig(ctx context.Context, ncpu, cpuSockets int32, vmemMB int64, guestId string, version string) error {
	changed := false
	configSpec := types.VirtualMachineConfigSpec{}
	if int(ncpu) != svm.GetVcpuCount() {
		configSpec.NumCPUs = ncpu
		changed = true
	}
	if cpuSockets > 0 && int(cpuSockets) != svm.GetCpuSockets() {
		configSpec.NumCoresPerSocket = cpuSockets
		changed = true
	}
	if int(vmemMB) != svm.GetVmemSizeMB() {
		configSpec.MemoryMB = vmemMB
		changed = true
	}
	if len(guestId) > 0 && guestId != svm.GetGuestId() {
		configSpec.GuestId = guestId
		changed = true
	}
	if len(version) > 0 && version != svm.GetVersion() {
		configSpec.Version = version
		changed = true
	}
	if !changed {
		return nil
	}

	vm := svm.getVmObj()

	task, err := vm.Reconfigure(ctx, configSpec)
	if err != nil {
		return err
	}
	err = task.Wait(ctx)
	if err != nil {
		return err
	}
	return svm.Refresh()
}

func (svm *SVirtualMachine) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (svm *SVirtualMachine) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (svm *SVirtualMachine) GetCreatedAt() time.Time {
	moVM := svm.getVirtualMachine()
	ctm := moVM.Config.CreateDate
	if ctm != nil {
		return *ctm
	} else {
		return time.Time{}
	}
}

func (svm *SVirtualMachine) SetConfig(ctx context.Context, input cloudprovider.SInstanceUpdateOptions) error {
	setDescTask, err := svm.getVmObj().Reconfigure(ctx, types.VirtualMachineConfigSpec{
		Name:       input.NAME,
		Annotation: input.Description,
	})
	if err != nil {
		return errors.Wrap(err, "set task")
	}
	return setDescTask.Wait(ctx)
}

func (svm *SVirtualMachine) GetExpiredAt() time.Time {
	return time.Time{}
}

func (svm *SVirtualMachine) UpdateUserData(userData string) error {
	return nil
}

func (svm *SVirtualMachine) fetchHardwareInfo() error {
	svm.vnics = make([]SVirtualNIC, 0)
	svm.vdisks = make([]SVirtualDisk, 0)
	svm.cdroms = make([]SVirtualCdrom, 0)
	svm.devs = make(map[int32]SVirtualDevice)

	moVM := svm.getVirtualMachine()

	// MAX_TRIES := 3
	// for tried := 0; tried < MAX_TRIES && (moVM == nil || moVM.Config == nil || moVM.Config.Hardware.Device == nil); tried += 1 {
	// 	time.Sleep(time.Second)
	// }

	if moVM == nil || moVM.Config == nil || moVM.Config.Hardware.Device == nil {
		return fmt.Errorf("invalid vm")
	}

	// sort devices via their Key
	devices := moVM.Config.Hardware.Device
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].GetVirtualDevice().Key < devices[j].GetVirtualDevice().Key
	})
	for i := 0; i < len(devices); i += 1 {
		dev := devices[i]
		devType := reflect.Indirect(reflect.ValueOf(dev)).Type()

		etherType := reflect.TypeOf((*types.VirtualEthernetCard)(nil)).Elem()
		diskType := reflect.TypeOf((*types.VirtualDisk)(nil)).Elem()
		vgaType := reflect.TypeOf((*types.VirtualMachineVideoCard)(nil)).Elem()
		cdromType := reflect.TypeOf((*types.VirtualCdrom)(nil)).Elem()

		if reflectutils.StructContains(devType, etherType) {
			vnic := NewVirtualNIC(svm, dev, len(svm.vnics))
			svm.vnics = append(svm.vnics, vnic)
		} else if reflectutils.StructContains(devType, diskType) {
			svm.vdisks = append(svm.vdisks, NewVirtualDisk(svm, dev, len(svm.vdisks)))
		} else if reflectutils.StructContains(devType, vgaType) {
			svm.vga = NewVirtualVGA(svm, dev, 0)
		} else if reflectutils.StructContains(devType, cdromType) {
			svm.cdroms = append(svm.cdroms, NewVirtualCdrom(svm, dev, len(svm.cdroms)))
		}
		vdev := NewVirtualDevice(svm, dev, 0)
		svm.devs[vdev.getKey()] = vdev
	}
	svm.rigorous()
	sort.Sort(byDiskType(svm.vdisks))
	return nil
}

func (svm *SVirtualMachine) rigorous() {
	hasRoot := false
	for i := range svm.vdisks {
		if svm.vdisks[i].IsRoot {
			hasRoot = true
			break
		}
	}
	if !hasRoot && len(svm.vdisks) > 0 {
		svm.vdisks[0].IsRoot = true
	}
}

func (svm *SVirtualMachine) getVdev(key int32) SVirtualDevice {
	return svm.devs[key]
}

func (svm *SVirtualMachine) getNetTags() string {
	info := make([]string, 0)
	moVM := svm.getVirtualMachine()
	for _, net := range moVM.Guest.Net {
		mac := netutils.FormatMacAddr(net.MacAddress)
		ips := make([]string, 0)
		for _, ip := range net.IpAddress {
			if regutils.MatchIP4Addr(ip) && !strings.HasPrefix(ip, "169.254.") {
				ips = append(ips, ip)
			}
		}
		if len(mac) > 0 && len(net.Network) > 0 && len(ips) > 0 {
			info = append(info, mac, net.Network)
			info = append(info, ips...)
		}
	}
	return strings.Join(info, "/")
}

func (svm *SVirtualMachine) fetchGuestIps() map[string]string {
	guestIps := make(map[string]string)
	moVM := svm.getVirtualMachine()
	for _, net := range moVM.Guest.Net {
		if len(net.Network) == 0 {
			continue
		}
		mac := netutils.FormatMacAddr(net.MacAddress)
		for _, ip := range net.IpAddress {
			if regutils.MatchIP4Addr(ip) && !strings.HasPrefix(ip, "169.254.") {
				if !vmIPV4Filter.Contains(ip) {
					continue
				}
				guestIps[mac] = ip
				break
			}
		}
	}
	return guestIps
}

func (svm *SVirtualMachine) getGuestIps() map[string]string {
	if svm.guestIps == nil {
		svm.guestIps = svm.fetchGuestIps()
	}
	return svm.guestIps
}

func (svm *SVirtualMachine) GetIps() []string {
	ips := make([]string, 0)
	for _, ip := range svm.getGuestIps() {
		ips = append(ips, ip)
	}
	return ips
}

func (svm *SVirtualMachine) GetVGADevice() string {
	return fmt.Sprintf("%s", svm.vga.String())
}

var (
	driverTable = map[string][]string{
		"sata":   {"ahci"},
		"scsi":   {"parascsi", "lsilogic", "lsilogicsas", "buslogic"},
		"pvscsi": {"parascsi"},
		"ide":    {"ide"},
	}
)

func (svm *SVirtualMachine) getDevsByDriver(driver string) []SVirtualDevice {
	devs := make([]SVirtualDevice, 0)
	for _, drv := range svm.devs {
		if strings.HasSuffix(drv.GetDriver(), fmt.Sprintf("%scontroller", driver)) {
			devs = append(devs, drv)
		}
	}
	return devs
}

func minDevKey(devs []SVirtualDevice) int32 {
	var minKey int32 = -1
	for i := 0; i < len(devs); i += 1 {
		if minKey < 0 || minKey > devs[i].getKey() {
			minKey = devs[i].getKey()
		}
	}
	return minKey
}

func minDiskKey(devs []SVirtualDisk) int32 {
	var minKey int32 = -1
	for i := 0; i < len(devs); i += 1 {
		if minKey < 0 || minKey > devs[i].getKey() {
			minKey = devs[i].getKey()
		}
	}
	return minKey
}

func (svm *SVirtualMachine) FindController(ctx context.Context, driver string) ([]SVirtualDevice, error) {
	aliasDrivers, ok := driverTable[driver]
	if !ok {
		return nil, fmt.Errorf("Unsupported disk driver %s", driver)
	}
	var devs []SVirtualDevice
	for _, alias := range aliasDrivers {
		devs = svm.getDevsByDriver(alias)
		if len(devs) > 0 {
			break
		}
	}
	return devs, nil
}

func (svm *SVirtualMachine) FindDiskByDriver(drivers ...string) []SVirtualDisk {
	disks := make([]SVirtualDisk, 0)
	for i := range svm.vdisks {
		if utils.IsInStringArray(svm.vdisks[i].GetDriver(), drivers) {
			disks = append(disks, svm.vdisks[i])
		}
	}
	return disks
}

func (svm *SVirtualMachine) devNumWithCtrlKey(ctrlKey int32) int {
	n := 0
	for _, dev := range svm.devs {
		if dev.getControllerKey() == ctrlKey {
			n++
		}
	}
	return n
}

func (svm *SVirtualMachine) getLayoutEx() *types.VirtualMachineFileLayoutEx {
	vm := svm.getVirtualMachine()
	if vm.LayoutEx != nil {
		return vm.LayoutEx
	}
	var nvm mo.VirtualMachine
	err := svm.manager.reference2Object(vm.Self, vmLayoutExProps, &nvm)
	if err != nil {
		log.Errorf("unable to fetch LayoutEx.File from vc: %v", err)
	}
	vm.LayoutEx = nvm.LayoutEx
	return vm.LayoutEx
}

func (svm *SVirtualMachine) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	if opts.Driver == "pvscsi" {
		opts.Driver = "scsi"
	}
	var ds *SDatastore
	var err error
	if opts.StorageId != "" {
		ihost := svm.getIHost()
		if ihost == nil {
			return "", fmt.Errorf("unable to get host of virtualmachine %s", svm.GetName())
		}
		ds, err = ihost.(*SHost).FindDataStoreById(opts.StorageId)
		if err != nil {
			return "", errors.Wrapf(err, "unable to find datastore %s", opts.StorageId)
		}
	}
	devs, err := svm.FindController(ctx, opts.Driver)
	if err != nil {
		return "", err
	}
	if len(devs) == 0 {
		return "", svm.createDriverAndDisk(ctx, ds, opts.SizeMb, opts.UUID, opts.Driver, opts.Preallocation)
	}
	numDevBelowCtrl := make([]int, len(devs))
	for i := range numDevBelowCtrl {
		numDevBelowCtrl[i] = svm.devNumWithCtrlKey(devs[i].getKey())
	}

	// find the min one
	ctrlKey := devs[0].getKey()
	unitNumber := numDevBelowCtrl[0]
	for i := 1; i < len(numDevBelowCtrl); i++ {
		if numDevBelowCtrl[i] >= unitNumber {
			continue
		}
		ctrlKey = devs[i].getKey()
		unitNumber = numDevBelowCtrl[i]
	}
	diskKey := svm.FindMinDiffKey(2000)

	// By default, the virtual SCSI controller is assigned to virtual device node (z:7),
	// so that device node is unavailable for hard disks or other devices.
	if unitNumber >= 7 && opts.Driver == "scsi" {
		unitNumber++
	}

	return "", svm.createDiskInternal(ctx, SDiskConfig{
		SizeMb:        int64(opts.SizeMb),
		Uuid:          opts.UUID,
		UnitNumber:    int32(unitNumber),
		ControllerKey: ctrlKey,
		Key:           diskKey,
		Datastore:     ds,
		Preallocation: opts.Preallocation,
	}, true)
}

// createDriverAndDisk will create a driver and disk associated with the driver
func (svm *SVirtualMachine) createDriverAndDisk(ctx context.Context, ds *SDatastore, sizeMb int, uuid string, driver, preallocation string) error {
	if driver != "scsi" && driver != "pvscsi" {
		return fmt.Errorf("Driver %s is not supported", driver)
	}

	deviceChange := make([]types.BaseVirtualDeviceConfigSpec, 0, 2)

	// find a suitable key for scsi or pvscsi driver
	scsiKey := svm.FindMinDiffKey(1000)
	deviceChange = append(deviceChange, addDevSpec(NewSCSIDev(scsiKey, 100, driver)))

	// find a suitable key for disk
	diskKey := svm.FindMinDiffKey(2000)

	if diskKey == scsiKey {
		// unarrivelable
		log.Errorf("there is no suitable key between 1000 and 2000???!")
	}

	return svm.createDiskWithDeviceChange(ctx, deviceChange,
		SDiskConfig{
			SizeMb:        int64(sizeMb),
			Uuid:          uuid,
			ControllerKey: scsiKey,
			UnitNumber:    0,
			Key:           scsiKey,
			ImagePath:     "",
			IsRoot:        false,
			Datastore:     ds,
			Preallocation: preallocation,
		}, true)
}

func (svm *SVirtualMachine) getDatastoreAndRootImagePath() (string, *SDatastore, error) {
	layoutEx := svm.getLayoutEx()
	if layoutEx == nil || len(layoutEx.File) == 0 {
		return "", nil, fmt.Errorf("invalid LayoutEx")
	}
	file := layoutEx.File[0].Name
	// find stroage
	host := svm.GetIHost()
	storages, err := host.GetIStorages()
	if err != nil {
		return "", nil, errors.Wrap(err, "host.GetIStorages")
	}
	var datastore *SDatastore
	for i := range storages {
		ds := storages[i].(*SDatastore)
		if ds.HasFile(file) {
			datastore = ds
			break
		}
	}
	if datastore == nil {
		return "", nil, fmt.Errorf("can't find storage associated with vm %q", svm.GetName())
	}
	path := datastore.cleanPath(file)
	vmDir := strings.Split(path, "/")[0]
	// TODO find a non-conflicting path
	return datastore.getPathString(fmt.Sprintf("%s/%s.vmdk", vmDir, vmDir)), datastore, nil
}

func (svm *SVirtualMachine) GetRootImagePath() (string, error) {
	path, _, err := svm.getDatastoreAndRootImagePath()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (svm *SVirtualMachine) CopyRootDisk(ctx context.Context, imagePath string) (string, error) {
	newImagePath, datastore, err := svm.getDatastoreAndRootImagePath()
	if err != nil {
		return "", errors.Wrapf(err, "GetRootImagePath")
	}
	fm := datastore.getDatastoreObj().NewFileManager(datastore.datacenter.getObjectDatacenter(), true)
	err = fm.Copy(ctx, imagePath, newImagePath)
	if err != nil {
		return "", errors.Wrapf(err, "unable to copy system disk %s -> %s", imagePath, newImagePath)
	}
	return newImagePath, nil
}

func (svm *SVirtualMachine) createDiskWithDeviceChange(ctx context.Context, deviceChange []types.BaseVirtualDeviceConfigSpec, config SDiskConfig, check bool) error {
	var err error
	// copy disk
	if len(config.ImagePath) > 0 {
		config.IsRoot = true
		config.ImagePath, err = svm.CopyRootDisk(ctx, config.ImagePath)
		if err != nil {
			return errors.Wrap(err, "unable to copyRootDisk")
		}
	}

	devSpec := NewDiskDev(int64(config.SizeMb), config)
	spec := addDevSpec(devSpec)
	if len(config.ImagePath) == 0 {
		spec.FileOperation = types.VirtualDeviceConfigSpecFileOperationCreate
	}
	configSpec := types.VirtualMachineConfigSpec{}
	configSpec.DeviceChange = append(deviceChange, spec)

	vmObj := svm.getVmObj()

	task, err := vmObj.Reconfigure(ctx, configSpec)
	if err != nil {
		return err
	}
	err = task.Wait(ctx)
	if err != nil {
		return err
	}
	if !check {
		return nil
	}
	oldDiskCnt := len(svm.vdisks)
	maxTries := 60
	for tried := 0; tried < maxTries; tried += 1 {
		time.Sleep(time.Second)
		svm.Refresh()
		if len(svm.vdisks) > oldDiskCnt {
			return nil
		}
	}
	return cloudprovider.ErrTimeout
}

func (svm *SVirtualMachine) createDiskInternal(ctx context.Context, config SDiskConfig, check bool) error {

	return svm.createDiskWithDeviceChange(ctx, nil, config, check)
}

func (svm *SVirtualMachine) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (svm *SVirtualMachine) GetProjectId() string {
	pool, err := svm.getResourcePool()
	if err != nil {
		return ""
	}
	if pool != nil {
		return pool.GetId()
	}
	return ""
}

func (svm *SVirtualMachine) GetError() error {
	return nil
}

func (svm *SVirtualMachine) getResourcePool() (*SResourcePool, error) {
	vm := svm.getVirtualMachine()
	morp := mo.ResourcePool{}
	if vm.ResourcePool == nil {
		return nil, errors.Error("nil resource pool")
	}
	err := svm.manager.reference2Object(*vm.ResourcePool, RESOURCEPOOL_PROPS, &morp)
	if err != nil {
		return nil, errors.Wrap(err, "svm.manager.reference2Object")
	}
	rp := NewResourcePool(svm.manager, &morp, svm.datacenter)
	return rp, nil
}

func (svm *SVirtualMachine) CheckFileInfo(ctx context.Context) error {
	layoutEx := svm.getLayoutEx()
	if layoutEx != nil && len(layoutEx.File) > 0 {
		file := layoutEx.File[0]
		host := svm.GetIHost()
		storages, err := host.GetIStorages()
		if err != nil {
			return errors.Wrap(err, "host.GetIStorages")
		}
		for i := range storages {
			ds := storages[i].(*SDatastore)
			if ds.HasFile(file.Name) {
				_, err := ds.CheckFile(ctx, file.Name)
				if err != nil {
					return errors.Wrap(err, "ds.CheckFile")
				}
				break
			}
		}
	}
	return nil
}

func (svm *SVirtualMachine) DoRename(ctx context.Context, name string) error {
	task, err := svm.getVmObj().Rename(ctx, name)
	if err != nil {
		return errors.Wrap(err, "object.VirtualMachine.Rename")
	}
	return task.Wait(ctx)
}

func (svm *SVirtualMachine) GetMoid() string {
	return svm.getVirtualMachine().Self.Value
}

func (svm *SVirtualMachine) GetToolsVersion() string {
	return svm.getVirtualMachine().Guest.ToolsVersion
}

type SServerNic struct {
	Name      string `json:"name"`
	Index     int    `json:"index"`
	Bridge    string `json:"bridge"`
	Domain    string `json:"domain"`
	Ip        string `json:"ip"`
	Vlan      int    `json:"vlan"`
	Driver    string `json:"driver"`
	Masklen   int    `json:"masklen"`
	Virtual   bool   `json:"virtual"`
	Manual    bool   `json:"manual"`
	WireId    string `json:"wire_id"`
	NetId     string `json:"net_id"`
	Mac       string `json:"mac"`
	BandWidth int    `json:"bw"`
	Mtu       int    `json:"mtu,omitempty"`
	Dns       string `json:"dns"`
	Ntp       string `json:"ntp"`
	Net       string `json:"net"`
	Interface string `json:"interface"`
	Gateway   string `json:"gateway"`
	Ifname    string `json:"ifname"`
	NicType   string `json:"nic_type,omitempty"`
	LinkUp    bool   `json:"link_up,omitempty"`
	TeamWith  string `json:"team_with,omitempty"`

	TeamingMaster *SServerNic   `json:"-"`
	TeamingSlaves []*SServerNic `json:"-"`
}

func (nicdesc SServerNic) getNicDns() []string {
	dnslist := []string{}
	if len(nicdesc.Dns) > 0 {
		dnslist = append(dnslist, nicdesc.Dns)
	}
	return dnslist
}

func (svm *SVirtualMachine) DoCustomize(ctx context.Context, params jsonutils.JSONObject) error {
	spec := new(types.CustomizationSpec)

	ipSettings := new(types.CustomizationGlobalIPSettings)
	domain := "local"
	if params.Contains("domain") {
		domain, _ = params.GetString("domain")
	}
	ipSettings.DnsSuffixList = []string{domain}

	// deal nics
	serverNics := make([]SServerNic, 0)
	err := params.Unmarshal(&serverNics, "nics")
	if err != nil {
		return errors.Wrap(err, "Unmarshal nics")
	}

	// find dnsServerList
	for i := range serverNics {
		dnsList := serverNics[i].getNicDns()
		if len(dnsList) != 0 {
			ipSettings.DnsServerList = dnsList
		}
	}
	spec.GlobalIPSettings = *ipSettings

	maps := make([]types.CustomizationAdapterMapping, 0, len(serverNics))
	for _, nic := range serverNics {
		conf := types.CustomizationAdapterMapping{}
		conf.MacAddress = nic.Mac
		if len(conf.MacAddress) == 0 {
			conf.MacAddress = "9e:46:27:21:a2:b2"
		}

		conf.Adapter = types.CustomizationIPSettings{}
		fixedIp := new(types.CustomizationFixedIp)
		fixedIp.IpAddress = nic.Ip
		if len(fixedIp.IpAddress) == 0 {
			fixedIp.IpAddress = "10.168.26.23"
		}
		conf.Adapter.Ip = fixedIp
		maskLen := nic.Masklen
		if maskLen == 0 {
			maskLen = 24
		}
		mask := netutils.Netlen2Mask(maskLen)
		conf.Adapter.SubnetMask = mask

		if len(nic.Gateway) != 0 {
			conf.Adapter.Gateway = []string{nic.Gateway}
		}
		dnsList := nic.getNicDns()
		if len(dnsList) != 0 {
			conf.Adapter.DnsServerList = dnsList
			dns := nic.Domain
			if len(dns) == 0 {
				dns = "local"
			}
			conf.Adapter.DnsDomain = dns
		}
		maps = append(maps, conf)
	}
	spec.NicSettingMap = maps

	var (
		osName   string
		name     = "yunionhost"
		hostname = name
	)
	if params.Contains("os_name") {
		osName, _ = params.GetString("os_name")
	}
	if params.Contains("name") {
		name, _ = params.GetString("name")
		hostname = name
	}
	if params.Contains("hostname") {
		hostname, _ = params.GetString("hostname")
	}
	// avoid spec.identity.hostName error
	hostname = func() string {
		ret := ""
		for _, s := range hostname {
			if unicode.IsDigit(s) || unicode.IsLetter(s) || s == '-' {
				ret += string(s)
			}
		}
		return ret
	}()
	if osName == "Linux" {
		linuxPrep := types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{Name: hostname},
			Domain:   domain,
			TimeZone: "Asia/Shanghai",
		}
		spec.Identity = &linuxPrep
	} else if osName == "Windows" {
		if len(hostname) > 15 {
			hostname = hostname[:15]
		}
		sysPrep := types.CustomizationSysprep{
			GuiUnattended: types.CustomizationGuiUnattended{
				TimeZone:  210,
				AutoLogon: false,
			},
			UserData: types.CustomizationUserData{
				FullName:  "Administrator",
				OrgName:   "Yunion",
				ProductId: "",
				ComputerName: &types.CustomizationFixedName{
					Name: hostname,
				},
			},
			Identification: types.CustomizationIdentification{},
		}
		spec.Identity = &sysPrep
	}
	log.Infof("customize spec: %#v", spec)
	task, err := svm.getVmObj().Customize(ctx, *spec)
	if err != nil {
		return errors.Wrap(err, "object.VirtualMachine.Customize")
	}
	return task.Wait(ctx)
}

func (svm *SVirtualMachine) ExportTemplate(ctx context.Context, idx int, diskPath string) error {
	lease, err := svm.getVmObj().Export(ctx)
	if err != nil {
		return errors.Wrap(err, "esxi.SVirtualMachine.DoExportTemplate")
	}
	info, err := lease.Wait(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "lease.Wait")
	}

	u := lease.StartUpdater(ctx, info)
	defer u.Done()

	if idx >= len(info.Items) {
		return errors.Error(fmt.Sprintf("No such Device whose index is %d", idx))
	}

	lr := newLeaseLogger("download vmdk", 5)
	lr.Log()
	defer lr.End()

	// filter vmdk item
	vmdkItems := make([]nfc.FileItem, 0, len(info.Items)/2)
	for i := range info.Items {
		if strings.HasSuffix(info.Items[i].Path, ".vmdk") {
			vmdkItems = append(vmdkItems, info.Items[i])
		} else {
			log.Infof("item.Path does not end in '.vmdk': %#v", info.Items[i])
		}
	}

	log.Debugf("download to %s start...", diskPath)
	err = lease.DownloadFile(ctx, diskPath, vmdkItems[idx], soap.Download{Progress: lr})
	if err != nil {
		return errors.Wrap(err, "lease.DownloadFile")
	}

	err = lease.Complete(ctx)
	if err != nil {
		return errors.Wrap(err, "lease.Complete")
	}
	log.Debugf("download to %s finish", diskPath)
	return nil
}

func (svm *SVirtualMachine) GetSerialOutput(port int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (svm *SVirtualMachine) ConvertPublicIpToEip() error {
	return cloudprovider.ErrNotSupported
}

func (svm *SVirtualMachine) IsAutoRenew() bool {
	return false
}

func (svm *SVirtualMachine) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (svm *SVirtualMachine) FindMinDiffKey(limit int32) int32 {
	if svm.devs == nil {
		svm.fetchHardwareInfo()
	}
	devKeys := make([]int32, 0, len(svm.devs))
	for key := range svm.devs {
		devKeys = append(devKeys, key)
	}
	sort.Slice(devKeys, func(i int, j int) bool {
		return devKeys[i] < devKeys[j]
	})
	for _, key := range devKeys {
		switch {
		case key < limit:
		case key == limit:
			limit += 1
		case key > limit:
			return limit
		}
	}
	return limit
}

func (svm *SVirtualMachine) relocate(hostId string) error {
	var targetHs *mo.HostSystem
	if hostId == "" {
		return errors.Wrap(fmt.Errorf("require hostId"), "relocate")
	}
	ihost, err := svm.manager.GetIHostById(hostId)
	if err != nil {
		return errors.Wrap(err, "svm.manager.GetIHostById(hostId)")
	}
	host := ihost.(*SHost)
	targetHs = host.object.(*mo.HostSystem)
	if len(targetHs.Datastore) < 1 {
		return errors.Wrap(fmt.Errorf("target host has no datastore"), "relocate")
	}
	rp, err := host.GetResourcePool()
	if err != nil {
		return errors.Wrapf(err, "GetResourcePool")
	}
	pool := rp.Reference()
	ctx := svm.manager.context
	config := types.VirtualMachineRelocateSpec{}
	config.Pool = &pool
	hrs := targetHs.Reference()
	config.Host = &hrs
	dss := []types.ManagedObjectReference{}
	var datastores []mo.Datastore
	for i := range svm.vdisks {
		ds := svm.vdisks[i].getBackingInfo().GetDatastore()
		if ds != nil {
			dss = append(dss, *ds)
		}
	}
	dc, err := host.GetDatacenter()
	if err != nil {
		return err
	}

	err = host.manager.references2Objects(dss, DATASTORE_PROPS, &datastores)
	if err != nil {
		return err
	}
	isShared := true
	for i := 0; i < len(datastores); i += 1 {
		ds := NewDatastore(host.manager, &datastores[i], dc)
		storageType := ds.GetStorageType()
		if !utils.IsInStringArray(storageType, []string{api.STORAGE_NAS, api.STORAGE_NFS, api.STORAGE_VSAN}) {
			isShared = false
			break
		}
	}
	if !isShared {
		config.Datastore = &targetHs.Datastore[0]
	}
	task, err := svm.getVmObj().Relocate(ctx, config, types.VirtualMachineMovePriorityDefaultPriority)
	if err != nil {
		return errors.Wrap(err, "Relocate")
	}
	err = task.Wait(ctx)
	if err != nil {
		return errors.Wrap(err, "task.wait")
	}
	return nil
}

func (svm *SVirtualMachine) MigrateVM(hostId string) error {
	return svm.relocate(hostId)
}

func (svm *SVirtualMachine) LiveMigrateVM(hostId string) error {
	return svm.relocate(hostId)
}

func (svm *SVirtualMachine) GetIHostId() string {
	ctx := svm.manager.context
	hs, err := svm.getVmObj().HostSystem(ctx)
	if err != nil {
		log.Errorf("get HostSystem %s", err)
		return ""
	}
	var moHost mo.HostSystem
	err = svm.manager.reference2Object(hs.Reference(), HOST_SYSTEM_PROPS, &moHost)
	if err != nil {
		log.Errorf("hostsystem reference2Object %s", err)
		return ""
	}
	shost := NewHost(svm.manager, &moHost, nil)
	return shost.GetGlobalId()
}

func (svm *SVirtualMachine) IsTemplate() bool {
	movm := svm.getVirtualMachine()
	if tempalteNameRegex != nil && tempalteNameRegex.MatchString(svm.GetName()) && movm.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff {
		return true
	}
	return movm.Config != nil && movm.Config.Template
}

func (svm *SVirtualMachine) fetchSnapshots() {
	movm := svm.getVirtualMachine()
	if movm.Snapshot == nil {
		return
	}
	svm.snapshots = svm.extractSnapshots(movm.Snapshot.RootSnapshotList, make([]SVirtualMachineSnapshot, 0, len(movm.Snapshot.RootSnapshotList)))
}

func (svm *SVirtualMachine) extractSnapshots(tree []types.VirtualMachineSnapshotTree, snapshots []SVirtualMachineSnapshot) []SVirtualMachineSnapshot {
	for i := range tree {
		snapshots = append(snapshots, SVirtualMachineSnapshot{
			snapshotTree: tree[i],
			vm:           svm,
		})
		snapshots = svm.extractSnapshots(tree[i].ChildSnapshotList, snapshots)
	}
	return snapshots
}

func (svm *SVirtualMachine) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	if svm.snapshots == nil {
		svm.fetchSnapshots()
	}
	ret := make([]cloudprovider.ICloudInstanceSnapshot, 0, len(svm.snapshots))
	for i := range svm.snapshots {
		ret = append(ret, &svm.snapshots[i])
	}
	return ret, nil
}

func (svm *SVirtualMachine) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	if svm.snapshots == nil {
		svm.fetchSnapshots()
	}
	for i := range svm.snapshots {
		if svm.snapshots[i].GetGlobalId() == idStr {
			// copyone
			sp := svm.snapshots[i]
			return &sp, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (svm *SVirtualMachine) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	ovm := svm.getVmObj()
	task, err := ovm.CreateSnapshot(ctx, name, desc, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSnapshot")
	}
	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.Wait")
	}
	sp := info.Result.(types.ManagedObjectReference)
	err = svm.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "create successfully")
	}
	svm.fetchSnapshots()
	for i := range svm.snapshots {
		if svm.snapshots[i].snapshotTree.Snapshot == sp {
			// copyone
			sp := svm.snapshots[i]
			return &sp, nil
		}
	}
	return nil, errors.Wrap(errors.ErrNotFound, "create successfully")
}

func (svm *SVirtualMachine) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	cloudIsp, err := svm.GetInstanceSnapshot(idStr)
	if err != nil {
		return errors.Wrap(err, "GetInstanceSnapshot")
	}
	isp := cloudIsp.(*SVirtualMachineSnapshot)
	req := types.RevertToSnapshot_Task{
		This: isp.snapshotTree.Snapshot.Reference(),
	}
	res, err := methods.RevertToSnapshot_Task(ctx, svm.manager.client.Client, &req)
	if err != nil {
		return errors.Wrap(err, "RevertToSnapshot_Task")
	}
	return object.NewTask(svm.manager.client.Client, res.Returnval).Wait(ctx)
}

func (vm *SVirtualMachine) GetDatastores() ([]*SDatastore, error) {
	dsList := make([]*SDatastore, 0)
	dss := vm.getVirtualMachine().Datastore
	for i := range dss {
		var moStore mo.Datastore
		err := vm.manager.reference2Object(dss[i].Reference(), DATASTORE_PROPS, &moStore)
		if err != nil {
			log.Errorf("datastore reference2Object %s", err)
			return nil, errors.Wrap(err, "reference2Object")
		}
		ds := NewDatastore(vm.manager, &moStore, vm.datacenter)
		dsList = append(dsList, ds)
	}
	return dsList, nil
}

func (vm *SVirtualMachine) GetDatastoreNames() []string {
	dss, err := vm.GetDatastores()
	if err != nil {
		log.Errorf("GetDatastores fail %s", err)
		return nil
	}
	names := make([]string, 0, len(dss))
	for i := range dss {
		names = append(names, dss[i].GetName())
	}
	return names
}
