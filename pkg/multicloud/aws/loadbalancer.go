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
	"time"

	"github.com/aws/aws-sdk-go/service/elbv2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

/*
https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/Welcome.html
*/

type SElb struct {
	multicloud.SResourceBase
	region *SRegion

	AvailabilityZones     []AvailabilityZone `xml:"AvailabilityZones>member"`
	CanonicalHostedZoneId string             `xml:"CanonicalHostedZoneId"`
	CreatedTime           time.Time          `xml:"CreatedTime"`
	CustomerOwnedIpv4Pool string             `xml:"CustomerOwnedIpv4Pool"`
	DNSName               string             `xml:"DNSName"`
	IpAddressType         string             `xml:"IpAddressType"`
	LoadBalancerArn       string             `xml:"LoadBalancerArn"`
	LoadBalancerName      string             `xml:"LoadBalancerName"`
	Scheme                string             `xml:"Scheme"`
	SecurityGroups        []string           `xml:"SecurityGroups"`
	State                 LoadBalancerState  `xml:"State"`
	Type                  string             `xml:"Type"`
	VpcId                 string             `xml:"VpcId"`
}

type LoadBalancerState struct {
	Code   string `xml:"Code"`
	Reason string `xml:"Reason"`
}

type AvailabilityZone struct {
	LoadBalancerAddresses []LoadBalancerAddress `xml:"LoadBalancerAddresses"`
	ZoneName              string                `xml:"ZoneName"`
	SubnetId              string                `xml:"SubnetId"`
}

type LoadBalancerAddress struct {
	IPAddress    string `xml:"IpAddress"`
	AllocationId string `xml:"AllocationId"`
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
		ret = append(ret, self.AvailabilityZones[i].SubnetId)
	}
	return ret
}

func (self *SElb) GetVpcId() string {
	return self.VpcId
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
	params := map[string]string{
		"LoadBalancerArn": elbId,
	}
	return self.elbRequest("DeleteLoadBalancer", params, nil)
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
	params := map[string]string{
		"TargetGroupArns.member.1": backendgroupId,
	}
	result := struct {
		NextMarker string             `xml:"NextMarker"`
		Groups     []SElbBackendGroup `xml:"TargetGroups>member"`
	}{}
	err := self.elbRequest("DescribeTargetGroups", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeTargetGroups")
	}
	for i := range result.Groups {
		if result.Groups[i].GetId() == backendgroupId {
			result.Groups[i].region = self
			return &result.Groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, backendgroupId)
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
	params := map[string]string{
		"Protocol":   strings.ToUpper(group.ListenType),
		"Port":       fmt.Sprintf("%d", group.ListenPort),
		"VpcId":      group.VpcId,
		"Name":       group.Name,
		"TargetType": "instance",
	}
	if group.HealthCheck != nil {
		params["HealthCheckPort"] = "traffic-port"
		params["HealthCheckProtocol"] = strings.ToUpper(group.HealthCheck.HealthCheckType)
		params["HealthCheckIntervalSeconds"] = fmt.Sprintf("%d", group.HealthCheck.HealthCheckInterval)
		params["HealthyThresholdCount"] = fmt.Sprintf("%d", group.HealthCheck.HealthCheckRise)
		if len(group.HealthCheck.HealthCheckURI) > 0 {
			params["HealthCheckPath"] = group.HealthCheck.HealthCheckURI
		}

		if utils.IsInStringArray(group.ListenType, []string{api.LB_HEALTH_CHECK_HTTP, api.LB_HEALTH_CHECK_HTTPS}) {
			params["HealthCheckTimeoutSeconds"] = fmt.Sprintf("%d", group.HealthCheck.HealthCheckTimeout)
			params["UnhealthyThresholdCount"] = fmt.Sprintf("%d", group.HealthCheck.HealthCheckFail)
			codes := ToAwsHealthCode(group.HealthCheck.HealthCheckHttpCode)
			if len(codes) > 0 {
				params["Matcher.HttpCode"] = codes
			}
		} else {
			// tcp & udp 健康检查阈值与不健康阈值需相同
			params["UnhealthyThresholdCount"] = fmt.Sprintf("%d", group.HealthCheck.HealthCheckRise)
		}
	}

	result := struct {
		Groups []SElbBackendGroup `xml:"TargetGroups>member"`
	}{}
	err := self.elbRequest("CreateTargetGroup", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateTargetGroup")
	}

	for i := range result.Groups {
		result.Groups[i].region = self
		return &result.Groups[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after create")
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
	params := map[string]string{
		"ResourceArns.1": arn,
	}
	result := struct {
		TagDescriptions []struct {
			ResourceArn string `xml:"ResourceArn"`
			Tags        []struct {
				Key   string `xml:"Key"`
				Value string `xml:"Value"`
			} `xml:"Tags>member"`
		} `xml:"TagDescriptions>member"`
	}{}
	err := self.elbRequest("DescribeTags", params, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeTags")
	}
	ret := map[string]string{}
	for _, desc := range result.TagDescriptions {
		for _, tag := range desc.Tags {
			ret[tag.Key] = tag.Value
		}
	}
	return ret, nil
}
