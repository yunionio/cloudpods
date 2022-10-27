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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKafka struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	QcloudTags
	region *SRegion

	InstanceId   string
	InstanceName string
	Vip          string
	Vport        string
	VipList      []struct {
		Vip   string
		Vport string
	}
	Status             int
	Bandwidth          int
	DiskSize           int
	ZoneId             int
	VpcId              string
	SubnetId           string
	RenewFlag          int
	Healthy            int
	HealthyMessage     string
	CreateTime         int
	ExpireTime         int
	IsInternal         int
	TopicNum           int
	Version            string
	ZoneIds            []int
	Cvm                int
	InstanceType       string
	DiskType           string
	MaxTopicNumber     int
	MaxPartitionNubmer int
	RebalanceTime      string
	MsgRetentionTime   int
}

func (self *SKafka) GetName() string {
	if len(self.InstanceName) > 0 {
		return self.InstanceName
	}
	return self.InstanceId
}

func (self *SKafka) GetId() string {
	return self.InstanceId
}

func (self *SKafka) GetGlobalId() string {
	return self.InstanceId
}

func (self *SKafka) GetVpcId() string {
	return self.VpcId
}

func (self *SKafka) GetNetworkId() string {
	return self.SubnetId
}

func (self *SKafka) IsAutoRenew() bool {
	return self.RenewFlag == 1
}

func (self *SKafka) GetBillingType() string {
	return billing_api.BILLING_TYPE_PREPAID
}

func (self *SKafka) GetCreatedAt() time.Time {
	return time.Unix(int64(self.CreateTime), 0)
}

func (self *SKafka) GetExpiredAt() time.Time {
	return time.Unix(int64(self.ExpireTime), 0)
}

func (self *SKafka) GetInstanceType() string {
	return self.InstanceType
}

func (self *SKafka) GetDiskSizeGb() int {
	return self.DiskSize
}

func (self *SKafka) GetVersion() string {
	return self.Version
}

func (self *SKafka) IsMultiAz() bool {
	return len(self.ZoneIds) > 1
}

func (self *SKafka) GetBandwidthMb() int {
	return self.Bandwidth / 8
}

func (self *SKafka) GetStorageType() string {
	return self.DiskType
}

func (self *SKafka) GetEndpoint() string {
	endpoints := []string{}
	var add = func(vip, vport string) {
		endpoint := fmt.Sprintf("%s:%s", vip, vport)
		if len(vip) > 0 && len(vport) > 0 && !utils.IsInStringArray(endpoint, endpoints) {
			endpoints = append(endpoints, endpoint)
		}
	}
	for _, ed := range self.VipList {
		add(ed.Vip, ed.Vport)
	}
	add(self.Vip, self.Vport)
	return strings.Join(endpoints, ",")
}

func (self *SKafka) GetStatus() string {
	switch self.Status {
	case 0:
		return api.KAFKA_STATUS_CREATING
	case 1:
		return api.KAFKA_STATUS_AVAILABLE
	case 2:
		return api.KAFKA_STATUS_DELETING
	default:
		return fmt.Sprintf("%d", self.Status)
	}
}

func (self *SKafka) GetMsgRetentionMinute() int {
	if self.MsgRetentionTime == 0 {
		self.Refresh()
	}
	return self.MsgRetentionTime
}

func (self *SKafka) GetZoneId() string {
	return fmt.Sprintf("%s-%d", self.region.Region, self.ZoneId)
}

func (self *SKafka) Refresh() error {
	kafka, err := self.region.GetKafka(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, kafka)
}

func (self *SKafka) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetICloudKafkaById(id string) (cloudprovider.ICloudKafka, error) {
	kafka, err := self.GetKafka(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKafka(%s)", id)
	}
	return kafka, nil
}

func (self *SRegion) GetICloudKafkas() ([]cloudprovider.ICloudKafka, error) {
	kafkas := []SKafka{}
	for {
		part, total, err := self.GetKafkas("", 20, len(kafkas))
		if err != nil {
			return nil, errors.Wrapf(err, "GetKafkas")
		}
		kafkas = append(kafkas, part...)
		if len(kafkas) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudKafka{}
	for i := range kafkas {
		kafkas[i].region = self
		ret = append(ret, &kafkas[i])
	}
	return ret, nil
}

func (self *SRegion) GetKafka(id string) (*SKafka, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	resp, err := self.kafkaRequest("DescribeInstanceAttributes", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstanceAttributes")
	}
	ret := SKafka{region: self}
	err = resp.Unmarshal(&ret, "Result")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return &ret, nil
}

func (self *SRegion) GetKafkas(id string, limit, offset int) ([]SKafka, int, error) {
	if limit < 1 || limit > 20 {
		limit = 20
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	if len(id) > 0 {
		params["InstanceId"] = id
	}
	resp, err := self.kafkaRequest("DescribeInstancesDetail", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeInstancesDetail")
	}
	ret := []SKafka{}
	err = resp.Unmarshal(&ret, "Result", "InstanceList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("Result", "TotalCount")
	return ret, int(totalCount), nil
}
