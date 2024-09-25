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

package cephfs

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_CEPHFS = api.CLOUD_PROVIDER_CEPHFS
)

type CephFSClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	host     string
	port     int
	username string
	password string
	fsId     string

	debug bool
}

func (cfg *CephFSClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *CephFSClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *CephFSClientConfig) Debug(debug bool) *CephFSClientConfig {
	cfg.debug = debug
	return cfg
}

func NewCephFSClientConfig(host string, port int, username, password, fsId string) *CephFSClientConfig {
	cfg := &CephFSClientConfig{
		host: host,
		port: port,

		username: username,
		password: password,

		fsId: fsId,
	}
	return cfg
}

type SCephFSClient struct {
	*CephFSClientConfig

	cloudprovider.SFakeOnPremiseRegion
	multicloud.SNoObjectStorageRegion
	multicloud.SRegion

	lock   sync.Mutex
	client *http.Client

	token string
}

func NewCephFSClient(cfg *CephFSClientConfig) (*SCephFSClient, error) {
	client := &SCephFSClient{
		CephFSClientConfig: cfg,
	}
	return client, client.auth()
}

func (cli *SCephFSClient) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(cli.GetName()).CN(cli.GetName())
	return table
}

func (cli *SCephFSClient) getDefaultClient() *http.Client {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if !gotypes.IsNil(cli.client) {
		return cli.client
	}
	cli.client = httputils.GetAdaptiveTimeoutClient()
	httputils.SetClientProxyFunc(cli.client, cli.cpcfg.ProxyFunc)
	ts, _ := cli.client.Transport.(*http.Transport)
	ts.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	cli.client.Transport = cloudprovider.GetCheckTransport(ts, func(req *http.Request) (func(resp *http.Response) error, error) {
		if cli.cpcfg.ReadOnly {
			if req.Method == "GET" || strings.HasSuffix(req.URL.Path, "/auth") {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return cli.client
}

func (cli *SCephFSClient) baseUrl() string {
	protocol := "http"
	if strings.Contains(fmt.Sprintf("%d", cli.port), "443") {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d/api", protocol, cli.host, cli.port)
}

func (cli *SCephFSClient) auth() error {
	client := cli.getDefaultClient()
	url := fmt.Sprintf("%s/auth", cli.baseUrl())
	header := http.Header{}
	header.Set("Accept", "application/vnd.ceph.api.v1.0+json")
	header.Set("Accept-Encoding", "gzip")
	body := jsonutils.Marshal(map[string]interface{}{
		"username": cli.username,
		"password": cli.password,
	})
	_, resp, err := httputils.JSONRequest(client, context.Background(), httputils.POST, url, header, body, cli.debug)
	if err != nil {
		return errors.Wrapf(err, "auth")
	}
	cli.token, err = resp.GetString("token")
	return err
}

func (cli *SCephFSClient) GetCapabilities() []string {
	return []string{
		cloudprovider.CLOUD_CAPABILITY_NAS,
	}
}

func (cli *SCephFSClient) list(res string, params url.Values) (jsonutils.JSONObject, error) {
	client := cli.getDefaultClient()
	url := fmt.Sprintf("%s/%s", cli.baseUrl(), res)
	if len(params) > 0 {
		url = fmt.Sprintf("%s/%s?%s", cli.baseUrl(), res, params.Encode())
	}
	header := http.Header{}
	header.Set("Accept", "application/vnd.ceph.api.v1.0+json")
	header.Set("Accept-Encoding", "gzip")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", cli.token))
	_, resp, err := httputils.JSONRequest(client, context.Background(), httputils.GET, url, header, nil, cli.debug)
	return resp, err
}

func (cli *SCephFSClient) post(res string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := cli.getDefaultClient()
	url := fmt.Sprintf("%s/%s", cli.baseUrl(), res)
	header := http.Header{}
	header.Set("Accept", "application/vnd.ceph.api.v1.0+json")
	header.Set("Accept-Encoding", "gzip")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", cli.token))
	_, resp, err := httputils.JSONRequest(client, context.Background(), httputils.POST, url, header, jsonutils.Marshal(params), cli.debug)
	return resp, err
}

func (cli *SCephFSClient) delete(res string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := cli.getDefaultClient()
	url := fmt.Sprintf("%s/%s", cli.baseUrl(), res)
	header := http.Header{}
	header.Set("Accept", "application/vnd.ceph.api.v1.0+json")
	header.Set("Accept-Encoding", "gzip")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", cli.token))
	_, resp, err := httputils.JSONRequest(client, context.Background(), httputils.DELETE, url, header, jsonutils.Marshal(params), cli.debug)
	return resp, err
}

func (cli *SCephFSClient) put(res string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	client := cli.getDefaultClient()
	url := fmt.Sprintf("%s/%s", cli.baseUrl(), res)
	header := http.Header{}
	header.Set("Accept", "application/vnd.ceph.api.v1.0+json")
	header.Set("Accept-Encoding", "gzip")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", cli.token))
	_, resp, err := httputils.JSONRequest(client, context.Background(), httputils.PUT, url, header, jsonutils.Marshal(params), cli.debug)
	return resp, err
}

func (cli *SCephFSClient) GetProvider() string {
	return api.CLOUD_PROVIDER_CEPHFS
}

func (cli *SCephFSClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	fss, err := cli.GetCephFSs()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.SSubAccount{}
	for _, fs := range fss {
		subAccount := cloudprovider.SSubAccount{
			Id:           cli.cpcfg.Id,
			Account:      fmt.Sprintf("%s/%s", cli.username, fs.Id),
			Name:         fs.Mdsmap.FsName,
			HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
		}
		ret = append(ret, subAccount)
	}
	return ret, nil
}

func (cli *SCephFSClient) GetAccountId() string {
	return fmt.Sprintf("%s@%s:%d", cli.username, cli.host, cli.port)
}
