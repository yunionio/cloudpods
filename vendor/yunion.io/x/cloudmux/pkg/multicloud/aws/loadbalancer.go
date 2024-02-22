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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

/*
https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/Welcome.html
*/

type SElbs struct {
	LoadBalancers []SElb `xml:"LoadBalancers>member"`
	NextMarker    string `xml:"NextMarker"`
}

type SElb struct {
	multicloud.SLoadbalancerBase
	region *SRegion

	AwsTags
	Type                  string             `xml:"Type"`
	Scheme                string             `xml:"Scheme"`
	IPAddressType         string             `xml:"IpAddressType"`
	VpcId                 string             `xml:"VpcId"`
	AvailabilityZones     []AvailabilityZone `xml:"AvailabilityZones>member"`
	CreatedTime           string             `xml:"CreatedTime"`
	CanonicalHostedZoneID string             `xml:"CanonicalHostedZoneId"`
	DNSName               string             `xml:"DNSName"`
	SecurityGroups        []string           `xml:"SecurityGroups>member"`
	LoadBalancerName      string             `xml:"LoadBalancerName"`
	State                 State              `xml:"State"`
	LoadBalancerArn       string             `xml:"LoadBalancerArn"`
}

type AvailabilityZone struct {
	LoadBalancerAddresses []LoadBalancerAddress `xml:"LoadBalancerAddresses>member"`
	ZoneName              string                `xml:"ZoneName"`
	SubnetId              string                `xml:"SubnetId"`
}

type LoadBalancerAddress struct {
	IPAddress    string `xml:"IpAddress"`
	AllocationID string `xml:"AllocationId"`
}

type State struct {
	Code string `xml:"Code"`
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
	lb, err := self.region.GetLoadBalancer(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, lb)
}

func (self *SElb) GetSysTags() map[string]string {
	data := map[string]string{}
	data["loadbalance_type"] = self.Type
	attrs, err := self.region.GetElbAttributes(self.GetId())
	if err != nil {
		return data
	}

	for k, v := range attrs {
		data[k] = v
	}
	return data
}

func (self *SElb) GetTags() (map[string]string, error) {
	tagBase, err := self.region.DescribeElbTags(self.LoadBalancerArn)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeElbTags")
	}
	return tagBase.GetTags()
}

func (self *SElb) GetAddress() string {
	return self.DNSName
}

func (lb *SElb) GetSecurityGroupIds() ([]string, error) {
	return lb.SecurityGroups, nil
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
	zoneNames := []string{}
	for i := range self.AvailabilityZones {
		zoneNames = append(zoneNames, self.AvailabilityZones[i].ZoneName)
	}
	sort.Strings(zoneNames)
	for i := range zoneNames {
		return zoneNames[i]
	}
	return ""
}

func (self *SElb) GetZone1Id() string {
	zoneNames := []string{}
	for i := range self.AvailabilityZones {
		zoneNames = append(zoneNames, self.AvailabilityZones[i].ZoneName)
	}
	sort.Strings(zoneNames)
	for i := range zoneNames {
		if i != 0 {
			return zoneNames[i]
		}
	}
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
	ret := []cloudprovider.ICloudLoadbalancerListener{}
	marker := ""
	for {
		part, marker, err := self.region.GetElbListeners(self.LoadBalancerArn, "", marker)
		if err != nil {
			return nil, err
		}
		for i := range part {
			part[i].lb = self
			ret = append(ret, &part[i])
		}
		if len(marker) == 0 || len(part) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SElb) GetILoadBalancerBackendGroups() ([]cloudprovider.ICloudLoadbalancerBackendGroup, error) {
	groups, err := self.region.GetElbBackendgroups(self.LoadBalancerArn, "")
	if err != nil {
		return nil, errors.Wrapf(err, "GetElbBackendgroups")
	}
	ret := []cloudprovider.ICloudLoadbalancerBackendGroup{}
	for i := range groups {
		groups[i].lb = self
		ret = append(ret, &groups[i])
	}
	return ret, nil
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
	lbbg, err := self.region.GetElbBackendgroup(groupId)
	if err != nil {
		return nil, err
	}
	lbbg.lb = self
	return lbbg, nil
}

func (self *SElb) CreateILoadBalancerListener(ctx context.Context, listener *cloudprovider.SLoadbalancerListenerCreateOptions) (cloudprovider.ICloudLoadbalancerListener, error) {
	ret, err := self.region.CreateElbListener(self.LoadBalancerArn, listener)
	if err != nil {
		return nil, errors.Wrap(err, "CreateElbListener")
	}

	ret.lb = self
	return ret, nil
}

func (self *SElb) GetILoadBalancerListenerById(listenerId string) (cloudprovider.ICloudLoadbalancerListener, error) {
	lis, err := self.region.GetElbListener(listenerId)
	if err != nil {
		return nil, err
	}
	lis.lb = self
	return lis, nil
}

func (self *SElb) GetIEIP() (cloudprovider.ICloudEIP, error) {
	return nil, nil
}

func (self *SRegion) DeleteElb(id string) error {
	params := map[string]string{"LoadBalancerArn": id}
	return self.elbRequest("DeleteLoadBalancer", params, nil)
}

func (self *SRegion) GetElbBackendgroups(elbId, id string) ([]SElbBackendGroup, error) {
	params := map[string]string{}
	if len(elbId) > 0 {
		params["LoadBalancerArn"] = elbId
	}
	if len(id) > 0 {
		params["TargetGroupArns.member.1"] = id
	}
	ret := []SElbBackendGroup{}
	for {
		part := struct {
			NextMarker   string             `xml:"NextMarker"`
			TargetGroups []SElbBackendGroup `xml:"TargetGroups>member"`
		}{}
		err := self.elbRequest("DescribeTargetGroups", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeTargetGroups")
		}
		ret = append(ret, part.TargetGroups...)
		if len(part.NextMarker) == 0 || len(part.TargetGroups) == 0 {
			break
		}
		params["Marker"] = part.NextMarker
	}

	return ret, nil
}

func (self *SRegion) GetElbBackendgroup(id string) (*SElbBackendGroup, error) {
	groups, err := self.GetElbBackendgroups("", id)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].TargetGroupArn == id {
			return &groups[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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
func (self *SRegion) CreateElbBackendgroup(opts *cloudprovider.SLoadbalancerBackendGroup) (*SElbBackendGroup, error) {
	params := map[string]string{
		"Protocol":   strings.ToUpper(opts.Protocol),
		"Name":       opts.Name,
		"Port":       fmt.Sprintf("%d", opts.ListenPort),
		"TargetType": "instance",
		"VpcId":      opts.VpcId,
	}
	ret := struct {
		TargetGroups []SElbBackendGroup `xml:"TargetGroups>member"`
	}{}
	err := self.elbRequest("CreateTargetGroup", params, &ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.TargetGroups {
		return &ret.TargetGroups[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SElb) SetTags(tags map[string]string, replace bool) error {
	return self.region.setElbTags(self.LoadBalancerArn, tags, replace)
}

func (self *SRegion) setElbTags(arn string, tags map[string]string, replace bool) error {
	tagBase, err := self.DescribeElbTags(arn)
	if err != nil {
		return errors.Wrapf(err, "DescribeElbTags")
	}
	oldTags, err := tagBase.GetTags()
	if err != nil {
		return errors.Wrap(err, "get tags")
	}
	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}
	if len(removed) > 0 {
		err = self.RemoveElbTags(arn, removed)
		if err != nil {
			return errors.Wrapf(err, "RemoveElbTags %s", removed)
		}
	}
	if len(added) > 0 {
		return self.AddElbTags(arn, added)
	}
	return nil
}

func (self *SRegion) AddElbTags(arn string, tags map[string]string) error {
	params := map[string]string{
		"ResourceArns.member.1": arn,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tags.member.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.member.%d.Value", idx)] = v
		idx++
	}
	ret := struct {
	}{}
	return self.elbRequest("AddTags", params, &ret)
}

func (self *SRegion) RemoveElbTags(arn string, tags map[string]string) error {
	params := map[string]string{
		"ResourceArns.member.1": arn,
	}
	idx := 1
	for k := range tags {
		params[fmt.Sprintf("TagKeys.member.%d", idx)] = k
		idx++
	}
	ret := struct {
	}{}
	return self.elbRequest("RemoveTags", params, &ret)
}

func (self *SRegion) DescribeElbTags(arn string) (*AwsTags, error) {
	ret := struct {
		TagDescriptions []struct {
			ResourceArn string `xml:"ResourceArn"`
			AwsTags
		} `xml:"TagDescriptions>member"`
	}{}
	err := self.elbRequest("DescribeTags", map[string]string{"ResourceArns.member.1": arn}, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeTags")
	}
	for _, res := range ret.TagDescriptions {
		if res.ResourceArn == arn {
			return &res.AwsTags, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetLoadbalancers(id, marker string) ([]SElb, string, error) {
	ret := &SElbs{}
	params := map[string]string{}
	if len(id) > 0 {
		params["LoadBalancerArns.member.1"] = id
	}
	if len(marker) > 0 {
		params["Marker"] = marker
	}
	err := self.elbRequest("DescribeLoadBalancers", params, ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "DescribeLoadBalancers")
	}
	return ret.LoadBalancers, ret.NextMarker, nil
}

func (self *SRegion) CreateLoadbalancer(opts *cloudprovider.SLoadbalancerCreateOptions) (*SElb, error) {
	ret := &SElbs{}
	params := map[string]string{
		"Name":          opts.Name,
		"Type":          opts.LoadbalancerSpec,
		"Scheme":        "internal",
		"IpAddressType": "ipv4",
	}
	if opts.AddressType == api.LB_ADDR_TYPE_INTERNET {
		params["Scheme"] = "internet-facing"
	}

	if opts.LoadbalancerSpec == api.LB_AWS_SPEC_APPLICATION && len(opts.NetworkIds) == 1 {
		nets, err := self.GetNetwroks(nil, opts.ZoneId, opts.VpcId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetNetworks(%s)", opts.VpcId)
		}
		for i := range nets {
			if !utils.IsInStringArray(nets[i].SubnetId, opts.NetworkIds) {
				opts.NetworkIds = append(opts.NetworkIds, nets[i].SubnetId)
				break
			}
		}
	}

	for i, net := range opts.NetworkIds {
		params[fmt.Sprintf("Subnets.member.%d", i+1)] = net
	}

	idx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.member.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.member.%d.Value", idx)] = v
		idx++
	}
	err := self.elbRequest("CreateLoadBalancer", params, ret)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateLoadBalancer")
	}
	for i := range ret.LoadBalancers {
		ret.LoadBalancers[i].region = self
		return &ret.LoadBalancers[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
}

func (self *SElb) GetDescription() string {
	tags, _ := self.region.DescribeElbTags(self.LoadBalancerArn)
	if tags == nil {
		return ""
	}
	return tags.GetDescription()
}
