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

package compute

import (
	"net/http"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	BUCKET_OPS_STATS_CHANGE = "stats_change"

	BUCKET_STATUS_START_CREATE = "start_create"
	BUCKET_STATUS_CREATING     = "creating"
	BUCKET_STATUS_READY        = "ready"
	BUCKET_STATUS_CREATE_FAIL  = "create_fail"
	BUCKET_STATUS_START_DELETE = "start_delete"
	BUCKET_STATUS_DELETING     = "deleting"
	BUCKET_STATUS_DELETED      = "deleted"
	BUCKET_STATUS_DELETE_FAIL  = "delete_fail"
	BUCKET_STATUS_UNKNOWN      = "unknown"

	BUCKET_UPLOAD_OBJECT_KEY_HEADER          = "X-Yunion-Bucket-Upload-Key"
	BUCKET_UPLOAD_OBJECT_ACL_HEADER          = "X-Yunion-Bucket-Upload-Acl"
	BUCKET_UPLOAD_OBJECT_STORAGECLASS_HEADER = "X-Yunion-Bucket-Upload-Storageclass"
)

type BucketCreateInput struct {
	apis.SharableVirtualResourceCreateInput
	CloudregionResourceInput
	CloudproviderResourceInput

	StorageClass string `json:"storage_class"`
}

type BucketDetails struct {
	apis.SharableVirtualResourceDetails
	ManagedResourceInfo
	CloudregionResourceInfo

	SBucket

	// 访问URL列表
	AccessUrls []cloudprovider.SBucketAccessUrl `json:"access_urls"`
}

type BucketObjectsActionInput struct {
	Key []string
}

type BucketAclInput struct {
	BucketObjectsActionInput

	Acl cloudprovider.TBucketACLType
}

func (input *BucketAclInput) Validate() error {
	switch input.Acl {
	case cloudprovider.ACLPrivate, cloudprovider.ACLAuthRead, cloudprovider.ACLPublicRead, cloudprovider.ACLPublicReadWrite:
		// do nothing
	default:
		return errors.Wrap(httperrors.ErrInputParameter, "acl")
	}
	return nil
}

type BucketMetadataInput struct {
	BucketObjectsActionInput

	Metadata http.Header
}

func (input *BucketMetadataInput) Validate() error {
	if len(input.Key) == 0 {
		return errors.Wrap(httperrors.ErrEmptyRequest, "key")
	}
	if len(input.Metadata) == 0 {
		return errors.Wrap(httperrors.ErrEmptyRequest, "metadata")
	}
	return nil
}

type BucketListInput struct {
	apis.SharableVirtualResourceListInput
	apis.ExternalizedResourceBaseListInput

	ManagedResourceListInput
	RegionalFilterListInput

	// STORAGE_CLASS
	StorageClass []string `json:"storage_class"`

	// 位置
	Location []string `json:"location"`

	// ACL
	Acl []string `json:"acl"`
}

type BucketSyncstatusInput struct {
}

type BucketUpdateInput struct {
	apis.SharableVirtualResourceBaseUpdateInput
}

type BucketPerformTempUrlInput struct {
	// 访问对象方法
	Method string `json:"method"`
	// 对象KEY
	// required:true
	Key string `json:"key"`
	// 过期时间，单位秒
	ExpireSeconds *int `json:"expire_seconds"`
}

type BucketPerformTempUrlOutput struct {
	// 生成的临时URL
	Url string `json:"url"`
}

type BucketPerformMakedirInput struct {
	// 目录对象KEY
	// required:true
	Key string `json:"key"`
}

type BucketPerformDeleteInput struct {
	// 待删除对象KEY
	// required:true
	Keys []string `json:"keys"`
}

type BucketGetAclInput struct {
	// 对象KEY
	// required:false
	Key string `json:"key"`
}

type BucketGetAclOutput struct {
	// ACL
	Acl string `json:"acl"`
}

type BucketGetObjectsInput struct {
	// Prefix
	Prefix string `json:"prefix"`
	// 是否模拟列举目录模式
	Recursive *bool `json:"recursive"`
	// 分页标识
	PagingMarker string `json:"paging_marker"`
	// 最大输出条目数
	Limit *int `json:"limit"`
}

type BucketGetObjectsOutput struct {
	// 对象列表
	Data []cloudprovider.SCloudObject `json:"data"`
	// 排序字段，总是key
	// example: key
	MarkerField string `json:"marker_field"`
	// 排序顺序，总是降序
	// example: DESC
	MarkerOrder string `json:"marker_order"`
	// 下一页请求的paging_marker标识
	NextMarker string `json:"next_marker"`
}

type BucketWebsiteRoutingRule struct {
	ConditionErrorCode string
	ConditionPrefix    string

	RedirectProtocol         string
	RedirectReplaceKey       string
	RedirectReplaceKeyPrefix string
}

type BucketWebsiteConf struct {
	// 主页
	Index string
	// 错误时返回的文档
	ErrorDocument string
	// http或https
	Protocol string

	Rules []BucketWebsiteRoutingRule
	// 访问网站url
	Url string
}

func (input *BucketWebsiteConf) Validate() error {
	if len(input.Index) == 0 {
		return httperrors.NewMissingParameterError("index")
	}
	if len(input.ErrorDocument) == 0 {
		return httperrors.NewMissingParameterError("error_document")
	}
	if len(input.Protocol) == 0 {
		return httperrors.NewMissingParameterError("protocol")
	}
	return nil
}

type BucketCORSRule struct {
	AllowedMethods []string
	// 允许的源站，可以是*
	AllowedOrigins []string
	AllowedHeaders []string
	MaxAgeSeconds  int
	ExposeHeaders  []string
	// 规则区别标识
	Id string
}

type BucketCORSRules struct {
	Data []BucketCORSRule `json:"data"`
}

type BucketCORSRuleDeleteInput struct {
	Id []string
}

type BucketPolicy struct {
	Data []BucketPolicyStatement
}

type BucketPolicyStatement struct {
	// 授权的目标主体
	Principal map[string][]string `json:"Principal,omitempty"`
	// 授权的行为
	Action []string `json:"Action,omitempty"`
	// Allow|Deny
	Effect string `json:"Effect,omitempty"`
	// 被授权的资源地址
	Resource []string `json:"Resource,omitempty"`
	// 触发授权的条件
	Condition map[string]map[string]interface{} `json:"Condition,omitempty"`

	// 解析字段，主账号id:子账号id
	PrincipalId []string
	// Read|ReadWrite|FullControl
	CannedAction string
	// 资源路径
	ResourcePath []string
	// 根据index 生成
	Id string
}

type BucketPolicyStatementInput struct {
	// 主账号id:子账号id
	PrincipalId []string
	// Read|ReadWrite|FullControl
	CannedAction string
	// Allow|Deny
	Effect string
	// 被授权的资源地址,/*
	ResourcePath []string
	// ip 条件
	IpEquals    []string
	IpNotEquals []string
}

func (input *BucketPolicyStatementInput) Validate() error {
	cannedAction := []string{"Read", "ReadWrite", "FullControl"}
	if !utils.IsInStringArray(input.CannedAction, cannedAction) {
		return httperrors.NewInputParameterError("invalid CannedAction %s ", input.CannedAction)
	}
	if input.Effect != "Allow" && input.Effect != "Deny" {
		return httperrors.NewInputParameterError("invalid Effect %s ", input.Effect)
	}
	for i := range input.IpEquals {
		if !regutils.MatchIP4Addr(input.IpEquals[i]) && !regutils.MatchCIDR(input.IpEquals[i]) {
			return httperrors.NewInputParameterError("invalid ipv4 %s ", input.IpEquals[i])
		}
	}
	for i := range input.IpNotEquals {
		if !regutils.MatchIP4Addr(input.IpNotEquals[i]) && !regutils.MatchCIDR(input.IpNotEquals[i]) {
			return httperrors.NewInputParameterError("invalid ipv4 %s ", input.IpNotEquals[i])
		}
	}
	return nil
}

type BucketPolicyDeleteInput struct {
	Id []string
}

func (input *BucketCORSRules) Validate() error {
	for i := range input.Data {
		if len(input.Data[i].AllowedOrigins) == 0 {
			return httperrors.NewMissingParameterError("allowed_origins")
		}
		if len(input.Data[i].AllowedMethods) == 0 {
			return httperrors.NewMissingParameterError("allowed_methods")
		}
	}
	return nil
}

type BucketRefererConf struct {
	// 白名单域名列表
	WhiteList []string
	// 黑名单域名列表
	BlackList []string
	// 是否允许空referer 访问
	AllowEmptyRefer bool
}

func (input *BucketRefererConf) Validate() error {
	return nil
}
