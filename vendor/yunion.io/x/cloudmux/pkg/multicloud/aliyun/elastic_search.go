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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElasticSearch struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	region *SRegion

	Tags []struct {
		TagKey   string
		TagValue string
	}
	AdvancedDedicateMaster bool
	AdvancedSetting        struct {
		gcName string
	}
	AliwsDicts []struct {
		FileSize   int
		Name       string
		SourceType string
		Type       string
	}
	ClientNodeConfiguration struct {
		Amount   int
		Disk     int
		DiskType string
		Spec     string
	}
	InstanceCategory string
	CreatedAt        time.Time
	DedicateMaster   bool
	Description      string
	DictList         []struct {
		FileSize   int
		Name       string
		SourceType string
		Type       string
	}
	Domain                       string
	ElasticDataNodeConfiguration struct {
		Amount   int
		Disk     int
		DiskType string
		Spec     string
	}
	EnableKibanaPrivateNetwork bool
	EnableKibanaPublicNetwork  bool
	EnablePublic               bool
	EsConfig                   map[string]string
	EsIPBlacklist              []string
	EsIPWhitelist              []string
	EsVersion                  string
	ExtendConfigs              []struct {
		ConfigType      string
		Value           string
		MaintainEndTime string
		AliVersion      string
	}
	HaveClientNode      bool
	HaveKibana          bool
	InstanceId          string
	KibanaConfiguration struct {
		Amount int
		Spec   string
	}
	KibanaDomain             string
	KibanaIPWhitelist        []string
	KibanaPort               int
	KibanaPrivateIPWhitelist []string
	MasterConfiguration      struct {
		Amount   int
		Disk     int
		DiskType string
		Spec     string
	}
	NetworkConfig struct {
		Type      string
		VpcId     string
		VsArea    string
		VswitchId string
	}
	NodeAmount int
	NodeSpec   struct {
		Disk           int
		DiskEncryption bool
		DiskType       string
		Spec           string
	}
	PaymentType               string
	Port                      int
	PrivateNetworkIpWhiteList []string
	Protocol                  string
	PublicDomain              string
	PublicIpWhitelist         []string
	PublicPort                int
	ResourceGroupId           string
	Status                    string
	SynonymsDicts             []struct {
		FileSize   int
		Name       string
		SourceType string
		Type       string
	}
	UpdatedAt             time.Time
	VpcInstanceId         string
	WarmNode              bool
	WarmNodeConfiguration struct {
		Amount         int
		Disk           int
		DiskEncryption bool
		DiskType       string
		Spec           string
	}
	ZoneCount int
	ZoneInfos []struct {
		status string
		zoneId string
	}
}

func (self *SElasticSearch) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") {
			continue
		}
		if len(tag.TagKey) > 0 {
			ret[tag.TagKey] = tag.TagValue
		}
	}
	return ret, nil
}

func (self *SElasticSearch) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.Tags {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			}
		}
	}
	return ret
}

func (self *SElasticSearch) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags(ALIYUN_SERVICE_ES, "INSTANCE", self.InstanceId, tags, replace)
}

func (self *SElasticSearch) GetId() string {
	return self.InstanceId
}

func (self *SElasticSearch) GetGlobalId() string {
	return self.InstanceId
}

func (self *SElasticSearch) GetName() string {
	if len(self.Description) > 0 {
		return self.Description
	}
	return self.InstanceId
}

func (self *SElasticSearch) GetDiskSizeGb() int {
	return self.NodeSpec.Disk
}

func (self *SElasticSearch) GetStorageType() string {
	return self.NodeSpec.DiskType
}

func (self *SElasticSearch) GetCategory() string {
	return self.InstanceCategory
}

func (self *SElasticSearch) GetVersion() string {
	return strings.Split(self.EsVersion, "_")[0]
}

func (self *SElasticSearch) GetVpcId() string {
	return self.NetworkConfig.VpcId
}

func (self *SElasticSearch) GetNetworkId() string {
	return self.NetworkConfig.VswitchId
}

func (self *SElasticSearch) GetZoneId() string {
	return self.NetworkConfig.VsArea
}

func (self *SElasticSearch) IsMultiAz() bool {
	return self.ZoneCount > 1
}

func (self *SElasticSearch) GetVcpuCount() int {
	spec, ok := esSpec[self.NodeSpec.Spec]
	if ok {
		return spec.VcpuCount
	}
	return 0
}

func (self *SElasticSearch) GetVmemSizeGb() int {
	spec, ok := esSpec[self.NodeSpec.Spec]
	if ok {
		return spec.VmemSizeGb
	}
	return 0
}

func (self *SElasticSearch) GetInstanceType() string {
	return self.NodeSpec.Spec
}

func (self *SElasticSearch) Refresh() error {
	es, err := self.region.GetElasitcSearch(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, es)
}

func (self *SElasticSearch) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SElasticSearch) GetBillingType() string {
	return self.PaymentType
}

func (self *SElasticSearch) GetProjectId() string {
	return self.ResourceGroupId
}

func (self *SElasticSearch) GetStatus() string {
	switch self.Status {
	case "active":
		return api.ELASTIC_SEARCH_STATUS_AVAILABLE
	case "activating":
		return api.ELASITC_SEARCH_STATUS_CREATING
	case "inactive":
		return api.ELASTIC_SEARCH_STATUS_UNAVAILABLE
	case "invalid":
		return api.ELASTIC_SEARCH_STATUS_DELETING
	default:
		return self.Status
	}
}

func (self *SElasticSearch) GetAccessInfo() (*cloudprovider.ElasticSearchAccessInfo, error) {
	return &cloudprovider.ElasticSearchAccessInfo{
		Domain:        self.PublicDomain,
		PrivateDomain: self.Domain,
		Port:          self.PublicPort,
		PrivatePort:   self.Port,
		KibanaUrl:     self.KibanaDomain,
	}, nil
}

func (self *SRegion) GetIElasticSearchs() ([]cloudprovider.ICloudElasticSearch, error) {
	ret := []SElasticSearch{}
	for {
		part, total, err := self.GetElasticSearchs(100, len(ret)/100+1)
		if err != nil {
			return nil, errors.Wrapf(err, "GetElasitcSearchs")
		}
		ret = append(ret, part...)
		if len(ret) >= total {
			break
		}
	}
	result := []cloudprovider.ICloudElasticSearch{}
	for i := range ret {
		ret[i].region = self
		result = append(result, &ret[i])
	}
	return result, nil
}

func (self *SRegion) GetIElasticSearchById(id string) (cloudprovider.ICloudElasticSearch, error) {
	es, err := self.GetElasitcSearch(id)
	if err != nil {
		return nil, err
	}
	return es, nil
}

func (self *SRegion) GetElasticSearchs(size, page int) ([]SElasticSearch, int, error) {
	if size < 1 || size > 100 {
		size = 100
	}
	if page < 1 {
		page = 1
	}
	params := map[string]string{
		"PathPattern": "/openapi/instances",
		"size":        fmt.Sprintf("%d", size),
		"page":        fmt.Sprintf("%d", page),
	}
	resp, err := self.esRequest("ListInstance", params, nil)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "ListInstance")
	}
	ret := []SElasticSearch{}
	err = resp.Unmarshal(&ret, "Result")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("Headers", "X-Total-Count")
	return ret, int(totalCount), nil
}

func (self *SRegion) GetElasitcSearch(id string) (*SElasticSearch, error) {
	if len(id) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty id")
	}
	params := map[string]string{
		"PathPattern": fmt.Sprintf("/openapi/instances/%s", id),
	}
	resp, err := self.esRequest("DescribeInstance", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstance")
	}
	ret := SElasticSearch{region: self}
	err = resp.Unmarshal(&ret, "Result")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return &ret, nil
}

func (self *SElasticSearch) Delete() error {
	return self.region.DeleteElasticSearch(self.InstanceId)
}

func (self *SRegion) DeleteElasticSearch(id string) error {
	params := map[string]string{
		"clientToken": utils.GenRequestId(20),
		"deleteType":  "immediate",
		"PathPattern": fmt.Sprintf("/openapi/instances/%s", id),
	}
	_, err := self.esRequest("DeleteInstance", params, nil)
	return errors.Wrapf(err, "DeleteInstance")
}
