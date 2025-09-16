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

package huawei

import (
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SCdnDomainOriginHost struct {
	DomainId        string `json:"domain_id"`        // 域名ID
	OriginHostType  string `json:"origin_host_type"` // 回源host的类型
	CustomizeDomain string `json:"customize_domain"` // 自定义回源host域名
}

type SCdnSource struct {
	IPOrDomain    string `json:"ip_or_domain"`   // 源站IP（非内网IP）或者域名
	DomainID      string `json:"domain_id"`      //
	OriginType    string `json:"origin_type"`    // 源站类型取值：ipaddr、 domain、obs_bucket，分别表示：源站IP、源站域名、OBS桶访问域名
	ActiveStandby int    `json:"active_standby"` // 主备状态（1代表主站；0代表备站）,主源站必须存在，备源站可选，OBS桶不能有备源站
}

type SCdnDomain struct {
	multicloud.SResourceBase
	HuaweiTags

	client *SHuaweiClient

	DomainOriginHost    SCdnDomainOriginHost `json:"domain_origin_host"`
	Sources             []SCdnSource         `json:"sources"`
	DomainName          string               `json:"domain_name"`           // 加速域名
	Cname               string               `json:"cname"`                 // 加速域名对应的CNAME
	Id                  string               `json:"id"`                    // 加速域名ID
	UserDomainId        string               `json:"user_domain_id"`        // 域名所属用户的domain_id
	BusinessType        string               `json:"business_type"`         // 域名业务类型，若为web，则表示类型为网站加速；若为download，则表示业务类型为文件下载加速；若为video，则表示业务类型为点播加速；若为wholeSite，则表示类型为全站加速
	ServiceArea         string               `json:"service_area"`          // 华为云CDN提供的加速服务范围，包含：mainland_china中国大陆、outside_mainland_china中国大陆境外、global全球
	DomainStatus        string               `json:"domain_status"`         // 加速域名状态。取值意义： - online表示“已开启” - offline表示“已停用” - configuring表示“配置中” - configure_failed表示“配置失败” - checking表示“审核中” - check_failed表示“审核未通过” - deleting表示“删除中”
	HttpsStatus         int                  `json:"https_status"`          // 是否开启HTTPS加速
	CreateTime          int64                `json:"create_time"`           // 域名创建时间，相对于UTC 1970-01-01到当前时间相隔的毫秒数
	ModifyTime          int64                `json:"modify_time"`           // 域名修改时间，相对于UTC 1970-01-01到当前时间相隔的毫秒数
	Disabled            int                  `json:"disabled"`              // 封禁状态（0代表未禁用；1代表禁用）
	Locked              int                  `json:"locked"`                // 锁定状态（0代表未锁定；1代表锁定）
	RangeStatus         string               `json:"range_status"`          // Range回源状态
	FollowStatus        string               `json:"follow_status"`         // 回源跟随状态
	OriginStatus        string               `json:"origin_status"`         // 是否暂停源站回源
	AutoRefreshPreheat  int                  `json:"auto_refresh_preheat"`  // 自动刷新预热（0代表关闭；1代表打开）
	EnterpriseProjectId string               `json:"enterprise_project_id"` // 企业项目ID

	//config *SCdnConfig
}

func (cd *SCdnDomain) GetCacheKeys() (*cloudprovider.SCDNCacheKeys, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetRangeOriginPull() (*cloudprovider.SCDNRangeOriginPull, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetCache() (*cloudprovider.SCDNCache, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetHTTPS() (*cloudprovider.SCDNHttps, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetForceRedirect() (*cloudprovider.SCDNForceRedirect, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetReferer() (*cloudprovider.SCDNReferer, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetMaxAge() (*cloudprovider.SCDNMaxAge, error) {
	return nil, nil
}

func (cd *SCdnDomain) GetArea() string {
	switch cd.ServiceArea {
	case "mainland_china":
		return api.CDN_DOMAIN_AREA_MAINLAND
	case "global":
		return api.CDN_DOMAIN_AREA_GLOBAL
	case "outside_mainland_china":
		return api.CDN_DOMAIN_AREA_OVERSEAS
	default:
		return cd.ServiceArea
	}
}

func (cd *SCdnDomain) GetCname() string {
	return cd.Cname
}

func (cd *SCdnDomain) GetEnabled() bool {
	if cd.Disabled == 0 {
		return true
	} else {
		return false
	}
}

func (cd *SCdnDomain) GetId() string {
	return cd.DomainName
}

func (cd *SCdnDomain) GetGlobalId() string {
	return cd.DomainName
}

func (cd *SCdnDomain) GetName() string {
	return cd.DomainName
}

func (cd *SCdnDomain) Refresh() error {
	domain, err := cd.client.GetCdnDomain(cd.Id, cd.EnterpriseProjectId)
	if err != nil {
		return errors.Wrapf(err, "GetCdnDomain")
	}
	cd.DomainStatus = domain.DomainStatus
	return nil
}

func (cd *SCdnDomain) GetOrigins() *cloudprovider.SCdnOrigins {
	ret := cloudprovider.SCdnOrigins{}
	for _, origin := range cd.Sources {
		ret = append(ret, cloudprovider.SCdnOrigin{
			Origin: origin.IPOrDomain,
		})
	}
	return &ret
}

func (cd *SCdnDomain) GetServiceType() string {
	return cd.BusinessType
}

func (cd *SCdnDomain) GetStatus() string {
	switch cd.DomainStatus {
	case "online", "offline":
		return cd.DomainStatus
	case "configuring", "checking":
		return api.CDN_DOMAIN_STATUS_PROCESSING
	case "configure_failed":
		return cd.DomainStatus
	case "check_failed":
		return api.CDN_DOMAIN_STATUS_REJECTED
	}
	return cd.DomainStatus
}

func (self *SCdnDomain) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (cd *SCdnDomain) Delete() error {
	_, err := cd.client.delete(SERVICE_CDN, "", "cdn/domains/"+cd.Id)
	return err
}

func (hc *SHuaweiClient) GetICloudCDNDomains() ([]cloudprovider.ICloudCDNDomain, error) {
	domains, err := hc.GetCdnDomains()
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomains")
	}

	ret := []cloudprovider.ICloudCDNDomain{}
	for i := range domains {
		domains[i].client = hc
		ret = append(ret, &domains[i])
	}
	return ret, nil
}

func (hc *SHuaweiClient) GetICloudCDNDomainByName(name string) (cloudprovider.ICloudCDNDomain, error) {
	return hc.GetCDNDomainByName(name)
}

func (hc *SHuaweiClient) GetCDNDomainByName(name string) (*SCdnDomain, error) {
	domains, total, err := hc.DescribeUserDomains(name, 1, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCdnDomain")
	}
	if total == 1 {
		domains[0].client = hc
		return &domains[0], nil
	}
	if total == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, name)
	}

	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, name)
}

func (hc *SHuaweiClient) GetCdnDomains() ([]SCdnDomain, error) {
	domains := make([]SCdnDomain, 0)
	for {
		part, total, err := hc.DescribeUserDomains("", 1000, len(domains)/1000+1)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeUserDomains")
		}

		domains = append(domains, part...)
		if len(domains) >= total || len(part) == 0 {
			break
		}
	}
	return domains, nil
}

func (hc *SHuaweiClient) DescribeUserDomains(domain string, pageSize, pageNumber int) ([]SCdnDomain, int, error) {
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 1000
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	params := url.Values{
		"page_size":             []string{fmt.Sprintf("%d", pageSize)},
		"page_number":           []string{fmt.Sprintf("%d", pageNumber)},
		"enterprise_project_id": []string{"ALL"},
	}
	if len(domain) > 0 {
		params["domain_name"] = []string{domain}
	}
	resp, err := hc.list(SERVICE_CDN, "", "cdn/domains", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "DescribeUserDomains")
	}

	domains := make([]SCdnDomain, 0)
	err = resp.Unmarshal(&domains, "domains")
	if err != nil {
		return nil, 0, errors.Wrap(err, "resp.Unmarshal")
	}

	totalCount, _ := resp.Int("total")
	return domains, int(totalCount), nil
}

func (hc *SHuaweiClient) GetCdnDomain(domainID, epID string) (*SCdnDomain, error) {
	params := url.Values{"enterprise_project_id": []string{epID}}
	resp, err := hc.list(SERVICE_CDN, "", "cdn/domains/"+domainID+"/detail", params)
	if err != nil {
		return nil, errors.Wrapf(err, "ShowDomainDetail")
	}

	domain := &SCdnDomain{client: hc}
	err = resp.Unmarshal(domain, "domain")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}

	return domain, nil
}
