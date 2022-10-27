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

package aws

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/elbv2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

/*
https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/Welcome.html
*/

type SElb struct {
	multicloud.SResourceBase
	region *SRegion

	Type                  string             `json:"Type"`
	Scheme                string             `json:"Scheme"`
	IPAddressType         string             `json:"IpAddressType"`
	VpcID                 string             `json:"VpcId"`
	AvailabilityZones     []AvailabilityZone `json:"AvailabilityZones"`
	CreatedTime           string             `json:"CreatedTime"`
	CanonicalHostedZoneID string             `json:"CanonicalHostedZoneId"`
	DNSName               string             `json:"DNSName"`
	SecurityGroups        []string           `json:"SecurityGroups"`
	LoadBalancerName      string             `json:"LoadBalancerName"`
	State                 State              `json:"State"`
	LoadBalancerArn       string             `json:"LoadBalancerArn"`
}

type AvailabilityZone struct {
	LoadBalancerAddresses []LoadBalancerAddress `json:"LoadBalancerAddresses"`
	ZoneName              string                `json:"ZoneName"`
	SubnetID              string                `json:"SubnetId"`
}

type LoadBalancerAddress struct {
	IPAddress    string `json:"IpAddress"`
	AllocationID string `json:"AllocationId"`
}

type State struct {
	Code string `json:"Code"`
}

func (self *SElb) GetId() string {
	return self.LoadBalancerArn
}

func (self *SElb) GetName() string {
	return self.LoadBalancerName
}

func (self *SElb) GetGlobalId() string {
	return self.GetId()
}

func (self *SElb) GetStatus() string {
	switch self.State.Code {
	case "provisioning":
		return api.LB_STATUS_INIT
	case "active":
		return api.LB_STATUS_ENABLED
	case "failed":
		return api.LB_STATUS_START_FAILED
	default:
		return api.LB_STATUS_UNKNOWN
	}
}

func (self *SElb) Refresh() error {
	ielb, err := self.region.GetILoadBalancerById(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, ielb)
	if err != nil {
		return err
	}

	return nil
}

func (self *SElb) IsEmulated() bool {
	return false
}

func (self *SElb) GetSysTags() map[string]string {
	data := map[string]string{}
	data["loadbalance_type"] = self.Type
	attrs, err := self.region.getElbAttributesById(self.GetId())
	if err != nil {
		log.Errorf("SElb GetSysTags %s", err)
		return data
	}

	for k, v := range attrs {
		data[k] = v
	}
	return data
}

func (self *SElb) GetTags() (map[string]string, error) {
	tags, err := self.region.FetchElbTags(self.LoadBalancerArn)
	if err != nil {
		return nil, errors.Wrap(err, "self.region.FetchElbTags")
	}
	return tags, nil
}

func (self *SElb) GetProjectId() string {
	return ""
}

func (self *SElb) GetAddress() string {
	return self.DNSName
}

func (self *SElb) GetAddressType() string {
	switch self.Scheme {
	case "internal":
		return api.LB_ADDR_TYPE_INTRANET
	case "internet-facing":
		return api.LB_ADDR_TYPE_INTERNET
	default:
		return api.LB_ADDR_TYPE_INTRANET
	}
}

func (self *SElb) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SElb) GetNetworkIds() []string {
	ret := []string{}
	for i := range self.AvailabilityZones {
		ret = append(ret, self.AvailabilityZones[i].SubnetID)
	}

	return ret
}

func (self *SElb) GetVpcId() string {
	return self.VpcID
}

func (self *SElb) GetZoneId() string {
	zones := []string{}
	for i := range self.AvailabilityZones {
		zones = append(zones, self.AvailabilityZones[i].ZoneName)
	}

	sort.Strings(zones)
	if len(zones) > 0 {
		z, err := self.region.getZoneById(zones[0])
		if err != nil {
			log.Infof("getZoneById %s %s", zones[0], err)
			return ""
		}

		return z.GetGlobalId()
	}

	return ""
}

func (self *SElb) GetZone1Id() string {
	return ""
}

func (self *SElb) GetLoadbalancerSpec() string {
	return self.Type
}

func (self *SElb) GetChargeType() string {
	return api.LB_CHARGE_TYPE_BY_TRAFFIC
}

func (self *SElb) GetEgressMbps() int {
	return 0
}

func (self *SElb) Delete(ctx context.Context) error {
	return self.region.DeleteElb(self.GetId())
}

func (self *SElb) Start() error {
	return nil
}

func (self *SElb) Stop() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElb) GetILoadBalancerListeners() ([]cloudprovider.ICloudLoadbalancerListener, error) {
	listeners, err := self.region.GetElbListeners(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetElbListeners")
	}

	ret := make([]cloudprovider.ICloudLoadbalancerListener, len(listeners))
	for i := range listeners {
		listeners[i].lb = self
		ret[i] = &listeners[i]
	}

	return ret, nil
}

func (self *SElb) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	backendgroups, err := self.region.GetElbBackendgroups(self.GetId(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetElbBackendgroups")
	}

	ibackendgroups := make([]cloudprovider.ICloudLoadbalancerBackendGroup, len(backendgroups))
	for i := range backendgroups {
		backendgroups[i].lb = self
		ibackendgroups[i] = &backendgroups[i]
	}

	return ibackendgroups, nil
}

func (self *SElb) CreateILoadBalancerBackendGroup(group *cloudprovider.SLoadbalancerBackendGroup) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	backendgroup, err := self.region.CreateElbBackendgroup(group)
	if err != nil {
		return nil, errors.Wrap(err, "CreateElbBackendgroup")
	}

	backendgroup.lb = self
	return backendgroup, nil
}

func (self *SElb) GetILoadBalancerBackendGroupById(groupId string) (cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	return self.region.GetElbBackendgroup(groupId)
}

func (self *SElb) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListener) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.CreateElbListener(listener)
	if err != nil {
		return nil, errors.Wrap(err, "CreateElbListener")
	}

	ret.lb = self
	return ret, nil
}

func (self *SElb) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	if listenerId == "" {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetILoadBalancerListenerById")
	}

	return self.region.GetElbListener(listenerId)
}

func (self *SElb) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, nil
}

func (self *SRegion) DeleteElb(elbId string) error {
	client, err := self.GetElbV2Client()
	if err != nil {
		return errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DeleteLoadBalancerInput{}
	params.SetLoadBalancerArn(elbId)
	_, err = client.DeleteLoadBalancer(params)
	if err != nil {
		return errors.Wrap(err, "DeleteLoadBalancer")
	}

	return nil
}

func (self *SRegion) GetElbBackendgroups(elbId string, backendgroupIds []string) ([]SElbBackendGroup, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DescribeTargetGroupsInput{}
	if len(elbId) > 0 {
		params.SetLoadBalancerArn(elbId)
	}

	if len(backendgroupIds) > 0 {
		v := make([]*string, len(backendgroupIds))
		for i := range backendgroupIds {
			v[i] = &backendgroupIds[i]
		}

		params.SetTargetGroupArns(v)
	}

	ret, err := client.DescribeTargetGroups(params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeTargetGroups")
	}

	backendgroups := []SElbBackendGroup{}
	err = unmarshalAwsOutput(ret, "TargetGroups", &backendgroups)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.TargetGroups")
	}

	for i := range backendgroups {
		backendgroups[i].region = self
	}

	return backendgroups, nil
}

func (self *SRegion) GetElbBackendgroup(backendgroupId string) (*SElbBackendGroup, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	params := &elbv2.DescribeTargetGroupsInput{}
	params.SetTargetGroupArns([]*string{&backendgroupId})

	ret, err := client.DescribeTargetGroups(params)
	if err != nil {
		if strings.Contains(err.Error(), "TargetGroupNotFound") {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, errors.Wrap(err, "DescribeTargetGroups")
	}

	backendgroups := []SElbBackendGroup{}
	err = unmarshalAwsOutput(ret, "TargetGroups", &backendgroups)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.TargetGroups")
	}

	if len(backendgroups) == 1 {
		backendgroups[0].region = self
		return &backendgroups[0], nil
	}

	return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetElbBackendgroup")
}

func ToAwsHealthCode(s string) string {
	ret := []string{}

	segs := strings.Split(s, ",")
	for _, seg := range segs {
		if seg == api.LB_HEALTH_CHECK_HTTP_CODE_4xx && !utils.IsInStringArray("400-499", ret) {
			ret = append(ret, "400-499")
		} else if seg == api.LB_HEALTH_CHECK_HTTP_CODE_3xx && !utils.IsInStringArray("300-399", ret) {
			ret = append(ret, "300-399")
		} else if seg == api.LB_HEALTH_CHECK_HTTP_CODE_2xx && !utils.IsInStringArray("200-299", ret) {
			ret = append(ret, "200-299")
		}
	}

	return strings.Join(ret, ",")
}

func ToOnecloudHealthCode(s string) string {
	ret := []string{}

	segs := strings.Split(s, ",")
	for _, seg := range segs {
		codes := strings.Split(seg, "-")
		for _, code := range codes {
			c, _ := strconv.Atoi(code)
			if c >= 400 && !utils.IsInStringArray(api.LB_HEALTH_CHECK_HTTP_CODE_4xx, ret) {
				ret = append(ret, api.LB_HEALTH_CHECK_HTTP_CODE_4xx)
			} else if c >= 300 && !utils.IsInStringArray(api.LB_HEALTH_CHECK_HTTP_CODE_3xx, ret) {
				ret = append(ret, api.LB_HEALTH_CHECK_HTTP_CODE_3xx)
			} else if c >= 200 && !utils.IsInStringArray(api.LB_HEALTH_CHECK_HTTP_CODE_2xx, ret) {
				ret = append(ret, api.LB_HEALTH_CHECK_HTTP_CODE_2xx)
			}
		}

		if len(codes) == 2 {
			min, _ := strconv.Atoi(codes[0])
			max, _ := strconv.Atoi(codes[1])

			if min >= 200 && max >= 400 {
				if !utils.IsInStringArray(api.LB_HEALTH_CHECK_HTTP_CODE_3xx, ret) {
					ret = append(ret, api.LB_HEALTH_CHECK_HTTP_CODE_3xx)
				}
			}
		}
	}

	return strings.Join(ret, ",")
}

// 目前只支持target type ：instance
func (self *SRegion) CreateElbBackendgroup(group *cloudprovider.SLoadbalancerBackendGroup) (*SElbBackendGroup, error) {
	params := &elbv2.CreateTargetGroupInput{}
	params.SetProtocol(strings.ToUpper(group.ListenType))
	params.SetPort(int64(group.ListenPort))
	params.SetVpcId(group.VpcId)
	params.SetName(group.Name)
	params.SetTargetType("instance")
	if group.HealthCheck != nil {
		params.SetHealthCheckPort("traffic-port")
		params.SetHealthCheckProtocol(strings.ToUpper(group.HealthCheck.HealthCheckType))
		params.SetHealthCheckIntervalSeconds(int64(group.HealthCheck.HealthCheckInterval))
		params.SetHealthyThresholdCount(int64(group.HealthCheck.HealthCheckRise))

		if len(group.HealthCheck.HealthCheckURI) > 0 {
			params.SetHealthCheckPath(group.HealthCheck.HealthCheckURI)
		}

		if utils.IsInStringArray(group.ListenType, []string{api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS}) {
			params.SetHealthCheckTimeoutSeconds(int64(group.HealthCheck.HealthCheckTimeout))
			params.SetUnhealthyThresholdCount(int64(group.HealthCheck.HealthCheckFail))

			codes := ToAwsHealthCode(group.HealthCheck.HealthCheckHttpCode)
			if len(codes) > 0 {
				matcher := &elbv2.Matcher{}
				matcher.SetHttpCode(codes)
				params.SetMatcher(matcher)
			}
		} else {
			// tcp & udp 健康检查阈值与不健康阈值需相同
			params.SetUnhealthyThresholdCount(int64(group.HealthCheck.HealthCheckRise))
		}
	}

	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}

	ret, err := client.CreateTargetGroup(params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateTargetGroup")
	}

	backendgroups := []SElbBackendGroup{}
	err = unmarshalAwsOutput(ret, "TargetGroups", &backendgroups)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalAwsOutput.TargetGroups")
	}

	if len(backendgroups) == 1 {
		backendgroups[0].region = self
		return &backendgroups[0], nil
	}

	return nil, fmt.Errorf("CreateElbBackendgroup error: %#v", backendgroups)
}

func (self *SElb) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := self.region.FetchElbTags(self.LoadBalancerArn)
	if err != nil {
		return errors.Wrapf(err, "self.region.FetchElbTags(%s)", self.LoadBalancerArn)
	}
	err = self.region.UpdateResourceTags(self.LoadBalancerArn, oldTags, tags, replace)
	if err != nil {
		return errors.Wrap(err, "self.region.UpdateResourceTags(self.LoadBalancerArn, oldTags, tags, replace)")
	}
	return nil
}

func (self *SRegion) FetchElbTags(arn string) (map[string]string, error) {
	client, err := self.GetElbV2Client()
	if err != nil {
		return nil, errors.Wrap(err, "GetElbV2Client")
	}
	params := elbv2.DescribeTagsInput{}
	params.SetResourceArns([]*string{&arn})
	output, err := client.DescribeTags(&params)
	if err != nil {
		return nil, errors.Wrapf(err, "client.DescribeTags(%s)", jsonutils.Marshal(params).String())
	}
	result := map[string]string{}
	for i := range output.TagDescriptions {
		if output.TagDescriptions[i].ResourceArn != nil && *output.TagDescriptions[i].ResourceArn == arn {
			for j := range output.TagDescriptions[i].Tags {
				if output.TagDescriptions[i].Tags[j].Key != nil && output.TagDescriptions[i].Tags[j].Value != nil {
					result[*output.TagDescriptions[i].Tags[j].Key] = *output.TagDescriptions[i].Tags[j].Value
				}
			}
			return result, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
