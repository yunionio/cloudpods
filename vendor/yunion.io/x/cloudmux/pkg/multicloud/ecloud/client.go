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

package ecloud

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_ECLOUD    = api.CLOUD_PROVIDER_ECLOUD
	CLOUD_PROVIDER_ECLOUD_CN = "移动云"
	CLOUD_PROVIDER_ECLOUD_EN = "Ecloud"
	CLOUD_API_VERSION        = "2016-12-05"

	ECLOUD_DEFAULT_REGION = "beijing-1"
)

type SEcloudClientConfig struct {
	cpcfg  cloudprovider.ProviderConfig
	signer ISigner

	debug bool
}

func NewEcloudClientConfig(signer ISigner) *SEcloudClientConfig {
	cfg := &SEcloudClientConfig{
		signer: signer,
	}
	return cfg
}

func (cfg *SEcloudClientConfig) SetCloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *SEcloudClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *SEcloudClientConfig) SetDebug(debug bool) *SEcloudClientConfig {
	cfg.debug = debug
	return cfg
}

type SEcloudClient struct {
	*SEcloudClientConfig

	httpClient *http.Client
	iregions   []cloudprovider.ICloudRegion
}

func NewEcloudClient(cfg *SEcloudClientConfig) (*SEcloudClient, error) {
	httpClient := cfg.cpcfg.AdaptiveTimeoutHttpClient()
	return &SEcloudClient{
		SEcloudClientConfig: cfg,
		httpClient:          httpClient,
	}, nil
}

func (self *SEcloudClient) GetAccessEnv() string {
	return api.CLOUD_ACCESS_ENV_ECLOUD_CHINA
}

func (ec *SEcloudClient) fetchRegions() {
	regions := make([]SRegion, 0, len(regionList))
	for id, name := range regionList {
		region := SRegion{}
		region.ID = id
		region.Name = name
		region.client = ec
		regions = append(regions, region)
	}
	iregions := make([]cloudprovider.ICloudRegion, len(regions))
	for i := range iregions {
		iregions[i] = &regions[i]
	}
	ec.iregions = iregions
	return
}

func (ec *SEcloudClient) TryConnect() error {
	iregions := ec.GetIRegions()
	if len(iregions) == 0 {
		return fmt.Errorf("no invalid region for ecloud")
	}
	_, err := iregions[0].GetIZones()
	if err != nil {
		return errors.Wrap(err, "try to connect failed")
	}
	return nil
}

func (ec *SEcloudClient) GetIRegions() []cloudprovider.ICloudRegion {
	if ec.iregions == nil {
		ec.fetchRegions()
	}
	return ec.iregions
}

func (ec *SEcloudClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	iregions := ec.GetIRegions()
	for i := range iregions {
		if iregions[i].GetGlobalId() == id {
			return iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (ec *SEcloudClient) GetRegionById(id string) (*SRegion, error) {
	iregions := ec.GetIRegions()
	for i := range iregions {
		if iregions[i].GetId() == id {
			return iregions[i].(*SRegion), nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (ec *SEcloudClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_NETWORK + cloudprovider.READ_ONLY_SUFFIX,
		cloudprovider.CLOUD_CAPABILITY_EIP + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}

func (ec *SEcloudClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{}
	subAccount.Name = ec.cpcfg.Name
	subAccount.Account = ec.signer.GetAccessKeyId()
	subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (ec *SEcloudClient) GetAccountId() string {
	return ec.signer.GetAccessKeyId()
}

func (ec *SEcloudClient) GetCloudRegionExternalIdPrefix() string {
	return CLOUD_PROVIDER_ECLOUD
}

func (ec *SEcloudClient) completeSingParams(request IRequest) (err error) {
	queryParams := request.GetQueryParams()
	queryParams["AccessKey"] = ec.signer.GetAccessKeyId()
	queryParams["Version"] = request.GetVersion()
	queryParams["Timestamp"] = request.GetTimestamp()
	queryParams["SignatureMethod"] = ec.signer.GetName()
	queryParams["SignatureVersion"] = ec.signer.GetVersion()
	queryParams["SignatureNonce"] = ec.signer.GetNonce()
	return
}

func (ec *SEcloudClient) buildStringToSign(request IRequest) string {
	signParams := request.GetQueryParams()
	queryString := getUrlFormedMap(signParams)
	queryString = strings.Replace(queryString, "+", "%20", -1)
	queryString = strings.Replace(queryString, "*", "%2A", -1)
	queryString = strings.Replace(queryString, "%7E", "~", -1)
	shaString := sha256.Sum256([]byte(queryString))
	summaryQuery := hex.EncodeToString(shaString[:])
	serverPath := strings.Replace(request.GetServerPath(), "/", "%2F", -1)
	return fmt.Sprintf("%s\n%s\n%s", request.GetMethod(), serverPath, summaryQuery)
}

func (ec *SEcloudClient) doGet(ctx context.Context, r IRequest, result interface{}) error {
	r.SetMethod("GET")
	data, err := ec.request(ctx, r)
	if err != nil {
		return err
	}
	return data.Unmarshal(result)
}

func (ec *SEcloudClient) doList(ctx context.Context, r IRequest, result interface{}) error {
	r.SetMethod("GET")
	// TODO Paging query
	data, err := ec.request(ctx, r)
	if err != nil {
		return err
	}
	var (
		datas *jsonutils.JSONArray
		ok    bool
	)

	if datas, ok = data.(*jsonutils.JSONArray); !ok {
		if !data.Contains("content") {
			return ErrMissKey{
				Key: "content",
				Jo:  data,
			}
		}
		content, _ := data.Get("content")
		datas, ok = content.(*jsonutils.JSONArray)
		if !ok {
			return fmt.Errorf("The return result should be an array, but:\n%s", content)
		}
	}
	return datas.Unmarshal(result)
}

func (ec *SEcloudClient) request(ctx context.Context, r IRequest) (jsonutils.JSONObject, error) {
	jrbody, err := ec.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	return r.ForMateResponseBody(jrbody)
}

func (ec *SEcloudClient) doRequest(ctx context.Context, r IRequest) (jsonutils.JSONObject, error) {
	// sign
	ec.completeSingParams(r)
	signature := ec.signer.Sign(ec.buildStringToSign(r), "BC_SIGNATURE&")
	query := r.GetQueryParams()
	query["Signature"] = signature
	header := r.GetHeaders()
	header["Content-Type"] = "application/json"
	var urlStr string
	port := r.GetPort()
	if len(port) > 0 {
		urlStr = fmt.Sprintf("%s://%s:%s%s", r.GetScheme(), r.GetEndpoint(), port, r.GetServerPath())
	} else {
		urlStr = fmt.Sprintf("%s://%s%s", r.GetScheme(), r.GetEndpoint(), r.GetServerPath())
	}
	queryString := getUrlFromedMapUnescaped(r.GetQueryParams())
	if len(queryString) > 0 {
		urlStr = urlStr + "?" + queryString
	}
	resp, err := httputils.Request(
		ec.httpClient,
		ctx,
		httputils.THttpMethod(r.GetMethod()),
		urlStr,
		convertHeader(header),
		r.GetBodyReader(),
		ec.debug,
	)
	defer httputils.CloseResponse(resp)
	if err != nil {
		return nil, err
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read body of response")
	}
	if ec.debug {
		fmt.Fprintf(os.Stderr, "Response body: %s\n", string(rbody))
	}
	rbody = bytes.TrimSpace(rbody)

	var jrbody jsonutils.JSONObject
	if len(rbody) > 0 && (rbody[0] == '{' || rbody[0] == '[') {
		var err error
		jrbody, err = jsonutils.Parse(rbody)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to parsing json: %s", rbody)
		}
	}
	return jrbody, nil
}

type ErrMissKey struct {
	Key string
	Jo  jsonutils.JSONObject
}

func (mk ErrMissKey) Error() string {
	return fmt.Sprintf("The response body should contain the %q key, but it doesn't. It is:\n%s", mk.Key, mk.Jo)
}

func convertHeader(mh map[string]string) http.Header {
	header := http.Header{}
	for k, v := range mh {
		header.Add(k, v)
	}
	return header
}

func getUrlFromedMapUnescaped(source map[string]string) string {
	kvs := make([]string, 0, len(source))
	for k, v := range source {
		kvs = append(kvs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(kvs, "&")
}

func getUrlFormedMap(source map[string]string) (urlEncoded string) {
	urlEncoder := url.Values{}
	for key, value := range source {
		urlEncoder.Add(key, value)
	}
	urlEncoded = urlEncoder.Encode()
	return
}
