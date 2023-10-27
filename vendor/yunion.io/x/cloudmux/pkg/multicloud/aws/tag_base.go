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
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SAwsTag struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type SAwsRdsTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type SAwsLbTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type AwsTags struct {
	TagSet []SAwsTag `xml:"tagSet>item"`
	// rds
	TagList []SAwsRdsTag `xml:"TagList>Tag"`
	// elb
	Tags []SAwsLbTag `xml:"Tags>member"`
}

func (self AwsTags) GetName() string {
	for _, tag := range self.TagSet {
		if strings.ToLower(tag.Key) == "name" {
			return tag.Value
		}
	}
	for _, tag := range self.TagList {
		if strings.ToLower(tag.Key) == "name" {
			return tag.Value
		}
	}
	for _, tag := range self.Tags {
		if strings.ToLower(tag.Key) == "name" {
			return tag.Value
		}
	}
	return ""
}

func (self AwsTags) GetDescription() string {
	for _, tag := range self.TagSet {
		if strings.ToLower(tag.Key) == "description" {
			return tag.Value
		}
	}
	for _, tag := range self.TagList {
		if strings.ToLower(tag.Key) == "description" {
			return tag.Value
		}
	}
	for _, tag := range self.Tags {
		if strings.ToLower(tag.Key) == "description" {
			return tag.Value
		}
	}
	return ""
}

func (self *AwsTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.TagSet {
		if tag.Key == "Name" || tag.Key == "Description" {
			continue
		}
		if strings.HasPrefix(tag.Key, "aws:") {
			continue
		}
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.TagList {
		if strings.ToLower(tag.Key) == "name" || strings.ToLower(tag.Key) == "description" {
			continue
		}
		if strings.HasPrefix(tag.Key, "aws:") {
			continue
		}
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.Tags {
		if strings.ToLower(tag.Key) == "name" || strings.ToLower(tag.Key) == "description" {
			continue
		}
		if strings.HasPrefix(tag.Key, "aws:") {
			continue
		}
		ret[tag.Key] = tag.Value
	}
	return ret, nil
}

func (self *AwsTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.TagSet {
		if strings.HasPrefix(tag.Key, "aws:") {
			ret[tag.Key] = tag.Value
		}
	}
	for _, tag := range self.TagList {
		if strings.HasPrefix(tag.Key, "aws:") {
			ret[tag.Key] = tag.Value
		}
	}
	for _, tag := range self.Tags {
		if strings.HasPrefix(tag.Key, "aws:") {
			ret[tag.Key] = tag.Value
		}
	}
	return ret
}

func (self *AwsTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
