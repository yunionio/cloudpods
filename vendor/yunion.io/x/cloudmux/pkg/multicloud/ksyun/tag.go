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

package ksyun

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

type STag struct {
	ResourceType string
	ResourceId   string
	Key          string
	Value        string
}

type TagSet []STag

func (vv TagSet) GetTags() map[string]string {
	ret := map[string]string{}
	for _, v := range vv {
		ret[v.Key] = v.Value
	}
	return ret
}

func (region *SRegion) ListTags(resType string, resId string) (*TagSet, error) {
	params := map[string]interface{}{
		"MaxResults":       "1000",
		"Filter.1.Name":    "resource-type",
		"Filter.1.Value.1": resType,
		"Filter.2.Name":    "resource-id",
		"Filter.2.Value.1": resId,
	}
	resp, err := region.tagRequest("DescribeTags", params)
	if err != nil {
		return nil, err
	}
	ret := &TagSet{}
	err = resp.Unmarshal(&ret, "TagSet")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) DeleteTags(resType, resId string, tags map[string]string) error {
	params := map[string]interface{}{
		"Resource.1.Type": resType,
		"Resource.1.Id":   resId,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tag.%d.Value", idx)] = v
		idx++
	}
	_, err := region.tagRequest("DeleteTags", params)
	if err != nil {
		return err
	}
	return nil
}

func (region *SRegion) CreateTags(resType, resId string, tags map[string]string) error {
	params := map[string]interface{}{
		"Resource.1.Type": resType,
		"Resource.1.Id":   resId,
	}
	idx := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tag.%d.Value", idx)] = v
		idx++
	}
	_, err := region.tagRequest("CreateTags", params)
	if err != nil {
		return err
	}
	return nil
}

func (region *SRegion) SetResourceTags(resType string, resId string, tags map[string]string, replace bool) error {
	_tags, err := region.ListTags(resType, resId)
	if err != nil {
		return errors.Wrapf(err, "ListTags")
	}
	if gotypes.IsNil(_tags) {
		_tags = &TagSet{}
	}
	keys, upperKeys := []string{}, []string{}
	for k := range tags {
		keys = append(keys, k)
		upperKeys = append(upperKeys, strings.ToUpper(k))
	}
	if replace {
		if len(tags) > 0 {
			removeKeys := map[string]string{}
			for _, k := range *_tags {
				if !utils.IsInStringArray(k.Key, keys) {
					removeKeys[k.Key] = k.Value
				}
			}
			if len(removeKeys) > 0 {
				err := region.DeleteTags(resType, resId, removeKeys)
				if err != nil {
					return errors.Wrapf(err, "DeleteTags")
				}
			}
		}
	} else {
		removeKeys := map[string]string{}
		for _, k := range *_tags {
			if !utils.IsInStringArray(k.Key, keys) && utils.IsInStringArray(strings.ToUpper(k.Key), upperKeys) {
				removeKeys[k.Key] = k.Value
			}
		}
		if len(removeKeys) > 0 {
			err := region.DeleteTags(resType, resId, removeKeys)
			if err != nil {
				return errors.Wrapf(err, "DeleteTags")
			}
		}
	}
	return region.CreateTags(resType, resId, tags)
}
