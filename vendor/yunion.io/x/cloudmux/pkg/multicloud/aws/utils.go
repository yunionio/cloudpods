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
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
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

func (self *TagSpec) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for k, v := range self.Tags {
		if k == "Name" || k == "Description" {
			continue
		}
		if strings.HasPrefix(k, "aws:") {
			continue
		}
		ret[k] = v
	}
	return ret, nil
}

func (self *TagSpec) GetSysTags() map[string]string {
	ret := map[string]string{}
	for k, v := range self.Tags {
		if !strings.HasPrefix(k, "aws:") {
			continue
		}
		ret[k] = v
	}
	return ret
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
		device := fmt.Sprintf("/dev/sd%c", byte(98+i))
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
		device := fmt.Sprintf("/dev/vxd%c", byte(98+i))
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
		return errors.Wrap(cloudprovider.ErrNotFound, "parseNotFoundError")
	} else {
		return err
	}
}
