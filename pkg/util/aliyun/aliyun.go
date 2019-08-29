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
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	v "yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_ALIYUN    = api.CLOUD_PROVIDER_ALIYUN
	CLOUD_PROVIDER_ALIYUN_CN = "阿里云"

	ALIYUN_DEFAULT_REGION = "cn-hangzhou"

	ALIYUN_API_VERSION     = "2014-05-26"
	ALIYUN_API_VERSION_VPC = "2016-04-28"
	ALIYUN_API_VERSION_LB  = "2014-05-15"

	ALIYUN_BSS_API_VERSION = "2017-12-14"

	ALIYUN_RAM_API_VERSION = "2015-05-01"
)

type SAliyunClient struct {
	providerId   string
	providerName string
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion

	Debug bool
}

func NewAliyunClient(providerId string, providerName string, accessKey string, secret string, isDebug bool) (*SAliyunClient, error) {
	client := SAliyunClient{
		providerId:   providerId,
		providerName: providerName,
		accessKey:    accessKey,
		secret:       secret,
		Debug:        isDebug,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func jsonRequest(client *sdk.Client, domain, apiVersion, apiName string, params map[string]string, debug bool) (jsonutils.JSONObject, error) {
	if debug {
		log.Debugf("request %s %s %s %s", domain, apiVersion, apiName, params)
	}
	for i := 1; i < 4; i++ {
		resp, err := _jsonRequest(client, domain, apiVersion, apiName, params)
		retry := false
		if err != nil {
			for _, code := range []string{"404 Not Found"} {
				if strings.Contains(err.Error(), code) {
					return nil, cloudprovider.ErrNotFound
				}
			}
			for _, code := range []string{"SignatureNonceUsed", "InvalidInstance.NotSupported", "try later", "BackendServer.configuring"} {
				if strings.Contains(err.Error(), code) {
					retry = true
					break
				}
			}
		}
		if retry {
			if debug {
				log.Debugf("Retry %d...", i)
			}
			time.Sleep(time.Second * time.Duration(i*10))
			continue
		}
		if debug {
			log.Debugf("Response: %s", resp)
		}
		return resp, err
	}
	return nil, fmt.Errorf("timeout for request %s params: %s", apiName, params)
}

func _jsonRequest(client *sdk.Client, domain string, version string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	req := requests.NewCommonRequest()
	req.Domain = domain
	req.Version = version
	req.ApiName = apiName
	if params != nil {
		for k, v := range params {
			req.QueryParams[k] = v
		}
	}
	req.Scheme = "https"
	req.GetHeaders()["User-Agent"] = "vendor/yunion-OneCloud@" + v.Get().GitVersion

	resp, err := processCommonRequest(client, req)
	if err != nil {
		log.Errorf("request error %s with params %s", err, params)
		return nil, err
	}
	body, err := jsonutils.Parse(resp.GetHttpContentBytes())
	if err != nil {
		log.Errorf("parse json fail %s", err)
		return nil, err
	}
	//{"Code":"InvalidInstanceType.ValueNotSupported","HostId":"ecs.aliyuncs.com","Message":"The specified instanceType beyond the permitted range.","RequestId":"0042EE30-0EDF-48A7-A414-56229D4AD532"}
	//{"Code":"200","Message":"successful","PageNumber":1,"PageSize":50,"RequestId":"BB4C970C-0E23-48DC-A3B0-EB21FFC70A29","RouterTableList":{"RouterTableListType":[{"CreationTime":"2017-03-19T13:37:40Z","Description":"","ResourceGroupId":"rg-acfmwie3cqoobmi","RouteTableId":"vtb-j6c60lectdi80rk5xz43g","RouteTableName":"","RouteTableType":"System","RouterId":"vrt-j6c00qrol733dg36iq4qj","RouterType":"VRouter","VSwitchIds":{"VSwitchId":["vsw-j6c3gig5ub4fmi2veyrus"]},"VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"}]},"Success":true,"TotalCount":1}
	if body.Contains("Code") {
		code, _ := body.GetString("Code")
		if len(code) > 0 && !utils.IsInStringArray(code, []string{"200"}) {
			return nil, fmt.Errorf(body.String())
		}
	}
	return body, nil
}

func (self *SAliyunClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

func (self *SAliyunClient) getDefaultClient() (*sdk.Client, error) {
	return sdk.NewClientWithAccessKey(ALIYUN_DEFAULT_REGION, self.accessKey, self.secret)
}

func (self *SAliyunClient) ecsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "ecs.aliyuncs.com", ALIYUN_API_VERSION, apiName, params, self.Debug)
}

func (self *SAliyunClient) fetchRegions() error {
	body, err := self.ecsRequest("DescribeRegions", map[string]string{"AcceptLanguage": "zh-CN"})
	if err != nil {
		log.Errorf("fetchRegions fail %s", err)
		return err
	}

	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions", "Region")
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}
	self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i += 1 {
		regions[i].client = self
		self.iregions[i] = &regions[i]
	}
	return nil
}

func (self *SAliyunClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SAliyunClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := self.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = self.providerName
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SAliyunClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAliyunClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = ALIYUN_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SAliyunClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAliyunClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SAliyunClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	// 阿里云并未公布资源组的api地址
	// params := map[string]string{}
	// body, err := region.ecsRequest("ListResourceGroups", params)
	return nil, cloudprovider.ErrNotImplemented
}
