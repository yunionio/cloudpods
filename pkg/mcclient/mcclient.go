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

package mcclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var listenerWorker *appsrv.SWorkerManager

type Client struct {
	authUrl string
	timeout int
	debug   bool

	httpconn        *http.Client
	_serviceCatalog IServiceCatalog

	catalogListeners []IServiceCatalogChangeListener
}

func init() {
	listenerWorker = appsrv.NewWorkerManager("client_catalog_listener_worker", 1, 2048, false)
}

func NewClient(authUrl string, timeout int, debug bool, insecure bool, certFile, keyFile string) *Client {
	var tlsConf *tls.Config

	if len(certFile) > 0 && len(keyFile) > 0 {
		var err error
		tlsConf, err = seclib2.InitTLSConfig(certFile, keyFile)
		if err != nil {
			log.Errorf("load TLS failed %s", err)
		}
	}

	if tlsConf == nil || gotypes.IsNil(tlsConf) {
		tlsConf = &tls.Config{}
	}
	tlsConf.InsecureSkipVerify = insecure

	tr := httputils.GetTransport(insecure)
	tr.TLSClientConfig = tlsConf
	tr.IdleConnTimeout = 5 * time.Second
	tr.TLSHandshakeTimeout = 10 * time.Second
	tr.ResponseHeaderTimeout = 0

	client := Client{authUrl: authUrl,
		timeout: timeout,
		debug:   debug,
		httpconn: &http.Client{
			Transport: tr,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}, // 不自动处理重定向请求
		},
	}

	return &client
}

func (client *Client) HttpClient() *http.Client {
	return client.httpconn
}

func (client *Client) SetHttpTransportProxyFunc(proxyFunc httputils.TransportProxyFunc) {
	httputils.SetClientProxyFunc(client.httpconn, proxyFunc)
}

func (client *Client) GetClient() *http.Client {
	return client.httpconn
}

func (client *Client) SetTransport(ts http.RoundTripper) {
	client.httpconn.Transport = ts
}

func (client *Client) SetDebug(debug bool) {
	client.debug = debug
}

func (client *Client) GetDebug() bool {
	return client.debug
}

func (client *Client) AuthVersion() string {
	pos := strings.LastIndexByte(client.authUrl, '/')
	if pos > 0 {
		return client.authUrl[pos+1:]
	} else {
		return ""
	}
}

func (client *Client) NewAuthTokenCredential() TokenCredential {
	if client.AuthVersion() == "v3" {
		return &TokenCredentialV3{}
	}
	return &TokenCredentialV2{}
}

func getDefaultHeader(header http.Header, token string) http.Header {
	if len(token) > 0 {
		if header == nil {
			header = http.Header{}
		}
		if len(header.Get(AUTH_TOKEN)) == 0 {
			header.Add(AUTH_TOKEN, token)
		}
	}
	return header
}

func joinUrl(baseUrl, path string) string {
	base, version := SplitVersionedURL(baseUrl)
	if len(version) > 0 {
		if strings.HasPrefix(path, fmt.Sprintf("/%s/", version)) {
			baseUrl = base
		}
	}
	return fmt.Sprintf("%s%s", baseUrl, path)
}

func FixContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	srvType := consts.GetServiceType()
	if len(srvType) > 0 && len(appctx.AppContextServiceName(ctx)) == 0 {
		ctx = context.WithValue(ctx, appctx.APP_CONTEXT_KEY_APPNAME, srvType)
	}
	return ctx
}

func (client *Client) rawRequest(ctx context.Context, endpoint string, token string, method httputils.THttpMethod, url string, header http.Header, body io.Reader) (*http.Response, error) {
	ctx = FixContext(ctx)
	return httputils.Request(client.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, client.debug)
}

func (client *Client) jsonRequest(ctx context.Context, endpoint string, token string, method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	ctx = FixContext(ctx)
	return httputils.JSONRequest(client.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, client.debug)
}

func (client *Client) _authV3(domainName, uname, passwd, projectId, projectName, projectDomain, token string, aCtx SAuthContext) (TokenCredential, error) {
	input := SAuthenticationInputV3{}
	if len(uname) > 0 && len(passwd) > 0 { // Password authentication
		input.Auth.Identity.Methods = []string{api.AUTH_METHOD_PASSWORD}
		input.Auth.Identity.Password.User.Name = uname
		input.Auth.Identity.Password.User.Password = passwd
		if len(domainName) > 0 {
			input.Auth.Identity.Password.User.Domain.Name = domainName
		}
		// else {
		//	input.Auth.Identity.Password.User.Domain.Name = api.DEFAULT_DOMAIN_ID
		//}
	} else if len(token) > 0 {
		input.Auth.Identity.Methods = []string{api.AUTH_METHOD_TOKEN}
		input.Auth.Identity.Token.Id = token
	}
	if len(projectId) > 0 {
		input.Auth.Scope.Project.Id = projectId
	}
	if len(projectName) > 0 {
		input.Auth.Scope.Project.Name = projectName
		if len(projectDomain) > 0 {
			input.Auth.Scope.Project.Domain.Name = projectDomain
		}
		// else {
		// 	input.Auth.Scope.Project.Domain.Id = api.DEFAULT_DOMAIN_ID
		// }
	}
	input.Auth.Context = aCtx
	return client._authV3Input(input)
}

func (client *Client) _authV3Input(input SAuthenticationInputV3) (TokenCredential, error) {
	hdr, rbody, err := client.jsonRequest(context.Background(), client.authUrl, "", "POST", "/auth/tokens", nil, jsonutils.Marshal(&input))
	if err != nil {
		return nil, err
	}

	tokenId := hdr.Get("X-Subject-Token")
	if len(tokenId) == 0 {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "No X-Subject-Token in header")
	}
	ret, err := client.unmarshalV3Token(rbody, tokenId)
	return ret, err
}

func (client *Client) _authV2(uname, passwd, tenantId, tenantName, token string, aCtx SAuthContext) (TokenCredential, error) {
	input := SAuthenticationInputV2{}
	input.Auth.PasswordCredentials.Username = uname
	input.Auth.PasswordCredentials.Password = passwd
	if len(tenantName) > 0 {
		input.Auth.TenantName = tenantName
	}
	if len(tenantId) > 0 {
		input.Auth.TenantId = tenantId
	}
	if len(token) > 0 {
		input.Auth.Token.Id = token
	}
	input.Auth.Context = aCtx
	_, rbody, err := client.jsonRequest(context.Background(), client.authUrl, "", "POST", "/tokens", nil, jsonutils.Marshal(&input))
	if err != nil {
		return nil, err
	}
	return client.unmarshalV2Token(rbody)
}

func (client *Client) Authenticate(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return client.AuthenticateApi(uname, passwd, domainName, tenantName, tenantDomain)
}

func (client *Client) AuthenticateApi(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return client.AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain, AuthSourceAPI)
}

func (client *Client) AuthenticateWeb(uname, passwd, domainName, tenantName, tenantDomain string, cliIp string) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: AuthSourceWeb,
		Ip:     cliIp,
	}
	return client.authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain, aCtx)
}

func (client *Client) AuthenticateOperator(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return client.AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain, AuthSourceOperator)
}

func (client *Client) AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain string, source string) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: source,
	}
	return client.authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain, aCtx)
}

func (client *Client) authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain string, aCtx SAuthContext) (TokenCredential, error) {
	if client.AuthVersion() == "v3" {
		return client._authV3(domainName, uname, passwd, "", tenantName, tenantDomain, "", aCtx)
	}
	return client._authV2(uname, passwd, "", tenantName, "", aCtx)
}

func (client *Client) unmarshalV3Token(rbody jsonutils.JSONObject, tokenId string) (cred TokenCredential, err error) {
	cred = &TokenCredentialV3{Id: tokenId}
	err = rbody.Unmarshal(cred)
	if err != nil {
		err = errors.Wrap(err, "Invalid response when unmarshal V3 Token")
	}
	cata := cred.GetServiceCatalog()
	if cata == nil || cata.Len() == 0 {
		log.Warningf("No service catalog avaiable")
	} else {
		client.SetServiceCatalog(cata)
	}
	return
}

func (client *Client) unmarshalV2Token(rbody jsonutils.JSONObject) (cred TokenCredential, err error) {
	access, err := rbody.Get("access")
	if err == nil {
		cred = &TokenCredentialV2{}
		err = access.Unmarshal(cred)
		if err != nil {
			err = errors.Wrap(err, "Invalid response when unmarshal V2 Token")
		}
		cata := cred.GetServiceCatalog()
		if cata == nil || cata.Len() == 0 {
			log.Warningf("No srvice catalog avaiable")
		} else {
			client.SetServiceCatalog(cata)
		}
		return
	}
	err = errors.Wrap(httperrors.ErrInvalidFormat, "Invalid response: no access object")
	return
}

func (client *Client) verifyV3(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	header.Add(api.AUTH_SUBJECT_TOKEN_HEADER, token)
	_, rbody, err := client.jsonRequest(context.Background(), client.authUrl, "", "GET", "/auth/tokens", header, nil)
	if err != nil {
		return nil, err
	}
	return client.unmarshalV3Token(rbody, token)
}

func (client *Client) verifyV2(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	verifyUrl := fmt.Sprintf("/tokens/%s", token)
	_, rbody, err := client.jsonRequest(context.Background(), client.authUrl, "", "GET", verifyUrl, header, nil)
	if err != nil {
		return nil, err
	}
	return client.unmarshalV2Token(rbody)
}

func (client *Client) Verify(adminToken, token string) (cred TokenCredential, err error) {
	if client.AuthVersion() == "v3" {
		return client.verifyV3(adminToken, token)
	}
	return client.verifyV2(adminToken, token)
}

func (client *Client) Invalidate(ctx context.Context, adminToken, token string) error {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	header.Add(api.AUTH_SUBJECT_TOKEN_HEADER, token)
	_, _, err := client.jsonRequest(ctx, client.authUrl, "", "DELETE", "/auth/tokens", header, nil)
	if err != nil {
		return errors.Wrap(err, "jsonRequest")
	}
	return nil
}

func (client *Client) FetchInvalidTokens(ctx context.Context, adminToken string) ([]string, error) {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	_, resp, err := client.jsonRequest(ctx, client.authUrl, "", "GET", "/auth/tokens/invalid", header, nil)
	if err != nil {
		return nil, errors.Wrap(err, "jsonRequest")
	}
	tokens := make([]string, 0)
	err = resp.Unmarshal(&tokens, "tokens")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return tokens, nil
}

func (client *Client) SetTenant(tenantId, tenantName, tenantDomain string, token TokenCredential) (TokenCredential, error) {
	return client.SetProject(tenantId, tenantName, tenantDomain, token)
}

func (client *Client) AuthenticateToken(token string, projName, projDomain string, source string) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: source,
	}
	if client.AuthVersion() == "v3" {
		return client._authV3("", "", "", "", projName, projDomain, token, aCtx)
	} else {
		return client._authV2("", "", "", projName, token, aCtx)
	}
}

func (client *Client) SetProject(tenantId, tenantName, tenantDomain string, token TokenCredential) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: token.GetLoginSource(),
		Ip:     token.GetLoginIp(),
	}
	if client.AuthVersion() == "v3" {
		return client._authV3("", "", "", tenantId, tenantName, tenantDomain, token.GetTokenString(), aCtx)
	} else {
		return client._authV2("", "", "", tenantName, token.GetTokenString(), aCtx)
	}
}

func (client *Client) GetCommonEtcdEndpoint(token TokenCredential, region, interfaceType string) (*api.EndpointDetails, error) {
	if client.AuthVersion() != "v3" {
		return nil, errors.Errorf("current version %s not support get internal etcd endpoint", client.AuthVersion())
	}

	_, err := client.GetServiceCatalog().getServiceURL(apis.SERVICE_TYPE_ETCD, region, "", interfaceType)
	if err != nil {
		return nil, err
	}

	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(interfaceType), "interface")
	params.Add(jsonutils.JSONTrue, "enabled")
	params.Add(jsonutils.NewString(apis.SERVICE_TYPE_ETCD), "service")
	params.Add(jsonutils.JSONTrue, "details")
	params.Add(jsonutils.NewString(region), "region")

	epUrl := "/endpoints?" + params.QueryString()
	_, rbody, err := client.jsonRequest(context.Background(), client.authUrl, token.GetTokenString(), httputils.GET, epUrl, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "get internal etcd endpoint")
	}
	rets, err := rbody.GetArray("endpoints")
	if err != nil {
		return nil, errors.Wrap(err, "get endpoints response")
	}
	if len(rets) == 0 {
		return nil, errors.Wrapf(httperrors.ErrNotFound, "not found service %s %s endpoint", apis.SERVICE_TYPE_ETCD, interfaceType)
	}
	if len(rets) > 1 {
		return nil, errors.Errorf("fond %d duplicate serivce %s %s endpoint", len(rets), apis.SERVICE_TYPE_ETCD, interfaceType)
	}
	endpoint := new(api.EndpointDetails)
	if err := rets[0].Unmarshal(endpoint); err != nil {
		return nil, errors.Wrap(err, "unmarshal endpoint")
	}
	return endpoint, nil
}

func (client *Client) GetCommonEtcdTLSConfig(endpoint *api.EndpointDetails) (*tls.Config, error) {
	if endpoint.CertId == "" {
		return nil, nil
	}
	caData := []byte(endpoint.CaCertificate)
	certData := []byte(endpoint.Certificate)
	keyData := []byte(endpoint.PrivateKey)
	return seclib2.InitTLSConfigByData(caData, certData, keyData)
}

func (client *Client) NewSession(ctx context.Context, region, zone, endpointType string, token TokenCredential) *ClientSession {
	cata := token.GetServiceCatalog()
	if client.GetServiceCatalog() == nil {
		if cata == nil || cata.Len() == 0 {
			log.Warningf("Missing service catalog in token")
		} else {
			client.SetServiceCatalog(cata)
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &ClientSession{
		ctx:                 ctx,
		client:              client,
		region:              region,
		zone:                zone,
		endpointType:        endpointType,
		token:               token,
		Header:              http.Header{},
		customizeServiceUrl: map[string]string{},
	}
}

type SCheckPoliciesInput struct {
	UserId    string
	ProjectId string
	LoginIp   string
}

type SFetchMatchPoliciesOutput struct {
	Names    map[rbacscope.TRbacScope][]string `json:"names"`
	Policies rbacutils.TPolicyGroup            `json:"policies"`
}

func (o *SFetchMatchPoliciesOutput) Decode(object jsonutils.JSONObject) error {
	err := object.Unmarshal(&o.Names, "names")
	if err != nil {
		return errors.Wrap(err, "unmarshal names")
	}
	pData, err := object.Get("policies")
	if err != nil {
		return errors.Wrap(err, "Get policies")
	}
	o.Policies, err = rbacutils.DecodePolicyGroup(pData)
	if err != nil {
		return errors.Wrap(err, "DecodePolicyGroup")
	}
	return nil
}

func (o SFetchMatchPoliciesOutput) Encode() jsonutils.JSONObject {
	output := jsonutils.NewDict()
	output.Set("names", jsonutils.Marshal(o.Names))
	output.Set("policies", o.Policies.Encode())
	return output
}

func (client *Client) FetchMatchPolicies(ctx context.Context, token TokenCredential) (*SFetchMatchPoliciesOutput, error) {
	header := http.Header{}
	if token.GetTokenString() != "" {
		header.Add(api.AUTH_TOKEN_HEADER, token.GetTokenString())
	}
	_, rbody, err := client.jsonRequest(ctx, client.authUrl, "", "GET", "/auth/policies", header, nil)
	if err != nil {
		return nil, errors.Wrap(err, "jsonRequest")
	}
	output := &SFetchMatchPoliciesOutput{}
	err = output.Decode(rbody)
	if err != nil {
		return nil, errors.Wrap(err, "SFetchMatchPoliciesOutput.Decode")
	}
	return output, nil
}

func (client *Client) CheckMatchPolicies(ctx context.Context, adminToken TokenCredential, input SCheckPoliciesInput) (*SFetchMatchPoliciesOutput, error) {
	_, rbody, err := client.jsonRequest(ctx, client.authUrl, adminToken.GetTokenString(), "POST", "/auth/policies", nil, jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "jsonRequest")
	}
	output := &SFetchMatchPoliciesOutput{}
	err = output.Decode(rbody)
	if err != nil {
		return nil, errors.Wrap(err, "SFetchMatchPoliciesOutput.Decode")
	}
	return output, nil
}
