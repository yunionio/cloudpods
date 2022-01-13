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

package nutanix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	NUTANIX_VERSION_V2 = "PrismGateway/services/rest/v2.0"
	NUTANIX_VERSION_V3 = "api/nutanix/v3"

	CLOUD_PROVIDER_NUTANIX = api.CLOUD_PROVIDER_NUTANIX
)

type NutanixClientConfig struct {
	cpcfg    cloudprovider.ProviderConfig
	username string
	password string
	host     string
	port     int
	debug    bool
}

func NewNutanixClientConfig(host, username, password string, port int) *NutanixClientConfig {
	cfg := &NutanixClientConfig{
		host:     host,
		username: username,
		password: password,
		port:     port,
	}
	return cfg
}

func (cfg *NutanixClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *NutanixClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *NutanixClientConfig) Debug(debug bool) *NutanixClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg NutanixClientConfig) Copy() NutanixClientConfig {
	return cfg
}

type SNutanixClient struct {
	*NutanixClientConfig
}

func NewNutanixClient(cfg *NutanixClientConfig) (*SNutanixClient, error) {
	client := &SNutanixClient{
		NutanixClientConfig: cfg,
	}
	return client, client.auth()
}

func (self *SNutanixClient) GetRegion() (*SRegion, error) {
	return &SRegion{cli: self}, nil
}

func (self *SNutanixClient) GetAccountId() string {
	return self.host
}

func (self *SNutanixClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
	}
}

func (self *SNutanixClient) auth() error {
	_, err := self.list("clusters", nil, nil)
	return err
}

func (self *SNutanixClient) getBaseDomain() string {
	return fmt.Sprintf("https://%s:%d/%s", self.host, self.port, NUTANIX_VERSION_V2)
}

func (cli *SNutanixClient) getDefaultClient() *http.Client {
	client := httputils.GetDefaultClient()
	proxy := func(req *http.Request) (*url.URL, error) {
		req.SetBasicAuth(cli.username, cli.password)
		if cli.cpcfg.ProxyFunc != nil {
			cli.cpcfg.ProxyFunc(req)
		}
		return nil, nil
	}
	httputils.SetClientProxyFunc(client, proxy)
	return client
}

func (self *SNutanixClient) _list(res string, params url.Values) (jsonutils.JSONObject, error) {
	url := fmt.Sprintf("%s/%s", self.getBaseDomain(), res)
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	return self.jsonRequest(httputils.GET, url, nil)
}

func (self *SNutanixClient) list(res string, params url.Values, retVal interface{}) (int, error) {
	resp, err := self._list(res, params)
	if err != nil {
		return 0, errors.Wrapf(err, "get %s", res)
	}
	if retVal != nil {
		err = resp.Unmarshal(retVal, "entities")
		if err != nil {
			return 0, errors.Wrapf(err, "resp.Unmarshal")
		}
	}
	total, err := resp.Int("metadata", "total_entities")
	if err != nil {
		return 0, errors.Wrapf(err, "get metadata total_entities")
	}
	return int(total), nil
}

func (self *SNutanixClient) listAll(res string, params url.Values, retVal interface{}) error {
	if len(params) == 0 {
		params = url.Values{}
	}
	entities := []jsonutils.JSONObject{}
	page, count := 1, 1024
	for {
		params.Set("count", fmt.Sprintf("%d", count))
		params.Set("page", fmt.Sprintf("%d", page))
		resp, err := self._list(res, params)
		if err != nil {
			return errors.Wrapf(err, "list %s", res)
		}
		_entities, err := resp.GetArray("entities")
		if err != nil {
			return errors.Wrapf(err, "resp get entities")
		}
		entities = append(entities, _entities...)
		totalEntities, err := resp.Int("metadata", "total_entities")
		if err != nil {
			return errors.Wrapf(err, "get resp total_entities")
		}
		if int64(page*count) >= totalEntities {
			break
		}
		page++
	}
	return jsonutils.Update(retVal, entities)
}

func (self *SNutanixClient) get(res string, id string, params url.Values, retVal interface{}) error {
	url := fmt.Sprintf("%s/%s/%s", self.getBaseDomain(), res, id)
	if len(params) > 0 {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
	}
	resp, err := self.jsonRequest(httputils.GET, url, nil)
	if err != nil {
		return errors.Wrapf(err, "get %s/%s", res, id)
	}
	if retVal != nil {
		return resp.Unmarshal(retVal)
	}
	return nil
}

func (self *SNutanixClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Account:      self.username,
		Name:         self.cpcfg.Name,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (self *SNutanixClient) jsonRequest(method httputils.THttpMethod, url string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	client := self.getDefaultClient()
	return _jsonRequest(client, method, url, nil, body, self.debug)
}

func _jsonRequest(cli *http.Client, method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject, debug bool) (jsonutils.JSONObject, error) {
	_, resp, err := httputils.JSONRequest(cli, context.Background(), method, url, header, body, debug)
	return resp, err
}

func (self *SNutanixClient) GetIRegions() []cloudprovider.ICloudRegion {
	region := &SRegion{cli: self}
	return []cloudprovider.ICloudRegion{region}
}
