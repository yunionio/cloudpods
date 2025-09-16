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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/huawei/obs"
)

const (
	CLOUD_PROVIDER_HUAWEI    = api.CLOUD_PROVIDER_HUAWEI
	CLOUD_PROVIDER_HUAWEI_CN = "华为云"
	CLOUD_PROVIDER_HUAWEI_EN = "Huawei"

	HUAWEI_DEFAULT_REGION = "cn-north-4"
	HUAWEI_API_VERSION    = "2018-12-25"

	SERVICE_IAM                = "iam"
	SERVICE_IAM_V3             = "iam_v3"
	SERVICE_IAM_V3_EXT         = "iam_v3_ext"
	SERVICE_ELB                = "elb"
	SERVICE_VPC                = "vpc"
	SERVICE_VPC_V2_0           = "vpc_v2.0"
	SERVICE_VPC_V3             = "vpc_v3"
	SERVICE_VPN                = "vpn"
	SERVICE_CES                = "ces"
	SERVICE_RDS                = "rds"
	SERVICE_ECS                = "ecs"
	SERVICE_ECS_V1_1           = "ecs_v1.1"
	SERVICE_ECS_V2_1           = "ecs_v2.1"
	SERVICE_EPS                = "eps"
	SERVICE_EVS                = "evs"
	SERVICE_EVS_V1             = "evs_v1"
	SERVICE_EVS_V2_1           = "evs_v2.1"
	SERVICE_BSS                = "bss"
	SERVICE_BSS_INTL           = "bss-intl"
	SERVICE_SFS                = "sfs-turbo"
	SERVICE_CTS                = "cts"
	SERVICE_NAT                = "nat"
	SERVICE_NAT_V2             = "nat_v2"
	SERVICE_BMS                = "bms"
	SERVICE_CCI                = "cci"
	SERVICE_CSBS               = "csbs"
	SERVICE_IMS                = "ims"
	SERVICE_IMS_V1             = "ims_v1"
	SERVICE_AS                 = "as"
	SERVICE_CCE                = "cce"
	SERVICE_DCS                = "dcs"
	SERVICE_MODELARTS          = "modelarts"
	SERVICE_MODELARTS_V1       = "modelarts_v1"
	SERVICE_SCM                = "scm"
	SERVICE_CDN                = "cdn"
	SERVICE_GAUSSDB            = "gaussdb"
	SERVICE_GAUSSDB_V3_1       = "gaussdb_v3.1"
	SERVICE_GAUSSDB_NOSQL      = "gaussdb-nosql"
	SERVICE_GAUSSDB_NOSQL_V3_1 = "gaussdb-nosql_v3.1"
	SERVICE_GAUSSDB_OPENGAUSS  = "gaussdb-opengauss"
	SERVICE_FUNCTIONGRAPH      = "functiongraph"
	SERVICE_APIG               = "apig"
	SERVICE_APIG_V1_0          = "apig_v1.0"
	SERVICE_MRS                = "mrs"
	SERVICE_DIS                = "dis"
	SERVICE_LTS                = "lts"
)

type HuaweiClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	accessKey    string
	accessSecret string

	debug bool
}

func NewHuaweiClientConfig(accessKey, accessSecret string) *HuaweiClientConfig {
	cfg := &HuaweiClientConfig{
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *HuaweiClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *HuaweiClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *HuaweiClientConfig) Debug(debug bool) *HuaweiClientConfig {
	cfg.debug = debug
	return cfg
}

type SHuaweiClient struct {
	*HuaweiClientConfig

	userId          string
	ownerId         string
	ownerName       string
	ownerCreateTime time.Time

	iBuckets []cloudprovider.ICloudBucket

	projects map[string]SProject
	regions  map[string]SRegion

	httpClient *http.Client

	orders map[string]SOrderResource
}

func NewHuaweiClient(cfg *HuaweiClientConfig) (*SHuaweiClient, error) {
	client := SHuaweiClient{
		HuaweiClientConfig: cfg,
		regions:            map[string]SRegion{},
		projects:           map[string]SProject{},
	}
	err := client.init()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (self *SHuaweiClient) init() error {
	_, err := self.getRegions()
	if err != nil {
		return errors.Wrapf(err, "GetRegions")
	}
	_, err = self.GetProjects()
	if err != nil {
		return errors.Wrapf(err, "GetProjects")
	}
	_, err = self.GetOwnerId()
	return err
}

func (self *SHuaweiClient) getDefaultClient() *http.Client {
	if self.httpClient != nil {
		return self.httpClient
	}
	self.httpClient = self.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := self.httpClient.Transport.(*http.Transport)
	self.httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		service, method, path := strings.Split(req.URL.Host, ".")[0], req.Method, req.URL.Path
		respCheck := func(resp *http.Response) error {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
				}
			}
			return nil
		}
		if self.cpcfg.ReadOnly {
			// get or metric skip read only check
			if req.Method == "GET" || strings.HasPrefix(req.URL.Host, "ces") {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})
	return self.httpClient
}

type sPageInfo struct {
	NextMarker string
}

type akClient struct {
	client *http.Client
	aksk   Signer
}

func (self *akClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Del("Accept")
	length := req.Header.Get("Content-Length")
	if length == "0" {
		req.Header.Del("Content-Length")
	}
	if strings.HasPrefix(req.Host, "modelarts") && req.Method == string(httputils.PATCH) {
		req.Header.Set("Content-Type", "application/merge-patch+json")
	}
	self.aksk.Sign(req)
	return self.client.Do(req)
}

func (self *SHuaweiClient) getAkClient() *akClient {
	return &akClient{
		client: self.getDefaultClient(),
		aksk: Signer{
			Key:    self.accessKey,
			Secret: self.accessSecret,
		},
	}
}

type sHuaweiError struct {
	RequestId string `json:"request_id"`
	ErrorMsg  string `json:"error_msg"`
	ErrorCode string `json:"error_code"`
	Code      string
	Message   string
	ErrorInfo struct {
		Message string
		Code    string
		Title   string
	} `json:"error"`
	Errorcode  []string `json:"errorcode"`
	ConvertMsg string
	RawMsg     jsonutils.JSONObject
}

func (self *sHuaweiError) Error() string {
	return jsonutils.Marshal(self).String()
}

// https://support.huaweicloud.com/api-iam/iam_02_0006.html
var convertMsg = map[string]string{
	"1101": "用户名校验失败,请检查用户名",
	"1103": "密码校验失败,请检查密码",
	"1104": "手机号校验失败,请检查手机号",
	"1108": "新密码不能与原密码相同,请修改新密码",
	"1109": "用户名已存在,请修改用户名",
	"1118": "密码是弱密码,重新选择密码",
}

func (self *sHuaweiError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
		// 特殊错误将返回原始错误信息
		if self.Error() == jsonutils.Marshal(sHuaweiError{}).String() {
			self.RawMsg = body
		}
	}
	for _, code := range self.Errorcode {
		self.ConvertMsg = convertMsg[code]
	}
	if statusCode == 404 {
		return errors.Wrapf(cloudprovider.ErrNotFound, self.Error())
	}
	return self
}

func (self *SHuaweiClient) request(method httputils.THttpMethod, regionId, service, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}

	if project, ok := self.projects[regionId]; ok {
		if len(project.Id) > 0 && service != SERVICE_EPS {
			header.Set("X-Project-Id", project.Id)
		}
	}
	if ((strings.Contains(url, "/OS-CREDENTIAL/") ||
		strings.Contains(url, "/users") ||
		strings.Contains(url, "/roles") ||
		strings.Contains(url, "/mappings") ||
		strings.Contains(url, "/identity_providers") ||
		strings.Contains(url, "/groups") && (utils.IsInStringArray(service, []string{SERVICE_IAM, SERVICE_IAM_V3, SERVICE_IAM_V3_EXT}))) ||
		service == SERVICE_EPS) && len(self.ownerId) > 0 {
		header.Set("X-Domain-Id", self.ownerId)
	}
	req := httputils.NewJsonRequest(method, url, body)
	req.SetHeader(header)
	hwErr := &sHuaweiError{}
	cli := httputils.NewJsonClient(client)
	_, resp, err := cli.Send(context.Background(), req, hwErr, self.debug)
	if err != nil {
		return nil, err
	}
	if gotypes.IsNil(resp) {
		return jsonutils.NewDict(), nil
	}
	return resp, nil
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=KeystoneListRegions
func (self *SHuaweiClient) getRegions() ([]SRegion, error) {
	if len(self.regions) > 0 {
		ret := []SRegion{}
		for _, region := range self.regions {
			ret = append(ret, region)
		}
		return ret, nil
	}
	resp, err := self.list(SERVICE_IAM_V3, "", "regions", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "list regions")
	}

	self.regions = map[string]SRegion{}
	regions := make([]SRegion, 0)
	err = resp.Unmarshal(&regions, "regions")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	for _, region := range regions {
		region.client = self
		self.regions[region.Id] = region
	}
	return regions, nil
}

func (self *SHuaweiClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SHuaweiClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func getOBSEndpoint(regionId string) string {
	return fmt.Sprintf("obs.%s.myhuaweicloud.com", regionId)
}

func (self *SHuaweiClient) getOBSClient(regionId string, signType obs.SignatureType) (*obs.ObsClient, error) {
	endpoint := getOBSEndpoint(regionId)
	cli, err := obs.New(self.accessKey, self.accessSecret, endpoint, obs.WithSignature(signType))
	if err != nil {
		return nil, err
	}
	client := cli.GetClient()
	ts, _ := client.Transport.(*http.Transport)
	client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		method, path := req.Method, req.URL.Path
		respCheck := func(resp *http.Response) error {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission("obs", fmt.Sprintf("%s %s", method, path))
				}
			}
			return nil
		}
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})

	return cli, nil
}

func (self *SHuaweiClient) fetchBuckets() error {
	obscli, err := self.getOBSClient(HUAWEI_DEFAULT_REGION, "")
	if err != nil {
		return errors.Wrap(err, "getOBSClient")
	}
	input := &obs.ListBucketsInput{QueryLocation: true}
	output, err := obscli.ListBuckets(input)
	if err != nil {
		return errors.Wrap(err, "obscli.ListBuckets")
	}
	self.ownerId = output.Owner.ID

	ret := make([]cloudprovider.ICloudBucket, 0)
	for i := range output.Buckets {
		bInfo := output.Buckets[i]
		region := self.GetRegion(bInfo.Location)
		if gotypes.IsNil(region) {
			log.Errorf("fail to find region %s", bInfo.Location)
			continue
		}
		b := SBucket{
			region: region,

			Name:         bInfo.Name,
			Location:     bInfo.Location,
			CreationDate: bInfo.CreationDate,
		}
		ret = append(ret, &b)
	}
	self.iBuckets = ret
	return nil
}

func (self *SHuaweiClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Id = self.GetAccountId()
	subAccount.Name = self.cpcfg.Name
	subAccount.Account = self.accessKey
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	subAccount.DefaultProjectId = "0"
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (client *SHuaweiClient) GetAccountId() string {
	ownerId, _ := client.GetOwnerId()
	return ownerId
}

func (client *SHuaweiClient) GetIamLoginUrl() string {
	return fmt.Sprintf("https://auth.huaweicloud.com/authui/login.html?account=%s#/login", client.ownerName)
}

func (self *SHuaweiClient) GetRegions() []SRegion {
	ret := []SRegion{}
	for id := range self.regions {
		if _, ok := self.projects[id]; !ok {
			continue
		}
		region := self.regions[id]
		region.client = self
		ret = append(ret, region)
	}
	for id := range self.projects {
		project := self.projects[id]
		if strings.Contains(project.Name, "_") {
			regionId := project.GetRegionId()
			if region, ok := self.regions[regionId]; ok {
				region.Id = project.Name
				region.client = self
				ret = append(ret, region)
			}
		}
	}
	return ret
}

func (self *SHuaweiClient) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions := self.GetRegions()
	ret := []cloudprovider.ICloudRegion{}
	for i := range regions {
		ret = append(ret, &regions[i])
	}
	return ret, nil
}

func (self *SHuaweiClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	regions, err := self.GetIRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].GetId() == id || regions[i].GetGlobalId() == id {
			return regions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SHuaweiClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = HUAWEI_DEFAULT_REGION
	}
	regions := self.GetRegions()
	for i := range regions {
		if regions[i].Id == regionId {
			return &regions[i]
		}
	}
	return nil
}

type SBalance struct {
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	AccountId        string  `json:"account_id"`
	AccountType      int64   `json:"account_type"`
	DesignatedAmount float64 `json:"designated_amount,omitempty"`
	CreditAmount     float64 `json:"credit_amount,omitempty"`
	MeasureUnit      int64   `json:"measure_unit"`
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/BSS/doc?api=ShowCustomerAccountBalances
func (self *SHuaweiClient) QueryAccountBalance() (*SBalance, error) {
	ret := struct {
		AccountBalances []SBalance
		Currency        string
	}{}
	for _, service := range []string{SERVICE_BSS, SERVICE_BSS_INTL} {
		resp, err := self.list(service, "", "accounts/customer-accounts/balances", nil)
		if err != nil {
			// 国际区账号会报错: {"error_code":"CBC.0150","error_msg":"Access denied. The customer does not belong to the website you are now at."}
			if e, ok := err.(*sHuaweiError); ok && (e.ErrorCode == "CBC.0150" || e.ErrorCode == "CBC.0156") {
				continue
			}
			return nil, err
		}
		err = resp.Unmarshal(&ret)
		if err != nil {
			return nil, err
		}
		break
	}
	for i := range ret.AccountBalances {
		if ret.AccountBalances[i].AccountType == 1 {
			return &ret.AccountBalances[i], nil
		}
	}
	return &SBalance{Currency: ret.Currency}, nil
}

func (self *SHuaweiClient) GetISSLCertificates() ([]cloudprovider.ICloudSSLCertificate, error) {
	ret, err := self.GetSSLCertificates()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSSLCertificates")
	}

	result := make([]cloudprovider.ICloudSSLCertificate, 0)
	for i := range ret {
		ret[i].client = self
		result = append(result, &ret[i])
	}
	return result, nil
}

func (self *SHuaweiClient) GetISSLCertificate(certId string) (cloudprovider.ICloudSSLCertificate, error) {
	var res cloudprovider.ICloudSSLCertificate
	res, err := self.GetSSLCertificate(certId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSSLCertificate")
	}
	return res, nil
}

func (self *SHuaweiClient) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHuaweiClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_NAT,
		cloudprovider.CLOUD_CAPABILITY_NAS,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_MODELARTES,
		cloudprovider.CLOUD_CAPABILITY_VPC_PEER,
		cloudprovider.CLOUD_CAPABILITY_CERT,
		cloudprovider.CLOUD_CAPABILITY_CDN + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=ShowPermanentAccessKey
func (self *SHuaweiClient) GetUserId() (string, error) {
	if len(self.userId) > 0 {
		return self.userId, nil
	}

	type cred struct {
		UserId string `json:"user_id"`
	}

	ret := &cred{}
	resp, err := self.list(SERVICE_IAM, "", "OS-CREDENTIAL/credentials/"+self.accessKey, nil)
	if err != nil {
		return "", errors.Wrapf(err, "show credential")
	}
	err = resp.Unmarshal(ret, "credential")
	if err != nil {
		return "", errors.Wrapf(err, "Unmarshal")
	}
	self.userId = ret.UserId
	return self.userId, nil
}

// owner id == domain_id == account id
// https://console.huaweicloud.com/apiexplorer/#/openapi/IAM/doc?api=ShowUser
func (self *SHuaweiClient) GetOwnerId() (string, error) {
	if len(self.ownerId) > 0 {
		return self.ownerId, nil
	}

	userId, err := self.GetUserId()
	if err != nil {
		return "", errors.Wrap(err, "SHuaweiClient.GetOwnerId.GetUserId")
	}

	type user struct {
		DomainId   string `json:"domain_id"`
		Name       string `json:"name"`
		CreateTime string
	}
	resp, err := self.list(SERVICE_IAM, "", "OS-USER/users/"+userId, nil)
	if err != nil {
		return "", errors.Wrapf(err, "show user")
	}
	ret := &user{}
	err = resp.Unmarshal(ret, "user")
	if err != nil {
		return "", errors.Wrapf(err, "Unmarshal")
	}

	self.ownerName = ret.Name
	// 2021-02-02 02:43:28.0
	self.ownerCreateTime, _ = timeutils.ParseTimeStr(strings.TrimSuffix(ret.CreateTime, ".0"))
	self.ownerId = ret.DomainId
	return self.ownerId, nil
}

func (self *SHuaweiClient) list(service, regionId, resource string, query url.Values) (jsonutils.JSONObject, error) {
	url, err := self.getUrl(service, regionId, resource, httputils.GET, nil)
	if err != nil {
		return nil, err
	}
	return self.request(httputils.GET, regionId, service, url, query, nil)
}

func (self *SHuaweiClient) delete(service, regionId, resource string) (jsonutils.JSONObject, error) {
	url, err := self.getUrl(service, regionId, resource, httputils.DELETE, nil)
	if err != nil {
		return nil, err
	}
	return self.request(httputils.DELETE, regionId, service, url, nil, nil)
}

func (self *SHuaweiClient) getUrl(service, regionId, resource string, method httputils.THttpMethod, params map[string]interface{}) (string, error) {
	url := ""
	resource = strings.TrimPrefix(resource, "/")
	if len(regionId) == 0 {
		regionId = HUAWEI_DEFAULT_REGION
	}
	projectId := ""
	project, ok := self.projects[regionId]
	if ok {
		regionId = project.GetRegionId()
		projectId = project.Id
	}
	switch service {
	case SERVICE_IAM:
		url = fmt.Sprintf("https://iam.myhuaweicloud.com/v3.0/%s", resource)
	case SERVICE_IAM_V3:
		url = fmt.Sprintf("https://iam.myhuaweicloud.com/v3/%s", resource)
	case SERVICE_IAM_V3_EXT:
		url = fmt.Sprintf("https://iam.myhuaweicloud.com/v3-ext/%s", resource)
	case SERVICE_ELB:
		url = fmt.Sprintf("https://elb.%s.myhuaweicloud.com/v3/%s/%s", regionId, projectId, resource)
	case SERVICE_VPC:
		url = fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v1/%s/%s", regionId, projectId, resource)
	case SERVICE_VPC_V2_0:
		url = fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v2.0/%s/%s", regionId, projectId, resource)
		if strings.Contains(resource, "/peerings") {
			url = fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v2.0/%s", regionId, resource)
		}
	case SERVICE_VPC_V3:
		url = fmt.Sprintf("https://vpc.%s.myhuaweicloud.com/v3/%s/%s", regionId, projectId, resource)
	case SERVICE_VPN:
		url = fmt.Sprintf("https://vpn.%s.myhuaweicloud.com/v5/%s/%s", regionId, projectId, resource)
	case SERVICE_CES:
		url = fmt.Sprintf("https://ces.%s.myhuaweicloud.com/V1.0/%s/%s", regionId, projectId, resource)
	case SERVICE_MODELARTS:
		url = fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v2/%s/%s", regionId, projectId, resource)
	case SERVICE_MODELARTS_V1:
		url = fmt.Sprintf("https://modelarts.%s.myhuaweicloud.com/v1/%s/%s", regionId, projectId, resource)
	case SERVICE_RDS:
		url = fmt.Sprintf("https://rds.%s.myhuaweicloud.com/v3/%s/%s", regionId, projectId, resource)
	case SERVICE_ECS:
		url = fmt.Sprintf("https://ecs.%s.myhuaweicloud.com/v1/%s/%s", regionId, projectId, resource)
	case SERVICE_ECS_V1_1:
		url = fmt.Sprintf("https://ecs.%s.myhuaweicloud.com/v1.1/%s/%s", regionId, projectId, resource)
	case SERVICE_ECS_V2_1:
		url = fmt.Sprintf("https://ecs.%s.myhuaweicloud.com/v2.1/%s/%s", regionId, projectId, resource)
	case SERVICE_EPS:
		url = fmt.Sprintf("https://eps.myhuaweicloud.com/v1.0/%s", resource)
	case SERVICE_EVS_V1:
		url = fmt.Sprintf("https://evs.%s.myhuaweicloud.com/v1/%s/%s", regionId, projectId, resource)
	case SERVICE_EVS:
		url = fmt.Sprintf("https://evs.%s.myhuaweicloud.com/v2/%s/%s", regionId, projectId, resource)
	case SERVICE_EVS_V2_1:
		url = fmt.Sprintf("https://evs.%s.myhuaweicloud.com/v2.1/%s/%s", regionId, projectId, resource)
	case SERVICE_BSS, SERVICE_BSS_INTL:
		url = fmt.Sprintf("https://%s.myhuaweicloud.com/v2/%s", service, resource)
	case SERVICE_SFS:
		url = fmt.Sprintf("https://sfs-turbo.%s.myhuaweicloud.com/v1/%s/%s", regionId, projectId, resource)
	case SERVICE_IMS:
		url = fmt.Sprintf("https://ims.%s.myhuaweicloud.com/v2/%s", regionId, resource)
	case SERVICE_IMS_V1:
		url = fmt.Sprintf("https://ims.%s.myhuaweicloud.com/v1/%s", regionId, resource)
	case SERVICE_DCS:
		url = fmt.Sprintf("https://dcs.%s.myhuaweicloud.com/v2/%s/%s", regionId, projectId, resource)
	case SERVICE_CTS:
		url = fmt.Sprintf("https://cts.%s.myhuaweicloud.com/v3/%s/%s", regionId, projectId, resource)
	case SERVICE_NAT:
		url = fmt.Sprintf("https://nat.%s.myhuaweicloud.com/v3/%s/%s", regionId, projectId, resource)
	case SERVICE_NAT_V2:
		url = fmt.Sprintf("https://nat.%s.myhuaweicloud.com/v2/%s/%s", regionId, projectId, resource)
	case SERVICE_SCM:
		url = fmt.Sprintf("https://scm.cn-north-4.myhuaweicloud.com/v3/%s", resource)
	case SERVICE_CDN:
		url = fmt.Sprintf("https://cdn.myhuaweicloud.com/v1.0/%s", resource)
	case SERVICE_GAUSSDB, SERVICE_GAUSSDB_NOSQL:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v3/%s/%s", service, regionId, projectId, resource)
	case SERVICE_GAUSSDB_V3_1:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v3.1/%s/%s", SERVICE_GAUSSDB, regionId, projectId, resource)
	case SERVICE_GAUSSDB_NOSQL_V3_1:
		url = fmt.Sprintf("https://gaussdb-nosql.%s.myhuaweicloud.com/v3.1/%s/%s", regionId, projectId, resource)
	case SERVICE_GAUSSDB_OPENGAUSS:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v3.1/%s/%s", service, regionId, projectId, resource)
	case SERVICE_FUNCTIONGRAPH:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v2/%s/%s", service, regionId, projectId, resource)
	case SERVICE_APIG:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v2/%s/%s", service, regionId, projectId, resource)
	case SERVICE_APIG_V1_0:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v1.0/%s", SERVICE_APIG, regionId, resource)
	case SERVICE_MRS:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v1.1/%s/%s", service, regionId, projectId, resource)
	case SERVICE_DIS:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v2/%s/%s", service, regionId, projectId, resource)
	case SERVICE_LTS:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/v2/%s/%s", service, regionId, projectId, resource)
	case SERVICE_CCE:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/api/v3/projects/%s/%s", service, regionId, projectId, resource)
	case SERVICE_AS:
		url = fmt.Sprintf("https://%s.%s.myhuaweicloud.com/autoscaling-api/v1/%s/%s", service, regionId, projectId, resource)
	default:
		return "", fmt.Errorf("invalid service %s", service)
	}
	return url, nil
}

func (self *SHuaweiClient) post(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	url, err := self.getUrl(service, regionId, resource, httputils.POST, params)
	if err != nil {
		return nil, err
	}
	return self.request(httputils.POST, regionId, service, url, nil, params)
}

func (self *SHuaweiClient) patch(service, regionId, resource string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	url, err := self.getUrl(service, regionId, resource, httputils.PATCH, params)
	if err != nil {
		return nil, err
	}
	return self.request(httputils.PATCH, regionId, service, url, query, params)
}

func (self *SHuaweiClient) put(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	url, err := self.getUrl(service, regionId, resource, httputils.PUT, params)
	if err != nil {
		return nil, err
	}
	return self.request(httputils.PUT, regionId, service, url, nil, params)
}
