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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SCdnOrigin struct {
	Origins            []string `json:"Origins"`
	OriginType         string   `json:"OriginType"`
	ServerName         string   `json:"ServerName"`
	CosPrivateAccess   string   `json:"CosPrivateAccess"`
	OriginPullProtocol string   `json:"OriginPullProtocol"`
	BackupOrigins      []string `json:"BackupOrigins"`
	BackupOriginType   string   `json:"BackupOriginType"`
	BackupServerName   string   `json:"BackupServerName"`
}

type SCdnDomain struct {
	multicloud.SResourceBase
	multicloud.QcloudTags

	client *SQcloudClient

	Area        string     `json:"Area"`
	Cname       string     `json:"Cname"`
	CreateTime  string     `json:"CreateTime"`
	Disable     string     `json:"Disable"`
	Domain      string     `json:"Domain"`
	Origin      SCdnOrigin `json:"Origin"`
	ProjectID   int        `json:"ProjectId"`
	Readonly    string     `json:"Readonly"`
	ResourceID  string     `json:"ResourceId"`
	ServiceType string     `json:"ServiceType"`
	Status      string     `json:"Status"`
	UpdateTime  string     `json:"UpdateTime"`

	config *SCdnConfig
}

func (self *SCdnDomain) GetConfig() (*SCdnConfig, error) {
	var err error = nil
	if self.config == nil {
		self.config, err = self.client.GetCdnConfig(self.ResourceID)
	}
	return self.config, err
}

type SCacheKey struct {
	FullUrlCache string
	IgnoreCase   string
	KeyRules     []struct {
		RulePaths    []string
		RuleType     string
		FullUrlCache string
		IgnoreCase   string
		QueryString  struct {
			Switch string
			Action string
			Value  string
		}
		RuleTag string
	}
}

type SCache struct {
	RuleCache []CdnCache
}

type CdnCache struct {
	CdnCacheCacheConfig CdnCacheCacheConfig `json:"CacheConfig"`
	RulePaths           []string            `json:"RulePaths"`
	RuleType            string              `json:"RuleType"`
}

type CdnCacheCacheConfig struct {
	CdnCacheCacheConfigCache        CdnCacheCacheConfigCache        `json:"Cache"`
	CdnCacheCacheConfigFollowOrigin CdnCacheCacheConfigFollowOrigin `json:"FollowOrigin"`
	CdnCacheCacheConfigNoCache      CdnCacheCacheConfigNoCache      `json:"NoCache"`
}

type CdnCacheCacheConfigFollowOrigin struct {
	CdnCacheCacheConfigFollowOriginHeuristicCache CdnCacheCacheConfigFollowOriginHeuristicCache `json:"HeuristicCache"`
	Switch                                        string                                        `json:"Switch"`
}

type CdnCacheCacheConfigFollowOriginHeuristicCache struct {
	CdnCacheCacheConfigFollowOriginHeuristicCacheCacheConfig CdnCacheCacheConfigFollowOriginHeuristicCacheCacheConfig `json:"CacheConfig"`
	Switch                                                   string                                                   `json:"Switch"`
}

type CdnCacheCacheConfigFollowOriginHeuristicCacheCacheConfig struct {
	HeuristicCacheTime       int    `json:"HeuristicCacheTime"`
	HeuristicCacheTimeSwitch string `json:"HeuristicCacheTimeSwitch"`
}

type CdnCacheCacheConfigNoCache struct {
	Revalidate string `json:"Revalidate"`
	Switch     string `json:"Switch"`
}

type CdnCacheCacheConfigCache struct {
	CacheTime          int    `json:"CacheTime"`
	CompareMaxAge      string `json:"CompareMaxAge"`
	IgnoreCacheControl string `json:"IgnoreCacheControl"`
	IgnoreSetCookie    string `json:"IgnoreSetCookie"`
	Switch             string `json:"Switch"`
}

type SRangeOriginPull struct {
	Switch     string
	RangeRules []struct {
		Switch    string
		RuleType  string
		RulePaths []string
	}
	Cache *SCache
}

type SCdnHttps struct {
	Switch string
	Http2  string
}

type SForceRedirect struct {
	Switch       string
	RedirectType string
}

type SCdnReferer struct {
	Switch       string
	RefererRules []struct {
		RuleType    string
		RulePaths   []string
		RefererType string
		Referers    []string
		AllowEmpty  bool
	}
}

type SMaxAge struct {
	Switch      string
	MaxAgeRules []struct {
		MaxAgeType     string
		MaxAgeContents []string
		MaxAgeTime     int
		FollowOrigin   string
	}
}

type SCdnConfig struct {
	CacheKey *SCacheKey

	RangeOriginPull *SRangeOriginPull

	Cache *SCache

	Https *SCdnHttps

	ForceRedirect *SForceRedirect

	Referer *SCdnReferer

	MaxAge *SMaxAge
}

func (self *SCdnDomain) GetName() string {
	return self.Domain
}

func (self *SCdnDomain) GetGlobalId() string {
	return self.Domain
}

func (self *SCdnDomain) GetId() string {
	return self.Domain
}

func (self *SCdnDomain) GetStatus() string {
	return self.Status
}

func (self *SCdnDomain) GetEnabled() bool {
	return self.Disable == "normal"
}

func (self *SCdnDomain) GetCname() string {
	return self.Cname
}

func (self *SCdnDomain) GetOrigins() *cloudprovider.SCdnOrigins {
	ret := cloudprovider.SCdnOrigins{}
	if self.Origin.OriginType == "cos" {
		self.Origin.OriginType = api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET
	}
	for _, org := range self.Origin.Origins {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Type:       self.Origin.OriginType,
			ServerName: self.Origin.ServerName,
			Protocol:   self.Origin.OriginPullProtocol,
			Origin:     org,
		})
	}
	for _, org := range self.Origin.BackupOrigins {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Type:       self.Origin.BackupOriginType,
			ServerName: self.Origin.BackupServerName,
			Origin:     org,
		})
	}
	return &ret
}

func (self *SCdnDomain) GetArea() string {
	return self.Area
}

func (self *SCdnDomain) GetServiceType() string {
	return self.ServiceType
}

func (self *SQcloudClient) GetCdnDomain(domain string) (*SCdnDomain, error) {
	domains, _, err := self.DescribeCdnDomains([]string{domain}, nil, "", 0, 100)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCdnDomains")
	}
	for i := range domains {
		if domains[i].Domain == domain {
			domains[i].client = self
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, domain)
}

func (self *SCdnDomain) Refresh() error {
	domain, err := self.client.GetCdnDomain(self.Domain)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, domain)
}

func (self *SCdnDomain) Delete() error {
	err := self.client.StopCdnDomain(self.Domain)
	if err != nil {
		return errors.Wrapf(err, "StopCdnDomain")
	}
	return self.client.DeleteCdnDomain(self.Domain)
}

func (self *SCdnDomain) SetTags(tags map[string]string, replace bool) error {
	region, err := self.client.getDefaultRegion()
	if err != nil {
		return errors.Wrapf(err, "getDefaultRegion")
	}
	return region.SetResourceTags("cdn", "domain", []string{self.Domain}, tags, replace)
}

func (self *SQcloudClient) StopCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("StopCdnDomain", params)
	return errors.Wrapf(err, "StopCdnDomain")
}

func (self *SQcloudClient) StartCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("StartCdnDomain", params)
	return errors.Wrapf(err, "StartCdnDomain")
}

func (self *SQcloudClient) DeleteCdnDomain(domain string) error {
	params := map[string]string{
		"Domain": domain,
	}
	_, err := self.cdnRequest("DeleteCdnDomain", params)
	return errors.Wrapf(err, "DeleteCdnDomain")
}

type SDomains struct {
	RequestID   string       `json:"RequestId"`
	Domains     []SCdnDomain `json:"Domains"`
	TotalNumber int          `json:"TotalNumber"`
}

func (self *SQcloudClient) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	cdns, err := self.DescribeAllCdnDomains(nil, nil, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudCDNDomain{}
	for i := range cdns {
		cdns[i].client = self
		ret = append(ret, &cdns[i])
	}
	return ret, nil
}

func (self *SQcloudClient) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	domains, _, err := self.DescribeCdnDomains([]string{name}, nil, "", 0, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCdnDomains")
	}
	for i := range domains {
		if domains[i].Domain == name {
			domains[i].client = self
			return &domains[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
}

func (client *SQcloudClient) AddCdnDomain(domain string, originType string, origins []string, cosPrivateAccess string) error {
	params := map[string]string{}
	params["Domain"] = domain
	params["ServiceType"] = "web"
	for i := range origins {
		params[fmt.Sprintf("Origin.Origins.%d", i)] = origins[i]
	}
	params["Origin.OriginType"] = originType
	params["Origin.CosPrivateAccess"] = cosPrivateAccess
	_, err := client.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return errors.Wrapf(err, `AddCdnDomain %s`, params)
	}
	return nil
}

func (client *SQcloudClient) DescribeCdnDomains(domains, origins []string, domainType string, offset int, limit int) ([]SCdnDomain, int, error) {
	params := map[string]string{}
	params["Offset"] = strconv.Itoa(offset)
	params["Limit"] = strconv.Itoa(limit)
	filterIndex := 0
	if len(domains) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domain"
		for i := range domains {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = domains[i]
		}
		filterIndex++
	}
	if len(origins) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "origin"
		for i := range origins {
			params[fmt.Sprintf("Filters.%d.Value.%d", filterIndex, i)] = origins[i]
		}
		filterIndex++
	}

	if len(domainType) > 0 {
		params[fmt.Sprintf("Filters.%d.Name", filterIndex)] = "domainType"
		params[fmt.Sprintf("Filters.%d.Value.0", filterIndex)] = domainType
		filterIndex++
	}

	resp, err := client.cdnRequest("DescribeDomainsConfig", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDomainsConfig %s", params)
	}
	cdnDomains := []SCdnDomain{}
	err = resp.Unmarshal(&cdnDomains, "Domains")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalcount, _ := resp.Float("TotalNumber")
	return cdnDomains, int(totalcount), nil
}

func (client *SQcloudClient) DescribeAllCdnDomains(domains, origins []string, domainType string) ([]SCdnDomain, error) {
	cdnDomains := make([]SCdnDomain, 0)
	for {
		part, total, err := client.DescribeCdnDomains(domains, origins, domainType, len(cdnDomains), 50)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeCdnDomains")
		}
		cdnDomains = append(cdnDomains, part...)
		if len(cdnDomains) >= total {
			break
		}
	}
	return cdnDomains, nil
}

func (self *SQcloudClient) CreateCDNDomain(opts *cloudprovider.CdnCreateOptions) (*SCdnDomain, error) {
	params := map[string]string{
		"Domain":      opts.Domain,
		"ServiceType": opts.ServiceType,
	}
	if len(opts.Area) > 0 {
		params["Area"] = opts.Area
	}
	originTypes := map[string][]string{}
	for _, origin := range opts.Origins {
		_, ok := originTypes[origin.Type]
		if !ok {
			originTypes[origin.Type] = []string{}
		}
		originTypes[origin.Type] = append(originTypes[origin.Type], origin.Origin)
	}
	for _, origin := range opts.Origins {
		params["Origin.OriginType"] = origin.Type
		if origin.Type == api.CDN_DOMAIN_ORIGIN_TYPE_BUCKET {
			params["Origin.OriginType"] = "cos"
		}
		if len(origin.ServerName) > 0 {
			params["Origin.ServerName"] = origin.ServerName
		} else {
			params["Origin.ServerName"] = origin.Origin
		}
		if len(origin.Protocol) > 0 {
			params["Origin.OriginPullProtocol"] = origin.Protocol
		}
		origins, ok := originTypes[origin.Type]
		if ok {
			for i, origin := range origins {
				params[fmt.Sprintf("Origin.Origins.%d", i)] = origin
			}
		}
	}
	_, err := self.cdnRequest("AddCdnDomain", params)
	if err != nil {
		return nil, errors.Wrapf(err, "AddCdnDomain")
	}
	return self.GetCdnDomain(opts.Domain)
}

func (self *SQcloudClient) GetCdnConfig(resourceId string) (*SCdnConfig, error) {
	params := map[string]string{
		"Filters.0.Name":    "resourceId",
		"Filters.0.Value.0": resourceId,
		"Limit":             "1",
	}
	resp, err := self.cdnRequest("DescribeDomainsConfig", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomainsConfig")
	}
	result := struct {
		Domains     []SCdnConfig
		TotalNumber int
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	for i := range result.Domains {
		return &result.Domains[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, resourceId)
}

func (self *SCdnDomain) GetCacheKeys() (*cloudprovider.SCDNCacheKeys, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.CacheKey == nil {
		return nil, nil
	}
	enabled, ignoreCase := false, false
	ret := &cloudprovider.SCDNCacheKeys{
		KeyRules: []cloudprovider.CacheKeyRule{},
	}
	if config.CacheKey.FullUrlCache == "on" {
		enabled = true
	}
	if config.CacheKey.IgnoreCase == "on" {
		ignoreCase = true
	}
	ret.Enabled, ret.IgnoreCase = &enabled, &ignoreCase
	for _, r := range config.CacheKey.KeyRules {
		rule := cloudprovider.CacheKeyRule{
			RulePaths:    r.RulePaths,
			RuleType:     r.RuleType,
			FullUrlCache: r.FullUrlCache == "on",
			IgnoreCase:   r.IgnoreCase == "on",
			RuleTag:      r.RuleTag,
			QueryString: cloudprovider.CacheKeyRuleQueryString{
				Enabled: r.QueryString.Switch == "on",
				Action:  r.QueryString.Action,
				Value:   r.QueryString.Value,
			},
		}
		ret.KeyRules = append(ret.KeyRules, rule)
	}
	return ret, nil
}

func (self *SCdnDomain) GetRangeOriginPull() (*cloudprovider.SCDNRangeOriginPull, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.RangeOriginPull == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNRangeOriginPull{RangeOriginPullRules: []cloudprovider.SRangeOriginPullRule{}}
	enabled := false
	if config.RangeOriginPull.Switch == "on" {
		enabled = true
	}
	ret.Enabled = &enabled
	for _, obj := range config.RangeOriginPull.RangeRules {
		rule := cloudprovider.SRangeOriginPullRule{
			Enabled:   false,
			RuleType:  obj.RuleType,
			RulePaths: obj.RulePaths,
		}
		if obj.Switch == "on" {
			rule.Enabled = true
		}
		ret.RangeOriginPullRules = append(ret.RangeOriginPullRules, rule)
	}
	return ret, nil
}

func (self *SCdnDomain) GetCache() (*cloudprovider.SCDNCache, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Cache == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNCache{
		RuleCache: []cloudprovider.SCacheRuleCache{},
	}
	for i, r := range config.Cache.RuleCache {
		rule := cloudprovider.SCacheRuleCache{
			Priority:    i + 1,
			RulePaths:   r.RulePaths,
			RuleType:    r.RuleType,
			CacheConfig: &cloudprovider.RuleCacheConfig{},
		}
		if r.CdnCacheCacheConfig.CdnCacheCacheConfigCache.Switch == "on" {
			rule.CacheConfig.Cache = &struct {
				Enabled            bool
				CacheTime          int
				CompareMaxAge      bool
				IgnoreCacheControl bool
				IgnoreSetCookie    bool
			}{
				Enabled:            true,
				CacheTime:          r.CdnCacheCacheConfig.CdnCacheCacheConfigCache.CacheTime,
				CompareMaxAge:      r.CdnCacheCacheConfig.CdnCacheCacheConfigCache.CompareMaxAge == "on",
				IgnoreCacheControl: r.CdnCacheCacheConfig.CdnCacheCacheConfigCache.IgnoreCacheControl == "on",
				IgnoreSetCookie:    r.CdnCacheCacheConfig.CdnCacheCacheConfigCache.IgnoreSetCookie == "on",
			}
		} else if r.CdnCacheCacheConfig.CdnCacheCacheConfigNoCache.Switch == "on" {
			rule.CacheConfig.NoCache = &struct {
				Enabled    bool
				Revalidate bool
			}{
				Enabled:    true,
				Revalidate: r.CdnCacheCacheConfig.CdnCacheCacheConfigNoCache.Revalidate == "on",
			}
		} else if r.CdnCacheCacheConfig.CdnCacheCacheConfigFollowOrigin.Switch == "on" {
			follow := r.CdnCacheCacheConfig.CdnCacheCacheConfigFollowOrigin
			rule.CacheConfig.FollowOrigin = &struct {
				Enabled        bool
				HeuristicCache struct {
					Enabled     bool
					CacheConfig struct {
						HeuristicCacheTimeSwitch bool
						HeuristicCacheTime       int
					}
				}
			}{
				Enabled: true,
			}
			rule.CacheConfig.FollowOrigin.HeuristicCache.Enabled = follow.Switch == "on"
			rule.CacheConfig.FollowOrigin.HeuristicCache.CacheConfig.HeuristicCacheTimeSwitch = follow.CdnCacheCacheConfigFollowOriginHeuristicCache.Switch == "on"
			rule.CacheConfig.FollowOrigin.HeuristicCache.CacheConfig.HeuristicCacheTime = follow.CdnCacheCacheConfigFollowOriginHeuristicCache.CdnCacheCacheConfigFollowOriginHeuristicCacheCacheConfig.HeuristicCacheTime
		}
		ret.RuleCache = append(ret.RuleCache, rule)
	}
	return ret, nil
}

func (self *SCdnDomain) GetHTTPS() (*cloudprovider.SCDNHttps, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Https == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNHttps{}
	enabled, enableHttp2 := false, false
	if config.Https.Switch == "on" {
		enabled = true
	}
	if config.Https.Http2 == "on" {
		enableHttp2 = true
	}
	ret.Enabled, ret.Http2 = &enabled, &enableHttp2
	return ret, nil
}

func (self *SCdnDomain) GetForceRedirect() (*cloudprovider.SCDNForceRedirect, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.ForceRedirect == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNForceRedirect{
		RedirectType: config.ForceRedirect.RedirectType,
	}
	enabled := false
	if config.ForceRedirect.Switch == "on" {
		enabled = true
	}
	ret.Enabled = &enabled
	return ret, nil
}

func (self *SCdnDomain) GetReferer() (*cloudprovider.SCDNReferer, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.Referer == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNReferer{
		RefererRules: []cloudprovider.RefererRule{},
	}
	enabled := false
	if config.Referer.Switch == "on" {
		enabled = true
	}
	ret.Enabled = &enabled
	for _, r := range config.Referer.RefererRules {
		rule := cloudprovider.RefererRule{
			RuleType:    r.RuleType,
			RulePaths:   r.RulePaths,
			RefererType: r.RuleType,
			Referers:    r.Referers,
			AllowEmpty:  &r.AllowEmpty,
		}
		ret.RefererRules = append(ret.RefererRules, rule)
	}
	return ret, nil
}

func (self *SCdnDomain) GetMaxAge() (*cloudprovider.SCDNMaxAge, error) {
	config, err := self.GetConfig()
	if err != nil {
		return nil, err
	}
	if config.MaxAge == nil {
		return nil, nil
	}
	ret := &cloudprovider.SCDNMaxAge{}
	enabled := false
	if config.MaxAge.Switch == "on" {
		enabled = true
	}
	ret.Enabled = &enabled
	for _, r := range config.MaxAge.MaxAgeRules {
		rule := cloudprovider.SMaxAgeRule{
			MaxAgeType:     r.MaxAgeType,
			MaxAgeContents: r.MaxAgeContents,
			MaxAgeTime:     r.MaxAgeTime,
			FollowOrigin:   false,
		}
		if r.FollowOrigin == "on" {
			rule.FollowOrigin = true
		}
		ret.MaxAgeRules = append(ret.MaxAgeRules, rule)
	}
	return ret, nil
}
