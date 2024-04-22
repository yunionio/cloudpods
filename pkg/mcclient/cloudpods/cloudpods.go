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

package cloudpods

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	CLOUD_PROVIDER_CLOUDPODS = api.CLOUD_PROVIDER_CLOUDPODS

	CLOUDPODS_DEFAULT_REGION = "default"
)

var (
	defaultParams map[string]interface{} = map[string]interface{}{
		"details":       true,
		"show_emulated": true,
		"scope":         "system",
		"cloud_env":     "onpremise",
	}
)

type ModelManager interface {
	List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error)
	Create(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Delete(session *mcclient.ClientSession, id string, param jsonutils.JSONObject) (jsonutils.JSONObject, error)
	DeleteWithParam(session *mcclient.ClientSession, id string, query jsonutils.JSONObject, body jsonutils.JSONObject) (jsonutils.JSONObject, error)
	PerformAction(session *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	Update(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
	GetKeyword() string
}

type CloudpodsClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	authURL        string
	region         string
	accessKey      string
	accessSecret   string
	adminProjectId string

	debug bool
}

func NewCloudpodsClientConfig(authURL, accessKey, accessSecret string) *CloudpodsClientConfig {
	cfg := &CloudpodsClientConfig{
		authURL:      authURL,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *CloudpodsClientConfig) Debug(debug bool) *CloudpodsClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg *CloudpodsClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *CloudpodsClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

type SCloudpodsClient struct {
	*CloudpodsClientConfig

	s *mcclient.ClientSession
}

func (self *SCloudpodsClient) auth() error {
	client := mcclient.NewClient(self.authURL, 0, self.debug, true, "", "")
	client.SetHttpTransportProxyFunc(self.cpcfg.ProxyFunc)
	ts, _ := client.GetClient().Transport.(*http.Transport)
	client.SetTransport(cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if self.cpcfg.ReadOnly {
			if req.Method == "GET" || req.Method == "HEAD" {
				return nil, nil
			}
			// 认证
			if req.Method == "POST" && req.URL.Path == "/v3/auth/tokens" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	}))
	token, err := client.AuthenticateByAccessKey(self.accessKey, self.accessSecret, "cli")
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrUnauthorized {
			return errors.Wrapf(cloudprovider.ErrInvalidAccessKey, err.Error())
		}
		return err
	}
	serviceRegion, endpoints := "", 0
	for _, region := range token.GetRegions() {
		if len(token.GetEndpoints(region, "")) > endpoints {
			serviceRegion = region
		}
	}
	endpoint := "publicURL"
	if strings.Contains(self.authURL, "api/s/identity/v3") {
		endpoint = "apigateway"
	}
	self.s = client.NewSession(context.Background(), serviceRegion, "", endpoint, token)
	if !self.s.GetToken().HasSystemAdminPrivilege() {
		return fmt.Errorf("no system admin privilege")
	}
	if self.s.GetProjectId() == self.adminProjectId {
		return fmt.Errorf("You can't manage yourself environment")
	}
	return nil
}

func NewCloudpodsClient(cfg *CloudpodsClientConfig) (*SCloudpodsClient, error) {
	cli := &SCloudpodsClient{
		CloudpodsClientConfig: cfg,
	}
	return cli, cli.auth()
}

func (self *SCloudpodsClient) GetRegion(regionId string) (*SRegion, error) {
	ret := &SRegion{cli: self}
	return ret, self.get(&compute.Cloudregions, regionId, nil, ret)
}

func (self *SCloudpodsClient) get(manager ModelManager, id string, params map[string]string, retVal interface{}) error {
	if len(id) == 0 {
		return errors.Wrap(cloudprovider.ErrNotFound, "empty id")
	}
	body := jsonutils.NewDict()
	for k, v := range params {
		body.Set(k, jsonutils.NewString(v))
	}
	resp, err := manager.Get(self.s, id, body)
	if err != nil {
		if strings.Contains(err.Error(), "NotFoundError") {
			return errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
		}
		return errors.Wrapf(err, "Get(%s)", id)
	}
	obj := resp.(*jsonutils.JSONDict)
	if manager.GetKeyword() == compute.Servers.GetKeyword() {
		obj.Remove("cdrom")
	}
	return obj.Unmarshal(retVal)
}

func (self *SCloudpodsClient) perform(manager ModelManager, id, action string, params interface{}) (jsonutils.JSONObject, error) {
	return manager.PerformAction(self.s, id, action, jsonutils.Marshal(params))
}

func (self *SCloudpodsClient) delete(manager ModelManager, id string) error {
	if len(id) == 0 {
		return nil
	}
	params := map[string]interface{}{"override_pending_delete": true}
	_, err := manager.DeleteWithParam(self.s, id, jsonutils.Marshal(params), nil)
	return err
}

func (self *SCloudpodsClient) update(manager ModelManager, id string, params interface{}) error {
	_, err := manager.Update(self.s, id, jsonutils.Marshal(params))
	return err
}

func (self *SCloudpodsClient) GetAccountId() string {
	return self.authURL
}

func (self *SCloudpodsClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	return []cloudprovider.SSubAccount{
		{
			Name:         self.cpcfg.Name,
			Account:      self.cpcfg.Account,
			HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
		},
	}, nil
}

func (self *SCloudpodsClient) GetVersion() string {
	version, err := modules.GetVersion(self.s, "compute_v2")
	if err != nil {
		return ""
	}
	return version
}

func (self *SCloudpodsClient) create(manager ModelManager, params interface{}, retVal interface{}) error {
	resp, err := manager.Create(self.s, jsonutils.Marshal(params))
	if err != nil {
		return err
	}
	obj := resp.(*jsonutils.JSONDict)
	if manager.GetKeyword() == compute.Servers.GetKeyword() {
		obj.Remove("cdrom")
	}
	return obj.Unmarshal(retVal)
}

func (self *SCloudpodsClient) list(manager ModelManager, params map[string]interface{}, retVal interface{}) error {
	if params == nil {
		params = map[string]interface{}{}
	}
	for k, v := range defaultParams {
		if _, ok := params[k]; !ok {
			params[k] = v
		}
	}
	ret := []jsonutils.JSONObject{}
	for {
		params["offset"] = len(ret)
		part, err := manager.List(self.s, jsonutils.Marshal(params))
		if err != nil {
			return errors.Wrapf(err, "list")
		}
		for i := range part.Data {
			data := part.Data[i].(*jsonutils.JSONDict)
			if manager.GetKeyword() == compute.Servers.GetKeyword() {
				data.Remove("cdrom")
			}
			ret = append(ret, data)
		}
		if len(ret) >= part.Total {
			break
		}
	}
	return jsonutils.Update(retVal, ret)
}

func (self *SCloudpodsClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	regions, err := self.GetRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		regions[i].cli = self
		if regions[i].GetGlobalId() == id {
			return &regions[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SCloudpodsClient) GetIRegions() ([]cloudprovider.ICloudRegion, error) {
	regions, err := self.GetRegions()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudRegion{}
	for i := range regions {
		regions[i].cli = self
		ret = append(ret, &regions[i])
	}
	return ret, nil
}

func (self *SCloudpodsClient) GetRegions() ([]SRegion, error) {
	ret := []SRegion{}
	return ret, self.list(&compute.Cloudregions, nil, &ret)
}

func (self *SCloudpodsClient) GetCloudRegionExternalIdPrefix() string {
	return fmt.Sprintf("%s/%s", api.CLOUD_PROVIDER_CLOUDPODS, self.cpcfg.Id)
}

func (self *SCloudpodsClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
		cloudprovider.CLOUD_CAPABILITY_EIP,
	}
}
