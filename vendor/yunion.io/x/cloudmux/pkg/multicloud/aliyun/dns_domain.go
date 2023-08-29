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

package aliyun

import (
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDomain struct {
	multicloud.SVirtualResourceBase
	AliyunTags
	client      *SAliyunClient
	ttlMinValue int64
	PunyCode    string     `json:"PunyCode"`
	VersionCode string     `json:"VersionCode"`
	InstanceId  string     `json:"InstanceId"`
	AliDomain   bool       `json:"AliDomain"`
	DomainName  string     `json:"DomainName"`
	DomainId    string     `json:"DomainId"`
	DNSServers  DNSServers `json:"DnsServers"`
	GroupID     string     `json:"GroupId"`
}

type SDomains struct {
	PageNumber int `json:"PageNumber"`
	TotalCount int `json:"TotalCount"`
	PageSize   int `json:"PageSize"`
	// RequestID  string  `json:"RequestId"`
	Domains sDomains `json:"Domains"`
}

type DNSServers struct {
	DNSServer []string `json:"DnsServer"`
}

type sDomains struct {
	Domain []SDomain `json:"Domain"`
}

// https://help.aliyun.com/document_detail/29758.html?spm=a2c4g.11186623.6.653.2ad93b59euq4oF
type SDNSProduct struct {
	client                *SAliyunClient
	InstanceId            string `json:"InstanceId"`
	VersionCode           string `json:"VersionCode"`
	VersionName           string `json:"VersionName"`
	StartTime             int64  `json:"StartTime"`
	EndTime               int64  `json:"EndTime"`
	Domain                string `json:"Domain"`
	BindCount             int64  `json:"BindCount"`
	BindUsedCount         int64  `json:"BindUsedCount"`
	TTLMinValue           int64  `json:"TTLMinValue"`
	SubDomainLevel        int64  `json:"SubDomainLevel"`
	DNSSLBCount           int64  `json:"DnsSLBCount"`
	URLForwardCount       int64  `json:"URLForwardCount"`
	DDosDefendFlow        int64  `json:"DDosDefendFlow"`
	DDosDefendQuery       int64  `json:"DDosDefendQuery"`
	OverseaDDosDefendFlow int64  `json:"OverseaDDosDefendFlow"`
	SearchEngineLines     string `json:"SearchEngineLines"`
	ISPLines              string `json:"ISPLines"`
	ISPRegionLines        string `json:"ISPRegionLines"`
	OverseaLine           string `json:"OverseaLine"`
}

type sDNSProducts struct {
	DNSProduct []SDNSProduct `json:"DnsProduct"`
}

type SDNSProducts struct {
	// RequestID   string       `json:"RequestId"`
	TotalCount  int          `json:"TotalCount"`
	PageNumber  int          `json:"PageNumber"`
	PageSize    int          `json:"PageSize"`
	DNSProducts sDNSProducts `json:"DnsProducts"`
}

// https://help.aliyun.com/document_detail/29758.html?spm=a2c4g.11186623.6.653.4b6d6970WphX2t
func (client *SAliyunClient) DescribeDnsProductInstances(pageNumber int, pageSize int) (SDNSProducts, error) {
	sproducts := SDNSProducts{}
	params := map[string]string{}
	params["Action"] = "DescribeDnsProductInstances"
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.alidnsRequest("DescribeDnsProductInstances", params)
	if err != nil {
		return sproducts, errors.Wrap(err, "DescribeDnsProductInstances")
	}
	err = resp.Unmarshal(&sproducts, "DnsProducts")
	if err != nil {
		return sproducts, errors.Wrap(err, "resp.Unmarshal")
	}
	return sproducts, nil
}

func (client *SAliyunClient) GetAllDnsProductInstances() ([]SDNSProduct, error) {
	pageNumber := 0
	sproducts := []SDNSProduct{}
	for {
		pageNumber++
		products, err := client.DescribeDnsProductInstances(pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeDnsProductInstances(%d, 20)", pageNumber)
		}
		sproducts = append(sproducts, products.DNSProducts.DNSProduct...)
		if len(sproducts) >= products.TotalCount {
			break
		}
	}
	for i := 0; i < len(sproducts); i++ {
		sproducts[i].client = client
	}
	return sproducts, nil
}

// https://help.aliyun.com/document_detail/29751.html?spm=a2c4g.11186623.6.638.55563230d00kzJ
func (client *SAliyunClient) DescribeDomains(pageNumber int, pageSize int) (SDomains, error) {
	sdomains := SDomains{}
	params := map[string]string{}
	params["Action"] = "DescribeDomains"
	params["PageNumber"] = strconv.Itoa(pageNumber)
	params["PageSize"] = strconv.Itoa(pageSize)
	resp, err := client.alidnsRequest("DescribeDomains", params)
	if err != nil {
		return sdomains, errors.Wrap(err, "DescribeDomains")
	}
	err = resp.Unmarshal(&sdomains)
	if err != nil {
		return sdomains, errors.Wrap(err, "resp.Unmarshal")
	}
	return sdomains, nil
}

func (client *SAliyunClient) GetAllDomains() ([]SDomain, error) {
	pageNumber := 0
	sdomains := []SDomain{}
	for {
		pageNumber += 1
		domains, err := client.DescribeDomains(pageNumber, 20)
		if err != nil {
			return nil, errors.Wrapf(err, "client.DescribeDomains(%d, 20)", pageNumber)
		}
		sdomains = append(sdomains, domains.Domains.Domain...)
		if len(sdomains) >= domains.TotalCount {
			break
		}
	}
	for i := 0; i < len(sdomains); i++ {
		sdomains[i].client = client
	}
	return sdomains, nil
}

func (client *SAliyunClient) GetPublicICloudDnsZones() ([]cloudprovider.ICloudDnsZone, error) {
	izones := []cloudprovider.ICloudDnsZone{}
	sdomains, err := client.GetAllDomains()
	if err != nil {
		return nil, errors.Wrap(err, "client.GetAllDomains()")
	}
	for i := 0; i < len(sdomains); i++ {
		izones = append(izones, &sdomains[i])
	}
	return izones, nil
}

func (client *SAliyunClient) DescribeDomainInfo(domainName string) (*SDomain, error) {
	sdomain := &SDomain{client: client}
	params := map[string]string{}
	params["Action"] = "DescribeDomainInfo"
	params["DomainName"] = domainName
	resp, err := client.alidnsRequest("DescribeDomainInfo", params)
	if err != nil {
		return sdomain, errors.Wrap(err, "DescribeDomainInfo")
	}
	err = resp.Unmarshal(sdomain)
	if err != nil {
		return sdomain, errors.Wrap(err, "resp.Unmarshal")
	}
	return sdomain, nil
}

func (client *SAliyunClient) GetPublicICloudDnsZoneById(id string) (cloudprovider.ICloudDnsZone, error) {
	izones, err := client.GetPublicICloudDnsZones()
	if err != nil {
		return nil, errors.Wrapf(err, "client.DescribeDomainInfo(%s)", id)
	}
	for i := 0; i < len(izones); i++ {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SAliyunClient) AddDomain(domainName string) (*SDomain, error) {
	sdomain := &SDomain{client: client}
	params := map[string]string{}
	params["Action"] = "AddDomain"
	params["DomainName"] = domainName
	resp, err := client.alidnsRequest("AddDomain", params)
	if err != nil {
		return sdomain, errors.Wrap(err, "AddDomain")
	}
	err = resp.Unmarshal(sdomain)
	if err != nil {
		return sdomain, errors.Wrapf(err, "%s:resp.Unmarshal()", resp)
	}
	return sdomain, nil
}

func (client *SAliyunClient) CreatePublicICloudDnsZone(opts *cloudprovider.SDnsZoneCreateOptions) (cloudprovider.ICloudDnsZone, error) {
	sdomain, err := client.AddDomain(opts.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "client.AddDomain(%s)", opts.Name)
	}
	return sdomain, nil
}

func (client *SAliyunClient) DeleteDomain(domainName string) error {
	params := map[string]string{}
	params["Action"] = "DeleteDomain"
	params["DomainName"] = domainName
	_, err := client.alidnsRequest("DeleteDomain", params)
	if err != nil {
		return errors.Wrap(err, "DeleteDomain")
	}
	return nil
}

func (self *SDomain) GetId() string {
	return self.DomainId
}

func (self *SDomain) GetName() string {
	if len(self.PunyCode) > 0 {
		return self.PunyCode
	}
	return self.DomainName
}

func (self *SDomain) GetGlobalId() string {
	return self.GetId()
}

func (self *SDomain) GetStatus() string {
	return api.DNS_ZONE_STATUS_AVAILABLE
}

func (self *SDomain) Refresh() error {
	sdomain, err := self.client.DescribeDomainInfo(self.DomainName)
	if err != nil {
		return errors.Wrapf(err, "self.client.DescribeDomainInfo(%s)", self.DomainName)
	}
	return jsonutils.Update(self, sdomain)
}

func (self *SDomain) GetZoneType() cloudprovider.TDnsZoneType {
	return cloudprovider.PublicZone
}

func (self *SDomain) GetICloudVpcIds() ([]string, error) {
	return nil, nil
}

func (self *SDomain) AddVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDomain) RemoveVpc(vpc *cloudprovider.SPrivateZoneVpc) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDomain) GetIDnsRecords() ([]cloudprovider.ICloudDnsRecord, error) {
	irecords := []cloudprovider.ICloudDnsRecord{}
	records, err := self.client.GetAllDomainRecords(self.DomainName)
	if err != nil {
		return nil, errors.Wrapf(err, "self.client.GetAllDomainRecords(%s)", self.DomainName)
	}
	for i := 0; i < len(records); i++ {
		records[i].domain = self
		irecords = append(irecords, &records[i])
	}
	return irecords, nil
}

func (self *SDomain) GetIDnsRecordById(id string) (cloudprovider.ICloudDnsRecord, error) {
	records, err := self.GetIDnsRecords()
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].GetGlobalId() == id {
			return records[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SDomain) AddDnsRecord(opts *cloudprovider.DnsRecord) (string, error) {
	recordId, err := self.client.AddDomainRecord(self.DomainName, opts)
	if err != nil {
		return "", errors.Wrapf(err, "AddDomainRecord(%s)", self.DomainName)
	}
	if !opts.Enabled {
		// Enable: 启用解析 Disable: 暂停解析
		self.client.SetDomainRecordStatus(recordId, "Disable")
	}
	if len(opts.Desc) > 0 {
		self.client.UpdateDomainRecordRemark(recordId, opts.Desc)
	}
	return recordId, nil
}

func (self *SDomain) Delete() error {
	return self.client.DeleteDomain(self.DomainName)
}

func TDnsProductType(productName string) cloudprovider.TDnsProductType {
	switch productName {
	case "企业旗舰版":
		return cloudprovider.DnsProductEnterpriseUltimate
	case "企业标准版":
		return cloudprovider.DnsProductEnterpriseStandard
	case "个人版":
		return cloudprovider.DnsProductPersonalProfessional
	default:
		return cloudprovider.DnsProductFree
	}
}

func (self *SDomain) GetDnsProductType() cloudprovider.TDnsProductType {
	sproducts, err := self.client.GetAllDnsProductInstances()
	if err != nil {
		log.Errorf("self.client.GetAllDnsProductInstances():%s", err)
		return cloudprovider.DnsProductFree
	}
	// https://help.aliyun.com/document_detail/29806.html?spm=a2c4g.11186623.4.1.67728197c8SCN9
	// 免费版，最低600
	self.ttlMinValue = 600
	for i := 0; i < len(sproducts); i++ {
		if sproducts[i].Domain == self.DomainName {
			return TDnsProductType(sproducts[i].VersionName)
		}
	}
	return cloudprovider.DnsProductFree
}

func (self *SDomain) fetchTTLMinValue() (int64, error) {
	if self.ttlMinValue != 0 {
		return self.ttlMinValue, nil
	}
	sproducts, err := self.client.GetAllDnsProductInstances()
	if err != nil {
		return 0, errors.Wrap(err, "self.client.GetAllDnsProductInstances()")
	}
	// https://help.aliyun.com/document_detail/29806.html?spm=a2c4g.11186623.4.1.67728197c8SCN9
	// 免费版，最低600
	self.ttlMinValue = 600
	for i := 0; i < len(sproducts); i++ {
		if sproducts[i].Domain == self.DomainName {
			self.ttlMinValue = sproducts[i].TTLMinValue
			return self.ttlMinValue, nil
		}
	}
	return self.ttlMinValue, nil
}

func (self *SDomain) GetProperlyTTL(ttl int64) int64 {
	ttlMin, err := self.fetchTTLMinValue()
	if err != nil {
		log.Errorf("self.fetchTTLMinValue():%s", err)
		ttlMin = 600
	}
	if ttl <= ttlMin {
		return ttlMin
	}
	if ttl > 86400 {
		return 86400
	}
	return ttl
}
