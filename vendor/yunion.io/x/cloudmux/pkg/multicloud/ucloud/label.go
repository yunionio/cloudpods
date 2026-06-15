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

package ucloud

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	UCLOUD_LABEL_CATEGORY_CUSTOM = "custom"
	UCLOUD_LABEL_CATEGORY_SYSTEM = "system"
)

type SResourceLabel struct {
	ResourceId string `json:"ResourceId"`
	Key        string `json:"Key"`
	Value      string `json:"Value"`
	Category   string `json:"Category"`
}

func setLabelsParams(params *SParams, prefix string, labels map[string]string) {
	idx := 0
	for k, v := range labels {
		params.Set(fmt.Sprintf("%s.%d.Key", prefix, idx), k)
		params.Set(fmt.Sprintf("%s.%d.Value", prefix, idx), v)
		idx++
	}
}

func setResourceIdsParams(params *SParams, resourceIds []string) {
	for i, id := range resourceIds {
		params.Set(fmt.Sprintf("ResourceIds.%d", i), id)
	}
}

// https://docs.ucloud.cn/api/label-api/list_labels_by_resource_ids
func (client *SUcloudClient) fetchAllLabelsByResourceIds(resourceIds []string) ([]SResourceLabel, error) {
	if len(resourceIds) == 0 {
		return nil, nil
	}
	ret := []SResourceLabel{}
	offset := 0
	limit := 100
	for {
		params := client.commonParams(NewUcloudParams())
		setResourceIdsParams(&params, resourceIds)
		params.Set("Offset", offset)
		params.Set("Limit", limit)
		params.SetAction("ListLabelsByResourceIds")
		resp, err := jsonRequest(client, params)
		if err != nil {
			return nil, errors.Wrap(err, "ListLabelsByResourceIds")
		}
		part := []SResourceLabel{}
		err = resp.Unmarshal(&part, "Labels")
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal Labels")
		}
		ret = append(ret, part...)
		total, _ := resp.Int("TotalCount")
		if int(total) > 0 && len(ret) >= int(total) {
			break
		}
		if len(part) < limit {
			break
		}
		offset += limit
	}
	return ret, nil
}

func filterCustomLabels(labels []SResourceLabel, resourceId string) map[string]string {
	ret := map[string]string{}
	for i := range labels {
		if labels[i].ResourceId != resourceId {
			continue
		}
		if labels[i].Category == UCLOUD_LABEL_CATEGORY_SYSTEM {
			continue
		}
		ret[labels[i].Key] = labels[i].Value
	}
	return ret
}

// https://docs.ucloud.cn/api/label-api/create_labels
func (client *SUcloudClient) createLabels(labels map[string]string) error {
	if len(labels) == 0 {
		return nil
	}
	params := client.commonParams(NewUcloudParams())
	setLabelsParams(&params, "Labels", labels)
	params.SetAction("CreateLabels")
	_, err := jsonRequest(client, params)
	if err != nil {
		if uerr, ok := err.(*SUcloudError); ok && strings.Contains(strings.ToLower(uerr.Message), "duplicate") {
			return nil
		}
		return errors.Wrap(err, "CreateLabels")
	}
	return nil
}

// https://docs.ucloud.cn/api/label-api/bind_labels
func (client *SUcloudClient) bindLabels(resourceIds []string, labels map[string]string) error {
	if len(resourceIds) == 0 || len(labels) == 0 {
		return nil
	}
	params := client.commonParams(NewUcloudParams())
	setResourceIdsParams(&params, resourceIds)
	setLabelsParams(&params, "Labels", labels)
	params.SetAction("BindLabels")
	_, err := jsonRequest(client, params)
	if err != nil {
		return errors.Wrap(err, "BindLabels")
	}
	return nil
}

// https://docs.ucloud.cn/api/label-api/unbind_labels
func (client *SUcloudClient) unbindLabels(resourceIds []string, labels map[string]string) error {
	if len(resourceIds) == 0 || len(labels) == 0 {
		return nil
	}
	params := client.commonParams(NewUcloudParams())
	setResourceIdsParams(&params, resourceIds)
	setLabelsParams(&params, "Labels", labels)
	params.SetAction("UnbindLabels")
	_, err := jsonRequest(client, params)
	if err != nil {
		return errors.Wrap(err, "UnbindLabels")
	}
	return nil
}

func (self *SRegion) GetResourceTags(resourceId string) (map[string]string, error) {
	if len(resourceId) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrMissingParameter, "resourceId")
	}
	labels, err := self.client.fetchAllLabelsByResourceIds([]string{resourceId})
	if err != nil {
		return nil, errors.Wrap(err, "fetchAllLabelsByResourceIds")
	}
	return filterCustomLabels(labels, resourceId), nil
}

func (self *SRegion) SetResourceTags(resourceId string, tags map[string]string, replace bool) error {
	if len(resourceId) == 0 {
		return errors.Wrap(cloudprovider.ErrMissingParameter, "resourceId")
	}
	if len(tags) == 0 && !replace {
		return nil
	}
	resourceIds := []string{resourceId}
	existing, err := self.GetResourceTags(resourceId)
	if err != nil {
		return errors.Wrap(err, "GetResourceTags")
	}
	toUnbind := map[string]string{}
	for k, v := range existing {
		nv, ok := tags[k]
		if replace {
			if !ok || nv != v {
				toUnbind[k] = v
			}
		} else if ok && nv != v {
			toUnbind[k] = v
		}
	}
	if err := self.client.unbindLabels(resourceIds, toUnbind); err != nil {
		return errors.Wrap(err, "unbindLabels")
	}
	if len(tags) == 0 {
		return nil
	}
	if err := self.client.createLabels(tags); err != nil {
		return errors.Wrap(err, "createLabels")
	}
	return self.client.bindLabels(resourceIds, tags)
}

// for shell debug
func (self *SRegion) FetchResourceTags(resourceIds []string) (map[string]map[string]string, error) {
	labels, err := self.client.fetchAllLabelsByResourceIds(resourceIds)
	if err != nil {
		return nil, err
	}
	ret := map[string]map[string]string{}
	for _, id := range resourceIds {
		ret[id] = filterCustomLabels(labels, id)
	}
	return ret, nil
}
