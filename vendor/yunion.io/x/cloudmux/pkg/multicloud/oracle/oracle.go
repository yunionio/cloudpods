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

package oracle

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"

	"github.com/oracle/oci-go-sdk/common"
)

const (
	CLOUD_PROVIDER_ORACLE_CN = "甲骨文"
	ORACLE_DEFAULT_REGION    = "ap-singapore-1"
	DEFAULT_API_VERSION      = "20160918"
	ISO8601                  = "2006-01-02T15:04:05Z"

	SERVICE_IAAS     = "iaas"
	SERVICE_IDENTITY = "identity"
)

type OracleClientConfig struct {
	cpcfg       cloudprovider.ProviderConfig
	tenancyOCID string
	userOCID    string
	compartment string
	key         *rsa.PrivateKey
	fingerprint string

	debug bool
}

type SOracleClient struct {
	*OracleClientConfig

	client *http.Client
	lock   sync.Mutex
	ctx    context.Context

	regions []SRegion
}

func NewOracleClientConfig(tenancy, user, compartment, privateKey string) (*OracleClientConfig, error) {
	cfg := &OracleClientConfig{
		tenancyOCID: tenancy,
		userOCID:    user,
		compartment: compartment,
	}
	err := cfg.parsePrivateKey(privateKey)
	if err != nil {
		return nil, errors.Wrapf(err, "parsePrivateKey %s", privateKey)
	}
	err = cfg.keyFingerprint()
	if err != nil {
		return nil, errors.Wrapf(err, "keyFingerprint")
	}
	return cfg, nil
}

func (self *OracleClientConfig) Debug(debug bool) *OracleClientConfig {
	self.debug = debug
	return self
}

func (self *OracleClientConfig) keyFingerprint() error {
	der, err := x509.MarshalPKIXPublicKey(&self.key.PublicKey)
	if err != nil {
		return errors.Wrapf(err, "MarshalPKIXPublicKey")
	}
	var ret bytes.Buffer
	fp := md5.Sum(der)
	for i, b := range fp {
		ret.WriteString(fmt.Sprintf("%02x", b))
		if i < len(fp)-1 {
			ret.WriteString(":")
		}
	}
	self.fingerprint = ret.String()
	return nil
}

func (self *OracleClientConfig) parsePrivateKey(key string) error {
	var err error
	if pemBlock, _ := pem.Decode([]byte(key)); pemBlock != nil {
		decrypted := pemBlock.Bytes
		if x509.IsEncryptedPEMBlock(pemBlock) {
			return fmt.Errorf("private key password is required for encrypted private keys")
		}

		self.key, err = x509.ParsePKCS1PrivateKey(decrypted)
		if err == nil {
			return nil
		}
		_key, err := x509.ParsePKCS8PrivateKey(decrypted)
		if err == nil {
			switch _key.(type) {
			case *rsa.PrivateKey:
				self.key = _key.(*rsa.PrivateKey)
				return nil
			default:
				return fmt.Errorf("unsupportesd private key type in PKCS8 wrapping")
			}
		}
		return err
	}
	return fmt.Errorf("failed to parse private key")
}

func (self *OracleClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *OracleClientConfig {
	self.cpcfg = cpcfg
	return self
}

func NewOracleClient(cfg *OracleClientConfig) (*SOracleClient, error) {
	client := &SOracleClient{
		OracleClientConfig: cfg,
		ctx:                context.Background(),
	}
	client.ctx = context.WithValue(client.ctx, "time", time.Now())
	_, err := client.GetRegions()
	return client, err
}

func (self *SOracleClient) GetRegions() ([]SRegion, error) {
	if len(self.regions) > 0 {
		return self.regions, nil
	}
	resp, err := self.list(SERVICE_IDENTITY, ORACLE_DEFAULT_REGION, "regions", nil)
	if err != nil {
		return nil, err
	}
	self.regions = []SRegion{}
	err = resp.Unmarshal(&self.regions)
	if err != nil {
		return nil, err
	}
	return self.regions, nil
}

func (self *SOracleClient) GetRegion(id string) (*SRegion, error) {
	regions, err := self.GetRegions()
	if err != nil {
		return nil, err
	}
	for i := range regions {
		if regions[i].Name == id {
			regions[i].client = self
			return &regions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SOracleClient) getUrl(service, regionId, resource string) (string, error) {
	if len(regionId) == 0 {
		regionId = ORACLE_DEFAULT_REGION
	}
	switch service {
	case "iaas", "identity":
		return fmt.Sprintf("https://%s.%s.oraclecloud.com/%s/%s", service, regionId, DEFAULT_API_VERSION, strings.TrimPrefix(resource, "/")), nil
	default:
		return "", errors.Wrapf(cloudprovider.ErrNotSupported, service)
	}
}

func (cli *SOracleClient) getDefaultClient() *http.Client {
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
			if req.Method == "GET" {
				return nil, nil
			}
			return nil, errors.Wrapf(cloudprovider.ErrAccountReadOnly, "%s %s", req.Method, req.URL.Path)
		}
		return nil, nil
	})
	return cli.client
}

type sOracleError struct {
	StatusCode int    `json:"statusCode"`
	RequestId  string `json:"requestId"`
	Code       string
	Message    string
}

func (self *sOracleError) Error() string {
	return jsonutils.Marshal(self).String()
}

func (self *sOracleError) ParseErrorFromJsonResponse(statusCode int, status string, body jsonutils.JSONObject) error {
	if body != nil {
		body.Unmarshal(self)
	}
	self.StatusCode = statusCode
	return self
}

func (self *SOracleClient) TenancyOCID() (string, error) {
	return self.tenancyOCID, nil
}

func (self *SOracleClient) UserOCID() (string, error) {
	return self.userOCID, nil
}

func (self *SOracleClient) KeyFingerprint() (string, error) {
	return self.fingerprint, nil
}

func (self *SOracleClient) Region() (string, error) {
	return ORACLE_DEFAULT_REGION, nil
}

func (self *SOracleClient) PrivateRSAKey() (*rsa.PrivateKey, error) {
	return self.OracleClientConfig.key, nil
}

func (self *SOracleClient) KeyID() (string, error) {
	return fmt.Sprintf("%s/%s/%s", self.tenancyOCID, self.userOCID, self.fingerprint), nil
}

func (self *SOracleClient) Do(req *http.Request) (*http.Response, error) {
	client := self.getDefaultClient()

	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	signer := common.DefaultRequestSigner(self)

	signer.Sign(req)

	return client.Do(req)
}

func (self *SOracleClient) list(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	if service == SERVICE_IAAS && len(self.compartment) > 0 {
		params["compartmentId"] = self.compartment
	}
	return self.request(httputils.GET, service, regionId, resource, params)
}

func (self *SOracleClient) post(service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return self.request(httputils.POST, service, regionId, resource, params)
}

func (self *SOracleClient) request(method httputils.THttpMethod, service, regionId, resource string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	uri, err := self.getUrl(service, regionId, resource)
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	values := url.Values{}
	if method == httputils.GET {
		for k, v := range params {
			values.Set(k, v.(string))
		}
		if len(values) > 0 {
			uri = fmt.Sprintf("%s?%s", uri, values.Encode())
		}
		params = nil
	}

	req := httputils.NewJsonRequest(method, uri, params)
	bErr := &sOracleError{}
	client := httputils.NewJsonClient(self)
	_, resp, err := client.Send(self.ctx, req, bErr, self.debug)
	return resp, err
}

func (self *SOracleClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	compartments, err := self.GetCompartments()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.SSubAccount{}
	for _, compartment := range compartments {
		if compartment.LifecycleState != "ACTIVE" {
			continue
		}
		subAccount := cloudprovider.SSubAccount{}
		subAccount.Id = compartment.Id
		subAccount.Name = compartment.Name
		subAccount.Account = fmt.Sprintf("%s/%s", self.userOCID, compartment.Id)
		subAccount.HealthStatus = api.CLOUD_PROVIDER_HEALTH_NORMAL
		ret = append(ret, subAccount)
	}
	return ret, nil
}

type Compartment struct {
	CompartmentId  string
	Id             string
	Name           string
	Description    string
	TimeCreated    time.Time
	LifecycleState string
}

func (self *SOracleClient) GetCompartments() ([]Compartment, error) {
	resp, err := self.list(SERVICE_IDENTITY, ORACLE_DEFAULT_REGION, "compartments", map[string]interface{}{"compartmentId": self.tenancyOCID})
	if err != nil {
		return nil, err
	}
	ret := []Compartment{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SOracleClient) GetAccountId() string {
	return self.tenancyOCID
}

type CashBalance struct {
	CashBalance float64
}

func (self *SOracleClient) QueryBalance() (*CashBalance, error) {
	resp, err := self.post("billing", "", "/v1/finance/cash/balance", nil)
	if err != nil {
		return nil, err
	}
	ret := &CashBalance{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SOracleClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_COMPUTE + cloudprovider.READ_ONLY_SUFFIX,
	}
	return caps
}
