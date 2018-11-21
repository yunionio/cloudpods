package esxi

import (
	"context"
	"fmt"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	"reflect"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/yunioncloud/pkg/util/regutils"
)

var VIRTUAL_MACHINE_PROPS = []string{"name", "parent", "runtime", "summary", "config", "guest"}

type SVirtualMachine struct {
	SManagedObject

	vnics  []SVirtualNIC
	vdisks []SVirtualDisk
	vga    SVirtualVGA
	cdroms []SVirtualCdrom
	devs   map[int32]SVirtualDevice

	guestIps map[string]string

	host *SHost
}

func NewVirtualMachine(manager *SESXiClient, vm *mo.VirtualMachine, dc *SDatacenter, host *SHost) *SVirtualMachine {
	svm := &SVirtualMachine{SManagedObject: newManagedObject(manager, vm, dc), host: host}
	svm.fetchHardwareInfo()
	return svm
}

func (self *SVirtualMachine) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SVirtualMachine) getVirtualMachine() *mo.VirtualMachine {
	return self.object.(*mo.VirtualMachine)
}

func (self *SVirtualMachine) GetGlobalId() string {
	return self.getUuid()
}

func (self *SVirtualMachine) SyncSecurityGroup(secgroupId, name string, rules []secrules.SecurityRule) error {
	return nil
}

func (self *SVirtualMachine) GetStatus() string {
	vm := object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
	state, err := vm.PowerState(self.manager.context)
	if err != nil {
		return models.VM_UNKNOWN
	}
	switch state {
	case types.VirtualMachinePowerStatePoweredOff:
		return models.VM_READY
	case types.VirtualMachinePowerStatePoweredOn:
		return models.VM_RUNNING
	case types.VirtualMachinePowerStateSuspended:
		return models.VM_SUSPEND
	default:
		return models.VM_UNKNOWN
	}
}

func (self *SVirtualMachine) Refresh() error {
	return nil
}

func (self *SVirtualMachine) IsEmulated() bool {
	return false
}

func (self *SVirtualMachine) DeployVM(ctx context.Context, name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) RebuildRoot(ctx context.Context, imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) DetachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) AttachDisk(ctx context.Context, diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) getUuid() string {
	return self.getVirtualMachine().Summary.Config.Uuid
}

func (self *SVirtualMachine) GetCreateTime() time.Time {
	moVM := self.getVirtualMachine()
	ctm := moVM.Config.CreateDate
	if ctm != nil {
		return *ctm
	} else {
		return time.Time{}
	}
}

func (self *SVirtualMachine) GetIHost() cloudprovider.ICloudHost {
	// moVM := self.getVirtualMachine()
	// log.Debugf("%#v", moVM.Parent)
	me := self.findInParents("HostSystem")
	if me == nil {
		log.Errorf("fail to find vm host??? %s", self.GetName())
		return self.host
	}
	ihost, err := self.manager.FindHostByMoId(me.Self.Value)
	if err != nil {
		log.Errorf("fail to find host %s for vm %s???", me.Self.Value, self.GetName())
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

func (self *SVirtualMachine) GetVcpuCount() int8 {
	return int8(self.getVirtualMachine().Summary.Config.NumCpu)
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

func (self *SVirtualMachine) GetGuestToolsRunningStatus() string {
	moVM := self.getVirtualMachine()
	return string(moVM.Guest.ToolsRunningStatus)
}

func (self *SVirtualMachine) GetOSType() string {
	return ""
}

func (self *SVirtualMachine) GetOSName() string {
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
	return models.HYPERVISOR_ESXI
}

func (self *SVirtualMachine) getVmObj() *object.VirtualMachine {
	return object.NewVirtualMachine(self.manager.client.Client, self.getVirtualMachine().Self)
}

func (self *SVirtualMachine) StartVM(ctx context.Context) error {
	err := self.makeNicsStartConnected(ctx)
	if err != nil {
		return err
	}
	task, err := self.getVmObj().PowerOn(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) makeNicsStartConnected(ctx context.Context) error {
	spec := types.VirtualMachineConfigSpec{}
	spec.DeviceChange = make([]types.BaseVirtualDeviceConfigSpec, len(self.vnics))
	for i := 0; i < len(self.vnics); i += 1 {
		spec.DeviceChange[i] = makeNicStartConnected(&self.vnics[i])
	}
	task, err := self.getVmObj().Reconfigure(ctx, spec)
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
	task, err := self.getVmObj().PowerOff(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) doDelete(ctx context.Context) error {
	task, err := self.getVmObj().Destroy(ctx)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

func (self *SVirtualMachine) DeleteVM(ctx context.Context) error {
	for i := 0; i < len(self.vdisks); i += 1 {
		err := self.doDetachAndDeleteDisk(ctx, &self.vdisks[i])
		if err != nil {
			return err
		}
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

	task, err := self.getVmObj().Reconfigure(ctx, spec)
	if err != nil {
		return err
	}

	err = task.Wait(ctx)
	if err != nil {
		return err
	}

	if !remove {
		return nil
	}

	return vdisk.Delete(ctx)
}

func (self *SVirtualMachine) GetVNCInfo() (jsonutils.JSONObject, error) {
	info, err := self.acquireWebmksTicket("webmks")
	if err != nil {
		info, err = self.acquireVmrcUrl()
	}
	return info, err
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
	/*
		ret.Add(jsonutils.NewString(ticketType), "type")
		ret.Add(jsonutils.NewString(ticket.Host), "host")
		ret.Add(jsonutils.NewInt(int64(ticket.Port)), "port")
		ret.Add(jsonutils.NewString(ticket.Ticket), "ticket")
		ret.Add(jsonutils.NewString(ticket.SslThumbprint), "slThumbprint")
		ret.Add(jsonutils.NewString(ticket.CfgFile), "cfgFile")
	*/
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

func (dc *SVirtualMachine) ChangeConfig(ctx context.Context, instanceId string, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
}

func (self *SVirtualMachine) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SVirtualMachine) UpdateUserData(userData string) error {
	return nil
}

func (self *SVirtualMachine) fetchHardwareInfo() {
	self.vnics = make([]SVirtualNIC, 0)
	self.vdisks = make([]SVirtualDisk, 0)
	self.cdroms = make([]SVirtualCdrom, 0)
	self.devs = make(map[int32]SVirtualDevice)

	moVM := self.getVirtualMachine()

	for i := 0; i < len(moVM.Config.Hardware.Device); i += 1 {
		dev := moVM.Config.Hardware.Device[i]
		devType := reflect.Indirect(reflect.ValueOf(dev)).Type()

		etherType := reflect.TypeOf((*types.VirtualEthernetCard)(nil)).Elem()
		diskType := reflect.TypeOf((*types.VirtualDisk)(nil)).Elem()
		vgaType := reflect.TypeOf((*types.VirtualMachineVideoCard)(nil)).Elem()
		cdromType := reflect.TypeOf((*types.VirtualCdrom)(nil)).Elem()

		if StructContains(devType, etherType) {
			self.vnics = append(self.vnics, NewVirtualNIC(self, dev, len(self.vnics)))
		} else if StructContains(devType, diskType) {
			self.vdisks = append(self.vdisks, NewVirtualDisk(self, dev, len(self.vnics)))
		} else if StructContains(devType, vgaType) {
			self.vga = NewVirtualVGA(self, dev, 0)
		} else if StructContains(devType, cdromType) {
			self.cdroms = append(self.cdroms, NewVirtualCdrom(self, dev, len(self.cdroms)))
		}
		vdev := NewVirtualDevice(self, dev, 0)
		self.devs[vdev.getKey()] = vdev
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
