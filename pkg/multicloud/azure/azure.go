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

package azure

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	azureenv "github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_AZURE    = api.CLOUD_PROVIDER_AZURE
	CLOUD_PROVIDER_AZURE_CN = "微软"

	AZURE_API_VERSION = "2016-02-01"
)

type TAzureResource string

var (
	GraphResource   = TAzureResource("graph")
	DefaultResource = TAzureResource("default")
)

type SAzureClient struct {
	*AzureClientConfig

	client  autorest.Client
	domain  string
	baseUrl string

	ressourceGroups []SResourceGroup

	env        azureenv.Environment
	authorizer autorest.Authorizer

	regions  []SRegion
	iBuckets []cloudprovider.ICloudBucket

	subscriptions []SSubscription

	debug bool
}

type AzureClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	envName      string
	tenantId     string
	clientId     string
	clientSecret string

	subscriptionId string

	debug bool
}

func NewAzureClientConfig(envName, tenantId, clientId, clientSecret string) *AzureClientConfig {
	cfg := &AzureClientConfig{
		envName:      envName,
		tenantId:     tenantId,
		clientId:     clientId,
		clientSecret: clientSecret,
	}
	return cfg
}

func (cfg *AzureClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *AzureClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *AzureClientConfig) SubscriptionId(id string) *AzureClientConfig {
	cfg.subscriptionId = id
	return cfg
}

func (cfg *AzureClientConfig) Debug(debug bool) *AzureClientConfig {
	cfg.debug = debug
	return cfg
}

func NewAzureClient(cfg *AzureClientConfig) (*SAzureClient, error) {
	client := SAzureClient{
		AzureClientConfig: cfg,
		debug:             cfg.debug,
	}
	var err error
	client.subscriptions, err = client.ListSubscriptions()
	if err != nil {
		return nil, errors.Wrap(err, "ListSubscriptions")
	}
	client.regions, err = client.ListRegions()
	if err != nil {
		return nil, errors.Wrapf(err, "ListRegions")
	}
	for i := range client.regions {
		client.regions[i].client = &client
	}
	client.ressourceGroups, err = client.ListResourceGroups()
	if err != nil {
		return nil, errors.Wrapf(err, "ListResourceGroups")
	}
	return &client, nil
}

func (self *SAzureClient) getClient(resource TAzureResource) (*autorest.Client, error) {
	client := autorest.NewClientWithUserAgent("Yunion API")
	conf := auth.NewClientCredentialsConfig(self.clientId, self.clientSecret, self.tenantId)
	env, err := azureenv.EnvironmentFromName(self.envName)
	if err != nil {
		return nil, errors.Wrapf(err, "azureenv.EnvironmentFromName(%s)", self.envName)
	}

	httpClient := self.cpcfg.HttpClient()
	client.Sender = httpClient

	self.env = env
	switch resource {
	case GraphResource:
		self.domain = env.GraphEndpoint
		conf.Resource = env.GraphEndpoint
	default:
		self.domain = env.ResourceManagerEndpoint
		conf.Resource = env.ResourceManagerEndpoint
	}
	conf.AADEndpoint = env.ActiveDirectoryEndpoint
	{
		spt, err := conf.ServicePrincipalToken()
		if err != nil {
			return nil, errors.Wrapf(err, "ServicePrincipalToken")
		}
		spt.SetSender(httpClient)
		client.Authorizer = autorest.NewBearerAuthorizer(spt)
	}
	if self.debug {
		client.RequestInspector = LogRequest()
		client.ResponseInspector = LogResponse()
	}

	return &client, nil
}

func (self *SAzureClient) getDefaultClient() (*autorest.Client, error) {
	return self.getClient(DefaultResource)
}

func (self *SAzureClient) getGraphClient() (*autorest.Client, error) {
	return self.getClient(GraphResource)
}

func (self *SAzureClient) jsonRequest(method, path string, body jsonutils.JSONObject, params url.Values) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, errors.Wrapf(err, "jsonRequest")
	}
	resp, err := jsonRequest(cli, method, self.domain, path, body, params)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonRequest")
	}

	azErr := func() *AzureError {
		for _, key := range []string{"error", "odata.error"} {
			if resp.Contains(key) {
				e := &AzureError{}
				resp.Unmarshal(e)
				return e
			}
		}
		return nil
	}()

	if azErr != nil {
		switch azErr.Code {
		case "SubscriptionNotRegistered":
			err = self.registerService("Microsoft.Network")
			if err != nil {
				return nil, errors.Wrapf(err, "self.registerService(Microsoft.Network)")
			}
		case "MissingSubscriptionRegistration":
			for _, serviceType := range azErr.Details {
				err = self.registerService(serviceType.Target)
				if err != nil {
					return nil, errors.Wrapf(err, "self.registerService(%s)", serviceType.Target)
				}
			}
		}
	}
	return resp, nil
}

func (self *SAzureClient) gjsonRequest(method, path string, body jsonutils.JSONObject, params url.Values) (jsonutils.JSONObject, error) {
	cli, err := self.getGraphClient()
	if err != nil {
		return nil, errors.Wrapf(err, "gjsonRequest")
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("api-version", "1.6")
	return jsonRequest(cli, method, self.domain, path, body, params)
}

func (self *SAzureClient) put(path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(path, params))
	return self.jsonRequest("PUT", path, body, params)
}

func (self *SAzureClient) post(path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(path, params))
	return self.jsonRequest("POST", path, body, params)
}

func (self *SAzureClient) patch(resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(resource, params))
	return self.jsonRequest("PATCH", resource, body, params)
}

func (self *SAzureClient) get(resourceId string, params url.Values, retVal interface{}) error {
	if len(resourceId) == 0 {
		return cloudprovider.ErrNotFound
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("api-version", self._apiVersion(resourceId, params))
	body, err := self.jsonRequest("GET", resourceId, nil, params)
	if err != nil {
		return err
	}
	err = body.Unmarshal(retVal)
	if err != nil {
		return err
	}
	return nil
}

func (self *SAzureClient) gcreate(resource string, body jsonutils.JSONObject, retVal interface{}) error {
	path := fmt.Sprintf("%s/%s", self.tenantId, resource)
	result, err := self.gjsonRequest("POST", path, body, url.Values{})
	if err != nil {
		return errors.Wrapf(err, "gjsonRequest")
	}
	if retVal != nil {
		return result.Unmarshal(retVal)
	}
	return nil
}

func (self *SAzureClient) gpatch(resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.gjsonRequest("PATCH", resource, body, nil)
}

func (self *SAzureClient) glist(resource string, params url.Values, retVal interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	err := self._glist(resource, params, retVal)
	if err != nil {
		return errors.Wrapf(err, "_glist(%s)", resource)
	}
	return nil
}

func (self *SAzureClient) _glist(resource string, params url.Values, retVal interface{}) error {
	path := fmt.Sprintf("%s/%s", self.tenantId, resource)
	body, err := self.gjsonRequest("GET", path, nil, params)
	if err != nil {
		return err
	}
	err = body.Unmarshal(retVal, "value")
	if err != nil {
		return errors.Wrapf(err, "body.Unmarshal")
	}
	return nil
}

func (self *SAzureClient) list(resource string, params url.Values, retVal interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	result := []jsonutils.JSONObject{}
	for {
		resp, err := self._list(resource, params)
		if err != nil {
			return errors.Wrapf(err, "_list(%s)", resource)
		}
		part, err := resp.GetArray("value")
		if err != nil {
			return errors.Wrapf(err, "resp.GetArray")
		}
		result = append(result, part...)
		nextLink, _ := resp.GetString("nextLink")
		if len(nextLink) == 0 {
			break
		}
		link, err := url.Parse(nextLink)
		if err != nil {
			return errors.Wrapf(err, "url.Parse(%s)", nextLink)
		}
		key, skiptoken := func() (string, string) {
			for _, key := range []string{"$skiptoken", "$skipToken"} {
				token := link.Query().Get(key)
				if len(token) > 0 {
					return key, token
				}
			}
			return "", ""
		}()
		if len(skiptoken) == 0 {
			break
		}
		params.Set(key, skiptoken)
	}
	return jsonutils.Update(retVal, result)
}

func (self *SAzureClient) _apiVersion(resource string, params url.Values) string {
	version := params.Get("api-version")
	if len(version) > 0 {
		return version
	}
	info := strings.Split(strings.ToLower(resource), "/")
	if utils.IsInStringArray("microsoft.compute", info) {
		if utils.IsInStringArray("virtualmachines", info) {
			return "2018-04-01"
		}
		if utils.IsInStringArray("skus", info) {
			return "2019-04-01"
		}
		return "2018-06-01"
	} else if utils.IsInStringArray("microsoft.classiccompute", info) {
	} else if utils.IsInStringArray("microsoft.network", info) {
		if utils.IsInStringArray("virtualnetworks", info) {
			return "2018-08-01"
		}
		if utils.IsInStringArray("publicipaddresses", info) {
			return "2018-03-01"
		}
		return "2018-06-01"
	} else if utils.IsInStringArray("microsoft.classicnetwork", info) {
		return "2016-04-01"
	} else if utils.IsInStringArray("microsoft.storage", info) {
		if utils.IsInStringArray("storageaccounts", info) {
			return "2016-12-01"
		}
		if utils.IsInStringArray("checknameavailability", info) {
			return "2019-04-01"
		}
		if utils.IsInStringArray("skus", info) {
			return "2019-04-01"
		}
		if utils.IsInStringArray("usages", info) {
			return "2018-07-01"
		}
	} else if utils.IsInStringArray("microsoft.classicstorage", info) {
		if utils.IsInStringArray("storageaccounts", info) {
			return "2016-04-01"
		}
	} else if utils.IsInStringArray("microsoft.billing", info) {
		return "2018-03-01-preview"
	} else if utils.IsInStringArray("microsoft.insights", info) {
		return "2017-03-01-preview"
	} else if utils.IsInStringArray("microsoft.authorization", info) {
		return "2018-01-01-preview"
	}
	return AZURE_API_VERSION
}

func (self *SAzureClient) _list(resource string, params url.Values) (jsonutils.JSONObject, error) {
	subId := self.subscriptionId
	if len(subId) == 0 {
		for _, sub := range self.subscriptions {
			if sub.State == "Enabled" {
				subId = sub.SubscriptionId
			}
		}
	}
	path := "/subscriptions"
	switch resource {
	case "subscriptions":
	case "locations", "resourcegroups", "providers":
		if len(subId) == 0 {
			return nil, fmt.Errorf("no avaiable subscriptions")
		}
		path = fmt.Sprintf("subscriptions/%s/%s", subId, resource)
	default:
		if len(subId) == 0 {
			return nil, fmt.Errorf("no avaiable subscriptions")
		}
		path = fmt.Sprintf("subscriptions/%s/providers/%s", self.subscriptionId, resource)
	}
	params.Set("api-version", self._apiVersion(resource, params))
	return self.jsonRequest("GET", path, nil, params)
}

func (self *SAzureClient) del(resourceId string) error {
	_, err := self.jsonRequest("DELETE", resourceId, nil, url.Values{})
	return err
}

func (self *SAzureClient) GDelete(resourceId string) error {
	return self.gdel(resourceId)
}

func (self *SAzureClient) gdel(resourceId string) error {
	_, err := self.gjsonRequest("DELETE", resourceId, nil, url.Values{})
	if err != nil {
		return errors.Wrapf(err, "gdel(%s)", resourceId)
	}
	return nil
}

func (self *SAzureClient) perform(resourceId string, action string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("%s/%s", resourceId, action)
	return self.post(path, body)
}

func (self *SAzureClient) CreateIProject(name string) (cloudprovider.ICloudProject, error) {
	if len(self.regions) > 0 {
		_, err := self.regions[0].CreateResourceGroup(name)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateResourceGroup")
		}
		return self.regions[0].GetResourceGroupDetail(name)
	}
	return nil, fmt.Errorf("no region found ???")
}

func (self *SAzureClient) ListResourceGroups() ([]SResourceGroup, error) {
	resourceGroups := []SResourceGroup{}
	err := self.list("resourcegroups", url.Values{}, &resourceGroups)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	return resourceGroups, nil
}

type AzureErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Target  string `json:"target,omitempty"`
}

type AzureError struct {
	Code    string             `json:"code,omitempty"`
	Details []AzureErrorDetail `json:"details,omitempty"`
	Message string             `json:"message,omitempty"`
}

func (e *AzureError) Error() string {
	return jsonutils.Marshal(e).String()
}

func (self *SAzureClient) getUniqName(resourceGroup, resourceType, name string) (string, error) {
	prefix := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/", self.subscriptionId, resourceGroup, resourceType)
	newName := name
	for i := 0; i < 20; i++ {
		err := self.get(prefix+newName, nil, url.Values{})
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return newName, nil
		}
		info := strings.Split(newName, "-")
		num, _ := strconv.Atoi(info[len(info)-1])
		if num > 0 {
			info[len(info)-1] = fmt.Sprintf("%d", num+1)
		}
		newName = strings.Join(info, "-")
	}
	return "", fmt.Errorf("not find uniq name for %s[%s]", resourceType, name)
}

func (self *SAzureClient) create(resourceGroup, resourceType, name string, body jsonutils.JSONObject, retVal interface{}) error {
	resource := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", self.subscriptionId, resourceGroup, resourceType, name)
	params := url.Values{}
	params.Set("api-version", self._apiVersion(resourceType, params))
	resp, err := self.jsonRequest("PUT", resource, body, params)
	if err != nil {
		return errors.Wrapf(err, "jsonRequest")
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SAzureClient) CheckNameAvailability(resourceType string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/subscriptions/%s/providers/%s/checkNameAvailability", self.subscriptionId, resourceType)
	return self.post(path, body)
}

func (self *SAzureClient) update(body jsonutils.JSONObject, retVal interface{}) error {
	id, _ := body.GetString("id")
	if len(id) == 0 {
		return fmt.Errorf("failed to found id for update operation")
	}
	params := url.Values{}
	params.Set("api-version", self._apiVersion(id, params))
	resp, err := self.jsonRequest("PUT", id, body, params)
	if err != nil {
		return err
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SAzureClient) waitRegisterComplete(serviceType string) error {
	return cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		services, err := self.ListServices()
		if err != nil {
			return false, errors.Wrapf(err, "ListServices")
		}
		for _, service := range services {
			if service.Namespace == serviceType {
				if service.RegistrationState == "Registered" {
					return true, nil
				}
				log.Debugf("service %s status: %s", service.RegistrationState)
			}
		}
		return false, nil
	})
}

func (self *SAzureClient) registerService(serviceType string) error {
	resource := fmt.Sprintf("/subscriptions/%s/providers/%s/register", self.subscriptionId, serviceType)
	_, err := self.post(resource, nil)
	if err != nil {
		return errors.Wrapf(err, "post(%s)", resource)
	}
	return self.waitRegisterComplete(serviceType)
}

func jsonRequest(client *autorest.Client, method, domain, baseUrl string, body jsonutils.JSONObject, params url.Values) (jsonutils.JSONObject, error) {
	result, err := _jsonRequest(client, method, domain, baseUrl, body, params)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func waitForComplatetion(client *autorest.Client, req *http.Request, resp *http.Response, timeout time.Duration) (jsonutils.JSONObject, error) {
	location := resp.Header.Get("Location")
	asyncoperation := resp.Header.Get("Azure-Asyncoperation")
	startTime := time.Now()
	if len(location) > 0 || (len(asyncoperation) > 0 && resp.StatusCode != 200 || strings.Index(req.URL.String(), "enablevmaccess") > 0) {
		if len(asyncoperation) > 0 {
			location = asyncoperation
		}
		for {
			if strings.HasSuffix(location, "Microsoft.DirectoryServices.User") || strings.HasSuffix(location, "Microsoft.DirectoryServices.Group") {
				location = location + "?api-version=1.6"
			}
			asyncReq, err := http.NewRequest("GET", location, nil)
			if err != nil {
				return nil, err
			}
			asyncResp, err := client.Do(asyncReq)
			if err != nil {
				return nil, err
			}
			defer asyncResp.Body.Close()
			if asyncResp.StatusCode == 202 {
				if _location := asyncResp.Header.Get("Location"); len(_location) > 0 {
					location = _location
				}
				if time.Now().Sub(startTime) > timeout {
					return nil, fmt.Errorf("Process request %s %s timeout", req.Method, req.URL.String())
				}
				timeSleep := time.Second * 5
				if _timeSleep := asyncResp.Header.Get("Retry-After"); len(_timeSleep) > 0 {
					if _time, err := strconv.Atoi(_timeSleep); err != nil {
						timeSleep = time.Second * time.Duration(_time)
					}
				}
				time.Sleep(timeSleep)
				continue
			}
			if asyncResp.ContentLength == 0 {
				return nil, nil
			}
			data, err := ioutil.ReadAll(asyncResp.Body)
			if err != nil {
				return nil, err
			}
			asyncData, err := jsonutils.Parse(data)
			if err != nil {
				return nil, err
			}
			if len(asyncoperation) > 0 && asyncData.Contains("status") {
				status, _ := asyncData.GetString("status")
				switch status {
				case "InProgress":
					log.Debugf("process %s %s InProgress", req.Method, req.URL.String())
					time.Sleep(time.Second * 5)
					continue
				case "Succeeded":
					log.Debugf("process %s %s Succeeded", req.Method, req.URL.String())
					output, err := asyncData.Get("properties", "output")
					if err == nil {
						return output, nil
					}
					return nil, nil
				case "Failed":
					if asyncData.Contains("error") {
						azureError := AzureError{}
						if err := asyncData.Unmarshal(&azureError, "error"); err != nil {
							log.Errorf("process %s %s error: %s", req.Method, req.URL.String(), asyncData.String())
							return nil, fmt.Errorf("%s", asyncData.String())
						}
						switch azureError.Code {
						// 忽略创建机器时初始化超时问题
						case "OSProvisioningTimedOut", "OSProvisioningClientError", "OSProvisioningInternalError":
							// {"code":"OSProvisioningInternalError","message":"OS Provisioning failed for VM 'stress-testvm-azure-1' due to an internal error: [000004] cloud-init appears to be running, this is not expected, cannot continue."}
							log.Debugf("ignore OSProvisioning error: %s", azureError)
							return nil, nil
						default:
							log.Errorf("process %s %s error: %s", req.Method, req.URL.String(), azureError)
							return nil, &azureError
						}
					}
				default:
					log.Errorf("Unknow status %s when process %s %s", status, req.Method, req.URL.String())
					return nil, fmt.Errorf("Unknow status %s", status)
				}
				return nil, fmt.Errorf("Create failed: %s", data)
			}
			log.Debugf("process %s %s return: %s", req.Method, req.URL.String(), data)
			return asyncData, nil
		}
	}
	return nil, nil
}

func _jsonRequest(client *autorest.Client, method, domain, path string, body jsonutils.JSONObject, params url.Values) (result jsonutils.JSONObject, err error) {
	url := fmt.Sprintf("%s/%s?%s", strings.TrimSuffix(domain, "/"), strings.TrimPrefix(path, "/"), params.Encode())
	req := &http.Request{}
	if body != nil {
		req, err = http.NewRequest(method, url, strings.NewReader(body.String()))
		if err != nil {
			log.Errorf("Azure %s new request: %s body: %s error: %v", method, url, body, err)
			return nil, err
		}
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			log.Errorf("Azure %s new request: %s error: %v", method, url, err)
			return nil, err
		}
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Azure %s request: %s \nbody: %s error: %v", req.Method, req.URL.String(), body, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		data := []byte{}
		if resp.ContentLength != 0 {
			data, _ = ioutil.ReadAll(resp.Body)
		}
		log.Infof("failed find %s error: %s", url, string(data))
		return nil, cloudprovider.ErrNotFound
	}
	// 异步任务最多耗时半小时，否则以失败处理
	asyncData, err := waitForComplatetion(client, req, resp, time.Minute*30)
	if err != nil {
		return nil, err
	}
	if asyncData != nil {
		return asyncData, nil
	}
	if resp.ContentLength == 0 {
		return jsonutils.NewDict(), nil
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_data := strings.Replace(string(data), "\r", "", -1)
	return jsonutils.Parse([]byte(_data))
}

func (self *SAzureClient) ListRegions() ([]SRegion, error) {
	regions := []SRegion{}
	err := self.list("locations", url.Values{}, &regions)
	return regions, err
}

func (self *SAzureClient) GetRegions() []SRegion {
	return self.regions
}

func (self *SAzureClient) GetSubAccounts() (subAccounts []cloudprovider.SSubAccount, err error) {
	subAccounts = make([]cloudprovider.SSubAccount, len(self.subscriptions))
	for i, subscription := range self.subscriptions {
		subAccounts[i].Account = fmt.Sprintf("%s/%s", self.tenantId, subscription.SubscriptionId)
		subAccounts[i].State = subscription.State
		subAccounts[i].Name = subscription.DisplayName
		subAccounts[i].HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	}
	return subAccounts, nil
}

func (self *SAzureClient) GetAccountId() string {
	return self.tenantId
}

func (self *SAzureClient) GetIamLoginUrl() string {
	switch self.envName {
	case "AzureChinaCloud":
		return "http://portal.azure.cn"
	default:
		return "http://portal.azure.com"
	}
}

func (self *SAzureClient) GetIRegions() []cloudprovider.ICloudRegion {
	ret := []cloudprovider.ICloudRegion{}
	for i := range self.regions {
		ret = append(ret, &self.regions[i])
	}
	return ret
}

func (self *SAzureClient) getDefaultRegion() (cloudprovider.ICloudRegion, error) {
	if len(self.regions) > 0 {
		return &self.regions[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.regions); i += 1 {
		if self.regions[i].GetId() == id {
			return &self.regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.regions); i += 1 {
		if self.regions[i].GetGlobalId() == id {
			return &self.regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(self.regions); i += 1 {
		if self.regions[i].GetId() == regionId {
			return &self.regions[i]
		}
	}
	return nil
}

func (self *SAzureClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIVpcById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.regions); i += 1 {
		ihost, err := self.regions[i].GetIStorageById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SAccountBalance struct {
	AvailableAmount     float64
	AvailableCashAmount float64
	CreditAmount        float64
	MybankCreditAmount  float64
	Currency            string
}

func (self *SAzureClient) QueryAccountBalance() (*SAccountBalance, error) {
	return nil, cloudprovider.ErrNotSupported
}

func getResourceGroup(id string) string {
	if info := strings.Split(id, "/resourceGroups/"); len(info) == 2 {
		if resourcegroupInfo := strings.Split(info[1], "/"); len(resourcegroupInfo) > 0 {
			return strings.ToLower(resourcegroupInfo[0])
		}
	}
	return ""
}

func (self *SAzureClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	resourceGroups, err := self.ListResourceGroups()
	if err != nil {
		return nil, errors.Wrapf(err, "ListResourceGroups")
	}
	iprojects := []cloudprovider.ICloudProject{}
	for i := 0; i < len(resourceGroups); i++ {
		resourceGroups[i].client = self
		iprojects = append(iprojects, &resourceGroups[i])
	}
	return iprojects, nil
}

func (self *SAzureClient) GetStorageClasses(regionExtId string) ([]string, error) {
	var iRegion cloudprovider.ICloudRegion
	var err error
	if regionExtId == "" {
		iRegion, err = self.getDefaultRegion()
	} else {
		iRegion, err = self.GetIRegionById(regionExtId)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetIRegionById %s", regionExtId)
	}
	skus, err := iRegion.(*SRegion).GetStorageAccountSkus()
	if err != nil {
		return nil, errors.Wrap(err, "GetStorageAccountSkus")
	}
	ret := make([]string, 0)
	for i := range skus {
		ret = append(ret, skus[i].Name)
	}
	return ret, nil
}

func (self *SAzureClient) GetAccessEnv() string {
	env, _ := azureenv.EnvironmentFromName(self.envName)
	switch env.Name {
	case azureenv.PublicCloud.Name:
		return api.CLOUD_ACCESS_ENV_AZURE_GLOBAL
	case azureenv.ChinaCloud.Name:
		return api.CLOUD_ACCESS_ENV_AZURE_CHINA
	case azureenv.GermanCloud.Name:
		return api.CLOUD_ACCESS_ENV_AZURE_GERMAN
	case azureenv.USGovernmentCloud.Name:
		return api.CLOUD_ACCESS_ENV_AZURE_US_GOVERNMENT
	default:
		return api.CLOUD_ACCESS_ENV_AZURE_CHINA
	}
}

func (self *SAzureClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
	}
	return caps
}
