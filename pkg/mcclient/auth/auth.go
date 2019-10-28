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
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/cache"

	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	manager           *authManager
	defaultTimeout    int       = 600 // maybe time.Duration better
	defaultCacheCount int64     = 100000
	initCh            chan bool = make(chan bool)
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

func (c *TokenCacheVerify) Verify(cli *mcclient.Client, adminToken, token string) (mcclient.TokenCredential, error) {
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
	callbackAuthhooks(cred)
	// log.Debugf("Add token: %s", cred)
	return cred, nil
}

type authManager struct {
	client           *mcclient.Client
	info             *AuthInfo
	adminCredential  mcclient.TokenCredential
	tokenCacheVerify *TokenCacheVerify
	accessKeyCache   *sAccessKeyCache
}

func newAuthManager(cli *mcclient.Client, info *AuthInfo) *authManager {
	return &authManager{
		client:           cli,
		info:             info,
		tokenCacheVerify: NewTokenCacheVerify(),
		accessKeyCache:   newAccessKeyCache(),
	}
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

func (a *authManager) verify(token string) (mcclient.TokenCredential, error) {
	if a.adminCredential == nil {
		return nil, fmt.Errorf("No valid admin token credential")
	}
	cred, err := a.tokenCacheVerify.Verify(a.client, a.adminCredential.GetTokenString(), token)
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

func (a *authManager) reAuth() {
	redoSleepTime := 60 * time.Second
	for {
		err := a.authAdmin()
		if err == nil {
			break
		}
		log.Errorf("Reauth failed: %s, try it again after %v", err, redoSleepTime)
		time.Sleep(redoSleepTime)
	}
	expire := a.adminCredential.GetExpires()
	duration := expire.Sub(time.Now())
	time.AfterFunc(time.Duration(duration.Nanoseconds()/2), a.reAuth)
}

func (a *authManager) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return a.adminCredential.GetServiceURL(service, region, zone, endpointType)
}

func (a *authManager) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return a.adminCredential.GetServiceURLs(service, region, zone, endpointType)
}

func (a *authManager) getTokenString() string {
	return a.adminCredential.GetTokenString()
}

func (a *authManager) isExpired() bool {
	return time.Now().After(a.adminCredential.GetExpires())
}

func (a *authManager) isAuthed() bool {
	if a.adminCredential == nil || a.isExpired() {
		return false
	}
	return true
}

func (a *authManager) init() error {
	if err := a.authAdmin(); err != nil {
		initCh <- false
		log.Fatalf("Auth manager init err: %v", err)
	}
	log.Infof("Get token: %v", a.getTokenString())
	expire := a.adminCredential.GetExpires()
	duration := expire.Sub(time.Now())
	time.AfterFunc(time.Duration(duration.Nanoseconds()/2), a.reAuth)
	initCh <- true
	return nil
}

func GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return manager.adminCredential.GetCatalogData(serviceTypes, region)
}

func Verify(tokenId string) (mcclient.TokenCredential, error) {
	return manager.verify(tokenId)
}

func VerifyRequest(req http.Request, virtualHost bool) (mcclient.TokenCredential, error) {
	return manager.verifyRequest(req, virtualHost)
}

func GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return manager.GetServiceURL(service, region, zone, endpointType)
}

func GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return manager.GetServiceURLs(service, region, zone, endpointType)
}

func GetTokenString() string {
	return manager.getTokenString()
}

func IsAuthed() bool {
	return manager.isAuthed()
}

func Client() *mcclient.Client {
	return manager.client
}

func AdminCredential() mcclient.TokenCredential {
	return manager.adminCredential
}

func AdminSession(ctx context.Context, region, zone, endpointType, apiVersion string) *mcclient.ClientSession {
	cli := Client()
	if cli == nil {
		return nil
	}
	return cli.NewSession(ctx, region, zone, endpointType, AdminCredential(), apiVersion)
}

type AuthCompletedCallback func()

func (callback *AuthCompletedCallback) Run() {
	f := *callback
	for {
		initOk := <-initCh
		if initOk && manager.isAuthed() {
			log.V(10).Infof("Auth completed, run callback: %v", *callback)
			f()
			return
		}
		log.Warningf("Auth manager not ready, check it again...")
	}
}

func AsyncInit(info *AuthInfo, debug, insecure bool, certFile, keyFile string, callback AuthCompletedCallback) {
	cli := mcclient.NewClient(info.AuthUrl, defaultTimeout, debug, insecure, certFile, keyFile)
	manager = newAuthManager(cli, info)
	go manager.init()
	if callback != nil {
		go callback.Run()
	}
}

func Init(info *AuthInfo, debug, insecure bool, certFile, keyFile string) {
	done := make(chan bool, 1)
	f := func() {
		done <- true
	}
	AsyncInit(info, debug, insecure, certFile, keyFile, f)
	<-done
}

func GetAdminSession(ctx context.Context, region string,
	apiVersion string) *mcclient.ClientSession {
	return GetSession(ctx, manager.adminCredential, region, apiVersion)
}

func GetSession(ctx context.Context, token mcclient.TokenCredential, region string, apiVersion string) *mcclient.ClientSession {
	return manager.client.NewSession(ctx, region, "", "internal", token, apiVersion)
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
}
