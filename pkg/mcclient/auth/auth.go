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

package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cache"

	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/syncman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	manager           *authManager
	defaultTimeout    int   = 600 // maybe time.Duration better
	defaultCacheCount int64 = 100000
	// initCh             chan bool = make(chan bool)
	globalEndpointType string
)

type AuthInfo struct {
	AuthUrl string
	// Domain not need when v2 auth
	Domain   string
	Username string
	Passwd   string
	// Project is tenant when v2 auth
	Project       string
	ProjectDomain string
}

func SetTimeout(t time.Duration) {
	defaultTimeout = int(t)
}

func SetEndpointType(epType string) {
	globalEndpointType = epType
}

func NewV2AuthInfo(authUrl, user, passwd, tenant string) *AuthInfo {
	return NewAuthInfo(authUrl, "", user, passwd, tenant, "")
}

func NewAuthInfo(authUrl, domain, user, passwd, project, projectDomain string) *AuthInfo {
	return &AuthInfo{
		AuthUrl:       authUrl,
		Domain:        domain,
		Username:      user,
		Passwd:        passwd,
		Project:       project,
		ProjectDomain: projectDomain,
	}
}

type cacheItem struct {
	credential mcclient.TokenCredential
}

func (item *cacheItem) Size() int {
	return 1
}

type TokenCacheVerify struct {
	*cache.LRUCache
}

func NewTokenCacheVerify() *TokenCacheVerify {
	return &TokenCacheVerify{
		LRUCache: cache.NewLRUCache(defaultCacheCount),
	}
}

func (c *TokenCacheVerify) AddToken(cred mcclient.TokenCredential) error {
	item := &cacheItem{cred}
	c.Set(cred.GetTokenString(), item)
	return nil
}

func (c *TokenCacheVerify) GetToken(token string) (mcclient.TokenCredential, bool) {
	item, found := c.Get(token)

	if !found {
		return nil, false
	}

	return item.(*cacheItem).credential, true
}

func (c *TokenCacheVerify) DeleteToken(token string) bool {
	return c.Delete(token)
}

func (c *TokenCacheVerify) Verify(ctx context.Context, cli *mcclient.Client, adminToken, token string) (mcclient.TokenCredential, error) {
	cred, found := c.GetToken(token)
	if found {
		if cred.IsValid() {
			return cred, nil
		} else {
			c.DeleteToken(token)
			log.Infof("Remove expired cache token: %s", token)
		}
	}

	cred, err := cli.Verify(adminToken, token)
	if err != nil {
		return nil, err
	}
	cred = mcclient.SimplifyToken(cred)
	err = c.AddToken(cred)
	if err != nil {
		return nil, fmt.Errorf("Add %s credential to cache: %#v", cred.GetTokenString(), err)
	}
	callbackAuthhooks(ctx, cred)
	// log.Debugf("Add token: %s", cred)
	return cred, nil
}

type authManager struct {
	syncman.SSyncManager

	client           *mcclient.Client
	info             *AuthInfo
	adminCredential  mcclient.TokenCredential
	tokenCacheVerify *TokenCacheVerify
	accessKeyCache   *sAccessKeyCache
}

func newAuthManager(cli *mcclient.Client, info *AuthInfo) *authManager {
	authm := &authManager{
		client:           cli,
		info:             info,
		tokenCacheVerify: NewTokenCacheVerify(),
		accessKeyCache:   newAccessKeyCache(),
	}
	authm.InitSync(authm)
	return authm
}

func (a *authManager) verifyRequest(req http.Request, virtualHost bool) (mcclient.TokenCredential, error) {
	if a.adminCredential == nil {
		return nil, fmt.Errorf("No valid admin token credential")
	}
	cred, err := a.accessKeyCache.Verify(a.client, req, virtualHost)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (a *authManager) verify(ctx context.Context, token string) (mcclient.TokenCredential, error) {
	if a.adminCredential == nil {
		return nil, fmt.Errorf("No valid admin token credential")
	}
	cred, err := a.tokenCacheVerify.Verify(ctx, a.client, a.adminCredential.GetTokenString(), token)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (a *authManager) authAdmin() error {
	var token mcclient.TokenCredential
	var err error
	token, err = a.client.AuthenticateWithSource(
		a.info.Username, a.info.Passwd, a.info.Domain,
		a.info.Project, a.info.ProjectDomain, mcclient.AuthSourceSrv)
	if err != nil {
		log.Errorf("Admin auth failed: %s", err)
		return err
	}
	if token != nil {
		a.adminCredential = token
		return nil
	} else {
		return fmt.Errorf("Auth token is nil")
	}
}

func (a *authManager) DoSync(first bool) (time.Duration, error) {
	err := a.authAdmin()
	if err != nil {
		return time.Minute, errors.Wrap(err, "authAdmin")
	} else {
		return a.adminCredential.GetExpires().Sub(time.Now()) / 2, nil
	}
}

func (a *authManager) NeedSync(dat *jsonutils.JSONDict) bool {
	return true
}

func (a *authManager) Name() string {
	return "AuthManager"
}

func (a *authManager) reAuth() {
	a.SyncOnce()
}

func (a *authManager) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	if endpointType == "" && globalEndpointType != "" {
		endpointType = globalEndpointType
	}
	return a.adminCredential.GetServiceURL(service, region, zone, endpointType)
}

func (a *authManager) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	if endpointType == "" && globalEndpointType != "" {
		endpointType = globalEndpointType
	}
	return a.adminCredential.GetServiceURLs(service, region, zone, endpointType)
}

func (a *authManager) getServiceIPs(service, region, zone, endpointType string, needResolve bool) ([]string, error) {
	urls, err := a.GetServiceURLs(service, region, zone, endpointType)
	if err != nil {
		return nil, errors.Wrap(err, "GetServiceURLs")
	}
	ret := stringutils2.NewSortedStrings(nil)
	for _, url := range urls {
		slashIdx := strings.Index(url, "://")
		if slashIdx >= 0 {
			url = url[slashIdx+3:]
		}
		if needResolve {
			addrs, err := net.LookupHost(url)
			if err != nil {
				log.Errorf("Lookup host %s fail: %s", url, err)
			} else {
				ret = ret.Append(addrs...)
			}
		} else {
			ret = ret.Append(url)
		}
	}
	return ret, nil
}

func (a *authManager) getTokenString() string {
	return a.adminCredential.GetTokenString()
}

func (a *authManager) isExpired() bool {
	return time.Now().After(a.adminCredential.GetExpires())
}

func (a *authManager) isAuthed() bool {
	if a == nil {
		return false
	}
	if a.adminCredential == nil || a.isExpired() {
		return false
	}
	return true
}

func GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return manager.adminCredential.GetCatalogData(serviceTypes, region)
}

func Verify(ctx context.Context, tokenId string) (mcclient.TokenCredential, error) {
	return manager.verify(ctx, tokenId)
}

func VerifyRequest(req http.Request, virtualHost bool) (mcclient.TokenCredential, error) {
	return manager.verifyRequest(req, virtualHost)
}

func GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return manager.GetServiceURL(service, region, zone, endpointType)
}

func GetPublicServiceURL(service, region, zone string) (string, error) {
	return manager.GetServiceURL(service, region, zone, identity.EndpointInterfacePublic)
}

func GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return manager.GetServiceURLs(service, region, zone, endpointType)
}

func GetDNSServers(region, zone string) ([]string, error) {
	return manager.getServiceIPs("dns", region, zone, identity.EndpointInterfacePublic, false)
}

func GetNTPServers(region, zone string) ([]string, error) {
	return manager.getServiceIPs("ntp", region, zone, identity.EndpointInterfacePublic, true)
}

func GetTokenString() string {
	return manager.getTokenString()
}

func IsAuthed() bool {
	return manager != nil && manager.isAuthed()
}

func Client() *mcclient.Client {
	return manager.client
}

func AdminCredential() mcclient.TokenCredential {
	return manager.adminCredential
}

// Deprecated
func AdminSession(ctx context.Context, region, zone, endpointType, apiVersion string) *mcclient.ClientSession {
	cli := Client()
	if cli == nil {
		return nil
	}
	if endpointType == "" && globalEndpointType != "" {
		endpointType = globalEndpointType
	}
	return cli.NewSession(ctx, region, zone, endpointType, AdminCredential(), apiVersion)
}

// Deprecated
func AdminSessionWithInternal(ctx context.Context, region, zone, apiVersion string) *mcclient.ClientSession {
	return AdminSession(ctx, region, zone, "internal", apiVersion)
}

// Deprecated
func AdminSessionWithPublic(ctx context.Context, region, zone, apiVersion string) *mcclient.ClientSession {
	return AdminSession(ctx, region, zone, "public", apiVersion)
}

type AuthCompletedCallback func()

func AsyncInit(info *AuthInfo, debug, insecure bool, certFile, keyFile string, callback AuthCompletedCallback) {
	cli := mcclient.NewClient(info.AuthUrl, defaultTimeout, debug, insecure, certFile, keyFile)
	manager = newAuthManager(cli, info)
	err := manager.FirstSync()
	if err != nil {
		log.Fatalf("Auth manager init err: %v", err)
	} else if callback != nil {
		callback()
	}
}

func Init(info *AuthInfo, debug, insecure bool, certFile, keyFile string) {
	AsyncInit(info, debug, insecure, certFile, keyFile, nil)
}

func ReAuth() {
	manager.reAuth()
}

func GetAdminSession(ctx context.Context, region string,
	apiVersion string) *mcclient.ClientSession {
	return GetSession(ctx, manager.adminCredential, region, apiVersion)
}

func GetAdminSessionWithPublic(ctx context.Context, region string,
	apiVersion string) *mcclient.ClientSession {
	return GetSessionWithPublic(ctx, manager.adminCredential, region, apiVersion)
}

func GetAdminSessionWithInternal(
	ctx context.Context, region string, apiVersion string) *mcclient.ClientSession {
	return GetSessionWithInternal(ctx, manager.adminCredential, region, apiVersion)
}

func GetSession(ctx context.Context, token mcclient.TokenCredential, region string, apiVersion string) *mcclient.ClientSession {
	if len(globalEndpointType) != 0 {
		return getSessionByType(ctx, token, region, apiVersion, globalEndpointType)
	}
	return GetSessionWithInternal(ctx, token, region, apiVersion)
}

func GetSessionWithInternal(ctx context.Context, token mcclient.TokenCredential, region string, apiVersion string) *mcclient.ClientSession {
	return getSessionByType(ctx, token, region, apiVersion, identity.EndpointInterfaceInternal)
}

func GetSessionWithPublic(ctx context.Context, token mcclient.TokenCredential, region string, apiVersion string) *mcclient.ClientSession {
	return getSessionByType(ctx, token, region, apiVersion, identity.EndpointInterfacePublic)
}

func getSessionByType(ctx context.Context, token mcclient.TokenCredential, region string, apiVersion string, epType string) *mcclient.ClientSession {
	return manager.client.NewSession(ctx, region, "", epType, token, apiVersion)
}

// use for climc test only
func InitFromClientSession(session *mcclient.ClientSession) {
	cli := session.GetClient()
	token := session.GetToken()
	info := &AuthInfo{}
	manager = &authManager{
		client:          cli,
		info:            info,
		adminCredential: token,
	}

	SetEndpointType(session.GetEndpointType())
}

func RegisterCatalogListener(listener mcclient.IServiceCatalogChangeListener) {
	manager.client.RegisterCatalogListener(listener)
}
