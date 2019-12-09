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
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_GOOGLE    = api.CLOUD_PROVIDER_GOOGLE
	CLOUD_PROVIDER_GOOGLE_CN = "谷歌云"

	GOOGLE_DEFAULT_REGION = "asia-east1"

	GOOGLE_API_VERSION = "v1"
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
	err := self.listAll("regions", nil, &regions)
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

func (self *SGoogleClient) get(id string, retval interface{}) error {
	if !strings.HasPrefix(id, getUrlPrefix()) {
		id = getUrlPrefix() + id
	}
	data, err := jsonRequest(self.client, "GET", id, nil, self.Debug)
	if err != nil {
		if strings.Index(err.Error(), "not found") > 0 {
			return cloudprovider.ErrNotFound
		}
		return errors.Wrap(err, "JSONRequest")
	}
	err = data.Unmarshal(retval)
	if err != nil {
		return errors.Wrap(err, "Unmarshal")
	}
	return nil
}

func (self *SGoogleClient) listAll(resource string, params map[string]string, retval interface{}) error {
	var (
		items  *jsonutils.JSONArray   = jsonutils.NewArray()
		_items []jsonutils.JSONObject = []jsonutils.JSONObject{}

		maxResults    int    = 500
		nextPageToken string = ""
		err           error  = nil
	)
	for {
		_items, nextPageToken, err = self._listAll(resource, params, maxResults, nextPageToken)
		if err != nil {
			return errors.Wrapf(err, `_listAll("%s")`, resource)
		}
		items.Add(_items...)
		if len(nextPageToken) == 0 || len(_items) == 0 {
			break
		}
	}
	return items.Unmarshal(retval)
}

func (self *SGoogleClient) _listAll(resource string, params map[string]string, maxResults int, pageToken string) ([]jsonutils.JSONObject, string, error) {
	if params == nil {
		params = map[string]string{}
	}
	params["maxResults"] = fmt.Sprintf("%d", maxResults)
	params["pageToken"] = pageToken
	data, err := self._list(resource, params)
	if err != nil {
		return nil, "", err
	}
	items := []jsonutils.JSONObject{}
	if data.Contains("items") {
		items, err = data.GetArray("items")
		if err != nil {
			return nil, "", errors.Wrap(err, "data.GetArray")
		}
	}
	nextPageToken, _ := data.GetString("nextPageToken")
	return items, nextPageToken, nil
}

func (self *SGoogleClient) list(resource string, params map[string]string, maxResults int, pageToken string, retval interface{}) error {
	if maxResults == 0 && len(pageToken) == 0 {
		return self.listAll(resource, params, retval)
	}
	params["maxResults"] = fmt.Sprintf("%d", maxResults)
	params["pageToken"] = pageToken
	data, err := self._list(resource, params)
	if err != nil {
		return errors.Wrapf(err, "_list(%s)", resource)
	}
	if data.Contains("items") {
		err := data.Unmarshal(retval, "items")
		if err != nil {
			return errors.Wrap(err, "data.Unmarshal")
		}
	}
	return nil
}

func jsonRequest(client *http.Client, method httputils.THttpMethod, url string, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	_, data, err := httputils.JSONRequest(client, context.Background(), method, url, nil, body, debug)
	if err != nil {
		if strings.Index(err.Error(), "not found") > 0 {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, errors.Wrap(err, "JSONRequest")
	}
	return data, nil
}

func (self *SGoogleClient) _list(resource string, params map[string]string) (jsonutils.JSONObject, error) {
	baseUrl := fmt.Sprintf("%s%s/%s", getUrlPrefix(), self.projectId, resource)
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	if len(values) > 0 {
		baseUrl = fmt.Sprintf("%s?%s", baseUrl, values.Encode())
	}
	return jsonRequest(self.client, "GET", baseUrl, nil, self.Debug)
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

func getUrlPrefix() string {
	return fmt.Sprintf("https://www.googleapis.com/compute/%s/projects/", GOOGLE_API_VERSION)
}

func getGlobalId(id string) string {
	return strings.TrimPrefix(id, getUrlPrefix())
}
