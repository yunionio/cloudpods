package guestman

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"unicode"

	libvirtxml "github.com/libvirt/libvirt-go-xml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
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
	disks, err := createConfig.GuestDesc.GetArray("disks")
	if err != nil {
		return nil, err
	}

	disksPath := jsonutils.NewDict()
	for _, disk := range disks {
		diskId, _ := disk.GetString("disk_id")
		diskPath, err := createConfig.DisksPath.GetString(diskId)
		if err != nil {
			return nil, fmt.Errorf("Disks path missing disk %s", diskId)
		}
		storageId, _ := disk.GetString("storage_id")
		storage := storageman.GetManager().GetStorage(storageId)
		if storage == nil {
			return nil, fmt.Errorf("Host has no stroage %s", storageId)
		}
		iDisk := storage.CreateDisk(diskId)

		output, err := procutils.NewCommand("mv", diskPath, iDisk.GetPath()).Run()
		if err != nil {
			return nil, fmt.Errorf("Mv disk from %s to %s error %s", diskPath, iDisk.GetPath(), output)
		}
		disksPath.Set(diskId, jsonutils.NewString(iDisk.GetPath()))
	}
	guest := m.Servers[createConfig.Sid]
	if err = guest.SaveDesc(createConfig.GuestDesc); err != nil {
		return nil, err
	}

	ret := jsonutils.NewDict()
	ret.Set("disks_path", disksPath)
	return ret, nil
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
		if nic.Mac == mac {
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
		for mac, _ := range server.MacIp {
			if !IsMacInGuestConfig(guestConfig, mac) {
				Matched = false
				break
			} else {
				Matched = true
			}
		}

		if Matched {
			for idx, nic := range guestConfig.Nics {
				guestConfig.Nics[idx].Ip = server.MacIp[nic.Mac]
			}

			log.Infof("Import guest %s", guestConfig.Name)
			return i, nil
		}
	}
	return -1, fmt.Errorf("No guest %s found in import config", guestConfig.Id)
}

func (m *SGuestManager) GenerateDescFromXml(libvirtConfig *compute.SLibvirtHostConfig) (jsonutils.JSONObject, error) {
	xmlFiles, err := ioutil.ReadDir(libvirtConfig.XmlFilePath)
	if err != nil {
		log.Errorf("Read dir %s error: %s", libvirtConfig.XmlFilePath, err)
		return nil, err
	}

	libvirtServers := []*compute.SImportGuestDesc{}
	for _, f := range xmlFiles {
		if !f.Mode().IsRegular() {
			continue
		}
		xmlContent, err := ioutil.ReadFile(path.Join(libvirtConfig.XmlFilePath, f.Name()))
		if err != nil {
			log.Errorf("Read file %s error: %s", f.Name(), err)
			continue
		}
		domain := &libvirtxml.Domain{}
		err = domain.Unmarshal(string(xmlContent))
		if err != nil {
			log.Errorf("Unmarshal xml file %s error %s", f.Name(), err)
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
		if disk.Source == nil || disk.Source.File == nil {
			return nil, fmt.Errorf("Domain disk missing source file ?")
		}
		if disk.Target == nil {
			return nil, fmt.Errorf("Domain disk missing target config")
		}

		// XXX: Ignore backing file
		var diskConfig = compute.SImportDisk{
			AccessPath: disk.Source.File.File,
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
