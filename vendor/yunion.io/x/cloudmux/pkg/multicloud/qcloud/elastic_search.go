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

package qcloud

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElasticSearch struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	QcloudTags
	region *SRegion

	InstanceId   string
	InstanceName string
	Region       string
	Zone         string
	AppId        int
	Uin          string
	VpcUid       string
	SubnetUid    string
	Status       int
	ChargeType   string
	ChargePeriod int
	RenewFlag    string
	NodeType     string
	NodeNum      int
	CpuNum       int
	MemSize      int
	DiskType     string
	DiskSize     int
	EsDomain     string
	EsVip        string
	EsPort       int
	KibanaUrl    string
	EsVersion    string
	EsConfig     string
	EsAcl        struct {
		BlackIpList []string
		WhiteIpList []string
	}
	CreateTime   time.Time
	UpdateTime   time.Time
	Deadline     string
	InstanceType int
	IkConfig     struct {
		MainDict []struct {
			Key  string
			Name string
			Size int
		}
		Stopwords []struct {
			Key  string
			Name string
			Size int
		}
		QQDict []struct {
			Key  string
			Name string
			Size int
		}
		Synonym []struct {
			Key  string
			Name string
			Size int
		}
		UpdateType string
	}
	MasterNodeInfo struct {
		EnableDedicatedMaster bool
		MasterNodeType        string
		MasterNodeNum         int
		MasterNodeCpuNum      int
		MasterNodeMemSize     int
		MasterNodeDiskSize    int
		MasterNodeDiskType    string
	}
	CosBackup struct {
		IsAutoBackup bool
		BackupTime   string
	}
	AllowCosBackup    bool
	LicenseType       string
	EnableHotWarmMode bool
	WarmNodeType      string
	WarmNodeNum       int
	WarmCpuNum        int
	WarmMemSize       int
	WarmDiskType      string
	WarmDiskSize      int
	NodeInfoList      []struct {
		NodeNum       int
		NodeType      string
		Type          string
		DiskType      string
		DiskSize      int
		LocalDiskInfo struct {
			LocalDiskType  string
			LocalDiskSize  int
			LocalDiskCount int
		}
		DiskCount   int
		DiskEncrypt int
	}
	EsPublicUrl   string
	MultiZoneInfo []struct {
		Zone     string
		SubnetId string
	}
	DeployMode   int
	PublicAccess string
	EsPublicAcl  struct {
		BlackIpList []string
		WhiteIpList []string
	}
	KibanaPrivateUrl    string
	KibanaPublicAccess  string
	KibanaPrivateAccess string
	SecurityType        int
	SceneType           int
	KibanaConfig        string
	KibanaNodeInfo      struct {
		KibanaNodeType     string
		KibanaNodeNum      int
		KibanaNodeCpuNum   int
		KibanaNodeMemSize  int
		KibanaNodeDiskType string
		KibanaNodeDiskSize int
	}
}

func (self *SElasticSearch) GetId() string {
	return self.InstanceId
}

func (self *SElasticSearch) GetGlobalId() string {
	return self.InstanceId
}

func (self *SElasticSearch) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("es", "instance", []string{self.InstanceId}, tags, replace)
}

func (self *SElasticSearch) GetName() string {
	if len(self.InstanceName) > 0 {
		return self.InstanceName
	}
	return self.InstanceId
}

func (self *SElasticSearch) GetVersion() string {
	return self.EsVersion
}

func (self *SElasticSearch) Refresh() error {
	es, err := self.region.GetElasticSearch(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, es)
}

func (self *SElasticSearch) GetDiskSizeGb() int {
	return self.DiskSize
}

func (self *SElasticSearch) GetStorageType() string {
	return self.DiskType
}

func (self *SElasticSearch) GetCategory() string {
	return self.LicenseType
}

func (self *SElasticSearch) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SElasticSearch) GetVpcId() string {
	return self.VpcUid
}

func (self *SElasticSearch) GetNetworkId() string {
	for _, zone := range self.MultiZoneInfo {
		if len(zone.SubnetId) > 0 {
			return zone.SubnetId
		}
	}
	return self.SubnetUid
}

func (self *SElasticSearch) GetInstanceType() string {
	return self.NodeType
}

func (self *SElasticSearch) GetVcpuCount() int {
	return self.CpuNum
}

func (self *SElasticSearch) GetVmemSizeGb() int {
	return self.MemSize
}

func (self *SElasticSearch) GetZoneId() string {
	for _, zone := range self.MultiZoneInfo {
		if len(zone.Zone) > 0 {
			return zone.Zone
		}
	}
	return self.Zone
}

func (self *SElasticSearch) IsMultiAz() bool {
	return len(self.MultiZoneInfo) > 1
}

func (self *SElasticSearch) GetBillingType() string {
	switch self.ChargeType {
	case "PREPAID", "CDHPAID":
		return billing_api.BILLING_TYPE_PREPAID
	case "POSTPAID_BY_HOUR":
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SElasticSearch) GetStatus() string {
	switch self.Status {
	case 0:
		return api.ELASITC_SEARCH_STATUS_CREATING
	case 1:
		return api.ELASTIC_SEARCH_STATUS_AVAILABLE
	case -1:
		return api.ELASTIC_SEARCH_STATUS_UNAVAILABLE
	case -2, -3:
		return api.ELASTIC_SEARCH_STATUS_DELETING
	}
	return fmt.Sprintf("%d", self.Status)
}

func (self *SElasticSearch) IsAutoRenew() bool {
	return self.RenewFlag == "NOTIFY_AND_AUTO_RENEW"
}

func (self *SElasticSearch) Delete() error {
	return self.region.DeleteElasticSearch(self.InstanceId)
}

func (self *SElasticSearch) GetAccessInfo() (*cloudprovider.ElasticSearchAccessInfo, error) {
	return &cloudprovider.ElasticSearchAccessInfo{
		Port:             self.EsPort,
		Vip:              self.EsVip,
		Domain:           self.EsPublicUrl,
		PrivateDomain:    self.EsDomain,
		KibanaUrl:        self.KibanaUrl,
		KibanaPrivateUrl: self.KibanaPrivateUrl,
	}, nil
}

func (self *SRegion) GetIElasticSearchs() ([]cloudprovider.ICloudElasticSearch, error) {
	ess := []SElasticSearch{}
	for {
		part, total, err := self.GetElasticSearchs(nil, 100, len(ess))
		if err != nil {
			return nil, errors.Wrapf(err, "GetElasticSearchs")
		}
		ess = append(ess, part...)
		if len(ess) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudElasticSearch{}
	for i := range ess {
		ess[i].region = self
		ret = append(ret, &ess[i])
	}
	return ret, nil
}

func (self *SRegion) GetIElasticSearchById(id string) (cloudprovider.ICloudElasticSearch, error) {
	es, err := self.GetElasticSearch(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasitcSearch")
	}
	return es, nil
}

func (self *SRegion) GetElasticSearch(id string) (*SElasticSearch, error) {
	if len(id) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	ret, _, err := self.GetElasticSearchs([]string{id}, 1, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticSearchs")
	}
	for i := range ret {
		ret[i].region = self
		return &ret[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetElasticSearchs(ids []string, limit, offset int) ([]SElasticSearch, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	for i, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", i)] = id
	}
	resp, err := self.esRequest("DescribeInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeInstances")
	}
	ret := []SElasticSearch{}
	err = resp.Unmarshal(&ret, "InstanceList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SRegion) DeleteElasticSearch(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := self.esRequest("DeleteInstance", params)
	return errors.Wrapf(err, "DeleteDBInstance")
}
