// Copyright 2023 Yunion
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

package volcengine

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	tos "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	sdk "github.com/volcengine/volc-sdk-golang/base"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_VOLCENGINE    = api.CLOUD_PROVIDER_VOLCENGINE
	CLOUD_PROVIDER_VOLCENGINE_CN = "火山云"
	CLOUD_PROVIDER_VOLCENGINE_EN = "VolcEngine"

	VOLCENGINE_API_VERSION         = "2020-04-01"
	VOLCENGINE_IAM_API_VERSION     = "2021-08-01"
	VOLCENGINE_OBSERVE_API_VERSION = "2018-01-01"
	VOLCENGINE_BILLING_API_VERSION = "2022-01-01"

	VOLCENGINE_API         = "open.volcengineapi.com"
	VOLCENGINE_IAM_API     = "iam.volcengineapi.com"
	VOLCENGINE_TOS_API     = "tos-cn-beijing.volces.com"
	VOLCENGINE_BILLING_API = "billing.volcengineapi.com"

	VOLCENGINE_SERVICE_ECS       = "ecs"
	VOLCENGINE_SERVICE_VPC       = "vpc"
	VOLCENGINE_SERVICE_NAT       = "natgateway"
	VOLCENGINE_SERVICE_STORAGE   = "storage_ebs"
	VOLCENGINE_SERVICE_IAM       = "iam"
	VOLCENGINE_SERVICE_TOS       = "tos"
	VOLCENGINE_SERVICE_OBSERVICE = "Volc_Observe"
	VOLCENGINE_SERVICE_BILLING   = "billing"

	VOLCENGINE_DEFAULT_REGION = "cn-beijing"
)

type VolcEngineClientConfig struct {
	cpcfg     cloudprovider.ProviderConfig
	cloudEnv  string
	accessKey string
	secretKey string
	accountId string
	debug     bool

	client *http.Client
	lock   sync.Mutex
}

func NewVolcEngineClientConfig(accessKey, secretKey string) *VolcEngineClientConfig {
	cfg := &VolcEngineClientConfig{
		accessKey: accessKey,
		secretKey: secretKey,
	}
	return cfg
}

func (cfg *VolcEngineClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *VolcEngineClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *VolcEngineClientConfig) AccountId(id string) *VolcEngineClientConfig {
	cfg.accountId = id
	return cfg
}

func (cfg *VolcEngineClientConfig) Debug(debug bool) *VolcEngineClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg VolcEngineClientConfig) Copy() VolcEngineClientConfig {
	return cfg
}

type SVolcEngineClient struct {
	*VolcEngineClientConfig

	ownerId string

	projects []SProject
	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket
}

func NewVolcEngineClient(cfg *VolcEngineClientConfig) (*SVolcEngineClient, error) {
	client := SVolcEngineClient{
		VolcEngineClientConfig: cfg,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchReginos")
	}
	return &client, nil
}

// Regions
func (client *SVolcEngineClient) fetchRegions() error {
	body, err := client.ecsRequest("", "DescribeRegions", nil)
	if err != nil {
		return errors.Wrapf(err, "DescribeRegions")
	}
	regions := make([]SRegion, 0)
	err = body.Unmarshal(&regions, "Regions")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	client.iregions = make([]cloudprovider.ICloudRegion, len(regions))
	for i := 0; i < len(regions); i += 1 {
		regions[i].client = client
		client.iregions[i] = &regions[i]
	}
	return nil
}

func (client *SVolcEngineClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(client.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := client.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (client *SVolcEngineClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = VOLCENGINE_DEFAULT_REGION
	}
	for i := 0; i < len(client.iregions); i += 1 {
		if client.iregions[i].GetId() == regionId {
			return client.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (client *SVolcEngineClient) GetIRegions() []cloudprovider.ICloudRegion {
	return client.iregions
}

func (client *SVolcEngineClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(client.iregions); i += 1 {
		if (client.iregions[i].GetId() == id) || (client.iregions[i].GetGlobalId() == id) {
			return client.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (client *SVolcEngineClient) GetAccountId() string {
	if len(client.ownerId) > 0 {
		return client.ownerId
	}
	caller, err := client.GetCallerIdentity()
	if err != nil {
		return ""
	}
	client.ownerId = caller.AccountId
	return client.ownerId
}

func (client *SVolcEngineClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = client.GetAccountId()
	subAccount.Name = client.cpcfg.Name
	subAccount.Account = client.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	subAccount.DefaultProjectId = "default"
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (client *SVolcEngineClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := client.GetProjects()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudProject{}
	for i := range projects {
		ret = append(ret, &projects[i])
	}
	return ret, nil
}

func (client *SVolcEngineClient) GetProjects() ([]SProject, error) {
	if len(client.projects) > 0 {
		return client.projects, nil
	}
	limit, offset := 50, 0
	client.projects = []SProject{}
	for {
		parts, total, err := client.ListProjects(limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "GetProjects")
		}
		client.projects = append(client.projects, parts...)
		if len(client.projects) >= total {
			break
		}
		offset += total
	}
	return client.projects, nil
}

func (cli *SVolcEngineClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cli.cpcfg.ReadOnly {
			action := req.URL.Query().Get("Action")
			if len(action) > 0 {
				for _, prefix := range []string{"Get", "Describe", "List"} {
					if strings.HasPrefix(action, prefix) {
						return nil, nil
					}
				}
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return cli.client
}

type sCred struct {
	client *SVolcEngineClient

	cred sdk.Credentials
}

func (self *sCred) Do(req *http.Request) (*http.Response, error) {
	cli := self.client.getDefaultClient()

	req = self.cred.Sign(req)

	return cli.Do(req)
}

func (client *SVolcEngineClient) monitorRequest(regionId, apiName string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	cred := client.getSdkCredential(regionId, VOLCENGINE_SERVICE_OBSERVICE, "")
	return client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_OBSERVE_API_VERSION, apiName, params)
}

func (client *SVolcEngineClient) jsonRequest(cred sdk.Credentials, domain string, apiVersion string, apiName string, params interface{}) (jsonutils.JSONObject, error) {

	query := url.Values{
		"Action":  []string{apiName},
		"Version": []string{apiVersion},
	}

	var body interface{} = nil
	if _params, ok := params.(map[string]string); ok {
		for k, v := range _params {
			query.Set(k, v)
		}
	} else {
		body = params
	}

	u, err := url.Parse(fmt.Sprintf("http://%s?%s", domain, query.Encode()))
	if err != nil {
		return nil, errors.Wrapf(err, "url.Parse")
	}
	method := httputils.GET
	for prefix, _method := range map[string]httputils.THttpMethod{
		"Get":      httputils.GET,
		"Describe": httputils.GET,
		"List":     httputils.GET,
		"Delete":   httputils.GET,
		"Put":      httputils.PUT,
	} {
		if strings.HasPrefix(apiName, prefix) {
			method = _method
			break
		}
	}
	if strings.HasPrefix(domain, "bucketname") {
		if strings.HasPrefix(apiName, "Delete") {
			method = httputils.DELETE
		}
	}
	if cred.Service == VOLCENGINE_SERVICE_OBSERVICE {
		method = httputils.POST
	}

	req := httputils.NewJsonRequest(method, u.String(), body)
	vErr := &sVolcError{}
	_cli := &sCred{
		client: client,
		cred:   cred,
	}
	cli := httputils.NewJsonClient(_cli)
	_, resp, err := cli.Send(context.Background(), req, vErr, client.debug)
	if err != nil {
		return nil, errors.Wrapf(err, apiName)
	}
	if resp.Contains("Result") {
		return resp.Get("Result")
	}
	return resp, nil
}

func (client *SVolcEngineClient) getSdkCredential(region string, service string, token string) sdk.Credentials {
	cred := sdk.Credentials{
		AccessKeyID:     client.accessKey,
		SecretAccessKey: client.secretKey,
		Region:          region,
		Service:         service,
		SessionToken:    token,
	}
	return cred
}

func (client *SVolcEngineClient) getDefaultCredential(region string, service string) sdk.Credentials {
	if region == "" {
		region = VOLCENGINE_DEFAULT_REGION
	}
	cred := sdk.Credentials{
		AccessKeyID:     client.accessKey,
		SecretAccessKey: client.secretKey,
		Region:          region,
		Service:         service,
		SessionToken:    "",
	}
	return cred
}

type sVolcError struct {
	StatusCode   int
	RequestId    string
	Action       string
	Version      string
	Service      string
	Region       string
	ErrorMessage struct {
		Code    string
		Message string
	} `json:"Error"`
}

func (self *sVolcError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sVolcError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self, "ResponseMetadata")
	}
	self.StatusCode = statusCode
	return self
}

func (client *SVolcEngineClient) ecsRequest(region string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := client.getDefaultCredential(region, VOLCENGINE_SERVICE_ECS)
	return client.jsonRequest(cred, VOLCENGINE_API, VOLCENGINE_API_VERSION, apiName, params)
}

func (client *SVolcEngineClient) iamRequest(region string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := client.getDefaultCredential(region, VOLCENGINE_SERVICE_IAM)
	return client.jsonRequest(cred, VOLCENGINE_IAM_API, VOLCENGINE_IAM_API_VERSION, apiName, params)
}

func (client *SVolcEngineClient) getTosClient(regionId string) (*tos.ClientV2, error) {
	tosClient, err := tos.NewClientV2(VOLCENGINE_TOS_API, tos.WithRegion(regionId), tos.WithCredentials(tos.NewStaticCredentials(client.accessKey, client.secretKey)))
	return tosClient, err
}

func (client *SVolcEngineClient) billRequest(region string, apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cred := client.getDefaultCredential(region, VOLCENGINE_SERVICE_BILLING)
	domain := VOLCENGINE_API
	if apiName == "QueryBalanceAcct" {
		domain = VOLCENGINE_BILLING_API + "/open-apis/trade_balance"
	}
	return client.jsonRequest(cred, domain, VOLCENGINE_BILLING_API_VERSION, apiName, params)
}

// Buckets
func (client *SVolcEngineClient) invalidateIBuckets() {
	client.iBuckets = nil
}

func (client *SVolcEngineClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if client.iBuckets == nil {
		err := client.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return client.iBuckets, nil
}

func (client *SVolcEngineClient) fetchBuckets() error {
	toscli, err := client.getTosClient(VOLCENGINE_DEFAULT_REGION)
	if err != nil {
		return errors.Wrap(err, "client.getOssClient")
	}
	out, err := toscli.ListBuckets(context.Background(), &tos.ListBucketsInput{})
	if err != nil {
		return errors.Wrap(err, "tos.ListBuckets")
	}

	ret := make([]cloudprovider.ICloudBucket, 0)
	for _, bucket := range out.Buckets {
		regionId := bucket.Location
		region, err := client.GetIRegionById(regionId)
		if err != nil {
			log.Errorf("cannot find bucket's region %s", regionId)
			continue
		}
		t, err := time.Parse(time.RFC3339, bucket.CreationDate)
		if err != nil {
			return errors.Wrapf(err, "Prase CreationDate error")
		}
		b := SBucket{
			region:       region.(*SRegion),
			Name:         bucket.Name,
			Location:     bucket.Location,
			CreationDate: t,
		}
		ret = append(ret, &b)
	}
	client.iBuckets = ret
	return nil
}

func getTOSExternalDomain(regionId string) string {
	return "tos-cn-beijing.volces.com"
}

func getTOSInternalDomain(regionId string) string {
	return "tos-cn-beijing.ivolces.com"
}

func (region *SVolcEngineClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
	}
	return caps
}

func (client *SVolcEngineClient) GetAccessEnv() string {
	return api.CLOUD_ACCESS_ENV_VOLCENGINE_CHINA
}

type SBalance struct {
	AccountId        string
	ArrearsBalance   float64
	AvailableBalance float64
	CashBalance      float64
	CreditLimit      float64
	FreezeAmount     float64
}

func (client *SVolcEngineClient) QueryBalance() (*SBalance, error) {
	resp, err := client.billRequest("", "QueryBalanceAcct", nil)
	if err != nil {
		return nil, err
	}
	ret := &SBalance{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}
