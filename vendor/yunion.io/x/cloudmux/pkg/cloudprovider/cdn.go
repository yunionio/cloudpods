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

package cloudprovider

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

// +onecloud:model-api-gen
type SCdnDomain struct {
	// cdn加速域名
	Domain string
	// 状态 rejected(域名未审核)|processing(部署中)|online|offline
	Status string
	// 区域 mainland|overseas|global
	Area string
	// cdn Cname
	Cname string
	// 源站
	Origin string
	// 源站类型 domain|ip|bucket
	OriginType string
}

// +onecloud:model-api-gen
type SCdnOrigin struct {
	// 源站类型
	// domain: 域名类型, cos：对象存储源站, ip：IP 列表作为源站
	// enmu: domain, cos, ip
	// required: true
	Type string
	// 源站地址
	Origin string
	// 回主源站时 Host 头部
	ServerName string
	// 回源协议
	// enmu: http, follow, https
	Protocol string
	Path     string
	Port     int
	Enabled  string
	Priority int
}

type CdnCreateOptions struct {
	Domain      string
	ServiceType string
	Area        string

	Origins SCdnOrigins
}

// +onecloud:model-api-gen
type SCdnOrigins []SCdnOrigin

func (origins SCdnOrigins) IsZero() bool {
	return len(origins) == 0
}

func (origins SCdnOrigins) String() string {
	return jsonutils.Marshal(origins).String()
}

// 是否忽略参数
// +onecloud:model-api-gen
type SCDNCacheKeys struct {
	// 开启关闭忽略参数
	Enabled *bool
	// 是否忽略大小
	IgnoreCase *bool
	// 分路径缓存键配置
	KeyRules []CacheKeyRule
}

type CacheKeyRule struct {
	RulePaths    []string
	RuleType     string
	FullUrlCache bool
	IgnoreCase   bool
	QueryString  CacheKeyRuleQueryString
	RuleTag      string
}

type CacheKeyRuleQueryString struct {
	Enabled bool
	Action  string
	Value   string
}

func (keys SCDNCacheKeys) IsZero() bool {
	return jsonutils.Marshal(keys) == jsonutils.Marshal(&SCDNCacheKeys{})
}

func (keys SCDNCacheKeys) String() string {
	return jsonutils.Marshal(keys).String()
}

// 是否分片回源
// +onecloud:model-api-gen
type SCDNRangeOriginPull struct {
	Enabled              *bool
	RangeOriginPullRules []SRangeOriginPullRule
}

type SRangeOriginPullRule struct {
	Enabled   bool
	RuleType  string
	RulePaths []string
}

func (pull SCDNRangeOriginPull) IsZero() bool {
	return jsonutils.Marshal(pull) == jsonutils.Marshal(&SCDNRangeOriginPull{})
}

func (pull SCDNRangeOriginPull) String() string {
	return jsonutils.Marshal(pull).String()
}

type CacheRule struct {
	// 规则类型：
	// all：所有文件生效
	// file：指定文件后缀生效
	// directory：指定路径生效
	// path：指定绝对路径生效
	// index：首页
	CacheType string
	// CacheType 对应类型下的匹配内容
	CacheContents []string
	// 过期时间: 秒
	CacheTime int
}

// +onecloud:model-api-gen
type SCDNCache struct {
	RuleCache []SCacheRuleCache
}

type SCacheRuleCache struct {
	RulePaths   []string
	RuleType    string
	Priority    int
	CacheConfig *RuleCacheConfig
}

type RuleCacheConfig struct {
	Cache *struct {
		Enabled            bool
		CacheTime          int
		CompareMaxAge      bool
		IgnoreCacheControl bool
		IgnoreSetCookie    bool
	}
	NoCache *struct {
		Enabled    bool
		Revalidate bool
	}
	FollowOrigin *struct {
		Enabled        bool
		HeuristicCache struct {
			Enabled     bool
			CacheConfig struct {
				HeuristicCacheTimeSwitch bool
				HeuristicCacheTime       int
			}
		}
	}
}

func (cache SCDNCache) IsZero() bool {
	return jsonutils.Marshal(cache) == jsonutils.Marshal(&SCDNCache{})
}

func (cache SCDNCache) String() string {
	return jsonutils.Marshal(cache).String()
}

// +onecloud:model-api-gen
type SCDNHttps struct {
	// https 配置开关
	Enabled *bool
	// http2 配置开关
	Http2 *bool
}

func (https SCDNHttps) IsZero() bool {
	return jsonutils.Marshal(https) == jsonutils.Marshal(&SCDNHttps{})
}

func (https SCDNHttps) String() string {
	return jsonutils.Marshal(https).String()
}

// +onecloud:model-api-gen
type SCDNForceRedirect struct {
	// 访问强制跳转配置开关
	Enabled *bool
	// 访问强制跳转类型
	// enmu: http, https
	RedirectType string
}

func (redirect SCDNForceRedirect) IsZero() bool {
	return jsonutils.Marshal(redirect) == jsonutils.Marshal(&SCDNForceRedirect{})
}

func (redirect SCDNForceRedirect) String() string {
	return jsonutils.Marshal(redirect).String()
}

type RefererRule struct {
	// 规则类型：
	// all：所有文件生效
	// file：指定文件后缀生效
	// directory：指定路径生效
	// path：指定绝对路径生效
	RuleType    string
	RulePaths   []string
	RefererType string
	Referers    []string
	AllowEmpty  *bool
}

// +onecloud:model-api-gen
type SCDNReferer struct {
	// 是否开启防盗链
	Enabled *bool

	RefererRules []RefererRule
}

func (referer SCDNReferer) IsZero() bool {
	return jsonutils.Marshal(referer) == jsonutils.Marshal(&SCDNReferer{})
}

func (referer SCDNReferer) String() string {
	return jsonutils.Marshal(referer).String()
}

type SMaxAgeRule struct {
	MaxAgeType     string
	MaxAgeContents []string
	MaxAgeTime     int
	FollowOrigin   bool
}

// 浏览器缓存配置
// +onecloud:model-api-gen
type SCDNMaxAge struct {
	Enabled     *bool
	MaxAgeRules []SMaxAgeRule
}

func (maxAge SCDNMaxAge) IsZero() bool {
	return jsonutils.Marshal(maxAge) == jsonutils.Marshal(&SCDNMaxAge{})
}

func (maxAge SCDNMaxAge) String() string {
	return jsonutils.Marshal(maxAge).String()
}

type CacheClearOptions struct {
	PurgeEverything *bool
	Tags            []string
	Hosts           []string
	Prefixes        []string
	Files           []interface{}
}

type CacheConfig struct {
	DNSSECEnabled   *bool
	HTTPSEnabled    *bool
	CacheLevel      string
	BrowserCacheTTL *int
	HTTPSRewrites   *bool
	SSLSetting      string
}

type CustomHostname struct {
	Id                    string
	Hostname              string
	CustomOriginServer    string
	OwnershipVerification struct {
		Name  string
		Type  string
		Value string
	}
	OwnershipVerificationHttp struct {
		HttpBody string
		HttpUrl  string
	}
	SSL struct {
		BundleMethod         string
		CertificateAuthority string
		DcvDelegationRecords []struct {
			Cname       string
			CnameTarget string
		}
		Id       string
		Method   string
		Settings struct {
			MinTLSVersion string `json:"min_tls_version"`
		}
		Status            string
		Type              string
		ValidationRecords []struct {
			Status   string
			TxtName  string
			TxtValue string
		}
		Wildcard bool
	}
	Status string
}

type CustomHostnameCreateOptions struct {
	CustomOriginServer string
	Hostname           string
	SSL                struct {
		CertificateAuthority string
		Method               string
		BundleMethod         string
		CustomCertificate    string
		CustomKey            string
		Settings             struct {
			MinTLSVersion string `json:"min_tls_version"`
		}
		Type     string
		Wildcard bool
	}
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SCdnOrigins{}), func() gotypes.ISerializable {
		return &SCdnOrigins{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNCacheKeys{}), func() gotypes.ISerializable {
		return &SCDNCacheKeys{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNRangeOriginPull{}), func() gotypes.ISerializable {
		return &SCDNRangeOriginPull{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNCache{}), func() gotypes.ISerializable {
		return &SCDNCache{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNHttps{}), func() gotypes.ISerializable {
		return &SCDNHttps{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNForceRedirect{}), func() gotypes.ISerializable {
		return &SCDNForceRedirect{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNReferer{}), func() gotypes.ISerializable {
		return &SCDNReferer{}
	})

	gotypes.RegisterSerializable(reflect.TypeOf(&SCDNMaxAge{}), func() gotypes.ISerializable {
		return &SCDNMaxAge{}
	})
}
