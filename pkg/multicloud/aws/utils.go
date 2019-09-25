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
	"fmt"
	"net"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"
)

type portRange struct {
	Start int64
	End   int64
}

type TagSpec struct {
	ResourceType string // "customer-gateway"|"dedicated-host"|"dhcp-options"|"image"|"instance"|"internet-gateway"|"network-acl"|"network-interface"|"reserved-instances"|"route-table"|"snapshot"|"spot-instances-request"|"subnet"|"security-group"|"volume"|"vpc"|"vpn-connection"|"vpn-gateway"
	Tags         map[string]string
}

func (self *TagSpec) LoadingEc2Tags(tags []*ec2.Tag) {
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			self.SetTag(*tag.Key, *tag.Value)
		}
	}
}

func (self *TagSpec) GetTagSpecifications() (*ec2.TagSpecification, error) {
	if self.ResourceType == "" {
		return nil, fmt.Errorf("ResourceType should not be empty")
	}

	spec := &ec2.TagSpecification{ResourceType: &self.ResourceType}
	tags := []*ec2.Tag{}
	for k, v := range self.Tags {
		if len(v) > 255 {
			return nil, fmt.Errorf("%s value length should less than 255", k)
		}

		tag := &ec2.Tag{}
		tag.SetKey(k)
		tag.SetValue(v)
		tags = append(tags, tag)
	}

	spec.SetTags(tags)
	return spec, nil
}

func (self *TagSpec) SetTag(k, v string) {
	if self.Tags == nil {
		self.Tags = make(map[string]string)
	}
	self.Tags[k] = v
}

func (self *TagSpec) SetNameTag(v string) {
	self.SetTag("Name", v)
}

func (self *TagSpec) SetDescTag(v string) {
	self.SetTag("Description", v)
}

func (self *TagSpec) GetTag(k string) (string, error) {
	v, ok := self.Tags[k]
	if !ok {
		return "", fmt.Errorf("%s not found", k)
	}

	return v, nil
}

// 找不到的情况下返回传入的默认值
func (self *TagSpec) GetTagWithDefault(k, Default string) string {
	v, ok := self.Tags[k]
	if !ok {
		return Default
	}

	return v
}

func (self *TagSpec) GetNameTag() string {
	return self.GetTagWithDefault("Name", "")
}

func (self *TagSpec) GetDescTag() string {
	return self.GetTagWithDefault("Description", "")
}

func AppendFilter(filters []*ec2.Filter, name string, values []string) []*ec2.Filter {
	f := &ec2.Filter{}
	v := make([]*string, len(values))
	for _, value := range values {
		v = append(v, &value)
	}

	f.SetName(name)
	f.SetValues(v)
	return append(filters, f)
}

func AppendSingleValueFilter(filters []*ec2.Filter, name string, value string) []*ec2.Filter {
	f := &ec2.Filter{}
	f.SetName(name)
	f.SetValues([]*string{&value})
	return append(filters, f)
}

func ConvertedList(list []string) []*string {
	result := make([]*string, 0)
	for i := range list {
		if len(list[i]) > 0 {
			result = append(result, &list[i])
		}
	}

	return result
}

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func ConvertedPointList(list []*string) []string {
	result := make([]string, len(list))
	for _, item := range list {
		if item != nil {
			result = append(result, *item)
		}
	}

	return result
}

func StrVal(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

func IntVal(s *int64) int64 {
	if s != nil {
		return *s
	}

	return 0
}

// SecurityRuleSet to  allow list
// 将安全组规则全部转换为等价的allow规则
func SecurityRuleSetToAllowSet(srs secrules.SecurityRuleSet) secrules.SecurityRuleSet {
	inRuleSet := secrules.SecurityRuleSet{}
	outRuleSet := secrules.SecurityRuleSet{}

	for _, rule := range srs {
		if rule.Direction == secrules.SecurityRuleIngress {
			inRuleSet = append(inRuleSet, rule)
		}

		if rule.Direction == secrules.SecurityRuleEgress {
			outRuleSet = append(outRuleSet, rule)
		}
	}

	sort.Sort(inRuleSet)
	sort.Sort(outRuleSet)

	inRuleSet = inRuleSet.AllowList()
	outRuleSet = outRuleSet.AllowList()

	ret := secrules.SecurityRuleSet{}
	ret = append(ret, inRuleSet...)
	ret = append(ret, outRuleSet...)
	return ret
}

func isAwsPermissionAllPorts(p ec2.IpPermission) bool {
	if p.FromPort == nil || p.ToPort == nil {
		return false
	}

	//  全部端口范围： TCP/UDP （0，65535）    其他：（-1，-1）
	if (*p.IpProtocol == "tcp" || *p.IpProtocol == "udp") && *p.FromPort == 0 && *p.ToPort == 65535 {
		return true
	} else if *p.FromPort == -1 && *p.ToPort == -1 {
		return true
	} else {
		return false
	}
}

func awsProtocolToYunion(p ec2.IpPermission) string {
	if p.IpProtocol != nil && *p.IpProtocol == "-1" {
		return secrules.PROTO_ANY
	} else {
		return *p.IpProtocol
	}
}

func yunionProtocolToAws(r secrules.SecurityRule) string {
	if r.Protocol == secrules.PROTO_ANY {
		return "-1"
	} else {
		return r.Protocol
	}
}

func isYunionRuleAllPorts(r secrules.SecurityRule) bool {
	//  全部端口范围： TCP/UDP （0，65535）    其他：（-1，-1）
	if (r.Protocol == "tcp" || r.Protocol == "udp") && r.PortStart == 0 && r.PortEnd == 65535 {
		return true
	} else if r.PortStart == -1 && r.PortEnd == -1 {
		return true
	} else {
		return false
	}
}

func yunionPortRangeToAws(r secrules.SecurityRule) []portRange {
	// port 0 / -1 都代表所有端口
	portranges := []portRange{}
	if len(r.Ports) == 0 {
		var start, end = 0, 0
		if r.PortStart <= 0 {
			if r.Protocol == "tcp" || r.Protocol == "udp" {
				start = 0
			} else {
				start = -1
			}
		} else {
			start = r.PortStart
		}

		if r.PortEnd <= 0 {
			if r.Protocol == "tcp" || r.Protocol == "udp" {
				end = 65535
			} else {
				end = -1
			}
		} else {
			end = r.PortEnd
		}

		portranges = append(portranges, portRange{int64(start), int64(end)})
	}

	for i := range r.Ports {
		port := r.Ports[i]
		if port <= 0 && (r.Protocol == "tcp" || r.Protocol == "udp") {
			portranges = append(portranges, portRange{0, 65535})
		} else if port <= 0 {
			portranges = append(portranges, portRange{-1, -1})
		} else {
			portranges = append(portranges, portRange{int64(port), int64(port)})
		}
	}

	return portranges
}

// Security Rule Transform
func AwsIpPermissionToYunion(direction secrules.TSecurityRuleDirection, p ec2.IpPermission) ([]secrules.SecurityRule, error) {

	if len(p.UserIdGroupPairs) > 0 {
		return nil, fmt.Errorf("AwsIpPermissionToYunion not supported aws rule: UserIdGroupPairs specified")
	}

	if len(p.PrefixListIds) > 0 {
		return nil, fmt.Errorf("AwsIpPermissionToYunion not supported aws rule: PrefixListIds specified")
	}

	if len(p.Ipv6Ranges) > 0 {
		log.Debugf("AwsIpPermissionToYunion ignored IPV6 rule: %s", p.Ipv6Ranges)
	}

	rules := []secrules.SecurityRule{}
	isAllPorts := isAwsPermissionAllPorts(p)
	protocol := awsProtocolToYunion(p)
	for _, ip := range p.IpRanges {
		_, ipNet, err := net.ParseCIDR(*ip.CidrIp)
		if err != nil {
			log.Errorf("ParseCIDR failed, ignored IPV4 rule: %s", *ip.CidrIp)
			continue
		}

		var rule secrules.SecurityRule
		if isAllPorts {
			rule = secrules.SecurityRule{
				Action:      secrules.SecurityRuleAllow,
				IPNet:       ipNet,
				Protocol:    protocol,
				Direction:   direction,
				Priority:    1,
				Description: StrVal(ip.Description),
			}
		} else {
			rule = secrules.SecurityRule{
				Action:      secrules.SecurityRuleAllow,
				IPNet:       ipNet,
				Protocol:    protocol,
				Direction:   direction,
				Priority:    1,
				Description: StrVal(ip.Description),
			}

			if p.FromPort != nil {
				rule.PortStart = int(*p.FromPort)
			}

			if p.ToPort != nil {
				rule.PortEnd = int(*p.ToPort)
			}
		}

		rules = append(rules, rule)

	}

	return rules, nil
}

// YunionSecRuleToAws 不能保证无损转换
// 规则描述如果包含中文等字符，将被丢弃掉
func YunionSecRuleToAws(rule secrules.SecurityRule) ([]*ec2.IpPermission, error) {
	if rule.Action == secrules.SecurityRuleDeny {
		return nil, fmt.Errorf("YunionSecRuleToAws ignored  aws not supported deny rule")
	}

	iprange := rule.IPNet.String()
	if iprange == "<nil>" {
		return nil, fmt.Errorf("YunionSecRuleToAws ignored  ipnet should not be empty")
	}

	description := ""
	if match, err := regexp.MatchString("^[\\sa-zA-Z0-9. _:/()#,@\\]\\[+=&;{}!$*-]+$", rule.Description); err == nil && match {
		description = rule.Description
	}
	ipranges := []*ec2.IpRange{}
	ipranges = append(ipranges, &ec2.IpRange{CidrIp: &iprange, Description: &description})

	portranges := yunionPortRangeToAws(rule)
	protocol := yunionProtocolToAws(rule)
	permissions := []*ec2.IpPermission{}
	for i := range portranges {
		port := portranges[i]
		permission := ec2.IpPermission{
			FromPort:   &port.Start,
			IpProtocol: &protocol,
			IpRanges:   ipranges,
			ToPort:     &port.End,
		}

		permissions = append(permissions, &permission)
	}

	return permissions, nil
}

// fill a pointer struct with zero value.
func FillZero(i interface{}) error {
	V := reflect.Indirect(reflect.ValueOf(i))

	if !V.CanSet() {
		return fmt.Errorf("input is not addressable: %#v", i)
	}

	if V.Kind() != reflect.Struct {
		return fmt.Errorf("only accept struct type")
	}

	for i := 0; i < V.NumField(); i++ {
		field := V.Field(i)

		if field.Kind() == reflect.Ptr && field.IsNil() {
			if field.CanSet() {
				field.Set(reflect.New(field.Type().Elem()))
			}
		}

		vField := reflect.Indirect(field)
		switch vField.Kind() {
		case reflect.Map:
			vField.Set(reflect.MakeMap(vField.Type()))
		case reflect.Struct:
			if field.CanInterface() {
				err := FillZero(field.Interface())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NextDeviceName(curDeviceNames []string) (string, error) {
	currents := []string{}
	for _, item := range curDeviceNames {
		currents = append(currents, strings.ToLower(item))
	}

	for i := 0; i < 25; i++ {
		device := fmt.Sprintf("/dev/sd%s", string(98+i))
		found := false
		for _, item := range currents {
			if strings.HasPrefix(item, device) {
				found = true
			}
		}

		if !found {
			return device, nil
		}
	}

	for i := 0; i < 25; i++ {
		device := fmt.Sprintf("/dev/vxd%s", string(98+i))
		found := false
		for _, item := range currents {
			if !strings.HasPrefix(item, device) {
				return device, nil
			}
		}

		if !found {
			return device, nil
		}
	}

	return "", fmt.Errorf("disk devicename out of index, current deivces: %s", currents)
}

// fetch tags
func FetchTags(client *ec2.EC2, resourceId string) (*jsonutils.JSONDict, error) {
	result := jsonutils.NewDict()
	params := &ec2.DescribeTagsInput{}
	filters := []*ec2.Filter{}
	if len(resourceId) == 0 {
		return result, fmt.Errorf("resource id should not be empty")
	}
	// todo: add resource type filter
	filters = AppendSingleValueFilter(filters, "resource-id", resourceId)
	params.SetFilters(filters)

	ret, err := client.DescribeTags(params)
	if err != nil {
		return result, err
	}

	for _, tag := range ret.Tags {
		if tag.Key != nil && tag.Value != nil {
			result.Set(*tag.Key, jsonutils.NewString(*tag.Value))
		}
	}

	return result, nil
}

// error
func parseNotFoundError(err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), ".NotFound") {
		return ErrorNotFound()
	} else {
		return err
	}
}

func ErrorNotFound() error {
	return cloudprovider.ErrNotFound
}
