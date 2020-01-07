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

package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_GOOGLE    = api.CLOUD_PROVIDER_GOOGLE
	CLOUD_PROVIDER_GOOGLE_CN = "谷歌云"

	GOOGLE_DEFAULT_REGION = "asia-east1"

	GOOGLE_API_VERSION         = "v1"
	GOOGLE_MANAGER_API_VERSION = "v1"

	GOOGLE_STORAGE_API_VERSION    = "v1"
	GOOGLE_CLOUDBUILD_API_VERSION = "v1"
	GOOGLE_BILLING_API_VERSION    = "v1"

	GOOGLE_MANAGER_DOMAIN        = "https://cloudresourcemanager.googleapis.com"
	GOOGLE_COMPUTE_DOMAIN        = "https://www.googleapis.com/compute"
	GOOGLE_STORAGE_DOMAIN        = "https://storage.googleapis.com/storage"
	GOOGLE_CLOUDBUILD_DOMAIN     = "https://cloudbuild.googleapis.com"
	GOOGLE_STORAGE_UPLOAD_DOMAIN = "https://www.googleapis.com/upload/storage"
	GOOGLE_BILLING_DOMAIN        = "https://cloudbilling.googleapis.com"

	MAX_RETRY = 3
)

type SGoogleClient struct {
	providerId      string
	providerName    string
	projectId       string
	privateKey      string
	privateKeyId    string
	clientEmail     string
	iregions        []cloudprovider.ICloudRegion
	images          []SImage
	snapshots       map[string][]SSnapshot
	globalnetworks  []SGlobalNetwork
	resourcepolices []SResourcePolicy

	client *http.Client

	Debug bool
}

func NewGoogleClient(providerId string, providerName string, projectId, clientEmail, privateKeyId, privateKey string, isDebug bool) (*SGoogleClient, error) {
	client := SGoogleClient{
		providerId:   providerId,
		providerName: providerName,
		projectId:    projectId,
		privateKey:   strings.Replace(privateKey, "\\n", "\n", -1),
		privateKeyId: privateKeyId,
		clientEmail:  clientEmail,
		Debug:        isDebug,
	}
	conf := &jwt.Config{
		Email:        clientEmail,
		PrivateKeyID: privateKeyId,
		PrivateKey:   []byte(client.privateKey),
		Scopes: []string{
			"https://www.googleapis.com/auth/cloud-platform",
			"https://www.googleapis.com/auth/compute",
			"https://www.googleapis.com/auth/compute.readonly",
			"https://www.googleapis.com/auth/cloud-platform.read-only",
			"https://www.googleapis.com/auth/cloudplatformprojects",
			"https://www.googleapis.com/auth/cloudplatformprojects.readonly",

			"https://www.googleapis.com/auth/devstorage.full_control",
			"https://www.googleapis.com/auth/devstorage.read_write",
		},
		TokenURL: google.JWTTokenURL,
	}
	client.client = conf.Client(oauth2.NoContext)
	return &client, client.fetchRegions()
}

func (self *SGoogleClient) GetAccountId() string {
	return self.clientEmail
}

func (self *SGoogleClient) fetchRegions() error {
	regions := []SRegion{}
	err := self.ecsListAll("regions", nil, &regions)
	if err != nil {
		return err
	}

	self.iregions = []cloudprovider.ICloudRegion{}
	for i := 0; i < len(regions); i++ {
		regions[i].client = self
		self.iregions = append(self.iregions, &regions[i])
	}
	return nil
}

func jsonRequest(client *http.Client, method httputils.THttpMethod, domain, apiVersion, resource string, params map[string]string, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	resource = strings.TrimPrefix(resource, fmt.Sprintf("%s/%s/", domain, apiVersion))
	if len(resource) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	_url := fmt.Sprintf("%s/%s/%s", domain, apiVersion, resource)
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	if len(values) > 0 {
		_url = fmt.Sprintf("%s?%s", _url, values.Encode())
	}
	return _jsonRequest(client, method, _url, body, debug)
}

func (self *SGoogleClient) ecsGet(resource string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, nil, nil, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		err = resp.Unmarshal(retval)
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (self *SGoogleClient) ecsList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	return jsonRequest(self.client, "GET", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, nil, self.Debug)
}

func (self *SGoogleClient) managerList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_MANAGER_DOMAIN, GOOGLE_MANAGER_API_VERSION, resource, params, nil, self.Debug)
}

func (self *SGoogleClient) managerGet(resource string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_MANAGER_DOMAIN, GOOGLE_MANAGER_API_VERSION, resource, nil, nil, self.Debug)
}

func (self *SGoogleClient) ecsListAll(resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["maxResults"] = "500"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.ecsList(resource, params)
		if err != nil {
			return errors.Wrap(err, "ecsList")
		}
		if resp.Contains("items") {
			_items, err := resp.GetArray("items")
			if err != nil {
				return errors.Wrap(err, "resp.GetArray")
			}
			items.Add(_items...)
		}
		nextPageToken, _ = resp.GetString("nextPageToken")
		if len(nextPageToken) == 0 {
			break
		}
	}
	return items.Unmarshal(retval)
}

func (self *SGoogleClient) ecsDelete(id string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, id, nil, nil, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) ecsPatch(resource string, action string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	if len(action) > 0 {
		resource = fmt.Sprintf("%s/%s", resource, action)
	}
	resp, err := jsonRequest(self.client, "PATCH", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, body, self.Debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) ecsDo(resource string, action string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	resource = fmt.Sprintf("%s/%s", resource, action)
	resp, err := jsonRequest(self.client, "POST", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, body, self.Debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) ecsInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	if name, _ := body.GetString("name"); len(name) > 0 {
		generateName := ""
		for _, s := range name {
			if unicode.IsLetter(s) || unicode.IsDigit(s) {
				generateName = fmt.Sprintf("%s%c", generateName, s)
			} else {
				generateName = fmt.Sprintf("%s-", generateName)
			}
		}
		if name != generateName {
			err := jsonutils.Update(body, map[string]string{"name": generateName})
			if err != nil {
				log.Errorf("faild to generate google name from %s -> %s", name, generateName)
			}
		}
	}
	resp, err := jsonRequest(self.client, "POST", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, nil, body, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) storageInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resp, err := jsonRequest(self.client, "POST", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, body, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) storageUpload(resource string, header http.Header, body io.Reader) error {
	return rawRequest(self.client, "POST", GOOGLE_STORAGE_UPLOAD_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, header, body, self.Debug)
}

func (self *SGoogleClient) storageList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, params, nil, self.Debug)
}

func (self *SGoogleClient) storageListAll(resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["maxResults"] = "500"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.storageList(resource, params)
		if err != nil {
			return errors.Wrap(err, "storageList")
		}
		if resp.Contains("items") {
			_items, err := resp.GetArray("items")
			if err != nil {
				return errors.Wrap(err, "resp.GetArray")
			}
			items.Add(_items...)
		}
		nextPageToken, _ = resp.GetString("nextPageToken")
		if len(nextPageToken) == 0 {
			break
		}
	}
	return items.Unmarshal(retval)
}

func (self *SGoogleClient) storageGet(resource string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, nil, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		err = resp.Unmarshal(retval)
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (self *SGoogleClient) storageDelete(id string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, id, nil, nil, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) storageDo(resource string, action string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	resource = fmt.Sprintf("%s/%s", resource, action)
	resp, err := jsonRequest(self.client, "POST", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, params, body, self.Debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) cloudbuildGet(resource string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_CLOUDBUILD_DOMAIN, GOOGLE_CLOUDBUILD_API_VERSION, resource, nil, nil, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		err = resp.Unmarshal(retval)
		if err != nil {
			return errors.Wrap(err, "resp.Unmarshal")
		}
	}
	return nil
}

func (self *SGoogleClient) cloudbuildInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resp, err := jsonRequest(self.client, "POST", GOOGLE_CLOUDBUILD_DOMAIN, GOOGLE_CLOUDBUILD_API_VERSION, resource, nil, body, self.Debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) billingList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_BILLING_DOMAIN, GOOGLE_BILLING_API_VERSION, resource, params, nil, self.Debug)
}

func (self *SGoogleClient) billingListAll(resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["pageSize"] = "5000"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.billingList(resource, params)
		if err != nil {
			return errors.Wrap(err, "billingList")
		}
		if resp.Contains("skus") {
			_items, err := resp.GetArray("skus")
			if err != nil {
				return errors.Wrap(err, "resp.GetArray")
			}
			items.Add(_items...)
		}
		nextPageToken, _ = resp.GetString("nextPageToken")
		if len(nextPageToken) == 0 {
			break
		}
	}
	return items.Unmarshal(retval)
}

func rawRequest(client *http.Client, method httputils.THttpMethod, domain, apiVersion string, resource string, header http.Header, body io.Reader, debug bool) error {
	resource = strings.TrimPrefix(resource, fmt.Sprintf("%s/%s/", domain, apiVersion))
	resource = fmt.Sprintf("%s/%s/%s", domain, apiVersion, resource)
	_, err := httputils.Request(client, context.Background(), method, resource, header, body, debug)
	return err
}

func _jsonRequest(client *http.Client, method httputils.THttpMethod, url string, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	var (
		retry bool                 = false
		err   error                = nil
		data  jsonutils.JSONObject = nil
	)
	for i := 0; i < MAX_RETRY; i++ {
		_, data, err = httputils.JSONRequest(client, context.Background(), method, url, nil, body, debug)
		if err != nil {
			if body != nil {
				log.Errorf("%s %s params: %s error: %v", method, url, body.PrettyString(), err)
			} else {
				log.Errorf("%s %s error: %v", method, url, err)
			}
			for _, msg := range []string{
				"EOF",
				"i/o timeout",
				"TLS handshake timeout",
			} {
				if strings.Index(err.Error(), msg) >= 0 {
					retry = true
					break
				}
			}
			if !retry {
				break
			}
		}
		if !retry {
			break
		}
	}
	if err != nil {
		if strings.Index(strings.ToLower(err.Error()), "not found") > 0 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, errors.Wrap(err, "JSONRequest")
	}
	return data, nil
}

func (self *SGoogleClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = GOOGLE_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i++ {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (client *SGoogleClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := client.GetProjects()
	if err != nil {
		return nil, errors.Wrap(err, "GetProjects")
	}
	accounts := []cloudprovider.SSubAccount{}
	for _, project := range projects {
		subAccount := cloudprovider.SSubAccount{}
		subAccount.Name = client.providerName
		subAccount.Account = fmt.Sprintf("%s/%s", project.ProjectId, client.clientEmail)
		if project.LifecycleState == "ACTIVE" {
			subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
		} else {
			subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_ARREARS
		}
		accounts = append(accounts, subAccount)
	}
	return accounts, nil
}

func (self *SGoogleClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i++ {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i].(*SRegion), nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SGoogleClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SGoogleClient) fetchGlobalNetwork() ([]SGlobalNetwork, error) {
	if len(self.globalnetworks) > 0 {
		return self.globalnetworks, nil
	}
	globalnetworks, err := self.GetGlobalNetworks(0, "")
	if err != nil {
		return nil, err
	}
	self.globalnetworks = globalnetworks
	return globalnetworks, nil
}

func (self *SGoogleClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i++ {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SGoogleClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := self.GetProjects()
	if err != nil {
		return nil, err
	}

	iprojects := []cloudprovider.ICloudProject{}
	for i := range projects {
		iprojects = append(iprojects, &projects[i])
	}
	return iprojects, nil
}

func (self *SGoogleClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
