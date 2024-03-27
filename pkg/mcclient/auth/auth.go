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
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/cache"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/syncman"
	"yunion.io/x/onecloud/pkg/httperrors"
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

func (c *TokenCacheVerify) Remove(ctx context.Context, cli *mcclient.Client, adminToken, token string) error {
	c.DeleteToken(token)

	err := cli.Invalidate(ctx, adminToken, token)
	if err != nil {
		return errors.Wrap(err, "Invalidate")
	}

	return nil
}

type authManager struct {
	syncman.SSyncManager

	client           *mcclient.Client
	info             *AuthInfo
	adminCredential  mcclient.TokenCredential
	tokenCacheVerify *TokenCacheVerify
	accessKeyCache   *sAccessKeyCache
}

var (
	authManagerInstane *authManager
	authManagerLock    *sync.Mutex = &sync.Mutex{}
)

func newAuthManager(cli *mcclient.Client, info *AuthInfo) *authManager {
	authManagerLock.Lock()
	defer authManagerLock.Unlock()

	if authManagerInstane != nil {
		authManagerInstane.client = cli
		authManagerInstane.info = info
		return authManagerInstane
	}
	authManagerInstane = &authManager{
		client:           cli,
		info:             info,
		tokenCacheVerify: NewTokenCacheVerify(),
		accessKeyCache:   newAccessKeyCache(),
	}
	authManagerInstane.InitSync(authManagerInstane)
	go authManagerInstane.startRefreshRevokeTokens()
	return authManagerInstane
}

func (a *authManager) startRefreshRevokeTokens() {
	err := a.refreshRevokeTokens(context.Background())
	if err != nil {
		log.Errorf("%s", err)
	}
	time.AfterFunc(5*time.Minute, a.startRefreshRevokeTokens)
}

func (a *authManager) refreshRevokeTokens(ctx context.Context) error {
	if a.adminCredential == nil {
		return fmt.Errorf("refreshRevokeTokens: No valid admin token credential")
	}
	tokens, err := a.client.FetchInvalidTokens(getContext(ctx), a.adminCredential.GetTokenString())
	if err != nil {
		return errors.Wrap(err, "client.FetchInvalidTokens")
	}
	for _, token := range tokens {
		a.tokenCacheVerify.DeleteToken(token)
	}
	return nil
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
		a.reAuth()
		return nil, errors.Wrap(httperrors.ErrInvalidCredential, "No valid admin token credential")
	}
	cred, err := a.tokenCacheVerify.Verify(ctx, a.client, a.adminCredential.GetTokenString(), token)
	if err != nil {
		if httputils.ErrorCode(err) == 403 {
			// adminCredential need to be refresh
			a.reAuth()
		}
		return nil, errors.Wrap(err, "tokenCacheVerify.Verify")
	}
	return cred, nil
}

func (a *authManager) remove(ctx context.Context, token string) error {
	if a.adminCredential == nil {
		return errors.Wrap(httperrors.ErrInvalidCredential, "No valid admin token credential")
	}
	err := a.tokenCacheVerify.Remove(ctx, a.client, a.adminCredential.GetTokenString(), token)
	if err != nil {
		return errors.Wrap(err, "tokenCacheVerify.Remove")
	}
	return nil
}

var (
	defaultAuthSource = mcclient.AuthSourceSrv
)

func SetDefaultAuthSource(src string) {
	defaultAuthSource = src
}

func GetDefaultAuthSource() string {
	return defaultAuthSource
}

func (a *authManager) authAdmin() error {
	var token mcclient.TokenCredential
	var err error
	token, err = a.client.AuthenticateWithSource(
		a.info.Username, a.info.Passwd, a.info.Domain,
		a.info.Project, a.info.ProjectDomain, GetDefaultAuthSource())
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

func (a *authManager) DoSync(first bool, timeout bool) (time.Duration, error) {
	err := a.authAdmin()
	if err != nil {
		return time.Minute, errors.Wrap(err, "authAdmin")
	} else {
		return time.Until(a.adminCredential.GetExpires()) / 2, nil
	}
}

func (a *authManager) NeedSync(dat *jsonutils.JSONDict) bool {
	return true
}

func (a *authManager) Name() string {
	return "AuthManager"
}

func (a *authManager) reAuth() {
	a.SyncOnce(false, false)
}

func (a *authManager) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return a.getAdminSession(context.Background(), region, zone, endpointType).GetServiceURL(service, endpointType)
}

func (a *authManager) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return a.getAdminSession(context.Background(), region, zone, endpointType).GetServiceURLs(service, endpointType)
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

func (a *authManager) getAdminSession(ctx context.Context, region, zone, endpointType string) *mcclient.ClientSession {
	return a.getSession(ctx, manager.adminCredential, region, zone, endpointType)
}

func getContext(ctx context.Context) context.Context {
	return mcclient.FixContext(ctx)
}

func (a *authManager) getSession(ctx context.Context, token mcclient.TokenCredential, region, zone, endpointType string) *mcclient.ClientSession {
	cli := Client()
	if cli == nil {
		return nil
	}
	if endpointType == "" && globalEndpointType != "" {
		endpointType = globalEndpointType
	}
	return cli.NewSession(getContext(ctx), region, zone, endpointType, token)
}

func GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return manager.adminCredential.GetCatalogData(serviceTypes, region)
}

func Verify(ctx context.Context, tokenId string) (mcclient.TokenCredential, error) {
	return manager.verify(ctx, tokenId)
}

func Remove(ctx context.Context, tokenId string) error {
	return manager.remove(ctx, tokenId)
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
	if manager.adminCredential.GetExpires().Before(time.Now()) {
		manager.authAdmin()
	}
	return manager.adminCredential
}

// Deprecated
func AdminSession(ctx context.Context, region, zone, endpointType string) *mcclient.ClientSession {
	return manager.getAdminSession(ctx, region, zone, endpointType)
}

// Deprecated
func AdminSessionWithInternal(ctx context.Context, region, zone string) *mcclient.ClientSession {
	return AdminSession(ctx, region, zone, identity.EndpointInterfaceInternal)
}

// Deprecated
func AdminSessionWithPublic(ctx context.Context, region, zone string) *mcclient.ClientSession {
	return AdminSession(ctx, region, zone, identity.EndpointInterfacePublic)
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

func GetAdminSession(ctx context.Context, region string) *mcclient.ClientSession {
	return GetSession(ctx, manager.adminCredential, region)
}

func GetAdminSessionWithPublic(ctx context.Context, region string) *mcclient.ClientSession {
	return GetSessionWithPublic(ctx, manager.adminCredential, region)
}

func GetAdminSessionWithInternal(ctx context.Context, region string) *mcclient.ClientSession {
	return GetSessionWithInternal(ctx, manager.adminCredential, region)
}

func GetSession(ctx context.Context, token mcclient.TokenCredential, region string) *mcclient.ClientSession {
	if len(globalEndpointType) != 0 {
		return manager.getSession(ctx, token, region, "", globalEndpointType)
	}
	return GetSessionWithInternal(ctx, token, region)
}

func GetSessionWithInternal(ctx context.Context, token mcclient.TokenCredential, region string) *mcclient.ClientSession {
	return manager.getSession(ctx, token, region, "", identity.EndpointInterfaceInternal)
}

func GetSessionWithPublic(ctx context.Context, token mcclient.TokenCredential, region string) *mcclient.ClientSession {
	return manager.getSession(ctx, token, region, "", identity.EndpointInterfacePublic)
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
