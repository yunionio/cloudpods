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
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	azureenv "github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_AZURE    = api.CLOUD_PROVIDER_AZURE
	CLOUD_PROVIDER_AZURE_CN = "微软"

	AZURE_API_VERSION = "2016-02-01"
)

type SAzureClient struct {
	client         autorest.Client
	providerId     string
	providerName   string
	subscriptionId string
	tenantId       string
	clientId       string
	clientScret    string
	domain         string
	baseUrl        string
	// secret              string
	envName             string
	ressourceGroups     []SResourceGroup
	fetchResourceGroups bool
	env                 azureenv.Environment
	authorizer          autorest.Authorizer

	iregions []cloudprovider.ICloudRegion
	iBuckets []cloudprovider.ICloudBucket

	debug bool
}

var DEFAULT_API_VERSION = map[string]string{
	"vmSizes": "2018-06-01", //2015-05-01-preview,2015-06-15,2016-03-30,2016-04-30-preview,2016-08-30,2017-03-30,2017-12-01,2018-04-01,2018-06-01,2018-10-01
	"Microsoft.Compute/virtualMachineScaleSets":       "2017-12-01",
	"Microsoft.Compute/virtualMachines":               "2018-04-01",
	"Microsoft.ClassicCompute/virtualMachines":        "2017-04-01",
	"Microsoft.Compute/operations":                    "2018-10-01",
	"Microsoft.ClassicCompute/operations":             "2017-04-01",
	"Microsoft.Network/virtualNetworks":               "2018-08-01",
	"Microsoft.ClassicNetwork/virtualNetworks":        "2017-11-15", //avaliable 2014-01-01,2014-06-01,2015-06-01,2015-12-01,2016-04-01,2016-11-01,2017-11-15
	"Microsoft.Compute/disks":                         "2018-06-01", //avaliable 2016-04-30-preview,2017-03-30,2018-04-01,2018-06-01
	"Microsoft.Storage/storageAccounts":               "2016-12-01", //2018-03-01-preview,2018-02-01,2017-10-01,2017-06-01,2016-12-01,2016-05-01,2016-01-01,2015-06-15,2015-05-01-preview
	"Microsoft.ClassicStorage/storageAccounts":        "2016-04-01", //2014-01-01,2014-04-01,2014-04-01-beta,2014-06-01,2015-06-01,2015-12-01,2016-04-01,2016-11-01
	"Microsoft.Compute/snapshots":                     "2018-06-01", //2016-04-30-preview,2017-03-30,2018-04-01,2018-06-01
	"Microsoft.Compute/images":                        "2018-10-01", //2016-04-30-preview,2016-08-30,2017-03-30,2017-12-01,2018-04-01,2018-06-01,2018-10-01
	"Microsoft.Storage":                               "2016-12-01", //2018-03-01-preview,2018-02-01,2017-10-01,2017-06-01,2016-12-01,2016-05-01,2016-01-01,2015-06-15,2015-05-01-preview
	"Microsoft.Network/publicIPAddresses":             "2018-06-01", //2014-12-01-preview, 2015-05-01-preview, 2015-06-15, 2016-03-30, 2016-06-01, 2016-07-01, 2016-08-01, 2016-09-01, 2016-10-01, 2016-11-01, 2016-12-01, 2017-03-01, 2017-04-01, 2017-06-01, 2017-08-01, 2017-09-01, 2017-10-01, 2017-11-01, 2018-01-01, 2018-02-01, 2018-03-01, 2018-04-01, 2018-05-01, 2018-06-01, 2018-07-01, 2018-08-01
	"Microsoft.Network/networkSecurityGroups":         "2018-06-01",
	"Microsoft.Network/networkInterfaces":             "2018-06-01", //2014-12-01-preview, 2015-05-01-preview, 2015-06-15, 2016-03-30, 2016-06-01, 2016-07-01, 2016-08-01, 2016-09-01, 2016-10-01, 2016-11-01, 2016-12-01, 2017-03-01, 2017-04-01, 2017-06-01, 2017-08-01, 2017-09-01, 2017-10-01, 2017-11-01, 2018-01-01, 2018-02-01, 2018-03-01, 2018-04-01, 2018-05-01, 2018-06-01, 2018-07-01, 2018-08-01
	"Microsoft.Network":                               "2018-06-01",
	"Microsoft.ClassicNetwork/reservedIps":            "2016-04-01", //2014-01-01,2014-06-01,2015-06-01,2015-12-01,2016-04-01,2016-11-01
	"Microsoft.ClassicNetwork/networkSecurityGroups":  "2016-11-01", //2015-06-01,2015-12-01,2016-04-01,2016-11-01
	"Microsoft.ClassicCompute/domainNames":            "2015-12-01", //2014-01-01, 2014-06-01, 2015-06-01, 2015-10-01, 2015-12-01, 2016-04-01, 2016-11-01, 2017-11-01, 2017-11-15
	"Microsoft.Compute/locations":                     "2018-06-01",
	"microsoft.insights/eventtypes/management/values": "2017-03-01-preview",
}

func NewAzureClient(providerId string, providerName string, envName, tenantId, clientId, clientSecret, subscriptionId string, debug bool) (*SAzureClient, error) {
	client := SAzureClient{
		providerId:     providerId,
		providerName:   providerName,
		envName:        envName,
		tenantId:       tenantId,
		clientId:       clientId,
		clientScret:    clientSecret,
		subscriptionId: subscriptionId,
		debug:          debug,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, errors.Wrap(err, "fetchRegions")
	}
	if len(subscriptionId) > 0 {
		err = client.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return &client, nil
}

func (self *SAzureClient) getDefaultClient() (*autorest.Client, error) {
	client := autorest.NewClientWithUserAgent("Yunion API")
	conf := auth.NewClientCredentialsConfig(self.clientId, self.clientScret, self.tenantId)
	env, err := azureenv.EnvironmentFromName(self.envName)
	if err != nil {
		return nil, err
	}
	self.env = env
	self.domain = env.ResourceManagerEndpoint
	conf.Resource = env.ResourceManagerEndpoint
	conf.AADEndpoint = env.ActiveDirectoryEndpoint
	authorizer, err := conf.Authorizer()
	if err != nil {
		return nil, err
	}
	client.Authorizer = authorizer
	if self.debug {
		client.RequestInspector = LogRequest()
		client.ResponseInspector = LogResponse()
	}
	return &client, nil
}

func (self *SAzureClient) jsonRequest(method, url string, body string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, method, self.domain, url, self.subscriptionId, body)
}

func (self *SAzureClient) Put(url string, body jsonutils.JSONObject) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	resp, err := jsonRequest(cli, "PUT", self.domain, url, self.subscriptionId, body.String())
	if err != nil {
		return err
	}
	if self.debug {
		log.Debugf("%s", resp)
	}
	return nil
}

func (self *SAzureClient) Patch(url string, body jsonutils.JSONObject) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	resp, err := jsonRequest(cli, "PATCH", self.domain, url, self.subscriptionId, body.String())
	if err != nil {
		return err
	}
	if self.debug {
		log.Debugf("%s", resp)
	}
	return nil
}

func (self *SAzureClient) Get(resourceId string, params []string, retVal interface{}) error {
	if len(resourceId) == 0 {
		return cloudprovider.ErrNotFound
	}
	path := resourceId
	if len(params) > 0 {
		path += fmt.Sprintf("?%s", strings.Join(params, "&"))
	}
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	body, err := jsonRequest(cli, "GET", self.domain, path, self.subscriptionId, "")
	if err != nil {
		return err
	}
	//fmt.Println(body)
	err = body.Unmarshal(retVal)
	if err != nil {
		return err
	}
	return nil
}

func (self *SAzureClient) ListVmSizes(location string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	if len(self.subscriptionId) == 0 {
		return nil, fmt.Errorf("need subscription id")
	}
	url := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/vmSizes", self.subscriptionId, location)
	return jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "")
}

func (self *SAzureClient) ListClassicDisks() (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	if len(self.subscriptionId) == 0 {
		return nil, fmt.Errorf("need subscription id")
	}
	url := fmt.Sprintf("/subscriptions/%s/services/disks", self.subscriptionId)
	return jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "")
}

func (self *SAzureClient) ListAll(resourceType string, retVal interface{}) error {
	return self.ListResources(resourceType, retVal, []string{"value"})
}

func (self *SAzureClient) ListAllWithNextToken(resourceType string, retVal interface{}) (string, error) {
	return self.ListResourcesWithNextLink(resourceType, retVal, []string{"value"})
}

func (self *SAzureClient) ListResourcesWithNextLink(resourceType string, retVal interface{}, keys []string) (string, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return "", err
	}
	url := "/subscriptions"
	if len(self.subscriptionId) > 0 {
		url += fmt.Sprintf("/%s", self.subscriptionId)
	}
	if len(resourceType) > 0 {
		url += fmt.Sprintf("/providers/%s", resourceType)
	}
	body, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "")
	if err != nil {
		return "", err
	}
	// fmt.Printf("%s: %s\n", resourceType, body)
	if retVal != nil {
		err = body.Unmarshal(retVal, keys...)
		if err != nil {
			return "", err
		}
	}
	nextLink, _ := body.GetString("nextLink")
	return nextLink, nil
}

func (self *SAzureClient) ListResources(resourceType string, retVal interface{}, keys []string) error {
	_, err := self.ListResourcesWithNextLink(resourceType, retVal, keys)
	return err
}

func (self *SAzureClient) ListSubscriptions() (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "GET", self.domain, "/subscriptions", self.subscriptionId, "")
}

func (self *SAzureClient) List(golbalResource string, retVal interface{}) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	url := "/subscriptions"
	if len(self.subscriptionId) > 0 {
		url += fmt.Sprintf("/%s", self.subscriptionId)
	}
	if len(self.subscriptionId) > 0 && len(golbalResource) > 0 {
		url += fmt.Sprintf("/%s", golbalResource)
	}
	body, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "")
	if err != nil {
		return err
	}
	return body.Unmarshal(retVal, "value")
}

func (self *SAzureClient) ListByTypeWithResourceGroup(resourceGroupName string, Type string, retVal interface{}) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	if len(self.subscriptionId) == 0 {
		return fmt.Errorf("Missing subscription Info")
	}
	url := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s", self.subscriptionId, resourceGroupName, Type)
	body, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "")
	if err != nil {
		return err
	}
	return body.Unmarshal(retVal, "value")
}

func (self *SAzureClient) Delete(resourceId string) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	_, err = jsonRequest(cli, "DELETE", self.domain, resourceId, self.subscriptionId, "")
	return err
}

func (self *SAzureClient) PerformAction(resourceId string, action string, body string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", resourceId, action)
	return jsonRequest(cli, "POST", self.domain, url, self.subscriptionId, body)
}

func (self *SAzureClient) fetchResourceGroup(cli *autorest.Client, location string) error {
	if !self.fetchResourceGroups {
		err := self.List("resourcegroups", &self.ressourceGroups)
		if err != nil {
			log.Errorf("failed to list resourceGroups: %v", err)
			return err
		}
		self.fetchResourceGroups = true
	}
	if len(self.ressourceGroups) == 0 {
		//Create Default resourceGroup
		_url := fmt.Sprintf("/subscriptions/%s/resourcegroups/Default", self.subscriptionId)
		body, err := jsonRequest(cli, "PUT", self.domain, _url, self.subscriptionId, fmt.Sprintf(`{"name": "Default", "location": "%s"}`, location))
		if err != nil {
			return err
		}
		resourceGroup := SResourceGroup{}
		err = body.Unmarshal(&resourceGroup)
		if err != nil {
			return err
		}
		self.ressourceGroups = []SResourceGroup{resourceGroup}
	}
	return nil
}

func (self *SAzureClient) checkParams(body jsonutils.JSONObject, params []string) (map[string]string, error) {
	result := map[string]string{}
	for i := 0; i < len(params); i++ {
		data, err := body.GetString(params[i])
		if err != nil {
			return nil, fmt.Errorf("Missing %s params", params[i])
		}
		result[params[i]] = data
	}
	return result, nil
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

func (self *SAzureClient) getUniqName(cli *autorest.Client, resourceType, name string, body jsonutils.JSONObject) (string, string, error) {
	url := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", self.subscriptionId, self.ressourceGroups[0].Name, resourceType, name)
	if _, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, ""); err != nil {
		if err == cloudprovider.ErrNotFound {
			return url, body.String(), nil
		}
		return "", "", err
	}
	for i := 0; i < 20; i++ {
		url = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s-%d", self.subscriptionId, self.ressourceGroups[0].Name, resourceType, name, i)
		if _, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, ""); err == cloudprovider.ErrNotFound {
			if err == cloudprovider.ErrNotFound {
				data := body.(*jsonutils.JSONDict)
				data.Set("name", jsonutils.NewString(fmt.Sprintf("%s-%d", name, i)))
				return url, body.String(), nil
			}
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("not find uniq name for %s[%s]", resourceType, name)
}

func (self *SAzureClient) Create(body jsonutils.JSONObject, retVal interface{}) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	if len(self.subscriptionId) == 0 {
		return fmt.Errorf("Missing subscription info")
	}
	params, err := self.checkParams(body, []string{"type", "name", "location"})
	if err != nil {
		return fmt.Errorf("Azure create resource failed: %s", err.Error())
	}
	err = self.fetchResourceGroup(cli, params["location"])
	if err != nil {
		return err
	}
	if len(self.ressourceGroups) == 0 {
		return fmt.Errorf("Create Default resourceGroup error?")
	}

	url, reqString, err := self.getUniqName(cli, params["type"], params["name"], body)
	if err != nil {
		return err
	}
	result, err := jsonRequest(cli, "PUT", self.domain, url, self.subscriptionId, reqString)
	if err != nil {
		return err
	}
	if retVal != nil {
		return result.Unmarshal(retVal)
	}
	return nil
}

func (self *SAzureClient) CheckNameAvailability(Type string, body string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	if len(self.subscriptionId) == 0 {
		return nil, fmt.Errorf("Missing subscription ID")
	}
	url := fmt.Sprintf("/subscriptions/%s/providers/%s/checkNameAvailability", self.subscriptionId, Type)
	return jsonRequest(cli, "POST", self.domain, url, self.subscriptionId, body)
}

func (self *SAzureClient) Update(body jsonutils.JSONObject, retVal interface{}) error {
	cli, err := self.getDefaultClient()
	if err != nil {
		return err
	}
	url, err := body.GetString("id")
	if err != nil {
		return errors.Wrap(err, "failed to found id for update operation")
	}
	result, err := jsonRequest(cli, "PUT", self.domain, url, self.subscriptionId, body.String())
	if err != nil {
		return err
	}
	if retVal != nil {
		return result.Unmarshal(retVal)
	}
	return nil
}

func waitRegisterComplete(client *autorest.Client, domain, subscriptionId string, serviceType string) error {
	for i := 1; i < 10; i++ {
		result, err := _jsonRequest(client, "GET", domain, fmt.Sprintf("/subscriptions/%s/providers", subscriptionId), "")
		if err != nil {
			return err
		}
		value, err := result.GetArray("value")
		if err != nil {
			return err
		}
		for _, v := range value {
			namespace, _ := v.GetString("namespace")
			if namespace == serviceType {
				state, _ := v.GetString("registrationState")
				if state == "Registered" {
					return nil
				}
				log.Debugf("service %s state %s waite %d second ...", serviceType, state, i*10)
			}
		}
		time.Sleep(time.Second * time.Duration(i*10))
	}
	return fmt.Errorf("wait service %s register timeout", serviceType)
}

func registerService(client *autorest.Client, domain, subscriptionId string, serviceType string) error {
	registryUrl := fmt.Sprintf("/subscriptions/%s/providers/%s/register", subscriptionId, serviceType)
	result, err := _jsonRequest(client, "POST", domain, registryUrl, "")
	if err != nil || result.Contains("error") {
		return fmt.Errorf("failed to register %s service", serviceType)
	}
	if state, _ := result.GetString("registrationState"); state == "Registered" {
		return nil
	}
	return waitRegisterComplete(client, domain, subscriptionId, serviceType)
}

func recoverFromError(client *autorest.Client, domain, subscriptionId string, azureErr AzureError) bool {
	switch azureErr.Code {
	case "SubscriptionNotRegistered":
		services := []string{"Microsoft.Network"}
		for _, service := range services {
			if err := registerService(client, domain, subscriptionId, service); err != nil {
				log.Errorf("register %s error: %v", service, err)
				return false
			}
		}
		return true
	case "MissingSubscriptionRegistration":
		for _, detail := range azureErr.Details {
			log.Errorf("The subscription is not registered to use namespace '%s', try register it", detail.Target)
			if err := registerService(client, domain, subscriptionId, detail.Target); err != nil {
				log.Errorf("register %s error: %v", detail.Target, err)
				return false
			}
		}
		return true
	default:
		return false
	}
}

func jsonRequest(client *autorest.Client, method, domain, baseUrl string, subscriptionId string, body string) (jsonutils.JSONObject, error) {
	result, err := _jsonRequest(client, method, domain, baseUrl, body)
	if err != nil {
		return nil, err
	}
	if result.Contains("error") {
		azureError := AzureError{}
		if err := result.Unmarshal(&azureError, "error"); err != nil {
			return nil, fmt.Errorf(result.String())
		}
		if recoverFromError(client, domain, subscriptionId, azureError) {
			return _jsonRequest(client, method, domain, baseUrl, body)
		}
		log.Errorf("Azure %s request: %s \nbody: %s error: %v", method, baseUrl, body, result.String())
		return nil, fmt.Errorf(result.String())
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

func _jsonRequest(client *autorest.Client, method, domain, baseURL, body string) (result jsonutils.JSONObject, err error) {
	version := AZURE_API_VERSION
	for resourceType, _version := range DEFAULT_API_VERSION {
		if strings.Index(strings.ToLower(baseURL), strings.ToLower(resourceType)) > 0 {
			version = _version
		}
	}
	url := fmt.Sprintf("%s%s?api-version=%s", domain, baseURL, version)
	if strings.Index(baseURL, "?") > 0 {
		if strings.Contains(baseURL, "api-version") {
			url = domain + baseURL
		} else {
			url = fmt.Sprintf("%s%s&api-version=%s", domain, baseURL, version)
		}
	}
	req := &http.Request{}
	if len(body) != 0 {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
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

/*func (self *SAzureClient) UpdateAccount(envName, tenantId, appId, appKey, subscriptionId string) error {
	if self.tenantId != tenantId || self.secret != secret || self.envName != envName {
		if clientInfo, accountInfo := strings.Split(secret, "/"), strings.Split(tenantId, "/"); len(clientInfo) >= 2 && len(accountInfo) >= 1 {
			self.clientId, self.clientScret = clientInfo[0], strings.Join(clientInfo[1:], "/")
			self.tenantId = accountInfo[0]
			if len(accountInfo) == 2 {
				self.subscriptionId = accountInfo[1]
			}
			err := self.fetchRegions()
			if err != nil {
				return err
			}
			return nil
		} else {
			return httperrors.NewUnauthorizedError("clientId、clientScret or subscriptId input error")
		}
	}
	return nil
}*/

func (self *SAzureClient) fetchRegions() error {
	if len(self.subscriptionId) > 0 {
		regions := []SRegion{}
		err := self.List("locations", &regions)
		if err != nil {
			return err
		}
		self.iregions = make([]cloudprovider.ICloudRegion, len(regions))
		for i := 0; i < len(regions); i++ {
			regions[i].client = self
			regions[i].SubscriptionID = self.subscriptionId
			self.iregions[i] = &regions[i]
		}
	}
	_, err := self.ListSubscriptions()
	return err
}

func (self *SAzureClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SAzureClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func (client *SAzureClient) fetchBuckets() error {
	accounts := []SStorageAccount{}
	err := client.ListAll("Microsoft.Storage/storageAccounts", &accounts)
	if err != nil {
		return errors.Wrap(err, "client.ListAll")
	}
	buckets := make([]cloudprovider.ICloudBucket, 0)
	for i := range accounts {
		log.Debugf("%s %s %#v", jsonutils.Marshal(accounts[i]), accounts[i].Location, accounts[i])
		region, err := client.getIRegionByRegionId(accounts[i].Location)
		if err != nil {
			log.Errorf("fail to find region '%s'", accounts[i].Location)
			continue
		}
		accounts[i].region = region.(*SRegion)
		buckets = append(buckets, &accounts[i])
	}
	client.iBuckets = buckets
	return nil
}

func (self *SAzureClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SAzureClient) GetSubAccounts() (subAccounts []cloudprovider.SSubAccount, err error) {
	body, err := self.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	subscriptions, err := body.GetArray("value")
	if err != nil {
		return nil, err
	}
	subAccounts = make([]cloudprovider.SSubAccount, len(subscriptions))
	for i, subscription := range subscriptions {
		subscriptionId, err := subscription.GetString("subscriptionId")
		if err != nil {
			return nil, err
		}
		subAccounts[i].Account = fmt.Sprintf("%s/%s", self.tenantId, subscriptionId)
		subAccounts[i].State, err = subscription.GetString("state")
		if err != nil {
			return nil, err
		}
		subAccounts[i].Name, err = subscription.GetString("displayName")
		if err != nil {
			return nil, err
		}

		subAccounts[i].HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	}
	return subAccounts, nil
}

func (self *SAzureClient) GetAccountId() string {
	return self.tenantId
}

func (self *SAzureClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SAzureClient) getDefaultRegion() (cloudprovider.ICloudRegion, error) {
	if len(self.iregions) > 0 {
		return self.iregions[0], nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) getIRegionByRegionId(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SAzureClient) GetRegion(regionId string) *SRegion {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SAzureClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
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

func (self *SAzureClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
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

func (self *SAzureClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
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
	resourceGroups := []SResourceGroup{}
	if err := self.List("resourcegroups", &resourceGroups); err != nil {
		return nil, err
	}
	iprojects := []cloudprovider.ICloudProject{}
	for i := 0; i < len(resourceGroups); i++ {
		if groupInfo := strings.Split(resourceGroups[i].ID, "/"); len(groupInfo) > 0 {
			resourceGroups[i].ID = strings.ToLower(groupInfo[len(groupInfo)-1])
		}
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
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
