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

package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var (
	rxIso            = regexp.MustCompile(`(.*?),media`)
	rxDeviceID       = regexp.MustCompile(`\d+`)
	rxDiskName       = regexp.MustCompile(`(virtio|scsi|sata|ide)\d+`)
	rxDiskType       = regexp.MustCompile(`\D+`)
	rxUnusedDiskName = regexp.MustCompile(`^(unused)\d+`)
	rxNicName        = regexp.MustCompile(`net\d+`)
	rxSerialName     = regexp.MustCompile(`serial\d+`)
	rxUsbName        = regexp.MustCompile(`usb\d+`)
)

type (
	QemuDevices     map[string]map[string]interface{}
	QemuDevice      map[string]interface{}
	QemuDeviceParam []string
)

type Intermediate struct {
	HardwareAddress string `json:"hardware-address"`
	IPAddresses     []struct {
		IPAddress     string `json:"ip-address"`
		IPAddressType string `json:"ip-address-type"`
		Prefix        int    `json:"prefix"`
	} `json:"ip-addresses"`
	Name       string           `json:"name"`
	Statistics map[string]int64 `json:"statistics"`
}

type VmBase struct {
	Name         string `json:"name"`
	Description  string `json:"Description"`
	Tags         string `json:"tags"`
	Args         string `json:"args"`
	Bios         string `json:"bios"`
	OnBoot       int    `json:"onboot"`
	Startup      string `json:"startup"`
	Tablet       int    `json:"tablet"`
	Ostype       string `json:"ostype"`
	Memory       int64  `json:"memory"`
	Balloon      int64  `json:"balloon"`
	Cores        int64  `json:"cores"`
	Vcpus        int64  `json:"vcpus"`
	Sockets      int64  `json:"sockets"`
	Cpu          string `json:"cpu"`
	Numa         int    `json:"numa"`
	Hotplug      string `json:"hotplug"`
	Boot         string `json:"boot"`
	Bootdisk     string `json:"bootdisk"`
	Kvm          int    `json:"kvm"`
	Scsihw       string `json:"scsihw"`
	Hookscript   string `json:"hookscript"`
	Machine      string `json:"machine"`
	Ide2         string `json:"ide2,omitempty"`
	Ciuser       string `json:"ciuser"`
	Cipassword   string `json:"cipassword"`
	Cicustom     string `json:"cicustom"`
	Searchdomain string `json:"searchdomain"`
	Nameserver   string `json:"nameserver"`
	Sshkeys      string `json:"sshkeys"`
}

type SInstance struct {
	multicloud.SInstanceBase
	ProxmoxTags

	host *SHost

	QemuNetworks []SInstanceNic
	PowerState   string
	Node         string

	VmID            int         `json:"vmid"`
	Name            string      `json:"name"`
	Description     string      `json:"desc"`
	Pool            string      `json:"pool,omitempty"`
	Bios            string      `json:"bios"`
	EFIDisk         QemuDevice  `json:"efidisk,omitempty"`
	Machine         string      `json:"machine,omitempty"`
	Onboot          bool        `json:"onboot"`
	Startup         string      `json:"startup,omitempty"`
	Tablet          bool        `json:"tablet"`
	Agent           int         `json:"agent"`
	Memory          int         `json:"memory"`
	Balloon         int         `json:"balloon"`
	QemuOs          string      `json:"ostype"`
	QemuCores       int         `json:"cores"`
	QemuSockets     int         `json:"sockets"`
	QemuVcpus       int         `json:"vcpus"`
	QemuCpu         string      `json:"cpu"`
	QemuNuma        bool        `json:"numa"`
	QemuKVM         bool        `json:"kvm"`
	Hotplug         string      `json:"hotplug"`
	QemuIso         string      `json:"iso"`
	QemuPxe         bool        `json:"pxe"`
	FullClone       *int        `json:"fullclone"`
	Boot            string      `json:"boot"`
	BootDisk        string      `json:"bootdisk,omitempty"`
	Scsihw          string      `json:"scsihw,omitempty"`
	QemuDisks       QemuDevices `json:"disk"`
	QemuUnusedDisks QemuDevices `json:"unused_disk"`
	QemuVga         QemuDevice  `json:"vga,omitempty"`
	QemuSerials     QemuDevices `json:"serial,omitempty"`
	QemuUsbs        QemuDevices `json:"usb,omitempty"`
	Hookscript      string      `json:"hookscript,omitempty"`
	Tags            string      `json:"tags"`
	Args            string      `json:"args"`

	// Deprecated single disk.
	DiskSize    float64 `json:"diskGB"`
	Storage     string  `json:"storage"`
	StorageType string  `json:"storageType"` // virtio|scsi (cloud-init defaults to scsi)

	// Deprecated single nic.
	QemuNicModel string `json:"nic"`
	QemuBridge   string `json:"bridge"`
	QemuVlanTag  int    `json:"vlan"`
	QemuMacAddr  string `json:"mac"`

	// cloud-init options
	CIuser     string `json:"ciuser"`
	CIpassword string `json:"cipassword"`
	CIcustom   string `json:"cicustom"`

	Searchdomain string `json:"searchdomain"`
	Nameserver   string `json:"nameserver"`
	Sshkeys      string `json:"sshkeys"`
}

func (self *SInstance) GetName() string {
	return self.Name
}

func (self *SInstance) GetId() string {
	return strconv.Itoa(self.VmID)
}

func (self *SInstance) GetGlobalId() string {
	return self.GetId()
}

func (self *SInstance) Refresh() error {
	id := strconv.Itoa(int(self.VmID))
	ins, err := self.host.zone.region.GetInstance(id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ins)
}

func (self *SInstance) AssignSecurityGroup(id string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) AttachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.AttachDisk(self.VmID, diskId)
}

func (self *SInstance) CreateDisk(ctx context.Context, opts *cloudprovider.GuestDiskCreateOptions) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedVMChangeConfig) error {
	return self.host.zone.region.ChangeConfig(self.VmID, opts.Cpu, opts.MemoryMB)
}

func (self *SInstance) DeleteVM(ctx context.Context) error {
	return self.host.zone.region.DeleteVM(self.VmID)
}

func (self *SInstance) DeployVM(ctx context.Context, name string, username string, password string, publicKey string, deleteKeypair bool, description string) error {
	return self.host.zone.region.ResetVmPassword(self.VmID, username, password)
}

func (self *SInstance) DetachDisk(ctx context.Context, diskId string) error {
	return self.host.zone.region.DetachDisk(self.VmID, diskId)
}

func (self *SInstance) GetBios() cloudprovider.TBiosType {
	return cloudprovider.ToBiosType(self.Bios)
}

func (self *SInstance) GetBootOrder() string {
	return strings.ToLower(self.Boot)
}

func (self *SInstance) GetError() error {
	return nil
}

func (self *SInstance) GetHostname() string {
	return self.GetName()
}

func (self *SInstance) GetHypervisor() string {
	return api.HYPERVISOR_PROXMOX
}

func (self *SInstance) VMIdExists(vmId int) (bool, error) {
	resources, err := self.host.zone.region.GetClusterVmResources()
	if err != nil {
		return false, err
	}

	_, res := resources[vmId]
	return res, nil
}

func (self *SInstance) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	ret := []cloudprovider.ICloudDisk{}
	id := self.VmID

	exist, err := self.VMIdExists(self.VmID)
	if err != nil {
		return nil, err
	}

	if exist == false {
		return nil, nil
	}

	for k, v := range self.QemuDisks {
		disk, err := self.host.zone.region.GetDisk(k)
		if err != nil {
			continue
		}
		disk.VmId = id
		disk.DiskDriver = v["type"].(string)
		idx, _ := strconv.ParseInt(fmt.Sprintf("%d", v["slot"]), 10, 64)
		disk.DriverIdx = int(idx)
		if cache, ok := v["cache"].(string); ok {
			disk.CacheMode = cache
		}
		ret = append(ret, disk)
	}

	return ret, nil
}

func (self *SInstance) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetIHost() cloudprovider.ICloudHost {
	return self.host
}

func (self *SInstance) GetINics() ([]cloudprovider.ICloudNic, error) {
	ret := []cloudprovider.ICloudNic{}
	for i := range self.QemuNetworks {
		self.QemuNetworks[i].ins = self
		ret = append(ret, &self.QemuNetworks[i])
	}
	return ret, nil
}

func (self *SInstance) GetInstanceType() string {
	return fmt.Sprintf("ecs.g1.c%dm%d", self.GetVcpuCount(), self.GetVmemSizeMB()/1024)
}

func (self *SInstance) GetMachine() string {
	return self.Machine
}

func (self *SInstance) GetStatus() string {
	switch strings.ToLower(self.PowerState) {
	case "running":
		return api.VM_RUNNING
	case "stopped":
		return api.VM_READY
	case "paused":
		return api.VM_SUSPEND
	}
	return api.VM_UNKNOWN
}

func (self *SInstance) GetFullOsName() string {
	return ""
}

func (self *SInstance) GetOsType() cloudprovider.TOsType {
	isWin, _ := regexp.MatchString("(wxp|w2k|w2k3|w2k8|wvista|win7|win8|win10|win11)", self.QemuOs)
	if isWin == true {
		return cloudprovider.TOsType(osprofile.OS_TYPE_WINDOWS)
	} else {
		return cloudprovider.TOsType(osprofile.OS_TYPE_LINUX)
	}
}

func (ins *SInstance) GetOsArch() string {
	return "x86_64"
}

func (ins *SInstance) GetOsDist() string {
	return ""
}

func (ins *SInstance) GetOsVersion() string {
	return ""
}

func (ins *SInstance) GetOsLang() string {
	return ""
}

func (self *SInstance) GetProjectId() string {
	return ""
}

func (self *SInstance) GetVNCInfo(input *cloudprovider.ServerVncInput) (*cloudprovider.ServerVncOutput, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SInstance) GetVcpuCount() int {
	return int(self.QemuCores * self.QemuSockets)
}

func (self *SInstance) GetVmemSizeMB() int {
	return int(self.Memory)
}

func (self *SInstance) GetVga() string {
	return "std"
}

func (self *SInstance) GetVdi() string {
	return "vnc"
}

func (self *SInstance) RebuildRoot(ctx context.Context, desc *cloudprovider.SManagedVMRebuildRootConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SInstance) SetSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) StartVM(ctx context.Context) error {
	return self.host.zone.region.StartVm(self.VmID)
}

func (self *SInstance) StopVM(ctx context.Context, opts *cloudprovider.ServerStopOptions) error {
	if self.GetStatus() == api.VM_READY {
		return nil
	}
	return self.host.zone.region.StopVm(self.VmID)
}

func (self *SInstance) UpdateUserData(userData string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SInstance) UpdateVM(ctx context.Context, name string) error {
	return cloudprovider.ErrNotSupported
}

// readDeviceConfig - get standard sub-conf strings where `key=value` and update conf map.
func (confMap QemuDevice) readDeviceConfig(confList []string) error {
	// Add device config.
	for _, conf := range confList {
		key, value := ParseSubConf(conf, "=")
		confMap[key] = value
	}
	return nil
}

func (self *SRegion) GetVmAgentNetworkInterfaces(node string, VmId int) (map[string]string, error) {
	intermediates := []Intermediate{}
	ipMap := map[string]string{}
	res := fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", node, VmId)
	err := self.getAgent(res, url.Values{}, &intermediates)
	if err != nil {
		return nil, errors.Wrap(err, "GetVmAgentNetworkInterfaces")
	}

	for _, intermediate := range intermediates {
		for _, addr := range intermediate.IPAddresses {
			ipMap[intermediate.HardwareAddress] = addr.IPAddress
		}
	}

	return ipMap, nil
}

func (self *SRegion) GetVmPowerStatus(node string, VmId int) string {
	current := map[string]string{}
	res := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", node, VmId)
	err := self.get(res, url.Values{}, &current)
	if err != nil {
		return "unkown"
	}

	power := "unkown"
	if _, ok := current["qmpstatus"]; ok {
		power = current["qmpstatus"]
	}

	return power
}

func (self *SRegion) GetQemuConfig(node string, VmId int) (*SInstance, error) {
	//ret := &SInstance{}
	res := fmt.Sprintf("/nodes/%s/qemu/%d/config", node, VmId)
	vmConfig := map[string]interface{}{}
	vmBase := &VmBase{
		Bios:    "seabios",
		OnBoot:  1,
		Tablet:  1,
		Ostype:  "other",
		Memory:  0,
		Balloon: 0,
		Cores:   0,
		Vcpus:   0,
		Sockets: 0,
		Cpu:     "host",
		Numa:    0,
		Hotplug: "network,disk,usb",
		Boot:    "cdn",
		Kvm:     1,
		Scsihw:  "lsi",
		Machine: "i440fx",
	}
	err := self.get(res, url.Values{}, &vmConfig)
	if err != nil {
		return nil, err
	}
	byteArr, err := json.Marshal(&vmConfig)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(byteArr, &vmBase)
	if err != nil {
		return nil, err
	}

	config := SInstance{
		VmID:            int(VmId),
		Name:            vmBase.Name,
		Description:     strings.TrimSpace(vmBase.Description),
		Tags:            strings.TrimSpace(vmBase.Tags),
		Args:            strings.TrimSpace(vmBase.Args),
		Bios:            vmBase.Bios,
		EFIDisk:         QemuDevice{},
		Machine:         vmBase.Machine,
		Onboot:          Itob(vmBase.OnBoot),
		Startup:         vmBase.Startup,
		Tablet:          Itob(vmBase.Tablet),
		QemuOs:          vmBase.Ostype,
		Memory:          int(vmBase.Memory),
		QemuCores:       int(vmBase.Cores),
		QemuSockets:     int(vmBase.Sockets),
		QemuCpu:         vmBase.Cpu,
		QemuNuma:        Itob(vmBase.Numa),
		QemuKVM:         Itob(vmBase.Kvm),
		Hotplug:         vmBase.Hotplug,
		QemuVlanTag:     -1,
		Boot:            vmBase.Boot,
		BootDisk:        vmBase.Bootdisk,
		Scsihw:          vmBase.Scsihw,
		Hookscript:      vmBase.Hookscript,
		QemuDisks:       QemuDevices{},
		QemuUnusedDisks: QemuDevices{},
		QemuVga:         QemuDevice{},
		QemuNetworks:    []SInstanceNic{},
		QemuSerials:     QemuDevices{},
		QemuUsbs:        QemuDevices{},
		Node:            node,
		CIuser:          vmBase.Ciuser,
		CIpassword:      vmBase.Cipassword,
		Searchdomain:    vmBase.Searchdomain,
		Nameserver:      vmBase.Nameserver,
	}

	// vmConfig Sample: map[ cpu:host
	// net0:virtio=62:DF:XX:XX:XX:XX,bridge=vmbr0
	// ide2:local:iso/xxx-xx.iso,media=cdrom memory:2048
	// smbios1:uuid=8b3bf833-aad8-4545-xxx-xxxxxxx digest:aa6ce5xxxxx1b9ce33e4aaeff564d4 sockets:1
	// name:terraform-ubuntu1404-template bootdisk:virtio0
	// virtio0:ProxmoxxxxISCSI:vm-1014-disk-2,size=4G
	// description:Base image
	// cores:2 ostype:l26
	if vmConfig["ide2"] != nil {
		isoMatch := rxIso.FindStringSubmatch(vmConfig["ide2"].(string))
		config.QemuIso = isoMatch[1]
	}

	if _, ok := vmConfig["sshkeys"]; ok {
		config.Sshkeys, _ = url.PathUnescape(vmConfig["sshkeys"].(string))
	}

	agent := 0
	if _, ok := vmConfig["agent"]; ok {
		switch vmConfig["agent"].(type) {
		case int64:
			agent = int(vmConfig["agent"].(int64))
		case string:
			agentConfList := strings.Split(vmConfig["agent"].(string), ",")
			agent, _ = strconv.Atoi(agentConfList[0])
		}

	}
	config.Agent = agent

	config.PowerState = self.GetVmPowerStatus(node, VmId)

	// Add disks.
	diskNames := []string{}
	for k := range vmConfig {
		if diskName := rxDiskName.FindStringSubmatch(k); len(diskName) > 0 {
			diskNames = append(diskNames, diskName[0])
		}
	}

	for _, diskName := range diskNames {
		diskConfStr := vmConfig[diskName].(string)

		id := rxDeviceID.FindStringSubmatch(diskName)
		diskID, _ := strconv.Atoi(id[0])
		diskType := rxDiskType.FindStringSubmatch(diskName)[0]

		diskConfMap := ParsePMConf(diskConfStr, "volume")

		if diskConfMap["volume"].(string) == "none" {
			continue
		}

		diskConfMap["slot"] = diskID
		diskConfMap["type"] = diskType

		storageName, fileName := ParseSubConf(diskConfMap["volume"].(string), ":")
		diskConfMap["storage"] = storageName
		diskConfMap["file"] = fileName

		volId := diskConfMap["volume"].(string)

		// cloud-init disks not always have the size sent by the API, which results in a crash
		if diskConfMap["size"] == nil && strings.Contains(fileName.(string), "cloudinit") {
			diskConfMap["size"] = "4M" // default cloud-init disk size
		}

		var sizeInTerabytes = regexp.MustCompile(`[0-9]+T`)
		// Convert to gigabytes if disk size was received in terabytes
		matched := sizeInTerabytes.MatchString(diskConfMap["size"].(string))
		if matched {
			diskConfMap["size"] = fmt.Sprintf("%.0fG", DiskSizeGB(diskConfMap["size"]))
		}

		// And device config to disks map.
		if len(diskConfMap) > 0 {
			config.QemuDisks[volId] = diskConfMap
		}
	}

	// Add unused disks
	// unused0:local:100/vm-100-disk-1.qcow2
	unusedDiskNames := []string{}
	for k := range vmConfig {
		// look for entries from the config in the format "unusedX:<storagepath>" where X is an integer
		if unusedDiskName := rxUnusedDiskName.FindStringSubmatch(k); len(unusedDiskName) > 0 {
			unusedDiskNames = append(unusedDiskNames, unusedDiskName[0])
		}
	}
	if len(unusedDiskNames) > 0 {
		log.Debugf("[DEBUG] unusedDiskNames: %v", unusedDiskNames)
	}

	for _, unusedDiskName := range unusedDiskNames {
		unusedDiskConfStr := vmConfig[unusedDiskName].(string)
		finalDiskConfMap := QemuDevice{}

		// parse "unused0" to get the id '0' as an int
		id := rxDeviceID.FindStringSubmatch(unusedDiskName)
		slotID, err := strconv.Atoi(id[0])
		if err != nil {
			return nil, errors.Errorf("Unable to parse unused disk id from input string '%s' .", unusedDiskName)
		}
		finalDiskConfMap["slot"] = slotID

		// parse the attributes from the unused disk
		// extract the storage and file path from the unused disk entry
		parsedUnusedDiskMap := ParsePMConf(unusedDiskConfStr, "storage+file")
		storageName, fileName := ParseSubConf(parsedUnusedDiskMap["storage+file"].(string), ":")
		finalDiskConfMap["storage"] = storageName
		finalDiskConfMap["file"] = fileName
		volId := parsedUnusedDiskMap["storage+file"].(string)

		config.QemuUnusedDisks[volId] = finalDiskConfMap
	}

	//Display
	if vga, ok := vmConfig["vga"]; ok {
		vgaList := strings.Split(vga.(string), ",")
		vgaMap := QemuDevice{}

		// TODO: keep going if error?
		err = vgaMap.readDeviceConfig(vgaList)
		if err != nil {
			log.Debugf("[ERROR] %q", err)
		}
		if len(vgaMap) > 0 {
			config.QemuVga = vgaMap
		}
	}

	// Add networks.
	nicNames := []string{}
	ipMap := make(map[string]string)
	if config.PowerState == "running" && config.Agent == 1 {
		ipMap, _ = self.GetVmAgentNetworkInterfaces(node, VmId)
	}

	for k := range vmConfig {
		if nicName := rxNicName.FindStringSubmatch(k); len(nicName) > 0 {
			nicNames = append(nicNames, nicName[0])
		}
	}

	for _, nicName := range nicNames {
		nicConfStr := vmConfig[nicName]
		nicConfList := strings.Split(nicConfStr.(string), ",")

		//id := rxDeviceID.FindStringSubmatch(nicName)
		model, macaddr := ParseSubConf(nicConfList[0], "=")
		_, network := ParseSubConf(nicConfList[1], "=")
		//nicID := fmt.Sprintf("%d:%s", VmId, nicName)

		// Add model and MAC address.
		nicConf := SInstanceNic{
			NicId:     network.(string),
			Model:     model,
			MacAddr:   strings.ToLower(macaddr.(string)),
			NetworkId: fmt.Sprintf("network/%s/%s", node, network),
		}

		if ip, ok := ipMap[nicConf.MacAddr]; ok {
			nicConf.IpAddr = ip
		}

		// And device config to networks.
		config.QemuNetworks = append(config.QemuNetworks, nicConf)
	}

	// Add serials
	serialNames := []string{}

	for k := range vmConfig {
		if serialName := rxSerialName.FindStringSubmatch(k); len(serialName) > 0 {
			serialNames = append(serialNames, serialName[0])
		}
	}

	for _, serialName := range serialNames {
		id := rxDeviceID.FindStringSubmatch(serialName)
		serialID, _ := strconv.Atoi(id[0])

		serialConfMap := QemuDevice{
			"id":   serialID,
			"type": vmConfig[serialName],
		}

		// And device config to serials map.
		if len(serialConfMap) > 0 {
			config.QemuSerials[serialName] = serialConfMap
		}
	}

	// Add usbs
	usbNames := []string{}

	for k := range vmConfig {
		if usbName := rxUsbName.FindStringSubmatch(k); len(usbName) > 0 {
			usbNames = append(usbNames, usbName[0])
		}
	}

	for _, usbName := range usbNames {
		usbConfStr := vmConfig[usbName]
		usbConfList := strings.Split(usbConfStr.(string), ",")
		id := rxDeviceID.FindStringSubmatch(usbName)
		usbID, _ := strconv.Atoi(id[0])
		_, host := ParseSubConf(usbConfList[0], "=")

		usbConfMap := QemuDevice{
			"id":   usbID,
			"host": host,
		}

		err = usbConfMap.readDeviceConfig(usbConfList[1:])
		if err != nil {
			log.Debugf("[ERROR] %q", err)
		}
		if usbConfMap["usb3"] == 1 {
			usbConfMap["usb3"] = true
		}

		// And device config to usbs map.
		if len(usbConfMap) > 0 {
			config.QemuUsbs[usbName] = usbConfMap
		}
	}

	return &config, nil

}

func (self *SRegion) GetInstances(hostId string) ([]SInstance, error) {
	ret := []SInstance{}
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		if res.NodeId == hostId {
			instance, err := self.GetQemuConfig(res.Node, res.VmId)
			if err == nil {
				ret = append(ret, *instance)
			}

		}
	}

	return ret, nil
}

func (self *SRegion) GetInstance(id string) (*SInstance, error) {
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return nil, err
	}

	nodeName := ""
	vmId, _ := strconv.Atoi(id)
	if resource, ok := resources[vmId]; !ok {
		return nil, errors.Errorf("failed get Instance id %s", id)
	} else {
		nodeName = resource.Node
	}

	return self.GetQemuConfig(nodeName, vmId)
}

func (self *SRegion) StartVm(vmId int) error {
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return err
	}

	nodeName := ""
	if resource, ok := resources[vmId]; !ok {
		return errors.Errorf("start VM id %d", vmId)
	} else {
		nodeName = resource.Node
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", nodeName, vmId)
	params := url.Values{}

	_, err = self.post(res, params)
	return err

}

func (self *SRegion) StopVm(vmId int) error {
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return err
	}

	nodeName := ""
	if resource, ok := resources[vmId]; !ok {
		return errors.Errorf("start VM id %d", vmId)
	} else {
		nodeName = resource.Node
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", nodeName, vmId)
	params := url.Values{}

	_, err = self.post(res, params)
	return err
}

func (self *SRegion) AttachDisk(vmId int, diskId string) error {
	id := strconv.Itoa(int(vmId))
	vm1, err := self.GetInstance(id)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%d)", vmId)
	}
	if _, ok := vm1.QemuUnusedDisks[diskId]; !ok {
		return nil
	}

	slotsArr := []int{}
	for _, v := range vm1.QemuDisks {
		if v["type"] == "scsi" {
			slotIdx := v["slot"].(int)
			slotsArr = append(slotsArr, slotIdx)
		}
	}
	sort.Ints(slotsArr)
	minSlot := slotsArr[0]
	for idx, _ := range slotsArr {
		if slotsArr[idx] == minSlot {
			minSlot++
		} else {
			break
		}

	}

	body := map[string]string{}
	params := url.Values{}
	diskName := fmt.Sprintf("scsi%d", minSlot)
	body[diskName] = diskId
	res := fmt.Sprintf("/nodes/%s/qemu/%d/config", vm1.Node, vm1.VmID)
	err = self.put(res, params, jsonutils.Marshal(body), nil)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%d) self.put", vmId)
	}
	//clear
	vm1.QemuDisks = make(map[string]map[string]interface{})
	vm1.QemuUnusedDisks = make(map[string]map[string]interface{})

	vm2, err := self.GetInstance(id)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%d) vm2", vmId)
	}
	vm1.QemuDisks = vm2.QemuDisks
	vm1.QemuUnusedDisks = vm2.QemuUnusedDisks

	return nil

}

func (self *SRegion) DetachDisk(vmId int, diskId string) error {
	id := strconv.Itoa(int(vmId))
	vm1, err := self.GetInstance(id)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%d)", vmId)
	}
	if v, ok := vm1.QemuDisks[diskId]; !ok {
		return nil
	} else {
		diskName := fmt.Sprintf("%s%d", v["type"].(string), v["slot"].(int))
		body := map[string]string{}
		params := url.Values{}
		body["delete"] = diskName
		res := fmt.Sprintf("/nodes/%s/qemu/%d/config", vm1.Node, vm1.VmID)
		err := self.put(res, params, jsonutils.Marshal(body), nil)
		if err != nil {
			return errors.Wrapf(err, "GetInstance(%d) self.put", vmId)
		}
		//clear
		vm1.QemuDisks = make(map[string]map[string]interface{})
		vm1.QemuUnusedDisks = make(map[string]map[string]interface{})

		vm2, err := self.GetInstance(id)
		if err != nil {
			return errors.Wrapf(err, "GetInstance(%d) vm2", vmId)
		}
		vm1.QemuDisks = vm2.QemuDisks
		vm1.QemuUnusedDisks = vm2.QemuUnusedDisks

		return nil
	}

}

func (self *SRegion) ChangeConfig(vmId int, cpu int, memMb int) error {
	vm, err := self.GetInstance(strconv.Itoa(int(vmId)))
	body := map[string]interface{}{}
	if err != nil {
		return errors.Wrapf(err, "ChangeConfig(%d)", vmId)
	}

	changed := false
	if cpu > 0 {
		vm.QemuCores = 1
		vm.QemuSockets = cpu
		vm.QemuVcpus = cpu
		changed = true
		body["cores"] = 1
		body["sockets"] = cpu
		body["vcpus"] = cpu
	}
	if memMb > 0 {
		vm.Memory = memMb
		body["memory"] = memMb
		changed = true
	}
	if !changed {
		return nil
	}

	params := url.Values{}
	res := fmt.Sprintf("/nodes/%s/qemu/%d/config", vm.Node, vmId)
	return self.put(res, params, jsonutils.Marshal(body), nil)
}

func (self *SRegion) ResetVmPassword(vmId int, username, password string) error {
	resources, err := self.GetClusterVmResources()
	if err != nil {
		return err
	}

	nodeName := ""
	if resource, ok := resources[vmId]; !ok {
		return errors.Errorf("failed to ResetVmPassword VM id %d", vmId)
	} else {
		nodeName = resource.Node
	}

	params := url.Values{}
	body := map[string]interface{}{
		"username": username,
		"password": password,
	}

	res := fmt.Sprintf("/nodes/%s/qemu/%d/agent/set-user-password", nodeName, vmId)
	return self.put(res, params, jsonutils.Marshal(body), nil)

}

func (self *SRegion) DeleteVM(vmId int) error {
	id := strconv.Itoa(int(vmId))
	vm1, err := self.GetInstance(id)
	if err != nil {
		return errors.Wrapf(err, "GetInstance(%d)", vmId)
	}
	params := url.Values{}
	params.Set("purge", "1")

	res := fmt.Sprintf("/nodes/%s/qemu/%d", vm1.Node, vmId)
	return self.del(res, params, nil)
}
func (self *SRegion) GenVM(name, node string, cores, memMB int) (*SInstance, error) {

	vmId := self.GetClusterVmMaxId()
	if vmId == -1 {
		return nil, errors.Errorf("failed to get vm number by %d", vmId)
	} else {
		vmId++
	}

	body := map[string]interface{}{
		"vmid":        vmId,
		"name":        name,
		"ostype":      "other",
		"sockets":     1,
		"cores":       cores,
		"cpu":         "host",
		"kvm":         1,
		"hotplug":     "network,disk,usb",
		"memory":      memMB,
		"description": "",
		"scsihw":      "virtio-scsi-pci",
	}

	res := fmt.Sprintf("/nodes/%s/qemu", node)
	_, err := self.post(res, jsonutils.Marshal(body))
	if err != nil {
		return nil, err
	}

	vmIdRet := strconv.Itoa(vmId)
	vm, err := self.GetInstance(vmIdRet)
	if err != nil {
		return nil, err
	}

	return vm, nil
}
