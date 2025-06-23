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

package aliyun

import (
	"encoding/json"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type AliyunTags struct {
	// 使用RawMessage来延迟解析，兼容两种格式
	TagsRaw json.RawMessage `json:"Tags"`

	// 缓存解析后的标签
	parsedTags []multicloud.STag
	parsed     bool
}

var sysTags = []string{
	"aliyun", "creator",
	"acs:", "serverless/", "alloc_id", "virtual-kubelet",
	"diskId", "diskNum", "serverId", "restoreId", "cnfs-id", "from", "shadowId",
	"ack.aliyun.com", "cluster-id.ack.aliyun.com", "ack.alibabacloud.com",
	"k8s.io", "k8s.aliyun.com", "kubernetes.do.not.delete", "kubernetes.reused.by.user",
	"HBR InstanceId", "HBR Retention Days", "HBR Retention Type", "HBR JobId",
	"createdBy", "recoveryPointTime", "recoveryPointId",
	"eas_resource_group_name", "eas_tenant_name", "managedby",
}

// 解析标签数据，兼容两种格式
func (self *AliyunTags) parseTags() error {
	if self.parsed || len(self.TagsRaw) == 0 {
		return nil
	}

	// 先尝试解析为直接数组格式（ALB格式）
	var directTags []multicloud.STag
	if err := json.Unmarshal(self.TagsRaw, &directTags); err == nil {
		self.parsedTags = directTags
		self.parsed = true
		return nil
	}

	// 如果失败，尝试解析为嵌套格式（传统格式）
	var nested struct {
		Tag   []multicloud.STag
		TagVO []multicloud.STag `json:"TagVO"`
	}

	if err := json.Unmarshal(self.TagsRaw, &nested); err == nil {
		// 合并Tag和TagVO
		var allTags []multicloud.STag
		allTags = append(allTags, nested.Tag...)
		allTags = append(allTags, nested.TagVO...)
		self.parsedTags = allTags
		self.parsed = true
		return nil
	}

	return errors.Errorf("failed to parse tags")
}

// 获取所有标签
func (self *AliyunTags) getAllTags() []multicloud.STag {
	self.parseTags()
	return self.parsedTags
}

func (self *AliyunTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}

	// 获取所有标签（兼容两种格式）
	allTags := self.getAllTags()

	for _, tag := range allTags {
		if tag.IsSysTagPrefix(sysTags) {
			continue
		}
		if len(tag.TagKey) > 0 {
			ret[tag.TagKey] = tag.TagValue
		} else if len(tag.Key) > 0 {
			ret[tag.Key] = tag.Value
		}
	}

	return ret, nil
}

func (self *AliyunTags) GetSysTags() map[string]string {
	ret := map[string]string{}

	// 获取所有标签（兼容两种格式）
	allTags := self.getAllTags()

	for _, tag := range allTags {
		if tag.IsSysTagPrefix(sysTags) {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			} else if len(tag.Key) > 0 {
				ret[tag.Key] = tag.Value
			}
		}
	}
	return ret
}

func (self *AliyunTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type SAliyunTag struct {
	ResourceId   string
	ResourceType string
	TagKey       string
	TagValue     string
}
