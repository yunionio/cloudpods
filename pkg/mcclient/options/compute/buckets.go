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
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type BucketListOptions struct {
	options.BaseListOptions
	DistinctField string `help:"query specified distinct field"`
}

func (opts *BucketListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketGetPropertyOptions struct {
	DistinctField string `help:"query specified distinct field"`
}

func (opts *BucketGetPropertyOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

func (opts *BucketGetPropertyOptions) Property() string {
	return "distinct-field"
}

type BucketIdOptions struct {
	options.BaseIdOptions
}

type BucketUpdateOptions struct {
	BucketIdOptions

	Name          string `help:"new name of bucket" json:"name"`
	Desc          string `help:"Description of bucket" json:"description" token:"desc"`
	EnablePerfMon bool   `help:"enable performance monitor" json:"-"`
}

func (opts *BucketUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts)
	if opts.EnablePerfMon {
		params.(*jsonutils.JSONDict).Add(jsonutils.JSONTrue, "enable_perf_mon")
	}
	return params, nil
}

type BucketCreateOptions struct {
	NAME        string `help:"name of bucket" json:"name"`
	CLOUDREGION string `help:"location of bucket" json:"cloudregion"`
	MANAGER     string `help:"cloud provider" json:"manager"`

	StorageClass string `help:"bucket storage class"`
	Acl          string `help:"bucket ACL"`

	EnablePerfMon bool `help:"enable performance monitor"`
}

func (opts *BucketCreateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketListObjectsOptions struct {
	BucketIdOptions

	Prefix       string `help:"List objects with prefix"`
	Recursive    bool   `help:"List objects recursively"`
	Limit        int    `help:"maximal items per request"`
	PagingMarker string `help:"paging marker"`
}

func (opts *BucketListObjectsOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.ListStructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketDeleteObjectsOptions struct {
	BucketIdOptions

	KEYS []string `help:"List of objects to delete"`
}

func (opts *BucketDeleteObjectsOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketMakeDirOptions struct {
	BucketIdOptions

	KEY string `help:"DIR key to create"`
}

func (opts *BucketMakeDirOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketPresignObjectsOptions struct {
	BucketIdOptions

	KEY           string `help:"Key of object to upload"`
	Method        string `help:"Request method" choices:"GET|PUT|DELETE"`
	ExpireSeconds int    `help:"expire in seconds" default:"60"`
}

func (opts *BucketPresignObjectsOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketSetAclOptions struct {
	BucketIdOptions

	ACL string   `help:"ACL to set" choices:"default|private|public-read|public-read-write" json:"acl"`
	Key []string `help:"Optional object key" json:"key"`
}

func (opts *BucketSetAclOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketAclOptions struct {
	BucketIdOptions

	Key string `help:"Optional object key"`
}

func (opts *BucketAclOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketSyncOptions struct {
	BucketIdOptions

	StatsOnly bool `help:"sync statistics only"`
}

func (opts *BucketSyncOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketLimitOptions struct {
	BucketIdOptions

	SizeBytes   int64 `help:"size limit in bytes"`
	ObjectCount int64 `help:"object count limit"`
}

func (opts *BucketLimitOptions) Params() (jsonutils.JSONObject, error) {
	limit := jsonutils.Marshal(opts)
	params := jsonutils.NewDict()
	params.Set("limit", limit)
	return params, nil
}

type BucketAccessInfoOptions struct {
	BucketIdOptions
}

func (opts *BucketAccessInfoOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketSetMetadataOptions struct {
	BucketIdOptions

	Key []string `help:"Optional object key" json:"key"`

	objectstore.ObjectHeaderOptions
}

func (opts *BucketSetMetadataOptions) Params() (jsonutils.JSONObject, error) {
	input := compute.BucketMetadataInput{}
	input.Key = opts.Key
	input.Metadata = opts.ObjectHeaderOptions.Options2Header()
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(input), nil
}

type BucketSetWebsiteOption struct {
	BucketIdOptions
	// 主页
	Index string `help:"main page"`
	// 错误时返回的文档
	ErrorDocument string `help:"error return"`
	// http或https
	Protocol string `help:"force https" choices:"http|https"`
}

func (opts *BucketSetWebsiteOption) Params() (jsonutils.JSONObject, error) {
	conf := compute.BucketWebsiteConf{
		Index:         opts.Index,
		ErrorDocument: opts.ErrorDocument,
		Protocol:      opts.Protocol,
	}
	return jsonutils.Marshal(conf), nil
}

type BucketGetWebsiteConfOption struct {
	BucketIdOptions
}

func (opts *BucketGetWebsiteConfOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketDeleteWebsiteConfOption struct {
	BucketIdOptions
}

func (opts *BucketDeleteWebsiteConfOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketSetCorsOption struct {
	BucketIdOptions

	AllowedMethods []string `help:"allowed http method" choices:"PUT|GET|POST|DELETE|HEAD"`
	// 允许的源站，可以设为*
	AllowedOrigins []string
	AllowedHeaders []string
	MaxAgeSeconds  int
	ExposeHeaders  []string
	RuleId         string
}

func (args *BucketSetCorsOption) Params() (jsonutils.JSONObject, error) {
	rule := compute.BucketCORSRule{
		AllowedOrigins: args.AllowedOrigins,
		AllowedMethods: args.AllowedMethods,
		AllowedHeaders: args.AllowedHeaders,
		MaxAgeSeconds:  args.MaxAgeSeconds,
		ExposeHeaders:  args.ExposeHeaders,
		Id:             args.RuleId,
	}
	rules := compute.BucketCORSRules{Data: []compute.BucketCORSRule{rule}}
	return jsonutils.Marshal(rules), nil
}

type BucketGetCorsOption struct {
	BucketIdOptions
}

func (opts *BucketGetCorsOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketDeleteCorsOption struct {
	BucketIdOptions

	RuleId []string `help:"Id of rules to delete"`
}

func (opts *BucketDeleteCorsOption) Params() (jsonutils.JSONObject, error) {
	input := compute.BucketCORSRuleDeleteInput{}
	input.Id = opts.RuleId
	return jsonutils.Marshal(input), nil
}

type BucketSetRefererOption struct {
	BucketIdOptions
	// 域名列表
	DomainList []string
	// 是否允许空referer 访问
	AllowEmptyRefer bool `help:"all empty refer access"`
	Enabled         bool
	RerererType     string `help:"Referer type" choices:"Black-List|White-List"`
}

func (args *BucketSetRefererOption) Params() (jsonutils.JSONObject, error) {
	conf := compute.BucketRefererConf{
		Enabled:         args.Enabled,
		AllowEmptyRefer: args.AllowEmptyRefer,
		RefererType:     args.RerererType,
		DomainList:      args.DomainList,
	}
	return jsonutils.Marshal(conf), nil
}

type BucketGetRefererOption struct {
	BucketIdOptions
}

func (opts *BucketGetRefererOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketGetCdnDomainOption struct {
	BucketIdOptions
}

func (opts *BucketGetCdnDomainOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketGetPolicyOption struct {
	BucketIdOptions
}

func (opts *BucketGetPolicyOption) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

type BucketSetPolicyOption struct {
	BucketIdOptions
	// 格式主账号id:子账号id
	PrincipalId []string `help:"ext account id, accountId:subaccountId"`
	// Read|ReadWrite|FullControl
	CannedAction string `help:"authority action" choice:"Read|FullControl"`
	// Allow|Deny
	Effect string `help:"allow or deny" choice:"Allow|Deny"`
	// 被授权的资源地址
	ResourcePath []string
	// ip 条件
	IpEquals    []string
	IpNotEquals []string
}

func (args *BucketSetPolicyOption) Params() (jsonutils.JSONObject, error) {
	opts := compute.BucketPolicyStatementInput{}
	opts.CannedAction = args.CannedAction
	opts.Effect = args.Effect
	opts.IpEquals = args.IpEquals
	opts.IpNotEquals = args.IpNotEquals
	opts.ResourcePath = args.ResourcePath
	opts.PrincipalId = args.PrincipalId
	return jsonutils.Marshal(opts), nil
}

type BucketDeletePolicyOption struct {
	BucketIdOptions
	PolicyId []string `help:"policy id to delete"`
}

func (args *BucketDeletePolicyOption) Params() (jsonutils.JSONObject, error) {
	input := compute.BucketPolicyDeleteInput{}
	input.Id = args.PolicyId
	return jsonutils.Marshal(input), nil
}

type BucketUploadObjectsOptions struct {
	BucketIdOptions

	KEY  string `help:"Key of object to upload"`
	Path string `help:"Path to file to upload" required:"true"`

	ContentLength int64  `help:"Content lenght (bytes)" default:"-1"`
	StorageClass  string `help:"storage CLass"`
	Acl           string `help:"object acl." choices:"private|public-read|public-read-write"`

	objectstore.ObjectHeaderOptions
}

type BucketPerfMonOptions struct {
	BucketIdOptions

	Payload string `help:"test payload in bytes, e.g. 1, 32, 1024M" default:"4M"`
}
