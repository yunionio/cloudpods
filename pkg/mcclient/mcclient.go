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
	"net"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type Client struct {
	authUrl string
	timeout int
	debug   bool

	httpconn       *http.Client
	serviceCatalog IServiceCatalog
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

	tr := &http.Transport{
		TLSClientConfig: tlsConf,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

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

func (this *Client) SetDebug(debug bool) {
	this.debug = debug
}

func (this *Client) AuthVersion() string {
	pos := strings.LastIndexByte(this.authUrl, '/')
	if pos > 0 {
		return this.authUrl[pos+1:]
	} else {
		return ""
	}
}

func (this *Client) SetServiceCatalog(catalog IServiceCatalog) {
	this.serviceCatalog = catalog
}

func (this *Client) GetServiceCatalog() IServiceCatalog {
	return this.serviceCatalog
}

func (this *Client) NewAuthTokenCredential() TokenCredential {
	if this.AuthVersion() == "v3" {
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

func (this *Client) rawRequest(ctx context.Context, endpoint string, token string, method httputils.THttpMethod, url string, header http.Header, body io.Reader) (*http.Response, error) {
	return httputils.Request(this.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, this.debug)
}

func (this *Client) jsonRequest(ctx context.Context, endpoint string, token string, method httputils.THttpMethod, url string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return httputils.JSONRequest(this.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, this.debug)
}

func (this *Client) _authV3(domainName, uname, passwd, projectId, projectName, projectDomain, token string, aCtx SAuthContext) (TokenCredential, error) {
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
	hdr, rbody, err := this.jsonRequest(context.Background(), this.authUrl, "", "POST", "/auth/tokens", nil, jsonutils.Marshal(&input))
	if err != nil {
		return nil, err
	}

	tokenId := hdr.Get("X-Subject-Token")
	if len(tokenId) == 0 {
		return nil, fmt.Errorf("No X-Subject-Token in header")
	}
	ret, err := this.unmarshalV3Token(rbody, tokenId)
	return ret, err
}

func (this *Client) _authV2(uname, passwd, tenantId, tenantName, token string, aCtx SAuthContext) (TokenCredential, error) {
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
	_, rbody, err := this.jsonRequest(context.Background(), this.authUrl, "", "POST", "/tokens", nil, jsonutils.Marshal(&input))
	if err != nil {
		return nil, err
	}
	return this.unmarshalV2Token(rbody)
}

func (this *Client) Authenticate(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return this.AuthenticateApi(uname, passwd, domainName, tenantName, tenantDomain)
}

func (this *Client) AuthenticateApi(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return this.AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain, AuthSourceAPI)
}

func (this *Client) AuthenticateWeb(uname, passwd, domainName, tenantName, tenantDomain string, cliIp string) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: AuthSourceWeb,
		Ip:     cliIp,
	}
	return this.authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain, aCtx)
}

func (this *Client) AuthenticateOperator(uname, passwd, domainName, tenantName, tenantDomain string) (TokenCredential, error) {
	return this.AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain, AuthSourceOperator)
}

func (this *Client) AuthenticateWithSource(uname, passwd, domainName, tenantName, tenantDomain string, source string) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: source,
	}
	return this.authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain, aCtx)
}

func (this *Client) authenticateWithContext(uname, passwd, domainName, tenantName, tenantDomain string, aCtx SAuthContext) (TokenCredential, error) {
	if this.AuthVersion() == "v3" {
		return this._authV3(domainName, uname, passwd, "", tenantName, tenantDomain, "", aCtx)
	}
	return this._authV2(uname, passwd, "", tenantName, "", aCtx)
}

func (this *Client) unmarshalV3Token(rbody jsonutils.JSONObject, tokenId string) (cred TokenCredential, err error) {
	cred = &TokenCredentialV3{Id: tokenId}
	err = rbody.Unmarshal(cred)
	if err != nil {
		err = fmt.Errorf("Invalid response when unmarshal V3 Token: %v", err)
	}
	cata := cred.GetServiceCatalog()
	if cata == nil || cata.Len() == 0 {
		log.Warningf("No service catalog avaiable")
	} else {
		this.serviceCatalog = cata
	}
	return
}

func (this *Client) unmarshalV2Token(rbody jsonutils.JSONObject) (cred TokenCredential, err error) {
	access, err := rbody.Get("access")
	if err == nil {
		cred = &TokenCredentialV2{}
		err = access.Unmarshal(cred)
		if err != nil {
			err = fmt.Errorf("Invalid response when unmarshal V2 Token: %s", err)
		}
		cata := cred.GetServiceCatalog()
		if cata == nil || cata.Len() == 0 {
			log.Warningf("No srvice catalog avaiable")
		} else {
			this.serviceCatalog = cata
		}
		return
	}
	err = fmt.Errorf("Invalid response: no access object")
	return
}

func (this *Client) verifyV3(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	header.Add(api.AUTH_SUBJECT_TOKEN_HEADER, token)
	_, rbody, err := this.jsonRequest(context.Background(), this.authUrl, "", "GET", "/auth/tokens", header, nil)
	if err != nil {
		return nil, err
	}
	return this.unmarshalV3Token(rbody, token)
}

func (this *Client) verifyV2(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add(api.AUTH_TOKEN_HEADER, adminToken)
	verifyUrl := fmt.Sprintf("/tokens/%s", token)
	_, rbody, err := this.jsonRequest(context.Background(), this.authUrl, "", "GET", verifyUrl, header, nil)
	if err != nil {
		return nil, err
	}
	return this.unmarshalV2Token(rbody)
}

func (this *Client) Verify(adminToken, token string) (cred TokenCredential, err error) {
	if this.AuthVersion() == "v3" {
		return this.verifyV3(adminToken, token)
	}
	return this.verifyV2(adminToken, token)
}

func (this *Client) SetTenant(tenantId, tenantName, tenantDomain string, token TokenCredential) (TokenCredential, error) {
	return this.SetProject(tenantId, tenantName, tenantDomain, token)
}

func (this *Client) SetProject(tenantId, tenantName, tenantDomain string, token TokenCredential) (TokenCredential, error) {
	aCtx := SAuthContext{
		Source: token.GetLoginSource(),
		Ip:     token.GetLoginIp(),
	}
	if this.AuthVersion() == "v3" {
		return this._authV3("", "", "", tenantId, tenantName, tenantDomain, token.GetTokenString(), aCtx)
	} else {
		return this._authV2("", "", "", tenantName, token.GetTokenString(), aCtx)
	}
}

func (this *Client) NewSession(ctx context.Context, region, zone, endpointType string, token TokenCredential, apiVersion string) *ClientSession {
	cata := token.GetServiceCatalog()
	if this.serviceCatalog == nil {
		if cata == nil || cata.Len() == 0 {
			log.Warningf("Missing service catalog in token")
		} else {
			this.serviceCatalog = cata
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return &ClientSession{
		ctx:               ctx,
		client:            this,
		region:            region,
		zone:              zone,
		endpointType:      endpointType,
		token:             token,
		defaultApiVersion: apiVersion,
		Header:            http.Header{},
	}
}
