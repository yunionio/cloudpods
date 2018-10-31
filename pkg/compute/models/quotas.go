package models

import (
	"errors"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/tristate"
)

var QuotaManager *quotas.SQuotaManager

func init() {
	dbStore := quotas.NewDBQuotaStore()
	pendingStore := quotas.NewMemoryQuotaStore()

	QuotaManager = quotas.NewQuotaManager("quotas", SQuota{}, dbStore, pendingStore)
}

var (
	ErrOutOfCPU            = errors.New("out of CPU quota")
	ErrOutOfMemory         = errors.New("out of memory quota")
	ErrOutOfStorage        = errors.New("out of storage quota")
	ErrOutOfPort           = errors.New("out of internal port quota")
	ErrOutOfEip            = errors.New("out of eip quota")
	ErrOutOfEport          = errors.New("out of external port quota")
	ErrOutOfBw             = errors.New("out of internal bandwidth quota")
	ErrOutOfEbw            = errors.New("out of external bandwidth quota")
	ErrOutOfKeypair        = errors.New("out of keypair quota")
	ErrOutOfImage          = errors.New("out of image quota")
	ErrOutOfGroup          = errors.New("out of group quota")
	ErrOutOfSecgroup       = errors.New("out of secgroup quota")
	ErrOutOfIsolatedDevice = errors.New("out of isolated device quota")
)

type SQuota struct {
	Cpu            int
	Memory         int
	Storage        int
	Port           int
	Eip            int
	Eport          int
	Bw             int
	Ebw            int
	Keypair        int
	Image          int
	Group          int
	Secgroup       int
	IsolatedDevice int
}

func (self *SQuota) FetchSystemQuota() {
	self.Cpu = options.Options.DefaultCpuQuota
	self.Memory = options.Options.DefaultMemoryQuota
	self.Storage = options.Options.DefaultStorageQuota
	self.Port = options.Options.DefaultPortQuota
	self.Eip = options.Options.DefaultEipQuota
	self.Eport = options.Options.DefaultEportQuota
	self.Bw = options.Options.DefaultBwQuota
	self.Ebw = options.Options.DefaultEbwQuota
	self.Keypair = options.Options.DefaultKeypairQuota
	self.Image = options.Options.DefaultImageQuota
	self.Group = options.Options.DefaultGroupQuota
	self.Secgroup = options.Options.DefaultSecgroupQuota
	self.IsolatedDevice = options.Options.DefaultIsolatedDeviceQuota
}

func (self *SQuota) FetchUsage(projectId string) error {
	diskSize := totalDiskSize(projectId, tristate.None, tristate.None, false)
	net := totalGuestNicCount(projectId, nil, false)
	guest := totalGuestResourceCount(projectId, nil, nil, nil, false, false, "")
	eipUsage := ElasticipManager.TotalCount(projectId, nil, nil)
	// XXX
	// keypair belongs to user
	// keypair := totalKeypairCount(projectId)

	self.Cpu = guest.TotalCpuCount
	self.Memory = guest.TotalMemSize
	self.Storage = diskSize
	self.Port = net.InternalNicCount + net.InternalVirtualNicCount
	self.Eip = eipUsage.Total()
	self.Eport = net.ExternalNicCount + net.ExternalVirtualNicCount
	self.Bw = net.InternalBandwidth
	self.Ebw = net.ExternalBandwidth
	self.Keypair = 0 // keypair
	s := auth.GetAdminSession("", "")
	self.Image, _ = modules.Images.GetPrivateImageCount(s, projectId, true)
	self.Group = 0
	self.Secgroup = totalSecurityGroupCount(projectId)
	self.IsolatedDevice = guest.TotalIsolatedCount

	return nil
}

func (self *SQuota) IsEmpty() bool {
	if self.Cpu > 0 {
		return false
	}
	if self.Memory > 0 {
		return false
	}
	if self.Storage > 0 {
		return false
	}
	if self.Port > 0 {
		return false
	}
	if self.Eip > 0 {
		return false
	}
	if self.Eport > 0 {
		return false
	}
	if self.Bw > 0 {
		return false
	}
	if self.Ebw > 0 {
		return false
	}
	if self.Keypair > 0 {
		return false
	}
	if self.Image > 0 {
		return false
	}
	if self.Group > 0 {
		return false
	}
	if self.Secgroup > 0 {
		return false
	}
	if self.IsolatedDevice > 0 {
		return false
	}
	return true
}

func (self *SQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Cpu = self.Cpu + squota.Cpu
	self.Memory = self.Memory + squota.Memory
	self.Storage = self.Storage + squota.Storage
	self.Port = self.Port + squota.Port
	self.Eip = self.Eip + squota.Eip
	self.Eport = self.Eport + squota.Eport
	self.Bw = self.Bw + squota.Bw
	self.Ebw = self.Ebw + squota.Ebw
	self.Keypair = self.Keypair + squota.Keypair
	self.Image = self.Image + squota.Image
	self.Group = self.Group + squota.Group
	self.Secgroup = self.Secgroup + squota.Secgroup
	self.IsolatedDevice = self.IsolatedDevice + squota.IsolatedDevice
}

func nonNegative(val int) int {
	if val < 0 {
		return 0
	} else {
		return val
	}
}

func (self *SQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	self.Cpu = nonNegative(self.Cpu - squota.Cpu)
	self.Memory = nonNegative(self.Memory - squota.Memory)
	self.Storage = nonNegative(self.Storage - squota.Storage)
	self.Port = nonNegative(self.Port - squota.Port)
	self.Eip = nonNegative(self.Eip - squota.Eip)
	self.Eport = nonNegative(self.Eport - squota.Eport)
	self.Bw = nonNegative(self.Bw - squota.Bw)
	self.Ebw = nonNegative(self.Ebw - squota.Ebw)
	self.Keypair = nonNegative(self.Keypair - squota.Keypair)
	self.Image = nonNegative(self.Image - squota.Image)
	self.Group = nonNegative(self.Group - squota.Group)
	self.Secgroup = nonNegative(self.Secgroup - squota.Secgroup)
	self.IsolatedDevice = nonNegative(self.IsolatedDevice - squota.IsolatedDevice)
}

func (self *SQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SQuota)
	if squota.Cpu > 0 {
		self.Cpu = squota.Cpu
	}
	if squota.Memory > 0 {
		self.Memory = squota.Memory
	}
	if squota.Storage > 0 {
		self.Storage = squota.Storage
	}
	if squota.Port > 0 {
		self.Port = squota.Port
	}
	if squota.Eip > 0 {
		self.Eip = squota.Eip
	}
	if squota.Eport > 0 {
		self.Eport = squota.Eport
	}
	if squota.Bw > 0 {
		self.Bw = squota.Bw
	}
	if squota.Ebw > 0 {
		self.Ebw = squota.Ebw
	}
	if squota.Keypair > 0 {
		self.Keypair = squota.Keypair
	}
	if squota.Image > 0 {
		self.Image = squota.Image
	}
	if squota.Group > 0 {
		self.Group = squota.Group
	}
	if squota.Secgroup > 0 {
		self.Secgroup = squota.Secgroup
	}
	if squota.IsolatedDevice > 0 {
		self.IsolatedDevice = squota.IsolatedDevice
	}
}

func (self *SQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	sreq := request.(*SQuota)
	squota := quota.(*SQuota)
	if sreq.Cpu > 0 && self.Cpu > squota.Cpu {
		return ErrOutOfCPU
	}
	if sreq.Memory > 0 && self.Memory > squota.Memory {
		return ErrOutOfMemory
	}
	if sreq.Storage > 0 && self.Storage > squota.Storage {
		return ErrOutOfStorage
	}
	if sreq.Port > 0 && self.Port > squota.Port {
		return ErrOutOfPort
	}
	if sreq.Eip > 0 && self.Eip > squota.Eip {
		return ErrOutOfEip
	}
	if sreq.Eport > 0 && self.Eport > squota.Eport {
		return ErrOutOfEport
	}
	if sreq.Bw > 0 && self.Bw > squota.Bw {
		return ErrOutOfBw
	}
	if sreq.Ebw > 0 && self.Ebw > squota.Ebw {
		return ErrOutOfEbw
	}
	if sreq.Keypair > 0 && self.Keypair > squota.Keypair {
		return ErrOutOfKeypair
	}
	if sreq.Image > 0 && self.Image > squota.Image {
		return ErrOutOfImage
	}
	if sreq.Group > 0 && self.Group > squota.Group {
		return ErrOutOfGroup
	}
	if sreq.Secgroup > 0 && self.Secgroup > squota.Secgroup {
		return ErrOutOfSecgroup
	}
	if sreq.IsolatedDevice > 0 && self.IsolatedDevice > squota.IsolatedDevice {
		return ErrOutOfIsolatedDevice
	}
	return nil
}

func keyName(prefix, name string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("%s.%s", prefix, name)
	} else {
		return name
	}
}

func (self *SQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	if self.Cpu > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Cpu)), keyName(prefix, "cpu"))
	}
	if self.Memory > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Memory)), keyName(prefix, "memory"))
	}
	if self.Storage > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Storage)), keyName(prefix, "storage"))
	}
	if self.Port > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Port)), keyName(prefix, "port"))
	}
	if self.Eip > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Eip)), keyName(prefix, "eip"))
	}
	if self.Eport > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Eport)), keyName(prefix, "eport"))
	}
	if self.Bw > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Bw)), keyName(prefix, "bw"))
	}
	if self.Ebw > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Ebw)), keyName(prefix, "ebw"))
	}
	if self.Keypair > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Keypair)), keyName(prefix, "keypair"))
	}
	if self.Image > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Image)), keyName(prefix, "image"))
	}
	if self.Group > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Group)), keyName(prefix, "group"))
	}
	if self.Secgroup > 0 {
		ret.Add(jsonutils.NewInt(int64(self.Secgroup)), keyName(prefix, "secgroup"))
	}
	if self.IsolatedDevice > 0 {
		ret.Add(jsonutils.NewInt(int64(self.IsolatedDevice)), keyName(prefix, "isolated_device"))
	}
	return ret
}
