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

	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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
	vmSummaryProps = []string{"summary.runtime.powerState", "summary.config.uuid", "summary.config.memorySizeMB", "summary.config.numCpu"}
	// vmConfigProps   = []string{"config.template", "config.alternateGuestName", "config.hardware", "config.guestId", "config.guestFullName", "config.firmware", "config.version", "config.createDate"}
	vmGuestProps    = []string{"guest.net", "guest.guestState", "guest.toolsStatus", "guest.toolsRunningStatus", "guest.toolsVersion"}
	vmLayoutExProps = []string{"layoutEx.file"}
)

var VIRTUAL_MACHINE_PROPS = []string{"name", "parent", "resourcePool", "snapshot", "config"}

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

func (self *SVirtualMachine) GetSecurityGroupIds() ([]string, error) {
	return []string{}, cloudprovider.ErrNotSupported
}

func (self *SVirtualMachine) GetSysTags() map[string]string {
	meta := map[string]string{}
	meta["datacenter"] = self.GetDatacenterPathString()
	rp, _ := self.getResourcePool()
	if rp != nil {
		rpPath := rp.GetPath()
		rpOffset := -1
		for i := range rpPath {
			if rpPath[i] == "Resources" {
				if i > 0 {
					meta["cluster"] = rpPath[i-1]
					rpOffset = i
				}
			} else if rpOffset >= 0 && i > rpOffset {
				meta[fmt.Sprintf("pool%d", i-rpOffset-1)] = rpPath[i]
			}
		}
	}
	return meta
}

func (self *SVirtualMachine) getVirtualMachine() *mo.VirtualMachine {
	return self.object.(*mo.VirtualMachine)
}

func (self *SVirtualMachine) GetGlobalId() string {
	return self.getUuid()
}

func (self *SVirtualMachine) GetHostname() string {
	return self.GetName()
}

func (self *SVirtualMachine) GetStatus() string {
	// err := self.CheckFileInfo(context.Background())
	// if err != nil {
	// 	return api.VM_UNKNOWN
	// }
	vm := object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
	state, err := vm.PowerState(self.manager.context)
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

func (self *SVirtualMachine) Refresh() error {
	base := self.SManagedObject
	var moObj mo.VirtualMachine
	err := self.manager.reference2Object(self.object.Reference(), VIRTUAL_MACHINE_PROPS, &moObj)
	if err != nil {
		return err
	}
	base.object = &moObj
	*self = SVirtualMachine{}
	self.SManagedObject = base
	self.fetchHardwareInfo()
	return nil
}

func (self *SVirtualMachine) IsEmulated() bool {
	return false
}

func (self *SVirtualMachine) GetInstanceType() string {
	return ""
}

func (self *SVirtualMachine) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) DoRebuildRoot(ctx context.Context, imagePath string, uuid string) error {
	if len(self.vdisks) == 0 {
		return errors.ErrNotFound
	}
	return self.rebuildDisk(ctx, &self.vdisks[0], imagePath)
}

func (self *SVirtualMachine) rebuildDisk(ctx context.Context, disk *SVirtualDisk, imagePath string) error {
	uuid := disk.GetId()
	sizeMb := disk.GetDiskSizeMB()
	diskKey := disk.getKey()
	ctlKey := disk.getControllerKey()
	unitNumber := *disk.dev.GetVirtualDevice().UnitNumber

	err := self.doDetachAndDeleteDisk(ctx, disk)
	if err != nil {
		return err
	}
	return self.createDiskInternal(ctx, SDiskConfig{
		SizeMb:        int64(sizeMb),
		Uuid:          uuid,
		ControllerKey: ctlKey,
		UnitNumber:    unitNumber,
		Key:           diskKey,
		ImagePath:     imagePath,
		IsRoot:        len(imagePath) > 0,
	}, false)
}

func (self *SVirtualMachine) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

// TODO: detach disk to a separate directory, so as to keep disk independent of VM

func (self *SVirtualMachine) DetachDisk(ctx context.Context, diskId string) error {
	vdisk, err := self.GetIDiskById(diskId)
	if err != nil {
		return err
	}
	return self.doDetachDisk(ctx, vdisk.(*SVirtualDisk), false)
}

func (self *SVirtualMachine) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) getUuid() string {
	return self.getVirtualMachine().Summary.Config.Uuid
}

func (self *SVirtualMachine) GetIHost() cloudprovider.ICloudHost {
	if self.ihost == nil {
		self.ihost = self.getIHost()
	}
	return self.ihost
}

func (self *SVirtualMachine) getIHost() cloudprovider.ICloudHost {
	vm := self.getVmObj()

	hostsys, err := vm.HostSystem(self.manager.context)
	if err != nil {
		log.Errorf("fail to find host system for vm %s", err)
		return nil
	}
	ihost, err := self.manager.FindHostByMoId(moRefId(hostsys.Reference()))
	if err != nil {
		log.Errorf("fail to find host %s for vm %s???", hostsys.Name(), self.GetName())
		return nil
	}
	return ihost
}

func (self *SVirtualMachine) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	idisks := make([]cloudprovider.ICloudDisk, len(self.vdisks))
	for i := 0; i < len(self.vdisks); i += 1 {
		idisks[i] = &(self.vdisks[i])
	}
	return idisks, nil
}

func (self *SVirtualMachine) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	for i := 0; i < len(self.vdisks); i += 1 {
		if self.vdisks[i].MatchId(idStr) {
			return &self.vdisks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SVirtualMachine) GetINics() ([]cloudprovider.ICloudNic, error) {
	inics := make([]cloudprovider.ICloudNic, len(self.vnics))
	for i := 0; i < len(self.vnics); i += 1 {
		inics[i] = &(self.vnics[i])
	}
	return inics, nil
}

func (self *SVirtualMachine) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, nil
}

func (self *SVirtualMachine) GetVcpuCount() int {
	return int(self.getVirtualMachine().Summary.Config.NumCpu)
}

func (self *SVirtualMachine) GetVmemSizeMB() int {
	return int(self.getVirtualMachine().Summary.Config.MemorySizeMB)
}

func (self *SVirtualMachine) GetBootOrder() string {
	return "cdn"
}

func (self *SVirtualMachine) GetVga() string {
	return "vga"
}

func (self *SVirtualMachine) GetVdi() string {
	return "vmrc"
}

func (self *SVirtualMachine) GetGuestFamily() string {
	moVM := self.getVirtualMachine()
	return moVM.Config.AlternateGuestName
}

func (self *SVirtualMachine) GetGuestId() string {
	moVM := self.getVirtualMachine()
	return moVM.Config.GuestId
}

func (self *SVirtualMachine) GetGuestFullName() string {
	moVM := self.getVirtualMachine()
	return moVM.Config.GuestFullName
}

func (self *SVirtualMachine) GetGuestState() string {
	moVM := self.getVirtualMachine()
	return moVM.Guest.GuestState
}

func (self *SVirtualMachine) GetGuestToolsStatus() string {
	moVM := self.getVirtualMachine()
	return string(moVM.Guest.ToolsStatus)
}

func (self *SVirtualMachine) isToolsOk() bool {
	switch self.getVirtualMachine().Guest.ToolsStatus {
	case types.VirtualMachineToolsStatusToolsNotInstalled:
		return false
	case types.VirtualMachineToolsStatusToolsNotRunning:
		return false
	}
	return true
}

func (self *SVirtualMachine) GetGuestToolsRunningStatus() string {
	moVM := self.getVirtualMachine()
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
	// self.obj.config.firmware
	switch vm.getVirtualMachine().Config.Firmware {
	case "efi":
		return "UEFI"
	case "bios":
		return "BIOS"
	default:
		return "BIOS"
	}
}

func (self *SVirtualMachine) GetMachine() string {
	return "pc"
}

func (self *SVirtualMachine) GetHypervisor() string {
	return api.HYPERVISOR_ESXI
}

func (self *SVirtualMachine) getVmObj() *object.VirtualMachine {
	return object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
}

// ideopotent start
func (self *SVirtualMachine) StartVM(ctx context.Context) error {
	if self.GetStatus() == api.VM_RUNNING {
		return nil
	}
	return self.startVM(ctx)
}

func (self *SVirtualMachine) startVM(ctx context.Context) error {
	ihost := self.GetIHost()
	if ihost == nil {
		return errors.Wrap(cloudprovider.ErrInvalidStatus, "no valid host")
	}

	err := self.makeNicsStartConnected(ctx)
	if err != nil {
		log.Errorf("self.makeNicsStartConnected %s", err)
		return err
	}

	vm := self.getVmObj()

	task, err := vm.PowerOn(ctx)
	if err != nil {
		log.Errorf("vm.PowerOn %s", err)
		return err
	}
	err = task.Wait(ctx)
	if err != nil {
		log.Errorf("task.Wait %s", err)
		return err
	}
	return nil
}

func (self *SVirtualMachine) makeNicsStartConnected(ctx context.Context) error {
	spec := types.VirtualMachineConfigSpec{}
	spec.DeviceChange = make([]types.BaseVirtualDeviceConfigSpec, len(self.vnics))
	for i := 0; i < len(self.vnics); i += 1 {
		spec.DeviceChange[i] = makeNicStartConnected(&self.vnics[i])
	}

	vm := self.getVmObj()

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

func (self *SVirtualMachine) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if self.GetStatus() == api.VM_READY {
		return nil
	}
	if !opts.IsForce && self.isToolsOk() {
		return self.shutdownVM(ctx)
	} else {
		return self.poweroffVM(ctx)
	}
}

func (self *SVirtualMachine) SuspendVM(ctx context.Context) error {
	vm := self.getVmObj()
	task, err := vm.Suspend(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) ResumeVM(ctx context.Context) error {
	if self.GetStatus() == api.VM_RUNNING {
		return nil
	}
	vm := self.getVmObj()

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

func (self *SVirtualMachine) poweroffVM(ctx context.Context) error {
	vm := self.getVmObj()

	task, err := vm.PowerOff(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) shutdownVM(ctx context.Context) error {
	vm := self.getVmObj()

	err := vm.ShutdownGuest(ctx)
	if err != nil {
		return err
	}
	return err
}

func (self *SVirtualMachine) doDestroy(ctx context.Context) error {
	vm := self.getVmObj()
	task, err := vm.Destroy(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to destroy vm")
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) doDelete(ctx context.Context) error {
	// detach all disks first
	for i := range self.vdisks {
		err := self.doDetachAndDeleteDisk(ctx, &self.vdisks[i])
		if err != nil {
			return errors.Wrap(err, "doDetachAndDeteteDisk")
		}
	}

	return self.doDestroy(ctx)
}

func (self *SVirtualMachine) doUnregister(ctx context.Context) error {
	vm := self.getVmObj()

	err := vm.Unregister(ctx)
	if err != nil {
		log.Errorf("vm.Unregister(ctx) fail %s", err)
		return err
	}
	return nil
}

func (self *SVirtualMachine) DeleteVM(ctx context.Context) error {
	err := self.CheckFileInfo(ctx)
	if err != nil {
		return self.doUnregister(ctx)
	}
	return self.doDestroy(ctx)
}

func (self *SVirtualMachine) doDetachAndDeleteDisk(ctx context.Context, vdisk *SVirtualDisk) error {
	return self.doDetachDisk(ctx, vdisk, true)
}

func (self *SVirtualMachine) doDetachDisk(ctx context.Context, vdisk *SVirtualDisk, remove bool) error {
	removeSpec := types.VirtualDeviceConfigSpec{}
	removeSpec.Operation = types.VirtualDeviceConfigSpecOperationRemove
	removeSpec.Device = vdisk.dev

	spec := types.VirtualMachineConfigSpec{}
	spec.DeviceChange = []types.BaseVirtualDeviceConfigSpec{&removeSpec}

	vm := self.getVmObj()

	task, err := vm.Reconfigure(ctx, spec)
	if err != nil {
		log.Errorf("vm.Reconfigure fail %s", err)
		return err
	}

	err = task.Wait(ctx)
	if err != nil {
		log.Errorf("task.Wait(ctx) fail %s", err)
		return err
	}

	if !remove {
		return nil
	}
	return vdisk.Delete(ctx)
}

func (self *SVirtualMachine) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	hostVer := self.GetIHost().GetVersion()
	if version.GE(hostVer, "6.5") {
		info, err := self.acquireWebmksTicket("webmks")
		if err == nil {
			return info, nil
		}
	}
	return self.acquireVmrcUrl()
}

func (self *SVirtualMachine) GetVmrcInfo() (*cloudprovider.ServerVncOutput, error) {
	return self.acquireVmrcUrl()
}

func (self *SVirtualMachine) GetWebmksInfo() (*cloudprovider.ServerVncOutput, error) {
	return self.acquireWebmksTicket("webmks")
}

func (self *SVirtualMachine) acquireWebmksTicket(ticketType string) (*cloudprovider.ServerVncOutput, error) {
	vm := object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
	ticket, err := vm.AcquireTicket(self.manager.context, ticketType)
	if err != nil {
		return nil, err
	}

	host := ticket.Host
	if len(host) == 0 {
		host = self.manager.host
	}
	port := ticket.Port
	if port == 0 {
		port = int32(self.manager.port)
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

func (self *SVirtualMachine) acquireVmrcUrl() (*cloudprovider.ServerVncOutput, error) {
	ticket, err := self.manager.acquireCloneTicket()
	if err != nil {
		return nil, err
	}
	port := self.manager.port
	if port == 0 {
		port = 443
	}
	ret := &cloudprovider.ServerVncOutput{
		Url:      fmt.Sprintf("vmrc://clone:%s@%s:%d/?moid=%s", ticket, self.manager.host, port, self.GetId()),
		Protocol: "vmrc",
	}
	return ret, nil
}

func (self *SVirtualMachine) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedVMChangeConfig) error {
	return self.doChangeConfig(ctx, int32(config.Cpu), int64(config.MemoryMB), "", "")
}

func (self *SVirtualMachine) GetVersion() string {
	return self.getVirtualMachine().Config.Version
}

func (self *SVirtualMachine) doChangeConfig(ctx context.Context, ncpu int32, vmemMB int64, guestId string, version string) error {
	changed := false
	configSpec := types.VirtualMachineConfigSpec{}
	if int(ncpu) != self.GetVcpuCount() {
		configSpec.NumCPUs = ncpu
		changed = true
	}
	if int(vmemMB) != self.GetVmemSizeMB() {
		configSpec.MemoryMB = vmemMB
		changed = true
	}
	if len(guestId) > 0 && guestId != self.GetGuestId() {
		configSpec.GuestId = guestId
		changed = true
	}
	if len(version) > 0 && version != self.GetVersion() {
		configSpec.Version = version
		changed = true
	}
	if !changed {
		return nil
	}

	vm := self.getVmObj()

	task, err := vm.Reconfigure(ctx, configSpec)
	if err != nil {
		return err
	}
	err = task.Wait(ctx)
	if err != nil {
		return err
	}
	return self.Refresh()
}

func (self *SVirtualMachine) AssignSecurityGroup(secgroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SVirtualMachine) GetCreatedAt() time.Time {
	moVM := self.getVirtualMachine()
	ctm := moVM.Config.CreateDate
	if ctm != nil {
		return *ctm
	} else {
		return time.Time{}
	}
}

func (self *SVirtualMachine) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SVirtualMachine) UpdateUserData(userData string) error {
	return nil
}

func (self *SVirtualMachine) fetchHardwareInfo() error {
	self.vnics = make([]SVirtualNIC, 0)
	self.vdisks = make([]SVirtualDisk, 0)
	self.cdroms = make([]SVirtualCdrom, 0)
	self.devs = make(map[int32]SVirtualDevice)

	moVM := self.getVirtualMachine()

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
			self.vnics = append(self.vnics, NewVirtualNIC(self, dev, len(self.vnics)))
		} else if reflectutils.StructContains(devType, diskType) {
			self.vdisks = append(self.vdisks, NewVirtualDisk(self, dev, len(self.vdisks)))
		} else if reflectutils.StructContains(devType, vgaType) {
			self.vga = NewVirtualVGA(self, dev, 0)
		} else if reflectutils.StructContains(devType, cdromType) {
			self.cdroms = append(self.cdroms, NewVirtualCdrom(self, dev, len(self.cdroms)))
		}
		vdev := NewVirtualDevice(self, dev, 0)
		self.devs[vdev.getKey()] = vdev
	}
	self.rigorous()
	sort.Sort(byDiskType(self.vdisks))
	return nil
}

func (self *SVirtualMachine) rigorous() {
	hasRoot := false
	for i := range self.vdisks {
		if self.vdisks[i].IsRoot {
			hasRoot = true
			break
		}
	}
	if !hasRoot && len(self.vdisks) > 0 {
		self.vdisks[0].IsRoot = true
	}
}

func (self *SVirtualMachine) getVdev(key int32) SVirtualDevice {
	return self.devs[key]
}

func (self *SVirtualMachine) fetchGuestIps() map[string]string {
	guestIps := make(map[string]string)
	moVM := self.getVirtualMachine()
	for _, net := range moVM.Guest.Net {
		mac := netutils.FormatMacAddr(net.MacAddress)
		for _, ip := range net.IpAddress {
			if regutils.MatchIP4Addr(ip) {
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

func (self *SVirtualMachine) getGuestIps() map[string]string {
	if self.guestIps == nil {
		self.guestIps = self.fetchGuestIps()
	}
	return self.guestIps
}

func (self *SVirtualMachine) GetIps() []string {
	ips := make([]string, 0)
	for _, ip := range self.getGuestIps() {
		ips = append(ips, ip)
	}
	return ips
}

func (self *SVirtualMachine) GetVGADevice() string {
	return fmt.Sprintf("%s", self.vga.String())
}

var (
	driverTable = map[string][]string{
		"sata":   {"ahci"},
		"scsi":   {"parascsi", "lsilogic", "lsilogicsas", "buslogic"},
		"pvscsi": {"parascsi"},
		"ide":    {"ide"},
	}
)

func (self *SVirtualMachine) getDevsByDriver(driver string) []SVirtualDevice {
	devs := make([]SVirtualDevice, 0)
	for _, drv := range self.devs {
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

func (self *SVirtualMachine) FindController(ctx context.Context, driver string) ([]SVirtualDevice, error) {
	aliasDrivers, ok := driverTable[driver]
	if !ok {
		return nil, fmt.Errorf("Unsupported disk driver %s", driver)
	}
	var devs []SVirtualDevice
	for _, alias := range aliasDrivers {
		devs = self.getDevsByDriver(alias)
		if len(devs) > 0 {
			break
		}
	}
	return devs, nil
}

func (self *SVirtualMachine) FindDiskByDriver(drivers ...string) []SVirtualDisk {
	disks := make([]SVirtualDisk, 0)
	for i := range self.vdisks {
		if utils.IsInStringArray(self.vdisks[i].GetDriver(), drivers) {
			disks = append(disks, self.vdisks[i])
		}
	}
	return disks
}

func (self *SVirtualMachine) devNumWithCtrlKey(ctrlKey int32) int {
	n := 0
	for _, dev := range self.devs {
		if dev.getControllerKey() == ctrlKey {
			n++
		}
	}
	return n
}

func (self *SVirtualMachine) getLayoutEx() *types.VirtualMachineFileLayoutEx {
	vm := self.getVirtualMachine()
	if vm.LayoutEx != nil {
		return vm.LayoutEx
	}
	var nvm mo.VirtualMachine
	err := self.manager.reference2Object(vm.Self, vmLayoutExProps, &nvm)
	if err != nil {
		log.Errorf("unable to fetch LayoutEx.File from vc: %v", err)
	}
	vm.LayoutEx = nvm.LayoutEx
	return vm.LayoutEx
}

func (self *SVirtualMachine) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	if opts.Driver == "pvscsi" {
		opts.Driver = "scsi"
	}
	var ds *SDatastore
	var err error
	if opts.StorageId != "" {
		ihost := self.getIHost()
		if ihost == nil {
			return "", fmt.Errorf("unable to get host of virtualmachine %s", self.GetName())
		}
		ds, err = ihost.(*SHost).FindDataStoreById(opts.StorageId)
		if err != nil {
			return "", errors.Wrapf(err, "unable to find datastore %s", opts.StorageId)
		}
	}
	devs, err := self.FindController(ctx, opts.Driver)
	if err != nil {
		return "", err
	}
	if len(devs) == 0 {
		return "", self.createDriverAndDisk(ctx, ds, opts.SizeMb, opts.UUID, opts.Driver)
	}
	numDevBelowCtrl := make([]int, len(devs))
	for i := range numDevBelowCtrl {
		numDevBelowCtrl[i] = self.devNumWithCtrlKey(devs[i].getKey())
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
	diskKey := self.FindMinDiffKey(2000)

	// By default, the virtual SCSI controller is assigned to virtual device node (z:7),
	// so that device node is unavailable for hard disks or other devices.
	if unitNumber >= 7 && opts.Driver == "scsi" {
		unitNumber++
	}

	return "", self.createDiskInternal(ctx, SDiskConfig{
		SizeMb:        int64(opts.SizeMb),
		Uuid:          opts.UUID,
		UnitNumber:    int32(unitNumber),
		ControllerKey: ctrlKey,
		Key:           diskKey,
		Datastore:     ds,
	}, true)
}

// createDriverAndDisk will create a driver and disk associated with the driver
func (self *SVirtualMachine) createDriverAndDisk(ctx context.Context, ds *SDatastore, sizeMb int, uuid string, driver string) error {
	if driver != "scsi" && driver != "pvscsi" {
		return fmt.Errorf("Driver %s is not supported", driver)
	}

	deviceChange := make([]types.BaseVirtualDeviceConfigSpec, 0, 2)

	// find a suitable key for scsi or pvscsi driver
	scsiKey := self.FindMinDiffKey(1000)
	deviceChange = append(deviceChange, addDevSpec(NewSCSIDev(scsiKey, 100, driver)))

	// find a suitable key for disk
	diskKey := self.FindMinDiffKey(2000)

	if diskKey == scsiKey {
		// unarrivelable
		log.Errorf("there is no suitable key between 1000 and 2000???!")
	}

	return self.createDiskWithDeviceChange(ctx, deviceChange,
		SDiskConfig{
			SizeMb:        int64(sizeMb),
			Uuid:          uuid,
			ControllerKey: scsiKey,
			UnitNumber:    0,
			Key:           scsiKey,
			ImagePath:     "",
			IsRoot:        false,
			Datastore:     ds,
		}, true)
}

func (self *SVirtualMachine) copyRootDisk(ctx context.Context, imagePath string) (string, error) {
	layoutEx := self.getLayoutEx()
	if layoutEx == nil || len(layoutEx.File) == 0 {
		return "", fmt.Errorf("invalid LayoutEx")
	}
	file := layoutEx.File[0].Name
	// find stroage
	host := self.GetIHost()
	storages, err := host.GetIStorages()
	if err != nil {
		return "", errors.Wrap(err, "host.GetIStorages")
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
		return "", fmt.Errorf("can't find storage associated with vm %q", self.GetName())
	}
	path := datastore.cleanPath(file)
	vmDir := strings.Split(path, "/")[0]
	// TODO find a non-conflicting path
	newImagePath := datastore.getPathString(fmt.Sprintf("%s/%s.vmdk", vmDir, vmDir))

	fm := datastore.getDatastoreObj().NewFileManager(datastore.datacenter.getObjectDatacenter(), true)
	err = fm.Copy(ctx, imagePath, newImagePath)
	if err != nil {
		return "", errors.Wrap(err, "unable to copy system disk")
	}
	return newImagePath, nil
}

func (self *SVirtualMachine) createDiskWithDeviceChange(ctx context.Context, deviceChange []types.BaseVirtualDeviceConfigSpec, config SDiskConfig, check bool) error {
	var err error
	// copy disk
	if len(config.ImagePath) > 0 {
		config.IsRoot = true
		config.ImagePath, err = self.copyRootDisk(ctx, config.ImagePath)
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

	vmObj := self.getVmObj()

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
	oldDiskCnt := len(self.vdisks)
	maxTries := 60
	for tried := 0; tried < maxTries; tried += 1 {
		time.Sleep(time.Second)
		self.Refresh()
		if len(self.vdisks) > oldDiskCnt {
			return nil
		}
	}
	return cloudprovider.ErrTimeout
}

func (self *SVirtualMachine) createDiskInternal(ctx context.Context, config SDiskConfig, check bool) error {

	return self.createDiskWithDeviceChange(ctx, nil, config, check)
}

func (self *SVirtualMachine) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SVirtualMachine) GetProjectId() string {
	pool, err := self.getResourcePool()
	if err != nil {
		return ""
	}
	if pool != nil {
		return pool.GetId()
	}
	return ""
}

func (self *SVirtualMachine) GetError() error {
	return nil
}

func (self *SVirtualMachine) getResourcePool() (*SResourcePool, error) {
	vm := self.getVirtualMachine()
	morp := mo.ResourcePool{}
	if vm.ResourcePool == nil {
		return nil, errors.Error("nil resource pool")
	}
	err := self.manager.reference2Object(*vm.ResourcePool, RESOURCEPOOL_PROPS, &morp)
	if err != nil {
		return nil, errors.Wrap(err, "self.manager.reference2Object")
	}
	rp := NewResourcePool(self.manager, &morp, self.datacenter)
	return rp, nil
}

func (self *SVirtualMachine) CheckFileInfo(ctx context.Context) error {
	layoutEx := self.getLayoutEx()
	if layoutEx != nil && len(layoutEx.File) > 0 {
		file := layoutEx.File[0]
		host := self.GetIHost()
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

func (self *SVirtualMachine) DoRename(ctx context.Context, name string) error {
	task, err := self.getVmObj().Rename(ctx, name)
	if err != nil {
		return errors.Wrap(err, "object.VirtualMachine.Rename")
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) GetMoid() string {
	return self.getVirtualMachine().Self.Value
}

func (self *SVirtualMachine) GetToolsVersion() string {
	return self.getVirtualMachine().Guest.ToolsVersion
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

func (self *SVirtualMachine) DoCustomize(ctx context.Context, params jsonutils.JSONObject) error {
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
		osName string
		name   = "yunionhost"
	)
	if params.Contains("os_name") {
		osName, _ = params.GetString("os_name")
	}
	if params.Contains("name") {
		name, _ = params.GetString("name")
	}
	if osName == "Linux" {
		linuxPrep := types.CustomizationLinuxPrep{
			HostName: &types.CustomizationFixedName{Name: name},
			Domain:   domain,
			TimeZone: "Asia/Shanghai",
		}
		spec.Identity = &linuxPrep
	} else if osName == "Windows" {
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
					Name: name,
				},
			},
			Identification: types.CustomizationIdentification{},
		}
		spec.Identity = &sysPrep
	}
	log.Infof("customize spec: %#v", spec)
	task, err := self.getVmObj().Customize(ctx, *spec)
	if err != nil {
		return errors.Wrap(err, "object.VirtualMachine.Customize")
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) ExportTemplate(ctx context.Context, idx int, diskPath string) error {
	lease, err := self.getVmObj().Export(ctx)
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

func (self *SVirtualMachine) GetSerialOutput(port int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) ConvertPublicIpToEip() error {
	return cloudprovider.ErrNotSupported
}

func (self *SVirtualMachine) IsAutoRenew() bool {
	return false
}

func (self *SVirtualMachine) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SVirtualMachine) FindMinDiffKey(limit int32) int32 {
	if self.devs == nil {
		self.fetchHardwareInfo()
	}
	devKeys := make([]int32, 0, len(self.devs))
	for key := range self.devs {
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

func (self *SVirtualMachine) relocate(hostId string) error {
	var targetHs *mo.HostSystem
	if hostId == "" {
		return errors.Wrap(fmt.Errorf("require hostId"), "relocate")
	}
	ihost, err := self.manager.GetIHostById(hostId)
	if err != nil {
		return errors.Wrap(err, "self.manager.GetIHostById(hostId)")
	}
	targetHs = ihost.(*SHost).object.(*mo.HostSystem)
	if len(targetHs.Datastore) < 1 {
		return errors.Wrap(fmt.Errorf("target host has no datastore"), "relocate")
	}
	ctx := self.manager.context
	config := types.VirtualMachineRelocateSpec{}
	hrs := targetHs.Reference()
	config.Host = &hrs
	config.Datastore = &targetHs.Datastore[0]
	task, err := self.getVmObj().Relocate(ctx, config, types.VirtualMachineMovePriorityDefaultPriority)
	if err != nil {
		log.Errorf("vm.Migrate %s", err)
		return errors.Wrap(err, "SVirtualMachine Migrate")
	}
	err = task.Wait(ctx)
	if err != nil {
		log.Errorf("task.Wait %s", err)
		return errors.Wrap(err, "task.wait")
	}
	return nil
}

func (self *SVirtualMachine) MigrateVM(hostId string) error {
	return self.relocate(hostId)
}

func (self *SVirtualMachine) LiveMigrateVM(hostId string) error {
	return self.relocate(hostId)
}

func (self *SVirtualMachine) GetIHostId() string {
	ctx := self.manager.context
	hs, err := self.getVmObj().HostSystem(ctx)
	if err != nil {
		log.Errorf("get HostSystem %s", err)
		return ""
	}
	var moHost mo.HostSystem
	err = self.manager.reference2Object(hs.Reference(), HOST_SYSTEM_PROPS, &moHost)
	if err != nil {
		log.Errorf("hostsystem reference2Object %s", err)
		return ""
	}
	shost := NewHost(self.manager, &moHost, nil)
	return shost.GetGlobalId()
}

func (self *SVirtualMachine) IsTemplate() bool {
	movm := self.getVirtualMachine()
	if tempalteNameRegex != nil && tempalteNameRegex.MatchString(self.GetName()) && movm.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff {
		return true
	}
	return movm.Config != nil && movm.Config.Template
}

func (self *SVirtualMachine) fetchSnapshots() {
	movm := self.getVirtualMachine()
	if movm.Snapshot == nil {
		return
	}
	self.snapshots = self.extractSnapshots(movm.Snapshot.RootSnapshotList, make([]SVirtualMachineSnapshot, 0, len(movm.Snapshot.RootSnapshotList)))
}

func (self *SVirtualMachine) extractSnapshots(tree []types.VirtualMachineSnapshotTree, snapshots []SVirtualMachineSnapshot) []SVirtualMachineSnapshot {
	for i := range tree {
		snapshots = append(snapshots, SVirtualMachineSnapshot{
			snapshotTree: tree[i],
			vm:           self,
		})
		snapshots = self.extractSnapshots(tree[i].ChildSnapshotList, snapshots)
	}
	return snapshots
}

func (self *SVirtualMachine) GetInstanceSnapshots() ([]cloudprovider.ICloudInstanceSnapshot, error) {
	if self.snapshots == nil {
		self.fetchSnapshots()
	}
	ret := make([]cloudprovider.ICloudInstanceSnapshot, 0, len(self.snapshots))
	for i := range self.snapshots {
		ret = append(ret, &self.snapshots[i])
	}
	return ret, nil
}

func (self *SVirtualMachine) GetInstanceSnapshot(idStr string) (cloudprovider.ICloudInstanceSnapshot, error) {
	if self.snapshots == nil {
		self.fetchSnapshots()
	}
	for i := range self.snapshots {
		if self.snapshots[i].GetGlobalId() == idStr {
			// copyone
			sp := self.snapshots[i]
			return &sp, nil
		}
	}
	return nil, errors.ErrNotFound
}

func (self *SVirtualMachine) CreateInstanceSnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudInstanceSnapshot, error) {
	ovm := self.getVmObj()
	task, err := ovm.CreateSnapshot(ctx, name, desc, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "CreateSnapshot")
	}
	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "task.Wait")
	}
	sp := info.Result.(types.ManagedObjectReference)
	err = self.Refresh()
	if err != nil {
		return nil, errors.Wrap(err, "create successfully")
	}
	self.fetchSnapshots()
	for i := range self.snapshots {
		if self.snapshots[i].snapshotTree.Snapshot == sp {
			// copyone
			sp := self.snapshots[i]
			return &sp, nil
		}
	}
	return nil, errors.Wrap(errors.ErrNotFound, "create successfully")
}

func (self *SVirtualMachine) ResetToInstanceSnapshot(ctx context.Context, idStr string) error {
	cloudIsp, err := self.GetInstanceSnapshot(idStr)
	if err != nil {
		return errors.Wrap(err, "GetInstanceSnapshot")
	}
	isp := cloudIsp.(*SVirtualMachineSnapshot)
	req := types.RevertToSnapshot_Task{
		This: isp.snapshotTree.Snapshot.Reference(),
	}
	res, err := methods.RevertToSnapshot_Task(ctx, self.manager.client.Client, &req)
	if err != nil {
		return errors.Wrap(err, "RevertToSnapshot_Task")
	}
	return object.NewTask(self.manager.client.Client, res.Returnval).Wait(ctx)
}
