/*
 * This file is part of the libvirt-go-xml project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirtxml

import (
	"encoding/xml"
)

type CapsHostCPUTopology struct {
	Sockets int `xml:"sockets,attr"`
	Cores   int `xml:"cores,attr"`
	Threads int `xml:"threads,attr"`
}

type CapsHostCPUFeatureFlag struct {
	Name string `xml:"name,attr"`
}

type CapsHostCPUPageSize struct {
	Size int    `xml:"size,attr"`
	Unit string `xml:"unit,attr"`
}

type CapsHostCPUMicrocode struct {
	Version int `xml:"version,attr"`
}

type CapsHostCPU struct {
	XMLName      xml.Name                 `xml:"cpu"`
	Arch         string                   `xml:"arch,omitempty"`
	Model        string                   `xml:"model,omitempty"`
	Vendor       string                   `xml:"vendor,omitempty"`
	Topology     *CapsHostCPUTopology     `xml:"topology"`
	FeatureFlags []CapsHostCPUFeatureFlag `xml:"feature"`
	Features     *CapsHostCPUFeatures     `xml:"features"`
	PageSizes    []CapsHostCPUPageSize    `xml:"pages"`
	Microcode    *CapsHostCPUMicrocode    `xml:"microcode"`
}

type CapsHostCPUFeature struct {
}

type CapsHostCPUFeatures struct {
	PAE    *CapsHostCPUFeature `xml:"pae"`
	NonPAE *CapsHostCPUFeature `xml:"nonpae"`
	SVM    *CapsHostCPUFeature `xml:"svm"`
	VMX    *CapsHostCPUFeature `xml:"vmx"`
}

type CapsHostNUMAMemory struct {
	Size uint64 `xml:",chardata"`
	Unit string `xml:"unit,attr"`
}

type CapsHostNUMAPageInfo struct {
	Size  int    `xml:"size,attr"`
	Unit  string `xml:"unit,attr"`
	Count uint64 `xml:",chardata"`
}

type CapsHostNUMACPU struct {
	ID       int    `xml:"id,attr"`
	SocketID *int   `xml:"socket_id,attr"`
	CoreID   *int   `xml:"core_id,attr"`
	Siblings string `xml:"siblings,attr,omitempty"`
}

type CapsHostNUMASibling struct {
	ID    int `xml:"id,attr"`
	Value int `xml:"value,attr"`
}

type CapsHostNUMACell struct {
	ID        int                    `xml:"id,attr"`
	Memory    *CapsHostNUMAMemory    `xml:"memory"`
	PageInfo  []CapsHostNUMAPageInfo `xml:"pages"`
	Distances *CapsHostNUMADistances `xml:"distances"`
	CPUS      *CapsHostNUMACPUs      `xml:"cpus"`
}

type CapsHostNUMADistances struct {
	Siblings []CapsHostNUMASibling `xml:"sibling"`
}

type CapsHostNUMACPUs struct {
	Num  uint              `xml:"num,attr,omitempty"`
	CPUs []CapsHostNUMACPU `xml:"cpu"`
}

type CapsHostNUMATopology struct {
	Cells *CapsHostNUMACells `xml:"cells"`
}

type CapsHostNUMACells struct {
	Num   uint               `xml:"num,attr,omitempty"`
	Cells []CapsHostNUMACell `xml:"cell"`
}

type CapsHostSecModelLabel struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type CapsHostSecModel struct {
	Name   string                  `xml:"model"`
	DOI    string                  `xml:"doi"`
	Labels []CapsHostSecModelLabel `xml:"baselabel"`
}

type CapsHostMigrationFeatures struct {
	Live          *CapsHostMigrationLive          `xml:"live"`
	URITransports *CapsHostMigrationURITransports `xml:"uri_transports"`
}

type CapsHostMigrationLive struct {
}

type CapsHostMigrationURITransports struct {
	URI []string `xml:"uri_transport"`
}

type CapsHost struct {
	UUID              string                     `xml:"uuid,omitempty"`
	CPU               *CapsHostCPU               `xml:"cpu"`
	PowerManagement   *CapsHostPowerManagement   `xml:"power_management"`
	IOMMU             *CapsHostIOMMU             `xml:"iommu"`
	MigrationFeatures *CapsHostMigrationFeatures `xml:"migration_features"`
	NUMA              *CapsHostNUMATopology      `xml:"topology"`
	Cache             *CapsHostCache             `xml:"cache"`
	MemoryBandwidth   *CapsHostMemoryBandwidth   `xml:"memory_bandwidth"`
	SecModel          []CapsHostSecModel         `xml:"secmodel"`
}

type CapsHostPowerManagement struct {
	SuspendMem    *CapsHostPowerManagementMode `xml:"suspend_mem"`
	SuspendDisk   *CapsHostPowerManagementMode `xml:"suspend_disk"`
	SuspendHybrid *CapsHostPowerManagementMode `xml:"suspend_hybrid"`
}

type CapsHostPowerManagementMode struct {
}

type CapsHostIOMMU struct {
	Support string `xml:"support,attr"`
}

type CapsHostCache struct {
	Banks   []CapsHostCacheBank   `xml:"bank"`
	Monitor *CapsHostCacheMonitor `xml:"monitor"`
}

type CapsHostCacheBank struct {
	ID      uint                   `xml:"id,attr"`
	Level   uint                   `xml:"level,attr"`
	Type    string                 `xml:"type,attr"`
	Size    uint                   `xml:"size,attr"`
	Unit    string                 `xml:"unit,attr"`
	CPUs    string                 `xml:"cpus,attr"`
	Control []CapsHostCacheControl `xml:"control"`
}

type CapsHostCacheMonitor struct {
	Level          uint                          `xml:"level,attr,omitempty"`
	ResueThreshold uint                          `xml:"reuseThreshold,attr,omitempty"`
	MaxMonitors    uint                          `xml:"maxMonitors,attr"`
	Features       []CapsHostCacheMonitorFeature `xml:"feature"`
}

type CapsHostCacheMonitorFeature struct {
	Name string `xml:"name,attr"`
}

type CapsHostCacheControl struct {
	Granularity uint   `xml:"granularity,attr"`
	Min         uint   `xml:"min,attr,omitempty"`
	Unit        string `xml:"unit,attr"`
	Type        string `xml:"type,attr"`
	MaxAllows   uint   `xml:"maxAllocs,attr"`
}

type CapsHostMemoryBandwidth struct {
	Nodes   []CapsHostMemoryBandwidthNode   `xml:"node"`
	Monitor *CapsHostMemoryBandwidthMonitor `xml:"monitor"`
}

type CapsHostMemoryBandwidthNode struct {
	ID      uint                                `xml:"id,attr"`
	CPUs    string                              `xml:"cpus,attr"`
	Control *CapsHostMemoryBandwidthNodeControl `xml:"control"`
}

type CapsHostMemoryBandwidthNodeControl struct {
	Granularity uint `xml:"granularity,attr"`
	Min         uint `xml:"min,attr"`
	MaxAllocs   uint `xml:"maxAllocs,attr"`
}

type CapsHostMemoryBandwidthMonitor struct {
	MaxMonitors uint                                    `xml:"maxMonitors,attr"`
	Features    []CapsHostMemoryBandwidthMonitorFeature `xml:"feature"`
}

type CapsHostMemoryBandwidthMonitorFeature struct {
	Name string `xml:"name,attr"`
}

type CapsGuestMachine struct {
	Name      string `xml:",chardata"`
	MaxCPUs   int    `xml:"maxCpus,attr,omitempty"`
	Canonical string `xml:"canonical,attr,omitempty"`
}

type CapsGuestDomain struct {
	Type     string             `xml:"type,attr"`
	Emulator string             `xml:"emulator,omitempty"`
	Machines []CapsGuestMachine `xml:"machine"`
}

type CapsGuestArch struct {
	Name     string             `xml:"name,attr"`
	WordSize string             `xml:"wordsize"`
	Emulator string             `xml:"emulator"`
	Loader   string             `xml:"loader,omitempty"`
	Machines []CapsGuestMachine `xml:"machine"`
	Domains  []CapsGuestDomain  `xml:"domain"`
}

type CapsGuestFeatureCPUSelection struct {
}

type CapsGuestFeatureDeviceBoot struct {
}

type CapsGuestFeaturePAE struct {
}

type CapsGuestFeatureNonPAE struct {
}

type CapsGuestFeatureDiskSnapshot struct {
	Default string `xml:"default,attr,omitempty"`
	Toggle  string `xml:"toggle,attr,omitempty"`
}

type CapsGuestFeatureAPIC struct {
	Default string `xml:"default,attr,omitempty"`
	Toggle  string `xml:"toggle,attr,omitempty"`
}

type CapsGuestFeatureACPI struct {
	Default string `xml:"default,attr,omitempty"`
	Toggle  string `xml:"toggle,attr,omitempty"`
}

type CapsGuestFeatureIA64BE struct {
}

type CapsGuestFeatures struct {
	CPUSelection *CapsGuestFeatureCPUSelection `xml:"cpuselection"`
	DeviceBoot   *CapsGuestFeatureDeviceBoot   `xml:"deviceboot"`
	DiskSnapshot *CapsGuestFeatureDiskSnapshot `xml:"disksnapshot"`
	PAE          *CapsGuestFeaturePAE          `xml:"pae"`
	NonPAE       *CapsGuestFeatureNonPAE       `xml:"nonpae"`
	APIC         *CapsGuestFeatureAPIC         `xml:"apic"`
	ACPI         *CapsGuestFeatureACPI         `xml:"acpi"`
	IA64BE       *CapsGuestFeatureIA64BE       `xml:"ia64_be"`
}

type CapsGuest struct {
	OSType   string             `xml:"os_type"`
	Arch     CapsGuestArch      `xml:"arch"`
	Features *CapsGuestFeatures `xml:"features"`
}

type Caps struct {
	XMLName xml.Name    `xml:"capabilities"`
	Host    CapsHost    `xml:"host"`
	Guests  []CapsGuest `xml:"guest"`
}

func (c *CapsHostCPU) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), c)
}

func (c *CapsHostCPU) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (c *Caps) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), c)
}

func (c *Caps) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
