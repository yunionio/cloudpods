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

package hcs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/auth"
)

const (
	VERSION_SPEC = "x-version"

	CLOUD_PROVIDER_HCS    = api.CLOUD_PROVIDER_HCS
	CLOUD_PROVIDER_HCS_CN = "华为云Stack"
	CLOUD_PROVIDER_HCS_EN = "HCS"

	HCS_API_VERSION = ""
)

type HcsConfig struct {
	cpcfg   cloudprovider.ProviderConfig
	authUrl string

	projectId    string // 华为云项目ID.
	accessKey    string
	accessSecret string

	account  string
	password string

	token string

	debug bool
}

func (self *HcsConfig) isAccountValid() bool {
	return len(self.account) > 0 && len(self.password) > 0
}

func NewHcsConfig(accessKey, accessSecret, projectId, url string) *HcsConfig {
	cfg := &HcsConfig{
		projectId:    projectId,
		accessKey:    accessKey,
		accessSecret: accessSecret,
		authUrl:      url,
	}

	return cfg
}

func (cfg *HcsConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *HcsConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *HcsConfig) Debug(debug bool) *HcsConfig {
	cfg.debug = debug
	return cfg
}

func (cfg *HcsConfig) WithAccount(account, password string) *HcsConfig {
	cfg.account = account
	cfg.password = password
	return cfg
}

type SHcsClient struct {
	*HcsConfig

	signer auth.Signer

	isMainProject bool // whether the project is the main project in the region

	domainId    string
	projectName string

	regions  []SRegion
	projects []SProject
	buckets  []SBucket

	defaultRegion string

	httpClient *http.Client
	token      string
	lock       sThrottlingThreshold
}

func NewHcsClient(cfg *HcsConfig) (*SHcsClient, error) {
	client := SHcsClient{
		HcsConfig: cfg,
		lock:      sThrottlingThreshold{locked: false, lockTime: time.Time{}},
	}
	if len(client.projectId) > 0 {
		project, err := client.GetProjectById(client.projectId)
		if err != nil {
			return nil, err
		}
		client.domainId = project.DomainId
		client.projectName = project.Name
	}
	return &client, client.fetchRegions()
}

func (self *SHcsClient) GetAccountId() string {
	return self.authUrl
}

func (self *SHcsClient) GetRegion(id string) (*SRegion, error) {
	for i := range self.regions {
		if self.regions[i].GetGlobalId() == id || self.regions[i].GetId() == id {
			self.regions[i].client = self
			return &self.regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHcsClient) GetCloudRegionExternalIdPrefix() string {
	if len(self.projectId) > 0 {
		if iregions := self.GetIRegions(); len(iregions) > 0 {
			return iregions[0].GetGlobalId()
		}
	}
	return CLOUD_PROVIDER_HCS
}

type akClient struct {
	client *http.Client
	signer *Signer
}

func (self *SHcsClient) getDefaultClient() *http.Client {
	if self.httpClient != nil {
		return self.httpClient
	}
	self.httpClient = self.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := self.httpClient.Transport.(*http.Transport)
	self.httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response), error) {
		service, method, path := strings.Split(req.URL.Host, ".")[0], req.Method, req.URL.Path
		respCheck := func(resp *http.Response) {
			if resp.StatusCode == 403 {
				if self.cpcfg.UpdatePermission != nil {
					self.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
				}
			}
		}
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})
	return self.httpClient
}

func (self *akClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Del("Accept")
	if req.Method == string(httputils.GET) || req.Method == string(httputils.DELETE) || req.Method == string(httputils.PATCH) {
		req.Header.Del("Content-Length")
	}
	err := self.signer.Sign(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Sign")
	}
	return self.client.Do(req)
}

func (self *SHcsClient) getAkClient() *akClient {
	return &akClient{
		client: self.getDefaultClient(),
		signer: &Signer{
			Key:    self.accessKey,
			Secret: self.accessSecret,
		},
	}
}

type hcsError struct {
	Url      string
	Params   map[string]interface{}
	Code     int    `json:"code,omitzero"`
	Class    string `json:"class,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Details  string `json:"details,omitempty"`
}

func (ce *hcsError) Error() string {
	return jsonutils.Marshal(ce).String()
}

func (ce *hcsError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	body.Unmarshal(ce)
	if ce.Code == 0 {
		ce.Code = statusCode
	}
	if len(ce.Class) == 0 {
		ce.Class = http.StatusText(statusCode)
	}
	if len(ce.Details) == 0 {
		ce.Details = body.String()
	}
	return ce
}

type sThrottlingThreshold struct {
	locked   bool
	lockTime time.Time
}

func (t *sThrottlingThreshold) CheckingLock() {
	if !t.locked {
		return
	}

	for {
		if t.lockTime.Sub(time.Now()).Seconds() < 0 {
			return
		}
		log.Debugf("throttling threshold has been reached. release at %s", t.lockTime)
		time.Sleep(5 * time.Second)
	}
}

func (t *sThrottlingThreshold) Lock() {
	// 锁定至少15秒
	t.locked = true
	t.lockTime = time.Now().Add(15 * time.Second)
}

func (self *SHcsClient) request(method httputils.THttpMethod, url string, query url.Values, params map[string]interface{}) (jsonutils.JSONObject, error) {
	self.lock.CheckingLock()
	client := self.getAkClient()
	if len(query) > 0 {
		url = fmt.Sprintf("%s?%s", url, query.Encode())
	}
	url = strings.TrimPrefix(url, "http://")
	if !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("https://%s", url)
	}
	var body jsonutils.JSONObject = nil
	if len(params) > 0 {
		body = jsonutils.Marshal(params)
	}
	header := http.Header{}
	if len(self.projectId) > 0 && !strings.Contains(url, "v3/regions") {
		header.Set("X-Project-Id", self.projectId)
	}
	if len(self.domainId) > 0 && strings.Contains(url, "/peering") {
		header.Set("X-Domain-Id", self.domainId)
	}
	cli := httputils.NewJsonClient(client)
	req := httputils.NewJsonRequest(method, url, body)
	req.SetHeader(header)
	var resp jsonutils.JSONObject
	var err error
	for i := 0; i < 4; i++ {
		_, resp, err = cli.Send(context.Background(), req, &hcsError{Url: url, Params: params}, self.debug)
		if err == nil {
			break
		}
		if err != nil {
			e, ok := err.(*hcsError)
			if ok {
				if e.Code == 404 {
					return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
				}
				if e.Code == 429 {
					log.Errorf("request %s %v try later", url, err)
					self.lock.Lock()
					time.Sleep(time.Second * 15)
					continue
				}
			}
			return nil, err
		}
	}
	return resp, err
}

func (self *SHcsClient) iamGet(resource string, query url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("iam-apigateway-proxy.%s/%s", self.authUrl, resource)
	return self.request(httputils.GET, url, query, nil)
}

func (self *SHcsClient) fetchRegions() error {
	if len(self.regions) > 0 {
		return nil
	}
	resp, err := self.iamGet("v3/regions", nil)
	if err != nil {
		return err
	}
	self.regions = []SRegion{}
	err = resp.Unmarshal(&self.regions, "regions")
	if err != nil {
		return err
	}
	for i := range self.regions {
		self.defaultRegion = self.regions[i].Id
		if self.projectName == self.regions[i].Id {
			self.isMainProject = true
		}
	}
	return nil
}

func (hcscli *SHcsClient) rdsList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return hcscli.list("rds", "v3", regionId, resource, query, retVal)
}

func (hcscli *SHcsClient) rdsGet(regionId string, resource string, retVal interface{}) error {
	return hcscli.get("rds", "v3", regionId, resource, retVal)
}

func (hcscli *SHcsClient) rdsDelete(regionId string, resource string) error {
	return hcscli._delete("rds", "v3", regionId, resource)
}

func (hcscli *SHcsClient) rdsCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return hcscli._create("rds", "v3", regionId, resource, body, retVal)
}

func (self *SHcsClient) rdsPerform(regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self._perform("rds", "v3", regionId, resource, action, params, retVal)
}

func (hcscli *SHcsClient) rdsJobGet(regionId string, resource string, query url.Values, retVal interface{}) error {
	url := hcscli._url("rds", "v3", regionId, resource)
	resp, err := hcscli.request(httputils.GET, url, query, nil)
	if err != nil {
		return err
	}
	err = resp.Unmarshal(retVal)
	return err
}

func (hcscli *SHcsClient) rdsDBPrivvilegesDelete(regionId string, resource string, params map[string]interface{}) error {
	return hcscli._deleteWithBody("rds", "v3", regionId, resource, params)
}

func (hcscli *SHcsClient) rdsDBPrivilegesGrant(regionId string, resource string, params map[string]interface{}, retVal interface{}) error {
	url := hcscli._url("rds", "v3", regionId, resource)
	resp, err := hcscli.request(httputils.GET, url, nil, params)
	if err != nil {
		return err
	}
	err = resp.Unmarshal(retVal)
	return err
}

func (self *SHcsClient) ecsList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return self.list("ecs", "v2", regionId, resource, query, retVal)
}

func (self *SHcsClient) list(product, version, regionId string, resource string, query url.Values, retVal interface{}) error {
	resp, err := self._list(product, version, regionId, resource, query)
	if err != nil {
		return errors.Wrapf(err, "_list")
	}
	return resp.Unmarshal(retVal)
}

func (self *SHcsClient) ecsGet(regionId string, resource string, retVal interface{}) error {
	return self.get("ecs", "v2", regionId, resource, retVal)
}

func (self *SHcsClient) ecsDelete(regionId string, resource string) error {
	return self._delete("ecs", "v2", regionId, resource)
}

func (self *SHcsClient) ecsPerform(regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self._perform("ecs", "v2", regionId, resource, action, params, retVal)
}

func (self *SHcsClient) evsDelete(regionId string, resource string) error {
	return self._delete("evs", "v2", regionId, resource)
}

func (self *SHcsClient) evsList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return self.list("evs", "v2", regionId, resource, query, retVal)
}

func (self *SHcsClient) evsGet(regionId string, resource string, retVal interface{}) error {
	return self.get("evs", "v2", regionId, resource, retVal)
}

func (self *SHcsClient) evsPerform(regionId string, resource, action string, params map[string]interface{}) error {
	return self._perform("evs", "v2", regionId, resource, action, params, nil)
}

func (self *SHcsClient) ecsCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create("ecs", "v2", regionId, resource, body, retVal)
}

func (self *SHcsClient) evsCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create("evs", "v2", regionId, resource, body, retVal)
}

func (self *SHcsClient) vpcCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create("vpc", "v1", regionId, resource, body, retVal)
}

func (self *SHcsClient) vpcDelete(regionId string, resource string) error {
	return self._delete("vpc", "v1", regionId, resource)
}

func (self *SHcsClient) vpcGet(regionId string, resource string, retVal interface{}) error {
	return self.get("vpc", "v1", regionId, resource, retVal)
}

func (self *SHcsClient) vpcList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return self.list("vpc", "v1", regionId, resource, query, retVal)
}

func (self *SHcsClient) vpcUpdate(regionId string, resource string, body map[string]interface{}) error {
	return self._update("vpc", "v1", regionId, resource, body)
}

func (self *SHcsClient) imsCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create("ims", "v2", regionId, resource, body, retVal)
}

func (self *SHcsClient) imsDelete(regionId string, resource string) error {
	return self._delete("ims", "v2", regionId, resource)
}

func (self *SHcsClient) imsGet(regionId string, resource string, retVal interface{}) error {
	return self.get("ims", "v2", regionId, resource, retVal)
}

func (self *SHcsClient) imsList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return self.list("ims", "v2", regionId, resource, query, retVal)
}

func (self *SHcsClient) imsPerform(regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self._perform("ims", "v2", regionId, resource, action, params, retVal)
}

func (self *SHcsClient) imsUpdate(regionId string, resource string, body map[string]interface{}) error {
	return self._update("ims", "v2", regionId, resource, body)
}

func (self *SHcsClient) dcsCreate(regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create("dcs", "v1.0", regionId, resource, body, retVal)
}

func (self *SHcsClient) dcsDelete(regionId string, resource string) error {
	return self._delete("dcs", "v1.0", regionId, resource)
}

func (self *SHcsClient) dcsGet(regionId string, resource string, retVal interface{}) error {
	return self.get("dcs", "v1.0", regionId, resource, retVal)
}

func (self *SHcsClient) dcsList(regionId string, resource string, query url.Values, retVal interface{}) error {
	return self.list("dcs", "v1.0", regionId, resource, query, retVal)
}

func (self *SHcsClient) dcsPerform(regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self._perform("dcs", "v1.0", regionId, resource, action, params, retVal)
}

func (self *SHcsClient) dcsUpdate(regionId string, resource string, body map[string]interface{}) error {
	return self._update("dcs", "v1.0", regionId, resource, body)
}

func (self *SHcsClient) _url(product, version, regionId string, resource string) string {
	url := fmt.Sprintf("%s.%s.%s/%s/%s/%s", product, regionId, self.authUrl, version, self.projectId, resource)
	for _, prefix := range []string{
		"images", "cloudimages", "nat_gateways",
		"lbaas", "products", "snat_rules",
		"dnat_rules", "networks",
		"ports",
	} {
		if strings.HasPrefix(resource, prefix) {
			url = fmt.Sprintf("%s.%s.%s/%s/%s", product, regionId, self.authUrl, version, resource)
			break
		}
	}
	if version == "v2.0" && strings.HasPrefix(resource, "subnets") {
		url = fmt.Sprintf("%s.%s.%s/%s/%s", product, regionId, self.authUrl, version, resource)
	}
	return url
}

func (self *SHcsClient) _list(product, version, regionId string, resource string, query url.Values) (*jsonutils.JSONArray, error) {
	ret := jsonutils.NewArray()
	url := self._url(product, version, regionId, resource)
	offset, _ := strconv.Atoi(query.Get("offset"))
	total := int64(0)
	for {
		resp, err := self.request(httputils.GET, url, query, nil)
		if err != nil {
			return nil, err
		}
		if gotypes.IsNil(resp) {
			log.Warningf("%s return empty", resource)
			return ret, nil
		}
		objMap, err := resp.GetMap()
		if err != nil {
			return nil, errors.Wrapf(err, "resp.GetMap")
		}
		next := false
		for k, v := range objMap {
			if k == "links" {
				url, _ = v.GetString("next")
				next = true
				continue
			}
			if k == "count" {
				offset++
				query.Set("offset", fmt.Sprintf("%d", offset))
				total, _ = v.Int()
				next = true
				continue
			}
			if strings.Contains(resource, k) ||
				strings.Contains(strings.ReplaceAll(resource, "-", "_"), k) ||
				utils.IsInStringArray(k, []string{
					"availabilityZoneInfo",
					"vpc_peering_connections",
				}) {
				objs, err := v.GetArray()
				if err != nil {
					return nil, errors.Wrapf(err, "v.GetArray")
				}
				ret.Add(objs...)
			}
		}
		if !next || len(url) == 0 || (ret.Length() == int(total)) {
			break
		}
	}
	return ret, nil
}

func (self *SHcsClient) get(product, version, regionId string, resource string, retVal interface{}) error {
	resp, err := self._get(product, version, regionId, resource)
	if err != nil {
		return err
	}
	obj, err := resp.GetMap()
	if err != nil {
		return errors.Wrapf(err, "GetMap")
	}
	for v := range obj {
		if len(obj) == 1 {
			return obj[v].Unmarshal(retVal)
		}
	}
	return resp.Unmarshal(retVal)
}

func (self *SHcsClient) _getJob(product, regionId string, jobId string) (jsonutils.JSONObject, error) {
	if product == "ims" { // 保存镜像时，使用ecs查询job
		product = "ecs"
	}
	url := self._url(product, "v1", regionId, fmt.Sprintf("jobs/%s", jobId))
	resp, err := self.request(httputils.GET, url, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (self *SHcsClient) _get(product, version, regionId string, resource string) (jsonutils.JSONObject, error) {
	url := self._url(product, version, regionId, resource)
	resp, err := self.request(httputils.GET, url, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (self *SHcsClient) delete(product, version, regionId string, resource string) error {
	return self._delete(product, version, regionId, resource)
}

func (self *SHcsClient) _delete(product, version, regionId string, resource string) error {
	url := self._url(product, version, regionId, resource)
	resp, err := self.request(httputils.DELETE, url, nil, nil)
	if !gotypes.IsNil(resp) && resp.Contains("job_id") {
		jobId, _ := resp.GetString("job_id")
		_, err := self.waitJobSuccess(product, regionId, jobId, time.Second*10, time.Hour*2)
		if err != nil {
			return errors.Wrapf(err, "wait create %s %s job", product, resource)
		}
		return nil
	}
	return err
}

func (self *SHcsClient) _deleteWithBody(product, version, regionId string, resource string, params map[string]interface{}) error {
	url := self._url(product, version, regionId, resource)
	_, err := self.request(httputils.DELETE, url, nil, params)
	return err
}

func (self *SHcsClient) update(product, version, regionId string, resource string, params map[string]interface{}) error {
	return self._update(product, version, regionId, resource, params)
}

func (self *SHcsClient) perform(product, version, regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	return self._perform(product, version, regionId, resource, action, params, retVal)
}

func (self *SHcsClient) _perform(product, version, regionId string, resource, action string, params map[string]interface{}, retVal interface{}) error {
	url := self._url(product, version, regionId, resource+"/"+action)
	resp, err := self.request(httputils.POST, url, nil, params)
	if err != nil {
		return err
	}
	if !gotypes.IsNil(resp) && resp.Contains("job_id") {
		jobId, _ := resp.GetString("job_id")
		job, err := self.waitJobSuccess(product, regionId, jobId, time.Second*10, time.Hour*2)
		if err != nil {
			return errors.Wrapf(err, "wait create %s %s job", product, resource)
		}
		if retVal != nil {
			return jsonutils.Update(retVal, jsonutils.Marshal(job))
		}
		return nil
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SHcsClient) create(product, version, regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	return self._create(product, version, regionId, resource, body, retVal)
}

func (self *SHcsClient) _create(product, version, regionId string, resource string, body map[string]interface{}, retVal interface{}) error {
	url := self._url(product, version, regionId, resource)
	resp, err := self.request(httputils.POST, url, nil, body)
	if err != nil {
		return err
	}
	if resp.Contains("job_id") {
		jobId, _ := resp.GetString("job_id")
		job, err := self.waitJobSuccess(product, regionId, jobId, time.Second*10, time.Hour*2)
		if err != nil {
			return errors.Wrapf(err, "wait create %s %s job", product, resource)
		}
		if retVal != nil {
			return jsonutils.Update(retVal, jsonutils.Marshal(job))
		}
		return nil
	}
	obj, err := resp.GetMap()
	if err != nil {
		return errors.Wrapf(err, "GetMap")
	}
	for v := range obj {
		if len(obj) == 1 && retVal != nil {
			return obj[v].Unmarshal(retVal)
		}
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SHcsClient) _update(product, version, regionId string, resource string, body map[string]interface{}) error {
	url := self._url(product, version, regionId, resource)
	_, err := self.request(httputils.PUT, url, nil, body)
	return err
}

func (self *SHcsClient) GetRegions() []SRegion {
	return self.regions
}

func (self *SHcsClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	if len(self.projectId) == 0 {
		for i := range self.regions {
			self.regions[i].client = self
			self.defaultRegion = self.regions[i].Id
			ret = append(ret, &self.regions[i])
		}
		return ret
	} else {
		for i := range self.regions {
			if strings.Contains(self.projectName, self.regions[i].Id) {
				self.defaultRegion = self.regions[i].Id
				self.regions[i].client = self
				ret = append(ret, &self.regions[i])
			}
			if self.projectName == self.regions[i].Id {
				self.isMainProject = true
			}
		}
	}
	return ret
}

func (self *SHcsClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_NAT,
		cloudprovider.CLOUD_CAPABILITY_VPC_PEER,
	}
	// huawei objectstore is shared across projects(subscriptions)
	// to avoid multiple project access the same bucket
	// only main project is allow to access objectstore bucket
	if self.isMainProject {
		caps = append(caps, cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE)
	}
	return caps
}

func (self *SHcsClient) getOBSEndpoint(regionId string) string {
	return fmt.Sprintf("obsv3.%s.%s", regionId, self.authUrl)
}

func (self *SHcsClient) getOBSClient(regionId string) (*obs.ObsClient, error) {
	endpoint := self.getOBSEndpoint(regionId)
	ts, _ := self.httpClient.Transport.(*http.Transport)
	conf := obs.WithHttpTransport(ts)
	cli, err := obs.New(self.accessKey, self.accessSecret, endpoint, conf)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func (self *SHcsClient) GetBuckets() ([]SBucket, error) {
	if len(self.buckets) > 0 {
		return self.buckets, nil
	}
	obscli, err := self.getOBSClient(self.regions[0].Id)
	if err != nil {
		return nil, errors.Wrap(err, "getOBSClient")
	}
	input := &obs.ListBucketsInput{QueryLocation: true}
	output, err := obscli.ListBuckets(input)
	if err != nil {
		return nil, errors.Wrap(err, "obscli.ListBuckets")
	}
	self.buckets = []SBucket{}
	for i := range output.Buckets {
		bInfo := output.Buckets[i]
		region, err := self.GetRegion(bInfo.Location)
		if err != nil {
			log.Errorf("fail to find region %s", bInfo.Location)
			continue
		}
		b := SBucket{
			region: region,

			Name:         bInfo.Name,
			Location:     bInfo.Location,
			CreationDate: bInfo.CreationDate,
		}
		self.buckets = append(self.buckets, b)
	}
	return self.buckets, nil
}
