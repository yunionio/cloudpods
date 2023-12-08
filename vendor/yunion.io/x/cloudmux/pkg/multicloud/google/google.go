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
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
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
	GOOGLE_MONITOR_API_VERSION    = "v3"
	GOOGLE_DBINSTANCE_API_VERSION = "v1beta4"
	GOOGLE_IAM_API_VERSION        = "v1"

	GOOGLE_BIGQUERY_API_VERSION = "v2"

	GOOGLE_MANAGER_DOMAIN        = "https://cloudresourcemanager.googleapis.com"
	GOOGLE_COMPUTE_DOMAIN        = "https://www.googleapis.com/compute"
	GOOGLE_STORAGE_DOMAIN        = "https://storage.googleapis.com/storage"
	GOOGLE_CLOUDBUILD_DOMAIN     = "https://cloudbuild.googleapis.com"
	GOOGLE_STORAGE_UPLOAD_DOMAIN = "https://www.googleapis.com/upload/storage"
	GOOGLE_BILLING_DOMAIN        = "https://cloudbilling.googleapis.com"
	GOOGLE_MONITOR_DOMAIN        = "https://monitoring.googleapis.com"
	GOOGLE_DBINSTANCE_DOMAIN     = "https://www.googleapis.com/sql"
	GOOGLE_IAM_DOMAIN            = "https://iam.googleapis.com"
	GOOGLE_BIGQUERY_DOMAIN       = "https://bigquery.googleapis.com/bigquery"

	MAX_RETRY = 3
)

var (
	MultiRegions []string = []string{"us", "eu", "asia"}
	DualRegions  []string = []string{"nam4", "eur4"}
)

type GoogleClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	projectId    string
	clientEmail  string
	privateKeyId string
	privateKey   string

	debug bool
}

func NewGoogleClientConfig(projectId, clientEmail, privateKeyId, privateKey string) *GoogleClientConfig {
	privateKey = strings.Replace(privateKey, "\\n", "\n", -1)
	cfg := &GoogleClientConfig{
		projectId:    projectId,
		clientEmail:  clientEmail,
		privateKeyId: privateKeyId,
		privateKey:   privateKey,
	}
	return cfg
}

func (cfg *GoogleClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *GoogleClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *GoogleClientConfig) Debug(debug bool) *GoogleClientConfig {
	cfg.debug = debug
	return cfg
}

type SGoogleClient struct {
	*GoogleClientConfig

	iregions        []cloudprovider.ICloudRegion
	images          []SImage
	snapshots       map[string][]SSnapshot
	globalnetworks  []SGlobalNetwork
	resourcepolices []SResourcePolicy

	client   *http.Client
	iBuckets []cloudprovider.ICloudBucket
}

func NewGoogleClient(cfg *GoogleClientConfig) (*SGoogleClient, error) {
	client := SGoogleClient{
		GoogleClientConfig: cfg,
	}
	conf := &jwt.Config{
		Email:        cfg.clientEmail,
		PrivateKeyID: cfg.privateKeyId,
		PrivateKey:   []byte(cfg.privateKey),
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

	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	ts, _ := httpClient.Transport.(*http.Transport)
	httpClient.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		service := strings.Split(req.URL.Host, ".")[0]
		if service == "www" {
			service = strings.Split(req.URL.Path, "/")[0]
		}
		method, path := req.Method, req.URL.Path
		respCheck := func(resp *http.Response) error {
			if resp.StatusCode == 403 {
				if cfg.cpcfg.UpdatePermission != nil {
					cfg.cpcfg.UpdatePermission(service, fmt.Sprintf("%s %s", method, path))
				}
			}
			return nil
		}
		if cfg.cpcfg.ReadOnly {
			if req.Method == "GET" {
				return respCheck, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return respCheck, nil
	})

	ctx := context.Background()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	client.client = conf.Client(ctx)
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

	objectstoreCapability := []string{
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
	}
	for _, region := range MultiRegions {
		_region := SRegion{
			Name:         region,
			client:       self,
			capabilities: objectstoreCapability,
		}
		self.iregions = append(self.iregions, &_region)
	}
	for _, region := range DualRegions {
		_region := SRegion{
			Name:         region,
			client:       self,
			capabilities: objectstoreCapability,
		}
		self.iregions = append(self.iregions, &_region)
	}
	return nil
}

func (self *SGoogleClient) fetchBuckets() error {
	buckets := []SBucket{}
	params := map[string]string{
		"project": self.projectId,
	}
	err := self.storageListAll("b", params, &buckets)
	if err != nil {
		return errors.Wrap(err, "storageList")
	}
	self.iBuckets = []cloudprovider.ICloudBucket{}
	for i := range buckets {
		region := self.GetRegion(buckets[i].GetLocation())
		if region == nil {
			log.Errorf("failed to found region for bucket %s", buckets[i].GetName())
			continue
		}
		buckets[i].region = region
		self.iBuckets = append(self.iBuckets, &buckets[i])
	}
	return nil
}

func (self *SGoogleClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.fetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
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

func (self *SGoogleClient) ecsGet(resourceType, id string, retval interface{}) error {
	params := map[string]string{
		"filter": fmt.Sprintf(`id="%s"`, id),
	}
	resource := fmt.Sprintf("aggregated/%s", resourceType)
	if strings.Contains(resourceType, "/") {
		resource = resourceType
	}
	resp, err := self.ecsList(resource, params)
	if err != nil {
		return errors.Wrapf(err, "ecsList")
	}
	if resp.Contains("items") {
		if strings.HasPrefix(resource, "aggregated/") {
			items, err := resp.GetMap("items")
			if err != nil {
				return errors.Wrapf(err, "resp.GetMap(items)")
			}
			for _, values := range items {
				if values.Contains(resourceType) {
					arr, err := values.GetArray(resourceType)
					if err != nil {
						return errors.Wrapf(err, "v.GetArray(%s)", resourceType)
					}
					for i := range arr {
						if _id, _ := arr[i].GetString("id"); _id == id {
							return arr[i].Unmarshal(retval)
						}
					}

				}
			}
		} else if strings.HasPrefix(resource, "global/") {
			items, err := resp.GetArray("items")
			if err != nil {
				return errors.Wrapf(err, "resp.GetMap(items)")
			}
			for i := range items {
				if _id, _ := items[i].GetString("id"); _id == id {
					return items[i].Unmarshal(retval)
				}
			}
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SGoogleClient) ecsList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return self._ecsList("GET", resource, params)
}

func (self *SGoogleClient) _ecsList(method, resource string, params map[string]string) (jsonutils.JSONObject, error) {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	return jsonRequest(self.client, httputils.THttpMethod(method), GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) managerList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_MANAGER_DOMAIN, GOOGLE_MANAGER_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) managerGet(resource string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_MANAGER_DOMAIN, GOOGLE_MANAGER_API_VERSION, resource, nil, nil, self.debug)
}

func (self *SGoogleClient) managerPost(resource string, params map[string]string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "POST", GOOGLE_MANAGER_DOMAIN, GOOGLE_MANAGER_API_VERSION, resource, params, body, self.debug)
}

func (self *SGoogleClient) iamPost(resource string, params map[string]string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "POST", GOOGLE_IAM_DOMAIN, GOOGLE_IAM_API_VERSION, resource, params, body, self.debug)
}

func (self *SGoogleClient) iamList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_IAM_DOMAIN, GOOGLE_IAM_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) iamGet(resource string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_IAM_DOMAIN, GOOGLE_IAM_API_VERSION, resource, nil, nil, self.debug)
}

func (self *SGoogleClient) iamPatch(resource string, params map[string]string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "PATCH", GOOGLE_IAM_DOMAIN, GOOGLE_IAM_API_VERSION, resource, params, body, self.debug)
}

func (self *SGoogleClient) iamDelete(id string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_IAM_DOMAIN, GOOGLE_IAM_API_VERSION, id, nil, nil, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) iamListAll(resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["pageSize"] = "100"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.iamList(resource, params)
		if err != nil {
			return errors.Wrap(err, "iamList")
		}
		if resp.Contains("roles") {
			_items, err := resp.GetArray("roles")
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

func (self *SGoogleClient) ecsListAll(resource string, params map[string]string, retval interface{}) error {
	return self._ecsListAll("GET", resource, params, retval)
}

func (self *SGoogleClient) _ecsListAll(method string, resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["maxResults"] = "500"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self._ecsList(method, resource, params)
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
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, id, nil, nil, self.debug)
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
	resp, err := jsonRequest(self.client, "PATCH", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) ecsDo(resource string, action string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	resource = fmt.Sprintf("%s/%s", resource, action)
	resp, err := jsonRequest(self.client, "POST", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) rdsDelete(id string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, id, nil, nil, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) rdsDo(resource string, action string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	resource = fmt.Sprintf("%s/%s", resource, action)
	resp, err := jsonRequest(self.client, "POST", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) rdsPatch(resource string, body jsonutils.JSONObject) (string, error) {
	resp, err := jsonRequest(self.client, "PATCH", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, nil, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) rdsUpdate(resource string, params map[string]string, body jsonutils.JSONObject) (string, error) {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	resp, err := jsonRequest(self.client, "PUT", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) checkAndSetName(body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	name, _ := body.GetString("name")
	if len(name) > 0 {
		generateName := ""
		for _, s := range strings.ToLower(name) {
			if unicode.IsLetter(s) || unicode.IsDigit(s) || s == '-' {
				generateName = fmt.Sprintf("%s%c", generateName, s)
			}
		}
		if strings.HasSuffix(generateName, "-") {
			generateName += "1"
		}
		if name != generateName {
			err := jsonutils.Update(body, map[string]string{"name": generateName})
			if err != nil {
				return nil, fmt.Errorf("faild to generate google name from %s -> %s", name, generateName)
			}
		}
	}
	return body, nil
}

func (self *SGoogleClient) ecsInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	var err error
	body, err = self.checkAndSetName(body)
	if err != nil {
		return errors.Wrap(err, "checkAndSetName")
	}
	params := map[string]string{"requestId": stringutils.UUID4()}
	resp, err := jsonRequest(self.client, "POST", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) storageInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resp, err := jsonRequest(self.client, "POST", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, body, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) storageUpload(resource string, header http.Header, body io.Reader) (*http.Response, error) {
	resp, err := rawRequest(self.client, "POST", GOOGLE_STORAGE_UPLOAD_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, header, body, self.debug)
	if err != nil {
		return nil, errors.Wrap(err, "rawRequest")
	}
	if resp.StatusCode >= 400 {
		msg, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("StatusCode: %d %s", resp.StatusCode, string(msg))
	}
	return resp, nil
}

func (self *SGoogleClient) storageUploadPart(resource string, header http.Header, body io.Reader) (*http.Response, error) {
	resp, err := rawRequest(self.client, "PUT", GOOGLE_STORAGE_UPLOAD_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, header, body, self.debug)
	if err != nil {
		return nil, errors.Wrap(err, "rawRequest")
	}
	if resp.StatusCode >= 400 {
		msg, _ := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("StatusCode: %d %s", resp.StatusCode, string(msg))
	}
	return resp, nil
}

func (self *SGoogleClient) storageAbortUpload(resource string) error {
	resp, err := rawRequest(self.client, "DELETE", GOOGLE_STORAGE_UPLOAD_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, nil, self.debug)
	if err != nil {
		return errors.Wrap(err, "rawRequest")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("StatusCode: %d %s", resp.StatusCode, string(msg))
	}
	return nil
}

func (self *SGoogleClient) storageDownload(resource string, header http.Header) (io.ReadCloser, error) {
	resp, err := rawRequest(self.client, "GET", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, header, nil, self.debug)
	if err != nil {
		return nil, errors.Wrap(err, "rawRequest")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		msg, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("StatusCode: %d %s", resp.StatusCode, string(msg))
	}
	return resp.Body, err
}

func (self *SGoogleClient) storageList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, params, nil, self.debug)
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
	resp, err := jsonRequest(self.client, "GET", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, nil, self.debug)
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

func (self *SGoogleClient) storagePut(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resp, err := jsonRequest(self.client, "PUT", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, nil, body, self.debug)
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
	resp, err := jsonRequest(self.client, "DELETE", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, id, nil, nil, self.debug)
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
	resp, err := jsonRequest(self.client, "POST", GOOGLE_STORAGE_DOMAIN, GOOGLE_STORAGE_API_VERSION, resource, params, body, self.debug)
	if err != nil {
		return "", err
	}
	selfLink, _ := resp.GetString("selfLink")
	return selfLink, nil
}

func (self *SGoogleClient) cloudbuildGet(resource string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_CLOUDBUILD_DOMAIN, GOOGLE_CLOUDBUILD_API_VERSION, resource, nil, nil, self.debug)
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
	resp, err := jsonRequest(self.client, "POST", GOOGLE_CLOUDBUILD_DOMAIN, GOOGLE_CLOUDBUILD_API_VERSION, resource, nil, body, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) rdsInsert(resource string, body jsonutils.JSONObject, retval interface{}) error {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	var err error
	body, err = self.checkAndSetName(body)
	if err != nil {
		return errors.Wrap(err, "checkAndSetName")
	}
	resp, err := jsonRequest(self.client, "POST", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, nil, body, self.debug)
	if err != nil {
		return err
	}
	if retval != nil {
		return resp.Unmarshal(retval)
	}
	return nil
}

func (self *SGoogleClient) rdsGet(resource string, retval interface{}) error {
	resp, err := jsonRequest(self.client, "GET", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, nil, nil, self.debug)
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

func (self *SGoogleClient) rdsList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	resource = fmt.Sprintf("projects/%s/%s", self.projectId, resource)
	return jsonRequest(self.client, "GET", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) rdsListAll(resource string, params map[string]string, retval interface{}) error {
	if params == nil {
		params = map[string]string{}
	}
	items := jsonutils.NewArray()
	nextPageToken := ""
	params["maxResults"] = "500"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.rdsList(resource, params)
		if err != nil {
			return errors.Wrap(err, "rdsList")
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

func (self *SGoogleClient) billingList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_BILLING_DOMAIN, GOOGLE_BILLING_API_VERSION, resource, params, nil, self.debug)
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

func (self *SGoogleClient) monitorList(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	return jsonRequest(self.client, "GET", GOOGLE_MONITOR_DOMAIN, GOOGLE_MONITOR_API_VERSION, resource, params, nil, self.debug)
}

func (self *SGoogleClient) monitorListAll(resource string, params map[string]string) (*jsonutils.JSONArray, error) {
	if params == nil {
		params = map[string]string{}
	}
	timeSeries := jsonutils.NewArray()
	nextPageToken := ""
	params["pageSize"] = "5000"
	for {
		params["pageToken"] = nextPageToken
		resp, err := self.monitorList(resource, params)
		if err != nil {
			return nil, errors.Wrap(err, "monitorList")
		}
		if resp.Contains("timeSeries") {
			_series, err := resp.GetArray("timeSeries")
			if err != nil {
				return nil, errors.Wrap(err, "resp.GetTimeSeries")
			}
			timeSeries.Add(_series...)
		}
		nextPageToken, _ = resp.GetString("nextPageToken")
		if len(nextPageToken) == 0 {
			break
		}
	}
	return timeSeries, nil
}

func rawRequest(client *http.Client, method httputils.THttpMethod, domain, apiVersion string, resource string, header http.Header, body io.Reader, debug bool) (*http.Response, error) {
	resource = strings.TrimPrefix(resource, fmt.Sprintf("%s/%s/", domain, apiVersion))
	resource = fmt.Sprintf("%s/%s/%s", domain, apiVersion, resource)
	return httputils.Request(client, context.Background(), method, resource, header, body, debug)
}

/*
  "error": {
    "code": 400,
    "message": "Request contains an invalid argument.",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.cloudresourcemanager.v1.ProjectIamPolicyError",
        "type": "SOLO_MUST_INVITE_OWNERS",
        "member": "user:test",
        "role": "roles/owner"
      }
    ]
  }
*/

type gError struct {
	ErrorInfo struct {
		Code    int
		Message string
		Status  string
		Details jsonutils.JSONObject
	} `json:"error"`
	Class string
}

func (g *gError) Error() string {
	return jsonutils.Marshal(g).String()
}

func (g *gError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(g)
	}
	if g.ErrorInfo.Code == 0 {
		g.ErrorInfo.Code = statusCode
	}
	if g.ErrorInfo.Details == nil {
		g.ErrorInfo.Details = body
	}
	if len(g.Class) == 0 {
		g.Class = http.StatusText(statusCode)
	}
	if statusCode == 404 {
		return errors.Wrap(cloudprovider.ErrNotFound, g.Error())
	}
	return g
}

func _jsonRequest(cli *http.Client, method httputils.THttpMethod, url string, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	client := httputils.NewJsonClient(cli)
	req := httputils.NewJsonRequest(method, url, body)
	var err error
	var data jsonutils.JSONObject
	for i := 0; i < MAX_RETRY; i++ {
		var ge gError
		_, data, err = client.Send(context.Background(), req, &ge, debug)
		if err == nil {
			return data, nil
		}
		if body != nil {
			log.Errorf("%s %s params: %s error: %v", method, url, body.PrettyString(), err)
		} else {
			log.Errorf("%s %s error: %v", method, url, err)
		}
		retry := false
		for _, msg := range []string{
			"EOF",
			"i/o timeout",
			"TLS handshake timeout",
			"connection reset by peer",
		} {
			if strings.Index(err.Error(), msg) >= 0 {
				retry = true
				break
			}
		}
		if !retry {
			return nil, err
		}
		time.Sleep(time.Second * 10)
	}
	return nil, err
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
		subAccount.Name = project.Name
		subAccount.Account = fmt.Sprintf("%s/%s", project.ProjectId, client.clientEmail)
		subAccount.Id = project.ProjectId
		subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
		if project.LifecycleState != "ACTIVE" {
			continue
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
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_SECURITY_GROUP,
		cloudprovider.CLOUD_CAPABILITY_EIP,
		cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
		cloudprovider.CLOUD_CAPABILITY_CLOUDID,
		cloudprovider.CLOUD_CAPABILITY_QUOTA + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

func (self *SGoogleClient) GetSamlSpInitiatedLoginUrl(idpName string) string {
	// GOOGLE只支持一个IDP, 可以将organization名字存储在idpName里，避免因为权限不足，无法获取organization名称
	if len(idpName) == 0 {
		orgs, _ := self.ListOrganizations()
		if len(orgs) != 1 {
			log.Warningf("Organization count %d != 1, require assign the service account to ONE organization with organization viewer/admin role", len(orgs))
		} else {
			idpName = orgs[0].DisplayName
		}
	}
	if len(idpName) == 0 {
		log.Errorf("no valid organization name for this GCP account")
		return ""
	}
	return fmt.Sprintf("https://www.google.com/a/%s/ServiceLogin?continue=https://console.cloud.google.com", idpName)
}
