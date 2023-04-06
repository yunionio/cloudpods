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

// Copyright 2019 Yunion
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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest"
	azureenv "github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_AZURE    = api.CLOUD_PROVIDER_AZURE
	CLOUD_PROVIDER_AZURE_CN = "微软"
	CLOUD_PROVIDER_AZURE_EN = "Azure"

	AZURE_API_VERSION = "2016-02-01"
)

type TAzureResource string

var (
	GraphResource        = TAzureResource("graph")
	DefaultResource      = TAzureResource("default")
	LoganalyticsResource = TAzureResource("loganalytics")
)

type azureAuthClient struct {
	client *autorest.Client
	domain string
}

type SAzureClient struct {
	*AzureClientConfig

	clientCache map[TAzureResource]*azureAuthClient
	lock        sync.Mutex

	ressourceGroups []SResourceGroup

	regions  []SRegion
	iBuckets []cloudprovider.ICloudBucket

	subscriptions []SSubscription

	debug bool

	workspaces []SLoganalyticsWorkspace
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
		clientCache:       map[TAzureResource]*azureAuthClient{},
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

func (self *SAzureClient) getClient(resource TAzureResource) (*azureAuthClient, error) {
	_client, ok := self.clientCache[resource]
	if ok {
		return _client, nil
	}
	ret := &azureAuthClient{}
	client := autorest.NewClientWithUserAgent("Yunion API")
	conf := auth.NewClientCredentialsConfig(self.clientId, self.clientSecret, self.tenantId)
	env, err := azureenv.EnvironmentFromName(self.envName)
	if err != nil {
		return nil, errors.Wrapf(err, "azureenv.EnvironmentFromName(%s)", self.envName)
	}

	httpClient := self.cpcfg.AdaptiveTimeoutHttpClient()
	transport, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response), error) {
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" || (req.Method == "POST" && strings.HasSuffix(req.URL.Path, "oauth2/token")) {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	client.Sender = httpClient

	switch resource {
	case GraphResource:
		ret.domain = env.GraphEndpoint
		conf.Resource = env.GraphEndpoint
		if self.envName == "AzureChinaCloud" {
			ret.domain = "https://graph.chinacloudapi.cn/"
			conf.Resource = "https://graph.chinacloudapi.cn/"
		}
	case LoganalyticsResource:
		ret.domain = env.ResourceIdentifiers.OperationalInsights
		conf.Resource = env.ResourceIdentifiers.OperationalInsights
		if conf.Resource == "N/A" && self.envName == "AzureChinaCloud" {
			ret.domain = "https://api.loganalytics.azure.cn"
			conf.Resource = ret.domain
		}
	default:
		ret.domain = env.ResourceManagerEndpoint
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
	}
	ret.client = &client
	self.lock.Lock()
	defer self.lock.Unlock()
	self.clientCache[resource] = ret
	return ret, nil
}

func (self *SAzureClient) getDefaultClient() (*azureAuthClient, error) {
	return self.getClient(DefaultResource)
}

func (self *SAzureClient) getGraphClient() (*azureAuthClient, error) {
	return self.getClient(GraphResource)
}

func (self *SAzureClient) getLoganalyticsClient() (*azureAuthClient, error) {
	return self.getClient(LoganalyticsResource)
}

func (self *SAzureClient) jsonRequest(method, path string, body jsonutils.JSONObject, params url.Values, showErrorMsg bool) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getDefaultClient")
	}
	defer func() {
		if err != nil && showErrorMsg {
			bj := ""
			if body != nil {
				bj = body.PrettyString()
			}
			log.Errorf("%s %s?%s \n%s error: %v", method, path, params.Encode(), bj, err)
		}
	}()
	var resp jsonutils.JSONObject
	for i := 0; i < 2; i++ {
		resp, err = jsonRequest(cli.client, method, cli.domain, path, body, params, self.debug)
		if err != nil {
			if ae, ok := err.(*AzureResponseError); ok {
				switch ae.AzureError.Code {
				case "SubscriptionNotRegistered":
					service := self.getService(path)
					if len(service) == 0 {
						return nil, err
					}
					re := self.ServiceRegister("Microsoft.Network")
					if re != nil {
						return nil, errors.Wrapf(re, "self.registerService(Microsoft.Network)")
					}
					continue
				case "MissingSubscriptionRegistration":
					for _, serviceType := range ae.AzureError.Details {
						re := self.ServiceRegister(serviceType.Target)
						if err != nil {
							return nil, errors.Wrapf(re, "self.registerService(%s)", serviceType.Target)
						}
					}
					continue
				}
			}
			return resp, err
		}
		return resp, err
	}
	return resp, err
}

func (self *SAzureClient) ljsonRequest(method, path string, body jsonutils.JSONObject, params url.Values) (jsonutils.JSONObject, error) {
	cli, err := self.getLoganalyticsClient()
	if err != nil {
		return nil, errors.Wrapf(err, "getLoganalyticsClient")
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("api-version", "2021-12-01-preview")
	return jsonRequest(cli.client, method, cli.domain, path, body, params, self.debug)
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
	return jsonRequest(cli.client, method, cli.domain, path, body, params, self.debug)
}

func (self *SAzureClient) put(path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(path, params))
	return self.jsonRequest("PUT", path, body, params, true)
}

func (self *SAzureClient) post(path string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(path, params))
	return self.jsonRequest("POST", path, body, params, true)
}

func (self *SAzureClient) patch(resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(resource, params))
	return self.jsonRequest("PATCH", resource, body, params, true)
}

func (self *SAzureClient) _get(resourceId string, params url.Values, retVal interface{}, showErrorMsg bool) error {
	if len(resourceId) == 0 {
		return cloudprovider.ErrNotFound
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("api-version", self._apiVersion(resourceId, params))
	body, err := self.jsonRequest("GET", resourceId, nil, params, showErrorMsg)
	if err != nil {
		return err
	}
	err = body.Unmarshal(retVal)
	if err != nil {
		return err
	}
	return nil
}

func (self *SAzureClient) get(resourceId string, params url.Values, retVal interface{}) error {
	return self._get(resourceId, params, retVal, true)
}

func (self *SAzureClient) gcreate(resource string, body jsonutils.JSONObject, retVal interface{}) error {
	path := resource
	result, err := self.msGraphRequest("POST", path, body)
	if err != nil {
		return errors.Wrapf(err, "msGraphRequest")
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
	path := resource
	if len(params) > 0 {
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}
	body, err := self.msGraphRequest("GET", path, nil)
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
	var key, skipToken string
	for {
		resp, err := self._list(resource, params)
		if err != nil {
			return errors.Wrapf(err, "_list(%s)", resource)
		}
		keys := []string{}
		if resp.Contains("value") {
			keys = []string{"value"}
		}
		part, err := resp.GetArray(keys...)
		if err != nil {
			return errors.Wrapf(err, "resp.GetArray(%s)", keys)
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
		prevSkipToken := params.Get(key)
		key, skipToken = func() (string, string) {
			for _, _key := range []string{"$skipToken", "$skiptoken"} {
				tokens, ok := link.Query()[_key]
				if ok {
					for _, token := range tokens {
						if len(token) > 0 && token != prevSkipToken {
							return _key, token
						}
					}
				}
			}
			return "", ""
		}()
		if len(skipToken) == 0 {
			break
		}
		params.Del("$skipToken")
		params.Del("$skiptoken")
		params.Set(key, skipToken)
	}
	return jsonutils.Update(retVal, result)
}

func (self *SAzureClient) getService(path string) string {
	for _, service := range []string{
		"microsoft.compute",
		"microsoft.classiccompute",
		"microsoft.network",
		"microsoft.classicnetwork",
		"microsoft.storage",
		"microsoft.classicstorage",
		"microsoft.billing",
		"microsoft.insights",
		"microsoft.authorization",
	} {
		if strings.Contains(strings.ToLower(path), service) {
			return service
		}
	}
	return ""
}

func (self *SAzureClient) _apiVersion(resource string, params url.Values) string {
	version := params.Get("api-version")
	if len(version) > 0 {
		return version
	}
	info := strings.Split(strings.ToLower(resource), "/")
	if utils.IsInStringArray("microsoft.dbformariadb", info) {
		return "2018-06-01-preview"
	} else if utils.IsInStringArray("microsoft.dbformysql", info) {
		if utils.IsInStringArray("flexibleservers", info) {
			return "2020-07-01-privatepreview"
		}
		return "2017-12-01"
	} else if utils.IsInStringArray("microsoft.dbforpostgresql", info) {
		if utils.IsInStringArray("flexibleservers", info) {
			return "2020-02-14-preview"
		}
		return "2017-12-01"
	} else if utils.IsInStringArray("microsoft.sql", info) {
		return "2020-08-01-preview"
	} else if utils.IsInStringArray("microsoft.compute", info) {
		if utils.IsInStringArray("tags", info) {
			return "2020-06-01"
		}
		if utils.IsInStringArray("publishers", info) {
			return "2020-06-01"
		}
		if utils.IsInStringArray("virtualmachines", info) {
			return "2021-11-01"
		}
		if utils.IsInStringArray("skus", info) {
			return "2019-04-01"
		}
		return "2018-06-01"
	} else if utils.IsInStringArray("microsoft.classiccompute", info) {
		return "2016-04-01"
	} else if utils.IsInStringArray("microsoft.network", info) {
		if utils.IsInStringArray("virtualnetworks", info) {
			return "2018-08-01"
		}
		if utils.IsInStringArray("publicipaddresses", info) {
			return "2018-03-01"
		}
		if utils.IsInStringArray("frontdoorwebapplicationfirewallmanagedrulesets", info) {
			return "2020-11-01"
		}
		if utils.IsInStringArray("frontdoorwebapplicationfirewallpolicies", info) {
			return "2020-11-01"
		}
		if utils.IsInStringArray("applicationgatewaywebapplicationfirewallpolicies", info) {
			return "2021-01-01"
		}
		if utils.IsInStringArray("applicationgatewayavailablewafrulesets", info) {
			return "2018-06-01"
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
	} else if utils.IsInStringArray("microsoft.cache", info) {
		if utils.IsInStringArray("redisenterprise", info) {
			return "2021-03-01"
		}
		return "2020-06-01"
	} else if utils.IsInStringArray("microsoft.containerservice", info) {
		return "2021-05-01"
	} else if utils.IsInStringArray("microsoft.operationalinsights", info) {
		return "2021-12-01-preview"
	}
	return AZURE_API_VERSION
}

func (self *SAzureClient) _subscriptionId() string {
	if len(self.subscriptionId) > 0 {
		return self.subscriptionId
	}
	for _, sub := range self.subscriptions {
		if sub.State == "Enabled" {
			return sub.SubscriptionId
		}
	}
	return ""
}

func (self *SAzureClient) _list(resource string, params url.Values) (jsonutils.JSONObject, error) {
	subId := self._subscriptionId()
	path := "subscriptions"
	switch resource {
	case "subscriptions", "providers/Microsoft.Billing/enrollmentAccounts":
		path = resource
	case "locations", "resourcegroups", "providers":
		if len(subId) == 0 {
			return nil, fmt.Errorf("no avaiable subscriptions")
		}
		path = fmt.Sprintf("subscriptions/%s/%s", subId, resource)
	case "Microsoft.Network/frontdoorWebApplicationFirewallPolicies":
		path = fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/%s", subId, params.Get("resourceGroups"), resource)
		params.Del("resourceGroups")
	default:
		if strings.HasPrefix(resource, "subscriptions/") || strings.HasPrefix(resource, "/subscriptions/") {
			path = resource
		} else {
			if len(subId) == 0 {
				return nil, fmt.Errorf("no avaiable subscriptions")
			}
			path = fmt.Sprintf("subscriptions/%s/providers/%s", subId, resource)
		}
	}
	params.Set("api-version", self._apiVersion(resource, params))
	return self.jsonRequest("GET", path, nil, params, true)
}

func (self *SAzureClient) del(resourceId string) error {
	params := url.Values{}
	params.Set("api-version", self._apiVersion(resourceId, params))
	_, err := self.jsonRequest("DELETE", resourceId, nil, params, true)
	return err
}

func (self *SAzureClient) GDelete(resourceId string) error {
	return self.gdel(resourceId)
}

func (self *SAzureClient) gdel(resourceId string) error {
	_, err := self.msGraphRequest("DELETE", resourceId, nil)
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
	for i := range resourceGroups {
		resourceGroups[i].client = self
		resourceGroups[i].subId = self.subscriptionId
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
		err := self._get(prefix+newName, nil, url.Values{}, false)
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return newName, nil
		}
		info := strings.Split(newName, "-")
		num, err := strconv.Atoi(info[len(info)-1])
		if err != nil {
			info = append(info, "1")
		} else {
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
	resp, err := self.jsonRequest("PUT", resource, body, params, true)
	if err != nil {
		return errors.Wrapf(err, "jsonRequest")
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SAzureClient) CheckNameAvailability(resourceType, name string) (bool, error) {
	path := fmt.Sprintf("/subscriptions/%s/providers/%s/checkNameAvailability", self.subscriptionId, strings.Split(resourceType, "/")[0])
	body := map[string]string{
		"Name": name,
		"Type": resourceType,
	}
	resp, err := self.post(path, jsonutils.Marshal(body))
	if err != nil {
		return false, errors.Wrapf(err, "post(%s)", path)
	}
	output := sNameAvailableOutput{}
	err = resp.Unmarshal(&output)
	if err != nil {
		return false, errors.Wrap(err, "resp.Unmarshal")
	}
	if output.NameAvailable {
		return true, nil
	}
	if output.Reason == "AlreadyExists" {
		return false, nil
	}
	return true, nil
}

func (self *SAzureClient) update(body jsonutils.JSONObject, retVal interface{}) error {
	id := jsonutils.GetAnyString(body, []string{"Id", "id", "ID"})
	if len(id) == 0 {
		return fmt.Errorf("failed to found id for update operation")
	}
	params := url.Values{}
	params.Set("api-version", self._apiVersion(id, params))
	resp, err := self.jsonRequest("PUT", id, body, params, true)
	if err != nil {
		return err
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func jsonRequest(client *autorest.Client, method, domain, baseUrl string, body jsonutils.JSONObject, params url.Values, debug bool) (jsonutils.JSONObject, error) {
	result, err := _jsonRequest(client, method, domain, baseUrl, body, params, debug)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// {"odata.error":{"code":"Authorization_RequestDenied","message":{"lang":"en","value":"Insufficient privileges to complete the operation."},"requestId":"b776ba11-5cae-4fb9-b80d-29552e3caedd","date":"2020-10-29T09:05:23"}}
type sMessage struct {
	Lang  string
	Value string
}
type sOdataError struct {
	Code      string
	Message   sMessage
	RequestId string
	Date      time.Time
}
type AzureResponseError struct {
	OdataError sOdataError `json:"odata.error"`
	AzureError AzureError  `json:"error"`
	Code       string
	Message    string
}

func (ae AzureResponseError) Error() string {
	return jsonutils.Marshal(ae).String()
}

func (ae *AzureResponseError) ParseErrorFromJsonResponse(statusCode int, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(ae)
	}
	if statusCode == 404 {
		msg := ""
		if body != nil {
			msg = body.String()
		}
		return errors.Wrap(cloudprovider.ErrNotFound, msg)
	}
	if len(ae.OdataError.Code) > 0 || len(ae.AzureError.Code) > 0 || (len(ae.Code) > 0 && len(ae.Message) > 0) {
		return ae
	}
	return nil
}

func _jsonRequest(client *autorest.Client, method, domain, path string, body jsonutils.JSONObject, params url.Values, debug bool) (jsonutils.JSONObject, error) {
	uri := fmt.Sprintf("%s/%s?%s", strings.TrimSuffix(domain, "/"), strings.TrimPrefix(path, "/"), params.Encode())
	req := httputils.NewJsonRequest(httputils.THttpMethod(method), uri, body)
	ae := AzureResponseError{}
	cli := httputils.NewJsonClient(client)
	header, body, err := cli.Send(context.TODO(), req, &ae, debug)
	if err != nil {
		if strings.Contains(err.Error(), "azure.BearerAuthorizer#WithAuthorization") {
			return nil, errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
		}
		return nil, err
	}
	locationFunc := func(head http.Header) string {
		for _, k := range []string{"Azure-Asyncoperation", "Location"} {
			link := head.Get(k)
			if len(link) > 0 {
				return link
			}
		}
		return ""
	}
	location := locationFunc(header)
	if len(location) > 0 && (body == nil || body.IsZero() || !body.Contains("id")) {
		err = cloudprovider.Wait(time.Second*10, time.Minute*30, func() (bool, error) {
			locationUrl, err := url.Parse(location)
			if err != nil {
				return false, errors.Wrapf(err, "url.Parse(%s)", location)
			}
			if len(locationUrl.Query().Get("api-version")) == 0 {
				q, _ := url.ParseQuery(locationUrl.RawQuery)
				q.Set("api-version", params.Get("api-version"))
				locationUrl.RawQuery = q.Encode()
			}
			req := httputils.NewJsonRequest(httputils.GET, locationUrl.String(), nil)
			lae := AzureResponseError{}
			_header, _body, _err := cli.Send(context.TODO(), req, &lae, debug)
			if _err != nil {
				if utils.IsInStringArray(lae.AzureError.Code, []string{"OSProvisioningTimedOut", "OSProvisioningClientError", "OSProvisioningInternalError"}) {
					body = _body
					return true, nil
				}
				return false, errors.Wrapf(_err, "cli.Send(%s)", location)
			}
			if retryAfter := _header.Get("Retry-After"); len(retryAfter) > 0 {
				sleepTime, _ := strconv.Atoi(retryAfter)
				time.Sleep(time.Second * time.Duration(sleepTime))
				return false, nil
			}
			if _body != nil {
				task := struct {
					Status     string
					Properties struct {
						Output *jsonutils.JSONDict
					}
				}{}
				_body.Unmarshal(&task)
				if len(task.Status) == 0 {
					body = _body
					return true, nil
				}
				switch task.Status {
				case "InProgress":
					log.Debugf("process %s %s InProgress", method, path)
					return false, nil
				case "Succeeded":
					log.Debugf("process %s %s Succeeded", method, path)
					if task.Properties.Output != nil {
						body = task.Properties.Output
					}
					return true, nil
				case "Failed":
					return false, fmt.Errorf("%s %s failed", method, path)
				default:
					return false, fmt.Errorf("Unknow status %s %s %s", task.Status, method, path)
				}
			}
			return false, nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "time out for waiting %s %s", method, uri)
		}
	}
	return body, nil
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
		subAccounts[i].Name = subscription.DisplayName
		subAccounts[i].HealthStatus = subscription.GetHealthStatus()
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

func getResourceGroup(id string) string {
	info := strings.Split(strings.ToLower(id), "/")
	idx := -1
	for i := range info {
		if info[i] == "resourcegroups" {
			idx = i + 1
			break
		}
	}
	if idx > 0 && idx < len(info)-2 {
		return fmt.Sprintf("%s/%s", info[2], info[idx])
	}
	return ""
}

func (self *SAzureClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	subscriptionId := self.subscriptionId
	groups := []SResourceGroup{}
	for _, sub := range self.subscriptions {
		self.subscriptionId = sub.SubscriptionId
		resourceGroups, err := self.ListResourceGroups()
		if err != nil {
			return nil, errors.Wrapf(err, "ListResourceGroups")
		}
		groups = append(groups, resourceGroups...)
	}
	self.subscriptionId = subscriptionId
	iprojects := []cloudprovider.ICloudProject{}
	for i := range groups {
		groups[i].client = self
		iprojects = append(iprojects, &groups[i])
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
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CACHE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_SAML_AUTH,
		cloudprovider.CLOUD_CAPABILITY_WAF,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CACHE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_APP + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_CONTAINER + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

type TagParams struct {
	Properties TagProperties `json:"properties"`
	Operation  string        `json:"operation"`
}

type TagProperties struct {
	Tags map[string]string `json:"tags"`
}

func (self *SAzureClient) GetTags(resourceId string) (map[string]string, error) {
	path := fmt.Sprintf("/%s/providers/Microsoft.Resources/tags/default", resourceId)
	tags := &TagParams{}
	err := self.get(path, nil, tags)
	if err != nil {
		return nil, errors.Wrap(err, "self.get(path, nil, tags)")
	}
	return tags.Properties.Tags, nil
}

func (self *SAzureClient) SetTags(resourceId string, tags map[string]string) (jsonutils.JSONObject, error) {
	//reserved prefix 'microsoft', 'azure', 'windows'.
	for k := range tags {
		if strings.HasPrefix(k, "microsoft") || strings.HasPrefix(k, "azure") || strings.HasPrefix(k, "windows") {
			return nil, errors.Wrap(cloudprovider.ErrNotSupported, "reserved prefix microsoft, azure, windows")
		}
	}
	path := fmt.Sprintf("/%s/providers/Microsoft.Resources/tags/default", resourceId)
	input := TagParams{}
	input.Operation = "replace"
	input.Properties.Tags = tags
	if len(tags) == 0 {
		return nil, self.del(path)
	}
	return self.patch(path, jsonutils.Marshal(input))
}

func (self *SAzureClient) msGraphClient() *http.Client {
	conf := clientcredentials.Config{
		ClientID:     self.clientId,
		ClientSecret: self.clientSecret,

		TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", self.tenantId),
		Scopes:   []string{"https://graph.microsoft.com/.default"},
	}
	if self.envName == "AzureChinaCloud" {
		conf.TokenURL = fmt.Sprintf("https://login.partner.microsoftonline.cn/%s/oauth2/v2.0/token", self.tenantId)
		conf.Scopes = []string{"https://microsoftgraph.chinacloudapi.cn/.default"}
	}
	return conf.Client(context.TODO())
}

func (self *SAzureClient) msGraphRequest(method string, resource string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	client := self.msGraphClient()
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/%s", resource)
	if self.envName == "AzureChinaCloud" {
		url = fmt.Sprintf("https://microsoftgraph.chinacloudapi.cn/v1.0/%s", resource)
	}
	req := httputils.NewJsonRequest(httputils.THttpMethod(method), url, body)
	ae := AzureResponseError{}
	cli := httputils.NewJsonClient(client)
	_, body, err := cli.Send(context.TODO(), req, &ae, self.debug)
	if err != nil {
		return nil, err
	}
	return body, nil
}
