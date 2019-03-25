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

type DomainCaps struct {
	XMLName   xml.Name             `xml:"domainCapabilities"`
	Path      string               `xml:"path"`
	Domain    string               `xml:"domain"`
	Machine   string               `xml:"machine,omitempty"`
	Arch      string               `xml:"arch"`
	VCPU      *DomainCapsVCPU      `xml:"vcpu"`
	IOThreads *DomainCapsIOThreads `xml:"iothreads"`
	OS        *DomainCapsOS        `xml:"os"`
	CPU       *DomainCapsCPU       `xml:"cpu"`
	Devices   *DomainCapsDevices   `xml:"devices"`
	Features  *DomainCapsFeatures  `xml:"features"`
}

type DomainCapsVCPU struct {
	Max uint `xml:"max,attr"`
}

type DomainCapsOS struct {
	Supported string              `xml:"supported,attr"`
	Loader    *DomainCapsOSLoader `xml:"loader"`
}

type DomainCapsOSLoader struct {
	Supported string           `xml:"supported,attr"`
	Values    []string         `xml:"value"`
	Enums     []DomainCapsEnum `xml:"enum"`
}

type DomainCapsIOThreads struct {
	Supported string `xml:"supported,attr"`
}

type DomainCapsCPU struct {
	Modes []DomainCapsCPUMode `xml:"mode"`
}

type DomainCapsCPUMode struct {
	Name      string                 `xml:"name,attr"`
	Supported string                 `xml:"supported,attr"`
	Models    []DomainCapsCPUModel   `xml:"model"`
	Vendor    string                 `xml:"vendor,omitempty"`
	Features  []DomainCapsCPUFeature `xml:"feature"`
}

type DomainCapsCPUModel struct {
	Name     string `xml:",chardata"`
	Usable   string `xml:"usable,attr,omitempty"`
	Fallback string `xml:"fallback,attr,omitempty"`
}

type DomainCapsCPUFeature struct {
	Policy string `xml:"policy,attr,omitempty"`
	Name   string `xml:"name,attr"`
}

type DomainCapsEnum struct {
	Name   string   `xml:"name,attr"`
	Values []string `xml:"value"`
}

type DomainCapsDevices struct {
	Disk     *DomainCapsDevice `xml:"disk"`
	Graphics *DomainCapsDevice `xml:"graphics"`
	Video    *DomainCapsDevice `xml:"video"`
	HostDev  *DomainCapsDevice `xml:"hostdev"`
}

type DomainCapsDevice struct {
	Supported string           `xml:"supported,attr"`
	Enums     []DomainCapsEnum `xml:"enum"`
}

type DomainCapsFeatures struct {
	GIC        *DomainCapsFeatureGIC        `xml:"gic"`
	VMCoreInfo *DomainCapsFeatureVMCoreInfo `xml:"vmcoreinfo"`
	GenID      *DomainCapsFeatureGenID      `xml:"genid"`
	SEV        *DomainCapsFeatureSEV        `xml:"sev"`
}

type DomainCapsFeatureGIC struct {
	Supported string           `xml:"supported,attr"`
	Enums     []DomainCapsEnum `xml:"enum"`
}

type DomainCapsFeatureVMCoreInfo struct {
	Supported string `xml:"supported,attr"`
}

type DomainCapsFeatureGenID struct {
	Supported string `xml:"supported,attr"`
}

type DomainCapsFeatureSEV struct {
	Supported       string `xml:"supported,attr"`
	CBitPos         uint   `xml:"cbitpos,omitempty"`
	ReducedPhysBits uint   `xml:"reducedPhysBits,omitempty"`
}

func (c *DomainCaps) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), c)
}

func (c *DomainCaps) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
