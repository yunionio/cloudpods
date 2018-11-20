package esxi

import (
	"fmt"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

var VIRTUAL_MACHINE_PROPS = []string{"name", "parent", "runtime", "summary"}

type SVirtualMachine struct {
	SManagedObject

	host *SHost
}

func NewVirtualMachine(manager *SESXiClient, vm *mo.VirtualMachine, dc *SDatacenter, host *SHost) *SVirtualMachine {
	return &SVirtualMachine{SManagedObject: newManagedObject(manager, vm, dc), host: host}
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
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) IsEmulated() bool {
	return false
}

func (self *SVirtualMachine) GetInstanceType() string {
	return ""
}

func (self *SVirtualMachine) DeployVM(name string, password string, publicKey string, deleteKeypair bool, description string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) RebuildRoot(imageId string, passwd string, publicKey string, sysSizeGB int) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) UpdateVM(name string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) DetachDisk(diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) AttachDisk(diskId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) getUuid() string {
	return self.getVirtualMachine().Summary.Config.Uuid
}

func (self *SVirtualMachine) GetCreateTime() time.Time {
	return time.Time{}
}

func (self *SVirtualMachine) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SVirtualMachine) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) GetINics() ([]cloudprovider.ICloudNic, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, nil
}

func (self *SVirtualMachine) GetVcpuCount() int8 {
	// ret = self.obj.summary.config.numCpu
	return int8(self.getVirtualMachine().Summary.Config.NumCpu)
}

func (self *SVirtualMachine) GetVmemSizeMB() int {
	// self.obj.summary.config.memorySizeMB
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

// GetSecurityGroup() ICloudSecurityGroup

func (self *SVirtualMachine) StartVM() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) StopVM(isForce bool) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SVirtualMachine) DeleteVM() error {
	return cloudprovider.ErrNotImplemented
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

func (dc *SVirtualMachine) ChangeConfig(instanceId string, ncpu int, vmem int) error {
	return cloudprovider.ErrNotImplemented
}

func (dc *SVirtualMachine) ChangeConfig2(instanceId string, instanceType string) error {
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
