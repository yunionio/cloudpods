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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	cloudtypes "yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/version"
)

var VIRTUAL_MACHINE_PROPS = []string{"name", "parent", "runtime", "summary", "config", "guest", "resourcePool", "layoutEx"}

type SVirtualMachine struct {
	multicloud.SInstanceBase
	SManagedObject

	vnics  []SVirtualNIC
	vdisks []SVirtualDisk
	vga    SVirtualVGA
	cdroms []SVirtualCdrom
	devs   map[int32]SVirtualDevice
	ihost  cloudprovider.ICloudHost

	guestIps map[string]string
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

func (self *SVirtualMachine) GetMetadata() *jsonutils.JSONDict {
	meta := jsonutils.NewDict()
	meta.Set("datacenter", jsonutils.NewString(self.GetDatacenterPathString()))
	rp, _ := self.getResourcePool()
	if rp != nil {
		rpPath := rp.GetPath()
		rpOffset := -1
		for i := range rpPath {
			if rpPath[i] == "Resources" {
				if i > 0 {
					meta.Set("cluster", jsonutils.NewString(rpPath[i-1]))
					rpOffset = i
				}
			} else if rpOffset >= 0 && i > rpOffset {
				meta.Set(fmt.Sprintf("pool%d", i-rpOffset-1), jsonutils.NewString(rpPath[i]))
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

func (self *SVirtualMachine) GetStatus() string {
	err := self.CheckFileInfo(context.Background())
	if err != nil {
		return api.VM_UNKNOWN
	}
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
	index := disk.index
	diskKey := disk.getKey()
	ctlKey := disk.getControllerKey()

	err := self.doDetachAndDeleteDisk(ctx, disk)
	if err != nil {
		return err
	}
	return self.createDiskInternal(ctx, sizeMb, uuid, int32(index), diskKey, ctlKey, imagePath, false)
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
	sort.Sort(byDiskType(self.vdisks))
	for i := 0; i < len(self.vdisks); i += 1 {
		idisks[i] = &(self.vdisks[i])
	}
	return idisks, nil
}

func (self *SVirtualMachine) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	for i := 0; i < len(self.vdisks); i += 1 {
		if self.vdisks[i].GetGlobalId() == idStr {
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

func (self *SVirtualMachine) GetOSType() string {
	if osInfo, ok := GuestOsInfo[self.GetGuestId()]; ok {
		return string(osInfo.OsType)
	}
	return ""
}

func (self *SVirtualMachine) GetOSName() string {
	if osInfo, ok := GuestOsInfo[self.GetGuestId()]; ok {
		return osInfo.OsDistribution
	}
	return ""
}

func (self *SVirtualMachine) GetOSVersion() string {
	if osInfo, ok := GuestOsInfo[self.GetGuestId()]; ok {
		return osInfo.OsVersion
	}
	return ""
}

func (self *SVirtualMachine) GetOsArch() string {
	if osInfo, ok := GuestOsInfo[self.GetGuestId()]; ok {
		return string(osInfo.OsArch)
	}
	return ""
}

func (self *SVirtualMachine) GetOsDistribution() string {
	if osInfo, ok := GuestOsInfo[self.GetGuestId()]; ok {
		return osInfo.OsDistribution
	}
	return ""
}

func (self *SVirtualMachine) GetBios() string {
	// self.obj.config.firmware
	switch self.getVirtualMachine().Config.Firmware {
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
		return errors.Wrap(httperrors.ErrInvalidStatus, "no valid host")
	}

	lockman.LockRawObject(ctx, "host", ihost.GetGlobalId())
	defer lockman.ReleaseRawObject(ctx, "host", ihost.GetGlobalId())

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

func (self *SVirtualMachine) StopVM(ctx context.Context, isForce bool) error {
	if self.GetStatus() == api.VM_READY {
		return nil
	}
	if !isForce && self.isToolsOk() {
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

func (self *SVirtualMachine) doDelete(ctx context.Context) error {
	vm := self.getVmObj()
	// detach all disks first
	for i := range self.vdisks {
		err := self.doDetachAndDeleteDisk(ctx, &self.vdisks[i])
		if err != nil {
			return errors.Wrap(err, "doDetachAndDeteteDisk")
		}
	}

	task, err := vm.Destroy(ctx)
	if err != nil {
		log.Errorf("vm.Destroy(ctx) fail %s", err)
		return err
	}
	return task.Wait(ctx)
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
	return self.doDelete(ctx)
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

func (self *SVirtualMachine) GetVNCInfo() (jsonutils.JSONObject, error) {
	hostVer := self.GetIHost().GetVersion()
	if version.GE(hostVer, "6.5") {
		info, err := self.acquireWebmksTicket("webmks")
		if err == nil {
			return info, nil
		}
	}
	return self.acquireVmrcUrl()
}

func (self *SVirtualMachine) GetVmrcInfo() (jsonutils.JSONObject, error) {
	return self.acquireVmrcUrl()
}

func (self *SVirtualMachine) GetWebmksInfo() (jsonutils.JSONObject, error) {
	return self.acquireWebmksTicket("webmks")
}

func (self *SVirtualMachine) acquireWebmksTicket(ticketType string) (jsonutils.JSONObject, error) {
	vm := object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
	ticket, err := vm.AcquireTicket(self.manager.context, ticketType)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()

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
	url := fmt.Sprintf("wss://%s:%d/ticket/%s", host, port, ticket.Ticket)
	ret.Add(jsonutils.NewString("wmks"), "protocol")
	ret.Add(jsonutils.NewString(url), "url")
	return ret, nil
}

func (self *SVirtualMachine) acquireVmrcUrl() (jsonutils.JSONObject, error) {
	ticket, err := self.manager.acquireCloneTicket()
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString("vmrc"), "protocol")
	port := self.manager.port
	if port == 0 {
		port = 443
	}
	url := fmt.Sprintf("vmrc://clone:%s@%s:%d/?moid=%s", ticket, self.manager.host, port, self.GetId())
	ret.Add(jsonutils.NewString(url), "url")
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

	MAX_TRIES := 3
	for tried := 0; tried < MAX_TRIES && (moVM == nil || moVM.Config == nil || moVM.Config.Hardware.Device == nil); tried += 1 {
		time.Sleep(time.Second)
	}

	if moVM == nil || moVM.Config == nil || moVM.Config.Hardware.Device == nil {
		return fmt.Errorf("invalid vm")
	}

	for i := 0; i < len(moVM.Config.Hardware.Device); i += 1 {
		dev := moVM.Config.Hardware.Device[i]
		devType := reflect.Indirect(reflect.ValueOf(dev)).Type()

		etherType := reflect.TypeOf((*types.VirtualEthernetCard)(nil)).Elem()
		diskType := reflect.TypeOf((*types.VirtualDisk)(nil)).Elem()
		vgaType := reflect.TypeOf((*types.VirtualMachineVideoCard)(nil)).Elem()
		cdromType := reflect.TypeOf((*types.VirtualCdrom)(nil)).Elem()

		if reflectutils.StructContains(devType, etherType) {
			self.vnics = append(self.vnics, NewVirtualNIC(self, dev, len(self.vnics)))
		} else if reflectutils.StructContains(devType, diskType) {
			self.vdisks = append(self.vdisks, NewVirtualDisk(self, dev, len(self.vnics)))
		} else if reflectutils.StructContains(devType, vgaType) {
			self.vga = NewVirtualVGA(self, dev, 0)
		} else if reflectutils.StructContains(devType, cdromType) {
			self.cdroms = append(self.cdroms, NewVirtualCdrom(self, dev, len(self.cdroms)))
		}
		vdev := NewVirtualDevice(self, dev, 0)
		self.devs[vdev.getKey()] = vdev
	}
	// sort disk based on index
	sort.Slice(self.vdisks, func(i, j int) bool {
		return self.vdisks[i].GetIndex() < self.vdisks[j].GetIndex()
	})
	return nil
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

func (self *SVirtualMachine) CreateDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
	if driver == "pvscsi" {
		driver = "scsi"
	}
	devs, err := self.FindController(ctx, driver)
	if err != nil {
		return err
	}
	if len(devs) == 0 {
		return self.createDriverAndDisk(ctx, sizeMb, uuid, driver)
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
	if unitNumber >= 7 && driver == "scsi" {
		unitNumber++
	}

	return self.createDiskInternal(ctx, sizeMb, uuid, int32(unitNumber), diskKey, ctrlKey, "", true)
}

// createDriverAndDisk will create a driver and disk associated with the driver
func (self *SVirtualMachine) createDriverAndDisk(ctx context.Context, sizeMb int, uuid string, driver string) error {
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

	return self.createDiskWithDeviceChange(ctx, deviceChange, sizeMb, uuid, 0, diskKey, scsiKey, "", true)
}

func (self *SVirtualMachine) createDiskWithDeviceChange(ctx context.Context,
	deviceChange []types.BaseVirtualDeviceConfigSpec, sizeMb int,
	uuid string, index int32, diskKey int32, ctlKey int32, imagePath string, check bool) error {

	devSpec := NewDiskDev(int64(sizeMb), imagePath, uuid, index, 0, ctlKey, diskKey)
	spec := addDevSpec(devSpec)
	spec.FileOperation = types.VirtualDeviceConfigSpecFileOperationCreate
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

func (self *SVirtualMachine) createDiskInternal(ctx context.Context, sizeMb int, uuid string, index int32,
	diskKey int32, ctlKey int32, imagePath string, check bool) error {

	return self.createDiskWithDeviceChange(ctx, nil, sizeMb, uuid, index, diskKey, ctlKey, imagePath, check)
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
	vm := self.getVirtualMachine()
	if vm.LayoutEx != nil && len(vm.LayoutEx.File) > 0 {
		file := vm.LayoutEx.File[0]
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

func (self *SVirtualMachine) DoCustomize(ctx context.Context, params jsonutils.JSONObject) error {
	spec := new(types.CustomizationSpec)

	ipSettings := new(types.CustomizationGlobalIPSettings)
	domain := "local"
	if params.Contains("domain") {
		domain, _ = params.GetString("domain")
	}
	ipSettings.DnsSuffixList = []string{domain}

	// deal nics
	nics, _ := params.GetArray("nics")
	serverNics := make([]cloudtypes.SServerNic, len(nics))
	for i := range nics {
		var nicType cloudtypes.SServerNic
		nics[i].Unmarshal(&nicType)
		serverNics[i] = nicType
	}

	// find dnsServerList
	for i := range serverNics {
		dnsList := netutils2.GetNicDns(&serverNics[i])
		if len(dnsList) != 0 {
			ipSettings.DnsServerList = dnsList
		}
	}
	spec.GlobalIPSettings = *ipSettings

	maps := make([]types.CustomizationAdapterMapping, 0, len(nics))
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
		mask := netutils2.Netlen2Mask(maskLen)
		conf.Adapter.SubnetMask = mask

		if len(nic.Gateway) != 0 {
			conf.Adapter.Gateway = []string{nic.Gateway}
		}
		dnsList := netutils2.GetNicDns(&nic)
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
		osName = "Linux"
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
	} else {
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
	log.Debugf("download to %s start...", diskPath)
	err = lease.DownloadFile(ctx, diskPath, info.Items[idx], soap.Download{Progress: lr})
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

func (self *SVirtualMachine) SetAutoRenew(autoRenew bool) error {
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
	return movm.Config != nil && movm.Config.Template
}
