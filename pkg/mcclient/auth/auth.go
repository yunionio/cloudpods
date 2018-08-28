package auth

import (
	"fmt"
	"time"

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
	Project string
}

func SetTimeout(t time.Duration) {
	defaultTimeout = int(t)
}

func NewV2AuthInfo(authUrl, user, passwd, tenant string) *AuthInfo {
	return NewAuthInfo(authUrl, "", user, passwd, tenant)
}

func NewAuthInfo(authUrl, domain, user, passwd, project string) *AuthInfo {
	return &AuthInfo{
		AuthUrl:  authUrl,
		Domain:   domain,
		Username: user,
		Passwd:   passwd,
		Project:  project,
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
	// log.Debugf("Add token: %s", cred)
	return cred, nil
}

type authManager struct {
	client           *mcclient.Client
	info             *AuthInfo
	adminCredential  mcclient.TokenCredential
	tokenCacheVerify *TokenCacheVerify
}

func newAuthManager(cli *mcclient.Client, info *AuthInfo) *authManager {
	return &authManager{
		client:           cli,
		info:             info,
		tokenCacheVerify: NewTokenCacheVerify(),
	}
}

func (a *authManager) verify(token string) (mcclient.TokenCredential, error) {
	cred, err := a.tokenCacheVerify.Verify(a.client, a.adminCredential.GetTokenString(), token)
	if err != nil {
		return nil, err
	}
	return cred, nil
}

func (a *authManager) authAdmin() error {
	var err error
	a.adminCredential, err = a.client.Authenticate(
		a.info.Username, a.info.Passwd,
		a.info.Domain, a.info.Project)
	if err != nil {
		log.Errorf("Admin auth failed: %s", err)
		return err
	}
	return nil
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

func (a *authManager) getServiceURL(service, region, zone, endpointType string) (string, error) {
	return a.adminCredential.GetServiceURL(service, region, zone, endpointType)
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

func Verify(tokenId string) (mcclient.TokenCredential, error) {
	return manager.verify(tokenId)
}

func GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return manager.getServiceURL(service, region, zone, endpointType)
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

func AdminSession(region, zone, endpointType, apiVersion string) *mcclient.ClientSession {
	cli := Client()
	if cli == nil {
		return nil
	}
	return cli.NewSession(region, zone, endpointType, AdminCredential(), apiVersion)
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

func AsyncInit(info *AuthInfo, debug, insecure bool, callback AuthCompletedCallback) {
	cli := mcclient.NewClient(info.AuthUrl, defaultTimeout, debug, insecure)
	manager = newAuthManager(cli, info)
	go manager.init()
	if callback != nil {
		go callback.Run()
	}
}

func Init(info *AuthInfo, debug, insecure bool) {
	done := make(chan bool, 1)
	f := func() {
		done <- true
	}
	AsyncInit(info, debug, insecure, f)
	<-done
}

func GetAdminSession(region string, apiVersion string) *mcclient.ClientSession {
	return GetSession(manager.adminCredential, region, apiVersion)
}

func GetSession(token mcclient.TokenCredential, region string, apiVersion string) *mcclient.ClientSession {
	return manager.client.NewSession(region, "", "internal", token, apiVersion)
}
