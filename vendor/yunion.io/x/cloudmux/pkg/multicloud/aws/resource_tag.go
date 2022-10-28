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

	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SRegion) TagResources(arns []string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	client, err := self.getResourceGroupTagClient()
	if err != nil {
		return errors.Wrap(err, "self.getResourceGroupTagClient()")
	}
	params := resourcegroupstaggingapi.TagResourcesInput{}
	arnInput := []*string{}
	for i := range arns {
		arnInput = append(arnInput, &arns[i])
	}
	tagsInput := make(map[string]*string)
	tagValues := []string{}
	for k, v := range tags {
		tagValues = append(tagValues, v)
		tagsInput[k] = &tagValues[len(tagValues)-1]
	}
	params.SetResourceARNList(arnInput)
	params.SetTags(tagsInput)
	out, err := client.TagResources(&params)
	if err != nil {
		return errors.Wrapf(err, "client.TagResources(%s)", jsonutils.Marshal(params).String())
	}
	if out != nil && len(out.FailedResourcesMap) > 0 {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "client.TagResources(%s),error:%s", jsonutils.Marshal(params).String(), jsonutils.Marshal(out).String())
	}
	return nil
}

func (self *SRegion) UntagResources(arns []string, tagKeys []string) error {
	if len(tagKeys) == 0 {
		return nil
	}
	client, err := self.getResourceGroupTagClient()
	if err != nil {
		return errors.Wrap(err, "self.getResourceGroupTagClient()")
	}
	params := resourcegroupstaggingapi.UntagResourcesInput{}
	arnInput := []*string{}
	for i := range arns {
		arnInput = append(arnInput, &arns[i])
	}
	delTagKeysInput := []*string{}
	for i := range tagKeys {
		delTagKeysInput = append(delTagKeysInput, &tagKeys[i])
	}
	params.SetResourceARNList(arnInput)
	params.SetTagKeys(delTagKeysInput)
	out, err := client.UntagResources(&params)
	if err != nil {
		return errors.Wrapf(err, "client.UntagResources(%s)", jsonutils.Marshal(params).String())
	}
	if out != nil && len(out.FailedResourcesMap) > 0 {
		return errors.Wrapf(cloudprovider.ErrNotSupported, "client.UntagResources(%s),error:%s", jsonutils.Marshal(params).String(), jsonutils.Marshal(out).String())
	}
	return nil
}

func (self *SRegion) UpdateResourceTags(arn string, oldTags, tags map[string]string, replace bool) error {
	addTags := map[string]string{}
	for k, v := range tags {
		if strings.HasPrefix(k, "aws:") {
			return errors.Wrap(cloudprovider.ErrNotSupported, "The aws: prefix is reserved for AWS use")
		}
		if _, ok := oldTags[k]; !ok {
			addTags[k] = v
		} else {
			if oldTags[k] != v {
				addTags[k] = v
			}
		}
	}
	delTags := []string{}
	if replace {
		for k := range oldTags {
			if _, ok := tags[k]; !ok {
				if !strings.HasPrefix(k, "aws:") {
					delTags = append(delTags, k)
				}
			}
		}
	}
	err := self.UntagResources([]string{arn}, delTags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.UntagResources([]string{%s}, %s)", arn, jsonutils.Marshal(delTags).String())
	}
	err = self.TagResources([]string{arn}, addTags)
	if err != nil {
		return errors.Wrapf(err, "self.host.zone.region.TagResources([]string{%s}, %s)", arn, jsonutils.Marshal(addTags).String())
	}
	return nil
}
