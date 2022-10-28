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

package apsara

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type STagResource struct {
	ResourceType string `json:"ResourceType"`
	TagValue     string `json:"TagValue"`
	ResourceID   string `json:"ResourceId"`
	TagKey       string `json:"TagKey"`
}

func (self *SRegion) rawListTagResources(serviceType string, resourceType string, resIds []string, token string) ([]STagResource, string, error) {
	if len(resIds) > 50 {
		return nil, "", errors.Wrap(cloudprovider.ErrNotSupported, "resource count exceed 50 for one request")
	}
	params := make(map[string]string)
	params["ResourceType"] = resourceType
	for i := range resIds {
		params[fmt.Sprintf("ResourceId.%d", i+1)] = resIds[i]
	}
	if len(token) != 0 {
		params["NextToken"] = token
	}
	ret, err := self.tagRequest(serviceType, "ListTagResources", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, `self.tagRequest(%s,"ListTagResources", %s)`, serviceType, jsonutils.Marshal(params).String())
	}
	tagResources := []STagResource{}
	err = ret.Unmarshal(&tagResources, "TagResources", "TagResource")
	if err != nil {
		return nil, "", errors.Wrapf(err, "(%s).Unmarshal(&tagResources)", ret.String())
	}
	nextToken, _ := ret.GetString("NextToken")
	return tagResources, nextToken, nil
}

func splitStringSlice(resIds []string, stride int) [][]string {
	result := [][]string{}
	i := 0
	for i < len(resIds)/stride {
		result = append(result, resIds[i*stride:i*stride+stride])
		i++
	}
	remainder := len(resIds) % stride
	if remainder != 0 {
		result = append(result, resIds[i*stride:i*stride+remainder])
	}
	return result
}

func splitTags(tags map[string]string, stride int) []map[string]string {
	tagsGroups := []map[string]string{}
	tagsGroup := map[string]string{}
	for k, v := range tags {
		tagsGroup[k] = v
		if len(tagsGroup) == stride {
			tagsGroups = append(tagsGroups, tagsGroup)
			tagsGroup = map[string]string{}
		}
	}
	if len(tagsGroup) > 0 {
		tagsGroups = append(tagsGroups, tagsGroup)
	}
	return tagsGroups
}

func (self *SRegion) ListResourceTags(serviceType string, resourceType string, resIds []string) (map[string]*map[string]string, error) {
	tags := make(map[string]*map[string]string)
	tagReources := []STagResource{}
	nextToken := ""
	resIdsGroups := splitStringSlice(resIds, 50)
	for i := range resIdsGroups {
		for {
			_tagResource, nextToken, err := self.rawListTagResources(serviceType, resourceType, resIdsGroups[i], nextToken)
			if err != nil {
				return nil, errors.Wrapf(err, "self.rawListTagResources(%s,%s,%s)", resourceType, resIds, nextToken)
			}
			tagReources = append(tagReources, _tagResource...)
			if len(_tagResource) == 0 || len(nextToken) == 0 {
				break
			}

		}
	}
	for _, r := range tagReources {
		if tagMapPtr, ok := tags[r.ResourceID]; !ok {
			tagMap := map[string]string{
				r.TagKey: r.TagValue,
			}
			tags[r.ResourceID] = &tagMap
		} else {
			tagMap := *tagMapPtr
			tagMap[r.TagKey] = r.TagValue
		}
	}
	return tags, nil
}

func (self *SRegion) rawTagResources(serviceType string, resourceType string, resIds []string, tags map[string]string) error {
	if len(resIds) > 50 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "resource count exceed 50 for one request")
	}
	if len(tags) > 20 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "tags count exceed 20 for one request")
	}
	params := make(map[string]string)
	params["ResourceType"] = resourceType
	for i := range resIds {
		params[fmt.Sprintf("ResourceId.%d", i+1)] = resIds[i]
	}
	i := 0
	for k, v := range tags {
		params[fmt.Sprintf("Tag.%d.Key", i+1)] = k
		params[fmt.Sprintf("Tag.%d.Value", i+1)] = v
		i++
	}
	_, err := self.tagRequest(serviceType, "TagResources", params)
	if err != nil {
		return errors.Wrapf(err, `self.tagRequest(%s,"TagResources", %s)`, serviceType, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) TagResources(serviceType string, resourceType string, resIds []string, tags map[string]string) error {
	if len(resIds) == 0 || len(tags) == 0 {
		return nil
	}
	resIdsGroups := splitStringSlice(resIds, 50)
	tagsGroups := splitTags(tags, 20)
	for i := range resIdsGroups {
		for j := range tagsGroups {
			err := self.rawTagResources(serviceType, resourceType, resIdsGroups[i], tagsGroups[j])
			if err != nil {
				return errors.Wrapf(err, "self.rawTagResources(resourceType, resIdsGroups[i], tagsGroups[i])")
			}
		}
	}
	return nil
}

func (self *SRegion) rawUntagResources(serviceType string, resourceType string, resIds []string, tags []string) error {
	if len(resIds) > 50 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "resource count exceed 50 for one request")
	}
	if len(tags) > 20 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "tags count exceed 20 for one request")
	}
	params := make(map[string]string)
	params["ResourceType"] = resourceType
	for i := range resIds {
		params[fmt.Sprintf("ResourceId.%d", i+1)] = resIds[i]
	}
	for i := range tags {
		params[fmt.Sprintf("TagKey.%d", i+1)] = tags[i]
	}
	_, err := self.tagRequest(serviceType, "UntagResources", params)
	if err != nil {
		return errors.Wrapf(err, `self.tagRequest(%s,"UntagResources", %s)`, serviceType, jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) UntagResources(serviceType string, resourceType string, resIds []string, tags []string) error {
	if len(resIds) == 0 || len(tags) == 0 {
		return nil
	}
	resIdsGroups := splitStringSlice(resIds, 50)
	tagsGroups := splitStringSlice(tags, 20)
	for i := range resIdsGroups {
		for j := range tagsGroups {
			err := self.rawUntagResources(serviceType, resourceType, resIdsGroups[i], tagsGroups[j])
			if err != nil {
				return errors.Wrapf(err, "self.rawTagResources(resourceType, resIdsGroups[i], tagsGroups[i])")
			}
		}
	}
	return nil
}

func (self *SRegion) SetResourceTags(serviceType string, resourceType string, resIds []string, tags map[string]string, replace bool) error {
	oldTags, err := self.ListResourceTags(serviceType, resourceType, resIds)
	if err != nil {
		return errors.Wrapf(err, "self.ListResourceTags(%s,%s)", resourceType, resIds)
	}
	for i := range resIds {
		_, ok := oldTags[resIds[i]]
		if !ok {
			err := self.TagResources(serviceType, resourceType, []string{resIds[i]}, tags)
			if err != nil {
				return errors.Wrap(err, "self.TagResources(resourceType, []string{resIds[i]}, tags)")
			}
		} else {
			oldResourceTags := *oldTags[resIds[i]]
			addTags := map[string]string{}
			for k, v := range tags {
				if _, ok := oldResourceTags[k]; !ok {
					addTags[k] = v
				} else {
					if oldResourceTags[k] != v {
						addTags[k] = v
					}
				}
			}
			delTags := []string{}
			if replace {
				for k := range oldResourceTags {
					if _, ok := tags[k]; !ok {
						delTags = append(delTags, k)
					}
				}
			}
			err := self.UntagResources(serviceType, resourceType, []string{resIds[i]}, delTags)
			if err != nil {
				return errors.Wrap(err, "self.UntagResources(resourceType, []string{resIds[i]}, delTags)")
			}
			err = self.TagResources(serviceType, resourceType, []string{resIds[i]}, addTags)
			if err != nil {
				return errors.Wrap(err, "self.TagResources(resourceType, []string{resIds[i]}, addTags)")
			}
		}
	}
	return nil
}
