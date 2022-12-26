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

package ctyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CTYUN_API_HOST          = "https://api.ctyun.cn"
	CLOUD_PROVIDER_CTYUN    = api.CLOUD_PROVIDER_CTYUN
	CLOUD_PROVIDER_CTYUN_CN = "天翼云"
	CLOUD_PROVIDER_CTYUN_EN = "Ctyun"
	CTYUN_DEFAULT_REGION    = "cn-bj4"

	CTYUN_API_VERSION = "2019-11-22"
)

type CtyunClientConfig struct {
	cpcfg   cloudprovider.ProviderConfig
	options *cloudprovider.SCtyunExtraOptions

	projectId    string
	accessKey    string
	accessSecret string

	debug bool
}

func NewSCtyunClientConfig(accessKey, accessSecret string, options *cloudprovider.SCtyunExtraOptions) *CtyunClientConfig {
	cfg := &CtyunClientConfig{
		accessKey:    accessKey,
		accessSecret: accessSecret,
		options:      options,
	}
	return cfg
}

func (cfg *CtyunClientConfig) ProjectId(projectId string) *CtyunClientConfig {
	cfg.projectId = projectId
	return cfg
}

func (cfg *CtyunClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *CtyunClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *CtyunClientConfig) Debug(debug bool) *CtyunClientConfig {
	cfg.debug = debug
	return cfg
}

type SCtyunClient struct {
	*CtyunClientConfig

	httpClient *http.Client
	iregions   []cloudprovider.ICloudRegion
}

func NewSCtyunClient(cfg *CtyunClientConfig) (*SCtyunClient, error) {
	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		if cfg.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	client := &SCtyunClient{
		CtyunClientConfig: cfg,
		httpClient:        httpClient,
	}

	err := client.init()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client *SCtyunClient) init() error {
	err := client.fetchRegions()
	if err != nil {
		return err
	}

	return nil
}

func (client *SCtyunClient) fetchRegions() error {
	resp, err := client.DoGet("/apiproxy/v3/order/getZoneConfig", map[string]string{})
	if err != nil {
		return err
	}

	zones := []SZone{}
	err = resp.Unmarshal(&zones, "returnObj")
	if err != nil {
		return err
	}

	regions := map[string]SRegion{}
	for i := range zones {
		zone := zones[i]

		if len(client.projectId) > 0 && !strings.Contains(client.projectId, zone.RegionID) {
			continue
		}

		if region, ok := regions[zone.RegionID]; !ok {
			region = SRegion{
				client:         client,
				Description:    zone.ZoneName,
				ID:             zone.RegionID,
				ParentRegionID: zone.RegionID,
				RegionName:     zone.ZoneName,
				izones:         []cloudprovider.ICloudZone{&zone},
			}

			zone.region = &region
			zone.host = &SHost{zone: &zone}
			regions[zone.RegionID] = region
		} else {
			zone.region = &region
			zone.host = &SHost{zone: &zone}
			region.izones = append(region.izones, &zone)
		}
	}

	client.iregions = []cloudprovider.ICloudRegion{}
	for k := range regions {
		region := regions[k]
		client.iregions = append(client.iregions, &region)
	}
	return nil
}

func (client *SCtyunClient) DoGet(apiName string, queries map[string]string) (jsonutils.JSONObject, error) {
	return formRequest(client, httputils.GET, apiName, queries, nil)
}

func (client *SCtyunClient) DoPost(apiName string, params map[string]jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return formRequest(client, httputils.POST, apiName, nil, params)
}

func getCustiomInfo(t string, crmBizId string, accountId string) jsonutils.JSONObject {
	if len(t) == 0 {
		return nil
	}

	customeInfo := jsonutils.NewDict()
	//customeInfo.Set("name", jsonutils.NewString(""))
	//customeInfo.Set("email", jsonutils.NewString(""))
	//customeInfo.Set("phone", jsonutils.NewString(""))
	indentity := jsonutils.NewDict()
	if len(crmBizId) > 0 {
		indentity.Set("crmBizId", jsonutils.NewString(crmBizId))
	}

	if len(accountId) > 0 {
		indentity.Set("accountId", jsonutils.NewString(accountId))
	}

	if len(t) > 0 {
		customeInfo.Set("type", jsonutils.NewString(t))
		customeInfo.Set("identity", indentity)
	}

	return customeInfo
}

func formRequest(client *SCtyunClient, method httputils.THttpMethod, apiName string, queries map[string]string, params map[string]jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	header := http.Header{}
	// signer
	{
		content := []string{}
		for k, v := range queries {
			content = append(content, v)
			header.Set(k, v)
		}

		for _, v := range params {
			c, _ := v.GetString()
			content = append(content, c)
		}

		// contentMd5 := fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(content, "\n"))))
		// contentMd5 = base64.StdEncoding.EncodeToString([]byte(contentMd5))
		contentRaw := strings.Join(content, "\n")
		contentMd5 := utils.GetMD5Base64([]byte(contentRaw))

		// EEE, d MMM yyyy HH:mm:ss z
		// Mon, 2 Jan 2006 15:04:05 MST
		requestDate := time.Now().Format("Mon, 2 Jan 2006 15:04:05 MST")
		hashMac := hmac.New(sha1.New, []byte(client.accessSecret))
		hashRawString := strings.Join([]string{contentMd5, requestDate, apiName}, "\n")
		hashMac.Write([]byte(hashRawString))
		hsum := base64.StdEncoding.EncodeToString(hashMac.Sum(nil))

		header.Set("accessKey", client.accessKey)
		header.Set("contentMD5", contentMd5)
		header.Set("requestDate", requestDate)
		header.Set("hmac", hsum)
		// 平台类型，整数类型，取值范围：2或3，传2表示2.0自营资源，传3表示3.0合营资源，该参数不需要加密。
		header.Set("platform", "3")
		// crm账号需要设置customInfo。目前发现只有VPC列表查询需要用到这个参数，其他的接口还没有发现此要求。为了简便对于crm 统一设置此项
		if client.options != nil && len(client.options.CrmBizId) > 0 {
			customInfo := getCustiomInfo("1", client.options.CrmBizId, "")
			header.Set("customInfo", customInfo.String())
		}
	}

	var reqbody string
	ioData := strings.NewReader("")
	if method == httputils.GET {
		for k, v := range queries {
			header.Set(k, v)
		}
	} else {
		datas := url.Values{}
		for k, v := range params {
			c, _ := v.GetString()
			datas.Add(k, c)
		}

		reqbody = datas.Encode()
		ioData = strings.NewReader(reqbody)
	}

	header.Set("Content-Length", strconv.FormatInt(int64(ioData.Len()), 10))
	header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.Background()
	MAX_RETRY := 3
	retry := 0

	var err error
	for retry < MAX_RETRY {
		resp, err := httputils.Request(
			client.httpClient,
			ctx,
			method,
			CTYUN_API_HOST+apiName,
			header,
			ioData,
			client.debug)

		_, jsonResp, err := httputils.ParseJSONResponse(reqbody, resp, err, client.debug)
		if err == nil {
			if code, _ := jsonResp.Int("statusCode"); code != 800 {
				if strings.Contains(jsonResp.String(), "NotFound") {
					return nil, cloudprovider.ErrNotFound
				}

				return nil, &httputils.JSONClientError{Code: 400, Details: jsonResp.String()}
			}

			return jsonResp, nil
		}

		switch e := err.(type) {
		case *httputils.JSONClientError:
			if e.Code >= 499 {
				time.Sleep(3 * time.Second)
				retry += 1
				continue
			} else {
				return nil, err
			}
		default:
			return nil, err
		}
	}

	return nil, fmt.Errorf("timeout for request: %s \n\n with params: %s", err, params)
}

func (self *SCtyunClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SCtyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccounts := make([]cloudprovider.SSubAccount, 0)
	for i := range self.iregions {
		iregion := self.iregions[i]

		s := cloudprovider.SSubAccount{
			Name:         fmt.Sprintf("%s-%s", self.cpcfg.Name, iregion.GetId()),
			Account:      fmt.Sprintf("%s/%s", self.accessKey, iregion.GetId()),
			HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
		}

		subAccounts = append(subAccounts, s)
	}

	return subAccounts, nil
}

func (client *SCtyunClient) GetAccountId() string {
	return client.accessKey
}

func (self *SCtyunClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SCtyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SCtyunClient) GetAccessEnv() string {
	return api.CLOUD_ACCESS_ENV_CTYUN_CHINA
}

func (self *SCtyunClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SCtyunClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = CTYUN_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SCtyunClient) GetCloudRegionExternalIdPrefix() string {
	if len(self.projectId) > 0 {
		return self.iregions[0].GetGlobalId()
	} else {
		return CLOUD_PROVIDER_CTYUN
	}
}

func (self *SCtyunClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
