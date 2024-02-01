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

package guestman

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"unicode"

	libvirtxml "github.com/libvirt/libvirt-go-xml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

func (m *SGuestManager) GuestCreateFromLibvirt(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	createConfig, ok := params.(*SGuestCreateFromLibvirt)
	if !ok {
		return nil, hostutils.ParamsError
	}
	disks := createConfig.GuestDesc.Disks

	disksPath := jsonutils.NewDict()
	for _, disk := range disks {
		diskPath, err := createConfig.DisksPath.GetString(disk.DiskId)
		if err != nil {
			return nil, fmt.Errorf("Disks path missing disk %s", disk.DiskId)
		}
		storage := storageman.GetManager().GetStorage(disk.StorageId)
		if storage == nil {
			return nil, fmt.Errorf("Host has no stroage %s", disk.StorageId)
		}
		iDisk := storage.CreateDisk(disk.DiskId)

		// use symbol link replace mv, more security
		output, err := procutils.NewCommand("ln", "-s", diskPath, iDisk.GetPath()).Output()
		if err != nil {
			return nil, fmt.Errorf("Symbol link disk from %s to %s error %s", diskPath, iDisk.GetPath(), output)
		}
		disksPath.Set(disk.DiskId, jsonutils.NewString(iDisk.GetPath()))
	}
	guest, _ := m.GetKVMServer(createConfig.Sid)
	if err := SaveDesc(guest, createConfig.GuestDesc); err != nil {
		return nil, err
	}

	if len(createConfig.MonitorPath) > 0 {
		if pid := findGuestProcessPid(guest.getOriginId(), "[q]emu-kvm"); len(pid) > 0 {
			fileutils2.FilePutContents(guest.GetPidFilePath(), pid, false)
			guest.StartMonitorWithImportGuestSocketFile(ctx, createConfig.MonitorPath, nil)
			stopScript := guest.generateStopScript(nil)
			if err := fileutils2.FilePutContents(guest.GetStopScriptPath(), stopScript, false); err != nil {
				return nil, fmt.Errorf("Save stop script error %s", err)
			}
		}
	}

	ret := jsonutils.NewDict()
	ret.Set("disks_path", disksPath)
	return ret, nil
}

func findGuestProcessPid(originId, sufix string) string {
	output, err := procutils.NewCommand(
		"sh", "-c", fmt.Sprintf("ps -A -o pid,args | grep [q]emu | grep %s | grep %s", originId, sufix)).Output()
	if err != nil {
		log.Errorf("find guest %s error: %s", originId, output)
		return ""
	}
	var spid string
	s1 := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i := 0; i < len(s1); i++ {
		if len(s1[i]) > 0 {
			if len(spid) > 0 {
				log.Errorf("can't find guest %s pid, has multi process", originId)
				return ""
			}
			s2 := strings.Fields(s1[i])
			if len(s2) > 1 {
				spid = s2[0]
			}
		}
	}
	if len(spid) > 0 {
		_, err := strconv.Atoi(spid)
		if err != nil {
			log.Errorf("Pid atoi failed %s", err)
			return ""
		} else {
			return spid
		}
	} else {
		return ""
	}
}

func (m *SGuestManager) PrepareImportFromLibvirt(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	libvirtConfig, ok := params.(*compute.SLibvirtHostConfig)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guestDescs, err := m.GenerateDescFromXml(libvirtConfig)
	if err != nil {
		return nil, err
	}
	return guestDescs, nil
}

func IsMacInGuestConfig(guestConfig *compute.SImportGuestDesc, mac string) bool {
	for _, nic := range guestConfig.Nics {
		if netutils.FormatMacAddr(nic.Mac) == netutils.FormatMacAddr(mac) {
			return true
		}
	}
	return false
}

func setAttributeFromLibvirtConfig(
	guestConfig *compute.SImportGuestDesc, libvirtConfig *compute.SLibvirtHostConfig,
) (int, error) {
	var Matched = true
	for i, server := range libvirtConfig.Servers {
		macMap := make(map[string]string, 0)
		for mac := range server.MacIp {
			if !IsMacInGuestConfig(guestConfig, mac) {
				Matched = false
				break
			} else {
				macMap[netutils.FormatMacAddr(mac)] = server.MacIp[mac]
				Matched = true
			}
		}

		if Matched {
			for idx, nic := range guestConfig.Nics {
				guestConfig.Nics[idx].Ip = macMap[netutils.FormatMacAddr(nic.Mac)]
			}
			log.Infof("config monitor path is %s, guest config id %s", libvirtConfig.MonitorPath, guestConfig.Id)
			if len(libvirtConfig.MonitorPath) > 0 {
				files, _ := ioutil.ReadDir(libvirtConfig.MonitorPath)
				for i := 0; i < len(files); i++ {
					if files[i].Mode().IsDir() &&
						strings.HasPrefix(files[i].Name(), "domain") &&
						strings.HasSuffix(files[i].Name(), guestConfig.Name) {
						monitorPath := path.Join(libvirtConfig.MonitorPath, files[i].Name(), "monitor.sock")
						if fileutils2.Exists(monitorPath) && isServerRunning("[q]emu-kvm", guestConfig.Id) {
							guestConfig.MonitorPath = monitorPath
						}
						break
					}
				}
			}
			log.Infof("Import guest %s, monitor is %s", guestConfig.Name, guestConfig.MonitorPath)
			return i, nil
		}
	}
	return -1, fmt.Errorf("Config not match guest %s", guestConfig.Id)
}

func isServerRunning(sufix, uuid string) bool {
	return procutils.NewCommand("sh", "-c",
		fmt.Sprintf("ps -ef | grep [q]emu | grep %s | grep %s", uuid, sufix)).Run() == nil
}

func (m *SGuestManager) GenerateDescFromXml(libvirtConfig *compute.SLibvirtHostConfig) (jsonutils.JSONObject, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"find", libvirtConfig.XmlFilePath,
		"-type", "f", "-maxdepth", "1",
	).Output()
	if err != nil {
		log.Errorf("failed read dir %s", libvirtConfig.XmlFilePath)
		return nil, errors.Wrapf(err, "failed read dir %s", libvirtConfig.XmlFilePath)
	}

	libvirtServers := []*compute.SImportGuestDesc{}
	xmlsPath := strings.Split(string(out), "\n")
	for _, xmlPath := range xmlsPath {
		if len(xmlPath) == 0 {
			continue
		}

		xmlContent, err := procutils.NewRemoteCommandAsFarAsPossible("cat", xmlPath).Output()
		if err != nil {
			log.Errorf("Read file %s failed: %s %s", xmlPath, xmlContent, err)
			continue
		}

		// parse libvirt xml file
		domain := &libvirtxml.Domain{}
		err = domain.Unmarshal(strings.TrimSpace(string(xmlContent)))
		if err != nil {
			log.Errorf("Unmarshal xml file %s error %s", xmlPath, err)
			continue
		}
		guestConfig, err := m.LibvirtDomainToGuestDesc(domain)
		if err != nil {
			log.Errorf("Parse libvirt domain failed %s", err)
			continue
		}
		if idx, err := setAttributeFromLibvirtConfig(guestConfig, libvirtConfig); err != nil {
			log.Errorf("Import guest %s error %s", guestConfig.Id, err)
			continue
		} else {
			libvirtConfig.Servers = append(libvirtConfig.Servers[:idx], libvirtConfig.Servers[idx+1:]...)
			libvirtServers = append(libvirtServers, guestConfig)
		}
	}

	ret := jsonutils.NewDict()
	ret.Set("servers_not_match", jsonutils.Marshal(libvirtConfig.Servers))
	ret.Set("servers_matched", jsonutils.Marshal(libvirtServers))
	return ret, nil
}

// Read key infomation from domain xml
func (m *SGuestManager) LibvirtDomainToGuestDesc(domain *libvirtxml.Domain) (*compute.SImportGuestDesc, error) {
	if nil == domain {
		return nil, fmt.Errorf("Libvirt domain is nil")
	}
	if nil == domain.VCPU {
		return nil, fmt.Errorf("Libvirt domain missing VCPU config")
	}
	if nil == domain.CurrentMemory {
		return nil, fmt.Errorf("Libvirt domain missing CurrentMemory config")
	}
	if 0 == len(domain.CurrentMemory.Unit) {
		return nil, fmt.Errorf("Libvirt Memory config missing unit")
	}
	if nil == domain.Devices {
		return nil, fmt.Errorf("Libvirt domain missing Devices config")
	}
	if 0 == len(domain.Devices.Disks) {
		return nil, fmt.Errorf("Livbirt domain has no Disks")
	}
	if 0 == len(domain.Devices.Interfaces) {
		return nil, fmt.Errorf("Libvirt domain has no network Interfaces")
	}

	var memSizeMb uint
	switch unicode.ToLower(rune(domain.CurrentMemory.Unit[0])) {
	case 'k':
		memSizeMb = domain.CurrentMemory.Value / 1024
		if domain.CurrentMemory.Value%1024 > 0 {
			memSizeMb += 1
		}
	case 'm':
		memSizeMb = domain.CurrentMemory.Value
	case 'g':
		memSizeMb = domain.CurrentMemory.Value * 1024
	case 'b':
		memSizeMb = domain.CurrentMemory.Value / 1024 / 1024
		if domain.CurrentMemory.Value%(1024*1024) > 0 {
			memSizeMb += 1
		}
	default:
		return nil, fmt.Errorf("Unknown memory unit %s", domain.CurrentMemory.Unit)
	}

	disksConfig, err := m.LibvirtDomainDiskToDiskConfig(domain.Devices.Disks)
	if err != nil {
		return nil, err
	}

	nicsConfig, err := m.LibvirtDomainInterfaceToNicConfig(domain.Devices.Interfaces)
	if err != nil {
		return nil, err
	}

	return &compute.SImportGuestDesc{
		Id:        domain.UUID,
		Name:      domain.Name,
		Cpu:       domain.VCPU.Value,
		MemSizeMb: int(memSizeMb),
		Disks:     disksConfig,
		Nics:      nicsConfig,
	}, nil
}

func (m *SGuestManager) LibvirtDomainDiskToDiskConfig(
	domainDisks []libvirtxml.DomainDisk) ([]compute.SImportDisk, error) {
	var diskConfigs = []compute.SImportDisk{}
	for _, disk := range domainDisks {
		if disk.Device != "disk" {
			continue
		}
		if disk.Source == nil || disk.Source.File == nil {
			return nil, fmt.Errorf("Domain disk missing source file ?")
		}
		if disk.Target == nil {
			return nil, fmt.Errorf("Domain disk missing target config")
		}

		// XXX: Ignore backing file
		var diskConfig = compute.SImportDisk{
			AccessPath: strings.Trim(disk.Source.File.File, "\""),
			Index:      int(disk.Source.Index),
		}
		if disk.Target.Bus != "virtio" {
			diskConfig.Driver = "scsi"
		} else {
			diskConfig.Driver = disk.Target.Bus
		}

		if img, err := qemuimg.NewQemuImage(diskConfig.AccessPath); err != nil {
			return nil, fmt.Errorf("Domain disk %s open failed: %s", diskConfig.AccessPath, err)
		} else {
			diskConfig.SizeMb = img.GetSizeMB()
			diskConfig.Format = img.Format.String()
		}
		diskConfigs = append(diskConfigs, diskConfig)
	}
	return diskConfigs, nil
}

func (m *SGuestManager) LibvirtDomainInterfaceToNicConfig(
	domainInterfaces []libvirtxml.DomainInterface) ([]compute.SImportNic, error) {
	var nicConfigs = []compute.SImportNic{}
	for idx, nic := range domainInterfaces {
		if nic.MAC == nil {
			return nil, fmt.Errorf("Domain interface %#v missing mac address", nic)
		}
		nicConfigs = append(nicConfigs, compute.SImportNic{
			Index:  idx,
			Mac:    nic.MAC.Address,
			Driver: "virtio", //default driver
		})
	}
	return nicConfigs, nil
}
