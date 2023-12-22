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
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	ProxmoxTags
	zone *SZone

	Id   string
	Node string

	Uptime     int      `json:"uptime"`
	Wait       int      `json:"wait"`
	Idle       int      `json:"idle"`
	Kversion   string   `json:"kversion"`
	Pveversion string   `json:"pveversion"`
	CPU        int      `json:"cpu"`
	Loadavg    []string `json:"loadavg"`
	Rootfs     Rootfs   `json:"rootfs"`
	Swap       Swap     `json:"swap"`
	Memory     Memory   `json:"memory"`
	Cpuinfo    Cpuinfo  `json:"cpuinfo"`
	Ksm        Ksm      `json:"ksm"`
}

type Rootfs struct {
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
	Avail int64 `json:"avail"`
	Free  int64 `json:"free"`
}

type Swap struct {
	Free  int64 `json:"free"`
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

type Memory struct {
	Free  int64 `json:"free"`
	Used  int64 `json:"used"`
	Total int64 `json:"total"`
}

type Cpuinfo struct {
	Flags   string  `json:"flags"`
	Hvm     string  `json:"hvm"`
	Cores   int     `json:"cores"`
	Model   string  `json:"model"`
	Mhz     float64 `json:"mhz"`
	Cpus    int     `json:"cpus"`
	UserHz  int     `json:"user_hz"`
	Sockets int     `json:"sockets"`
}

type Ksm struct {
	Shared int `json:"shared"`
}

func (self *SHost) GetId() string {
	return self.Id
}

func (self *SHost) GetGlobalId() string {
	return self.Id
}

func (self *SHost) GetName() string {
	return self.Node
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) GetAccessIp() string {
	network := fmt.Sprintf("nodes/%s/network", self.Node)
	ret := []struct {
		Address string
	}{}
	err := self.zone.region.get(network, url.Values{}, &ret)
	if err != nil {
		return ""
	}
	for i := range ret {
		if len(ret[i].Address) > 0 {
			return ret[i].Address
		}
	}
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetCpuCount() int {
	return int(self.Cpuinfo.Cores)
}

func (self *SHost) GetNodeCount() int8 {
	return int8(self.Cpuinfo.Sockets)
}

func (self *SHost) GetCpuDesc() string {
	return self.Cpuinfo.Model
}

func (self *SHost) GetCpuMhz() int {
	return int(self.Cpuinfo.Mhz)
}

func (self *SHost) GetCpuCmtbound() float32 {
	return 1
}

func (self *SHost) GetMemSizeMB() int {
	return int(self.Memory.Total / 1024 / 1024)
}

func (self *SHost) GetMemCmtbound() float32 {
	return 1
}

func (self *SHost) GetReservedMemoryMb() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int64 {
	return self.Rootfs.Total / 1024 / 1024
}

func (self *SHost) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_PROXMOX
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return self.Pveversion
}

func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {

	vmId := self.zone.region.GetClusterVmMaxId()
	if vmId == -1 {
		return nil, errors.Errorf("failed to get vm number by %d", vmId)
	}
	vmId++

	storage, err := self.zone.region.GetStorage(opts.SysDisk.StorageExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetStorage")
	}

	body := map[string]interface{}{
		"vmid":        vmId,
		"name":        opts.Name,
		"ide2":        fmt.Sprintf("%s,media=cdrom", opts.ExternalImageId),
		"ostype":      "other",
		"sockets":     1,
		"cores":       opts.Cpu,
		"cpu":         "host",
		"kvm":         1,
		"hotplug":     "network,disk,usb",
		"memory":      opts.MemoryMB,
		"description": opts.OsDistribution,
		"scsihw":      "virtio-scsi-pci",
		"net0":        "virtio,bridge=vmbr0,firewall=1",
		"scsi0":       fmt.Sprintf("%s:%d", storage.Storage, opts.SysDisk.SizeGB),
	}
	for i, disk := range opts.DataDisks {
		storage, err := self.zone.region.GetStorage(disk.StorageExternalId)
		if err != nil {
			return nil, err
		}
		body[fmt.Sprintf("scsi%d", i+1)] = fmt.Sprintf("%s:%d", storage.Storage, opts.SysDisk.SizeGB)
	}

	res := fmt.Sprintf("/nodes/%s/qemu", self.Node)
	_, err = self.zone.region.post(res, jsonutils.Marshal(body))
	if err != nil {
		return nil, err
	}

	vmIdRet := strconv.Itoa(vmId)
	cloudprovider.Wait(time.Second*5, time.Minute, func() (bool, error) {
		_, err := self.zone.region.GetInstance(vmIdRet)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return false, nil
			}
			return false, errors.Wrapf(err, "after created")
		}
		return true, nil
	})

	vm, err := self.zone.region.GetInstance(vmIdRet)
	if err != nil {
		return nil, err
	}

	vm.host = self
	return vm, nil
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "getIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstances")
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		vms[i].host = self
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	hostId := fmt.Sprintf("node/%s", vm.Node)
	if hostId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	vm.host = self
	return vm, nil
}

func (self *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := self.zone.region.GetWires()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		wires[i].region = self.zone.region
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.zone.region.GetStoragesByHost(self.Node)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = self.zone
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.zone.region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	storage.zone = self.zone

	return storage, nil
}

func (self *SRegion) GetHosts() ([]SHost, error) {
	hosts := []SHost{}
	resources, err := self.GetClusterNodeResources()
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		host := &SHost{}
		status := fmt.Sprintf("nodes/%s/status", res.Node)
		err := self.get(status, url.Values{}, host)
		if err != nil {
			return nil, err
		}
		host.Id = res.Id
		host.Node = res.Node
		hosts = append(hosts, *host)
	}

	return hosts, nil
}

func (self *SRegion) GetHost(id string) (*SHost, error) {
	ret := &SHost{}
	nodeName := ""

	//"id": "node/nodeNAME",
	splited := strings.Split(id, "/")
	if len(splited) == 2 {
		nodeName = splited[1]
	}

	res := fmt.Sprintf("nodes/%s/status", nodeName)
	err := self.get(res, url.Values{}, ret)
	if err != nil {
		return nil, err
	}
	ret.Id = id
	ret.Node = nodeName

	return ret, nil
}
