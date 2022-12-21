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

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKafka struct {
	multicloud.SBillingBase
	multicloud.SVirtualResourceBase
	AliyunTags
	region *SRegion

	AllConfig                string `json:"AllConfig"`
	DeployType               int    `json:"DeployType"`
	SpecType                 string `json:"SpecType"`
	PaidType                 int    `json:"PaidType"`
	InstanceId               string `json:"InstanceId"`
	MsgRetain                int    `json:"MsgRetain"`
	ZoneId                   string `json:"ZoneId"`
	IoMax                    int    `json:"IoMax"`
	VSwitchId                string `json:"VSwitchId"`
	VpcId                    string `json:"VpcId"`
	UpgradeServiceDetailInfo struct {
		Current2OpenSourceVersion string `json:"Current2OpenSourceVersion"`
	} `json:"UpgradeServiceDetailInfo"`
	ServiceStatus int    `json:"ServiceStatus"`
	Name          string `json:"Name"`
	TopicNumLimit int    `json:"TopicNumLimit"`
	DiskSize      int    `json:"DiskSize"`
	RegionId      string `json:"RegionId"`
	CreateTime    int64  `json:"CreateTime"`
	SslEndPoint   string `json:"SslEndPoint"`
	EipMax        int    `json:"EipMax"`
	EndPoint      string `json:"EndPoint"`
	ExpiredTime   int64  `json:"ExpiredTime"`
	DiskType      int    `json:"DiskType"`
	SecurityGroup string `json:"SecurityGroup"`
}

func (self *SKafka) GetTags() (map[string]string, error) {
	return self.AliyunTags.GetTags()
}

func (self *SKafka) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.AliyunTags.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			}
		}
	}
	return ret
}

func (self *SKafka) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags(ALIYUN_SERVICE_KAFKA, "INSTANCE", self.InstanceId, tags, replace)
}

func (self *SKafka) GetName() string {
	return self.Name
}

func (self *SKafka) GetGlobalId() string {
	return self.InstanceId
}

func (self *SKafka) GetId() string {
	return self.InstanceId
}

func (self *SKafka) GetVpcId() string {
	return self.VpcId
}

func (self *SKafka) GetNetworkId() string {
	return self.VSwitchId
}

func (self *SKafka) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime/1000, self.CreateTime%1000)
}

func (self *SKafka) GetBillingType() string {
	if self.PaidType == 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SKafka) GetInstanceType() string {
	return self.SpecType
}

func (self *SKafka) GetDiskSizeGb() int {
	return self.DiskSize
}

func (self *SKafka) GetVersion() string {
	return self.UpgradeServiceDetailInfo.Current2OpenSourceVersion
}

func (self *SKafka) IsMultiAz() bool {
	return false
}

func (self *SKafka) GetStorageType() string {
	switch self.DiskType {
	case 0:
		return api.STORAGE_CLOUD_EFFICIENCY
	case 1:
		return api.STORAGE_CLOUD_SSD
	}
	return ""
}

func (self *SKafka) GetBandwidthMb() int {
	if self.EipMax > 0 {
		return self.EipMax
	}
	return self.IoMax
}

func (self *SKafka) GetEndpoint() string {
	ret := []string{}
	if len(self.EndPoint) > 0 {
		ret = append(ret, self.EndPoint)
	}
	if len(self.SslEndPoint) > 0 {
		ret = append(ret, self.SslEndPoint)
	}
	return strings.Join(ret, ",")
}

func (self *SKafka) GetMsgRetentionMinute() int {
	return self.MsgRetain * 60
}

func (self *SKafka) GetZoneId() string {
	if len(self.ZoneId) > 0 {
		return fmt.Sprintf("%s-%s", self.RegionId, strings.TrimPrefix(self.ZoneId, "zone"))
	}
	return ""
}

func (self *SKafka) GetStatus() string {
	switch self.ServiceStatus {
	case 0, 1, 2:
		return api.KAFKA_STATUS_CREATING
	case 5:
		return api.KAFKA_STATUS_AVAILABLE
	case 15:
		return api.KAFKA_STATUS_UNAVAILABLE
	}
	return api.KAFKA_STATUS_UNKNOWN
}

func (self *SKafka) Refresh() error {
	kafka, err := self.region.GetKafka(self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetKafka")
	}
	return jsonutils.Update(self, kafka)
}

func (self *SKafka) Delete() error {
	return self.region.DeleteKafka(self.InstanceId)
}

func (self *SRegion) GetICloudKafkaById(id string) (cloudprovider.ICloudKafka, error) {
	kafka, err := self.GetKafka(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKafka(%s)", id)
	}
	return kafka, nil
}

func (self *SRegion) GetICloudKafkas() ([]cloudprovider.ICloudKafka, error) {
	kafkas, err := self.GetKafkas(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKafkas")
	}
	ret := []cloudprovider.ICloudKafka{}
	for i := range kafkas {
		kafkas[i].region = self
		ret = append(ret, &kafkas[i])
	}
	return ret, nil
}

func (self *SRegion) GetKafka(id string) (*SKafka, error) {
	kafkas, err := self.GetKafkas([]string{id})
	if err != nil {
		return nil, errors.Wrapf(err, "GetKafkas")
	}
	for i := range kafkas {
		if kafkas[i].GetGlobalId() == id {
			kafkas[i].region = self
			return &kafkas[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetKafkas(ids []string) ([]SKafka, error) {
	params := map[string]string{}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceId.%d", idx)] = id
	}
	resp, err := self.kafkaRequest("GetInstanceList", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceList")
	}
	ret := struct {
		Code         int
		Message      string
		RequestId    string
		Success      bool
		InstanceList struct {
			InstanceVO []SKafka
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if ret.Code != 200 {
		return nil, errors.Errorf("message: %s requestId: %s", ret.Message, ret.RequestId)
	}

	for i := 0; i < len(ret.InstanceList.InstanceVO); i++ {
		ret.InstanceList.InstanceVO[i].region = self
	}

	return ret.InstanceList.InstanceVO, nil
}

func (self *SRegion) DeleteKafka(id string) error {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
	}
	_, err := self.kafkaRequest("DeleteInstance", params)
	return errors.Wrapf(err, "DeleteInstance")
}

func (self *SRegion) ReleaseKafka(id string) error {
	params := map[string]string{
		"RegionId":            self.RegionId,
		"InstanceId":          id,
		"ForceDeleteInstance": "true",
	}
	_, err := self.kafkaRequest("ReleaseInstance", params)
	return errors.Wrapf(err, "ReleaseInstance")
}

func (self *SKafka) GetTopics() ([]cloudprovider.SKafkaTopic, error) {
	ret := []cloudprovider.SKafkaTopic{}
	pageSize := 100
	for {
		part, total, err := self.region.GetKafkaTopics(self.InstanceId, len(ret)/pageSize+1, pageSize)
		if err != nil {
			return nil, errors.Wrapf(err, "GetKafkaTopics")
		}
		ret = append(ret, part...)
		if len(ret) >= total {
			break
		}
	}
	return ret, nil
}

func (self *SRegion) GetKafkaTopics(id string, page int, pageSize int) ([]cloudprovider.SKafkaTopic, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	params := map[string]string{
		"InstanceId":  id,
		"CurrentPage": fmt.Sprintf("%d", page),
		"PageSize":    fmt.Sprintf("%d", pageSize),
	}
	resp, err := self.kafkaRequest("GetTopicList", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetTopicList")
	}
	result := struct {
		Code      int
		Total     int
		Message   string
		TopicList struct {
			TopicVO []struct {
				CompactTopic bool
				CreateTime   int
				InstanceId   string
				LocalTopic   bool
				PartitionNum int
				RegionId     string
				Remark       string
				Status       int
				StatusName   string
				Topic        string
			}
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	ret := []cloudprovider.SKafkaTopic{}
	for _, topic := range result.TopicList.TopicVO {
		ret = append(ret, cloudprovider.SKafkaTopic{
			Id:          topic.Topic,
			Name:        topic.Topic,
			Description: topic.Remark,
		})
	}
	return ret, result.Total, nil
}
