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

package multicloud

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/encode"
)

type STagBase struct {
}

func (self STagBase) GetSysTags() map[string]string {
	return nil
}

func (self STagBase) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self STagBase) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type QcloudTags struct {
	TagSet []STag

	// Redis
	InstanceTags []STag
	// Elasticsearch
	TagList []STag
	// Kafka
	Tags []STag
	// Cdn
	Tag []STag
	// TDSQL
	ResourceTags []STag
}

func (self *QcloudTags) getTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.TagSet {
		if tag.Value == "null" {
			tag.Value = ""
		}
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.InstanceTags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.TagList {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.Tags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.Tag {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	for _, tag := range self.ResourceTags {
		if tag.TagValue == "null" {
			tag.TagValue = ""
		}
		ret[tag.TagKey] = tag.TagValue
	}
	return ret, nil
}

func (self *QcloudTags) GetTags() (map[string]string, error) {
	tags, _ := self.getTags()
	for k := range tags {
		if strings.HasPrefix(k, "tencentcloud:") {
			delete(tags, k)
		}
	}
	return tags, nil
}

func (self *QcloudTags) GetSysTags() map[string]string {
	tags, _ := self.getTags()
	ret := map[string]string{}
	for k, v := range tags {
		if strings.HasPrefix(k, "tencentcloud:") {
			ret[k] = v
		}
	}
	return ret
}

func (self *QcloudTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type STag struct {
	TagKey   string
	TagValue string

	Key   string
	Value string
}

type AliyunTags struct {
	Tags struct {
		Tag []STag

		// Kafka
		TagVO []STag `json:"TagVO" yunion-deprecated-by:"Tag"`
	}
}

func (self *AliyunTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasSuffix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") ||
			strings.HasSuffix(tag.Key, "ack.") { // k8s
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
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasPrefix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") ||
			strings.HasPrefix(tag.Key, "ack.") { // k8s
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

type ApsaraTags struct {
	Tags struct {
		Tag []STag
	}
}

func (self *ApsaraTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasSuffix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") {
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

func (self *ApsaraTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for _, tag := range self.Tags.Tag {
		if strings.HasPrefix(tag.TagKey, "aliyun") || strings.HasPrefix(tag.TagKey, "acs:") ||
			strings.HasPrefix(tag.Key, "aliyun") || strings.HasPrefix(tag.Key, "acs:") {
			if len(tag.TagKey) > 0 {
				ret[tag.TagKey] = tag.TagValue
			} else if len(tag.Key) > 0 {
				ret[tag.Key] = tag.Value
			}
		}
	}
	return ret
}

func (self *ApsaraTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type GoogleTags struct {
	Labels map[string]string
}

func (self *GoogleTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for k, v := range self.Labels {
		if strings.HasPrefix(k, "goog-") {
			continue
		}
		ret[encode.DecodeGoogleLable(k)] = encode.DecodeGoogleLable(v)
	}
	return ret, nil
}

func (self *GoogleTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for k, v := range self.Labels {
		if strings.HasPrefix(k, "goog-") {
			ret[k] = v
		}
	}
	return ret
}

func (self *GoogleTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type AzureTags struct {
	Tags map[string]string
}

func (self *AzureTags) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *AzureTags) GetSysTags() map[string]string {
	return nil
}

func (self *AzureTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type SAwsTag struct {
	Key   string `xml:"key"`
	Value string `xml:"value"`
}

type SAwsRdsTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type AwsTags struct {
	TagSet []SAwsTag `xml:"tagSet>item"`
	// rds
	TagList []SAwsRdsTag `xml:"TagList>Tag"`
}

func (self AwsTags) GetName() string {
	for _, tag := range self.TagSet {
		if strings.ToLower(tag.Key) == "name" {
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
		ret[tag.Key] = tag.Value
	}
	for _, tag := range self.TagList {
		if strings.ToLower(tag.Key) == "name" || strings.ToLower(tag.Key) == "description" {
			continue
		}
		ret[tag.Key] = tag.Value
	}
	return ret, nil
}

func (self *AwsTags) GetSysTags() map[string]string {
	return nil
}

func (self *AwsTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type CtyunTags struct {
}

func (self *CtyunTags) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self *CtyunTags) GetSysTags() map[string]string {
	return nil
}

func (self *CtyunTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type EcloudTags struct {
}

func (self *EcloudTags) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self *EcloudTags) GetSysTags() map[string]string {
	return nil
}

func (self *EcloudTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type JdcloudTags struct {
}

func (jt *JdcloudTags) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self *JdcloudTags) GetSysTags() map[string]string {
	return nil
}

func (self *JdcloudTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type HuaweiTags struct {
	Tags []string
}

func (self *HuaweiTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	for _, kv := range self.Tags {
		splited := strings.Split(kv, "=")
		if len(splited) == 2 {
			tags[splited[0]] = splited[1]
		}
	}
	return tags, nil
}

func (self *HuaweiTags) GetSysTags() map[string]string {
	return nil
}

func (self *HuaweiTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type HuaweiDiskTags struct {
	Tags map[string]string
}

func (self *HuaweiDiskTags) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *HuaweiDiskTags) GetSysTags() map[string]string {
	return nil
}

func (self *HuaweiDiskTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type OpenStackTags struct {
	Metadata map[string]string
}

func (self *OpenStackTags) GetTags() (map[string]string, error) {
	return self.Metadata, nil
}

func (self *OpenStackTags) GetSysTags() map[string]string {
	return nil
}

func (self *OpenStackTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type UcloudTags struct {
}

func (self *UcloudTags) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self *UcloudTags) GetSysTags() map[string]string {
	return nil
}

func (self *UcloudTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type ZStackTags struct {
}

func (self *ZStackTags) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self *ZStackTags) GetSysTags() map[string]string {
	return nil
}

func (self *ZStackTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type CloudpodsTags struct {
	Metadata map[string]string
}

func (self *CloudpodsTags) GetTags() (map[string]string, error) {
	metadatas := map[string]string{}
	for k, v := range self.Metadata {
		if strings.HasPrefix(k, apis.USER_TAG_PREFIX) {
			metadatas[strings.TrimPrefix(k, apis.USER_TAG_PREFIX)] = v
		}
	}
	return metadatas, nil
}

func (self *CloudpodsTags) GetSysTags() map[string]string {
	return nil
}

func (self *CloudpodsTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type BingoTags struct {
	TagSet []struct {
		Key   string
		Value string
	}
}

func (self *BingoTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	for _, tag := range self.TagSet {
		tags[tag.Key] = tag.Value
	}
	return tags, nil
}

func (self *BingoTags) GetSysTags() map[string]string {
	return nil
}

func (self *BingoTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type InCloudSphereTags struct {
}

func (self *InCloudSphereTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	return tags, nil
}

func (self *InCloudSphereTags) GetSysTags() map[string]string {
	return nil
}

func (self *InCloudSphereTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type ProxmoxTags struct {
}

func (self *ProxmoxTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	return tags, nil
}

func (self *ProxmoxTags) GetSysTags() map[string]string {
	return nil
}

func (self *ProxmoxTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type RemoteFileTags struct {
	Tags    map[string]string
	SysTags map[string]string
}

func (self *RemoteFileTags) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *RemoteFileTags) GetSysTags() map[string]string {
	return self.SysTags
}

func (self *RemoteFileTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type HcsTags struct {
	Tags []string
}

func (self *HcsTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	for _, kv := range self.Tags {
		splited := strings.Split(kv, "=")
		if len(splited) == 2 {
			tags[splited[0]] = splited[1]
		}
	}
	return tags, nil
}

func (self *HcsTags) GetSysTags() map[string]string {
	return nil
}

func (self *HcsTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
