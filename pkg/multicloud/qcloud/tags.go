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
	"strconv"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

const (
	QCLOUD_API_VERSION_TAGS = "2018-08-13"
)

func tagRequest(client *common.Client, apiName string, params map[string]string, updateFunc func(string, string), debug bool) (jsonutils.JSONObject, error) {
	domain := "tag.tencentcloudapi.com"
	return _jsonRequest(client, domain, QCLOUD_API_VERSION_TAGS, apiName, params, updateFunc, debug, true)
}

func (client *SQcloudClient) tagRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := client.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return tagRequest(cli, apiName, params, client.cpcfg.UpdatePermission, client.debug)
}

type SFetchTagRow struct {
	ServiceType string `json:"ServiceType"`
	TagKey      string `json:"TagKey"`
	TagKeyMd5   string `json:"TagKeyMd5"`
	TagValue    string `json:"TagValue"`
	TagValueMd5 string `json:"TagValueMd5"`
	ResourceId  string `json:"ResourceId"`
}

type SListInfo struct {
	TotalCount int `json:"TotalCount"`
	Offset     int `json:"Offset"`
	Limit      int `json:"Limit"`
}

type SFetchTagResponse struct {
	SListInfo
	Tags []SFetchTagRow `json:"Tags"`
}

func (region *SRegion) rawFetchTags(serviceType, resourceType string, resIds []string, limit, offset int) (*SFetchTagResponse, error) {
	params := make(map[string]string)
	params["ServiceType"] = serviceType
	params["ResourcePrefix"] = resourceType
	for i, resId := range resIds {
		params[fmt.Sprintf("ResourceIds.%d", i)] = resId
	}
	params["ResourceRegion"] = region.Region
	if limit > 0 {
		params["Limit"] = strconv.FormatInt(int64(limit), 10)
	}
	if offset > 0 {
		params["Offset"] = strconv.FormatInt(int64(offset), 10)
	}
	apiName := "DescribeResourceTagsByResourceIdsSeq"
	respJson, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return nil, errors.Wrap(err, apiName)
	}
	resp := SFetchTagResponse{}
	err = respJson.Unmarshal(&resp)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal Response")
	}
	return &resp, nil
}

func tagsLen(tags map[string]*map[string]string) int {
	ret := 0
	for _, kv := range tags {
		ret += len(*kv)
	}
	return ret
}

func (region *SRegion) FetchResourceTags(serviceType, resourceType string, resIds []string) (map[string]*map[string]string, error) {
	tags := make(map[string]*map[string]string)
	total := -1
	for total < 0 || tagsLen(tags) < total {
		resp, err := region.rawFetchTags(serviceType, resourceType, resIds, 100, tagsLen(tags))
		if err != nil {
			return nil, errors.Wrap(err, "rawFetchTags")
		}
		if total < 0 {
			total = resp.TotalCount
		}
		for _, r := range resp.Tags {
			if tagMapPtr, ok := tags[r.ResourceId]; !ok {
				tagMap := map[string]string{
					r.TagKey: r.TagValue,
				}
				tags[r.ResourceId] = &tagMap
			} else {
				tagMap := *tagMapPtr
				tagMap[r.TagKey] = r.TagValue
			}
		}
	}
	return tags, nil
}

func (region *SRegion) attachTag(serviceType, resourceType string, resIds []string, key, value string) error {
	params := make(map[string]string)
	params["ServiceType"] = serviceType
	params["ResourcePrefix"] = resourceType
	params["ResourceRegion"] = region.Region
	params["TagKey"] = key
	params["TagValue"] = value
	for i, resId := range resIds {
		params[fmt.Sprintf("ResourceIds.%d", i)] = resId
	}
	apiName := "AttachResourcesTag"
	_, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return errors.Wrap(err, apiName)
	}
	return nil
}

func (region *SRegion) detachTag(serviceType, resourceType string, resIds []string, key string) error {
	params := make(map[string]string)
	params["ServiceType"] = serviceType
	params["ResourcePrefix"] = resourceType
	params["ResourceRegion"] = region.Region
	params["TagKey"] = key
	for i, resId := range resIds {
		params[fmt.Sprintf("ResourceIds.%d", i)] = resId
	}
	apiName := "DetachResourcesTag"
	_, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return errors.Wrap(err, apiName)
	}
	return nil
}

func (region *SRegion) modifyTag(serviceType, resourceType string, resIds []string, key, value string) error {
	params := make(map[string]string)
	params["ServiceType"] = serviceType
	params["ResourcePrefix"] = resourceType
	params["ResourceRegion"] = region.Region
	params["TagKey"] = key
	params["TagValue"] = value
	for i, resId := range resIds {
		params[fmt.Sprintf("ResourceIds.%d", i)] = resId
	}
	apiName := "ModifyResourcesTagValue"
	_, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return errors.Wrap(err, apiName)
	}
	return nil
}

func (region *SRegion) createTag(key, value string) error {
	params := make(map[string]string)
	params["TagKey"] = key
	params["TagValue"] = value
	apiName := "CreateTag"
	_, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return errors.Wrap(err, apiName)
	}
	return nil
}

func (region *SRegion) deleteTag(key, value string) error {
	params := make(map[string]string)
	params["TagKey"] = key
	params["TagValue"] = value
	apiName := "DeleteTag"
	_, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return errors.Wrap(err, apiName)
	}
	return nil
}

type SDescribeTag struct {
	TagKey   string `json:"TagKey"`
	TagValue string `json:"TagValue"`
}

type SDescribeTagsSeqResponse struct {
	SListInfo
	Tags []SDescribeTag `json:"Tags"`
}

func (region *SRegion) fetchTags(keys []string, limit int, offset int) (int, []SDescribeTag, error) {
	if len(keys) == 0 {
		return 0, nil, nil
	}
	params := make(map[string]string)
	for i, k := range keys {
		params[fmt.Sprintf("TagKeys.%d", i)] = k
	}
	if limit > 0 {
		params["Limit"] = strconv.FormatInt(int64(limit), 10)
	}
	if offset > 0 {
		params["Offset"] = strconv.FormatInt(int64(offset), 10)
	}
	apiName := "DescribeTagValuesSeq"
	respJson, err := region.client.tagRequest(apiName, params)
	if err != nil {
		return -1, nil, errors.Wrap(err, apiName)
	}
	resp := SDescribeTagsSeqResponse{}
	err = respJson.Unmarshal(&resp)
	if err != nil {
		return -1, nil, errors.Wrap(err, "Unmarshal")
	}
	return resp.TotalCount, resp.Tags, nil
}

func (region *SRegion) fetchAllTags(keys []string) ([]SDescribeTag, error) {
	tags := make([]SDescribeTag, 0)
	total := -1
	for total < 0 || len(tags) < total {
		st, stags, err := region.fetchTags(keys, 100, len(tags))
		if err != nil {
			return nil, errors.Wrap(err, "fetchTags")
		}
		if total < 0 {
			total = st
		}
		tags = append(tags, stags...)
	}
	return tags, nil
}

func (region *SRegion) tagsExist(tags map[string]string) (map[string]string, map[string]string, error) {
	tagKeys := make([]string, 0, len(tags))
	for k := range tags {
		tagKeys = append(tagKeys, k)
	}
	tagRows, err := region.fetchAllTags(tagKeys)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetchAllTags")
	}
	existTags := make(map[string]string)
	for i := range tagRows {
		tagkv := tagRows[i]
		if v, ok := tags[tagkv.TagKey]; ok && (tagkv.TagValue == v || (utils.IsInStringArray(tagkv.TagValue, []string{"", "null"}) && len(v) == 0)) {
			// exist
			existTags[tagkv.TagKey] = tagkv.TagValue
		}
	}
	notexistTags := make(map[string]string)
	for k, v := range tags {
		if _, ok := existTags[k]; !ok {
			notexistTags[k] = v
		}
	}
	return existTags, notexistTags, nil
}

func (region *SRegion) SetResourceTags(serviceType, resoureType string, resIds []string, tags map[string]string, replace bool) error {
	allTags, notExist, err := region.tagsExist(tags)
	if err != nil {
		return errors.Wrapf(err, "tagsExist")
	}
	var getTagValue = func(k string) string {
		if len(tags[k]) > 0 {
			return tags[k]
		}
		if v, ok := allTags[k]; ok {
			return v
		}
		return "null"
	}

	for k, v := range notExist {
		err := region.createTag(k, getTagValue(k))
		if err != nil {
			return errors.Wrapf(err, "createTag %s %s", k, v)
		}
	}

	oldTags, err := region.FetchResourceTags(serviceType, resoureType, resIds)
	if err != nil {
		return errors.Wrap(err, "FetchTags")
	}
	delKeyIds := make(map[string][]string)
	modKeyIds := make(map[string][]string)
	addKeyIds := make(map[string][]string)
	for id, okvsPtr := range oldTags {
		okvs := *okvsPtr
		for k, v := range okvs {
			if nv, ok := tags[k]; !ok {
				// key not exist
				if replace {
					// need to delete
					if _, ok := delKeyIds[k]; !ok {
						delKeyIds[k] = []string{id}
					} else {
						delKeyIds[k] = append(delKeyIds[k], id)
					}
				}
			} else if nv != v {
				// need to modify
				if _, ok := modKeyIds[k]; !ok {
					modKeyIds[k] = []string{id}
				} else {
					modKeyIds[k] = append(modKeyIds[k], id)
				}
			}
		}
		for k := range tags {
			if _, ok := okvs[k]; !ok {
				// need to add
				if _, ok := addKeyIds[k]; !ok {
					addKeyIds[k] = []string{id}
				} else {
					addKeyIds[k] = append(addKeyIds[k], id)
				}
			}
		}
	}
	for k := range tags {
		for _, id := range resIds {
			if _, ok := oldTags[id]; !ok {
				if _, ok := addKeyIds[k]; !ok {
					addKeyIds[k] = []string{id}
				} else {
					addKeyIds[k] = append(addKeyIds[k], id)
				}
			}
		}
	}
	for k, ids := range delKeyIds {
		err := region.detachTag(serviceType, resoureType, ids, k)
		if err != nil {
			return errors.Wrapf(err, "detachTag %s fail %s", k, err)
		}
	}
	for k, ids := range modKeyIds {
		err := region.modifyTag(serviceType, resoureType, ids, k, getTagValue(k))
		if err != nil {
			return errors.Wrapf(err, "modifyTag %s %s fail %s", k, getTagValue(k), err)
		}
	}
	for k, ids := range addKeyIds {
		err := region.attachTag(serviceType, resoureType, ids, k, getTagValue(k))
		if err != nil {
			return errors.Wrapf(err, "addTag %s %s fail %s", k, getTagValue(k), err)
		}
	}
	return nil
}
