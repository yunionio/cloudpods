package tos

import (
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sync/singleflight"
)

type Credential struct {
	AccessKeyID     string
	AccessKeySecret string
	SecurityToken   string
}

// Credentials provides Credential
type Credentials interface {
	Credential() Credential
}

// StaticCredentials Credentials with static access-key and secret-key
type StaticCredentials struct {
	accessKey     string
	secretKey     string
	securityToken string
}

// NewStaticCredentials Credentials with static access-key and secret-key
//  use StaticCredentials.WithSecurityToken to set security-token
//
// you can use it as:
//  client, err := tos.NewClient(endpoint, tos.WithCredentials(tos.NewStaticCredentials(accessKey, secretKey)))
//  // do something more
//
// And you can use tos.WithPerRequestSigner set the 'Signer' for each request.
//
func NewStaticCredentials(accessKeyID, accessKeySecret string) *StaticCredentials {
	return &StaticCredentials{
		accessKey: accessKeyID,
		secretKey: accessKeySecret,
	}
}

// WithSecurityToken set security-token
func (sc *StaticCredentials) WithSecurityToken(securityToken string) {
	sc.securityToken = securityToken
}

func (sc *StaticCredentials) Credential() Credential {
	return Credential{
		AccessKeyID:     sc.accessKey,
		AccessKeySecret: sc.secretKey,
		SecurityToken:   sc.securityToken,
	}
}

// WithoutSecretKeyCredentials Credentials with static access-key and no secret-key
//
// If you don't want to use secret-key directly, but use signed-key, you can use it as:
//  signer := tos.NewSignV4(tos.NewWithoutSecretKeyCredentials(accessKey), region)
//  signer.WithSigningKey(func(*SigningKeyInfo) []byte { return signingKey})
//  client, err := tos.NewClient(endpoint, tos.WithSigner(signer))
//  // do something more
//
// And you can use tos.WithPerRequestSigner set the 'Signer' for each request.
//
type WithoutSecretKeyCredentials struct {
	accessKey     string
	securityToken string
}

func NewWithoutSecretKeyCredentials(accessKeyID string) *WithoutSecretKeyCredentials {
	return &WithoutSecretKeyCredentials{
		accessKey:     accessKeyID,
		securityToken: "",
	}
}

// WithSecurityToken set security-token
func (sc *WithoutSecretKeyCredentials) WithSecurityToken(securityToken string) {
	sc.securityToken = securityToken
}

func (sc *WithoutSecretKeyCredentials) Credential() Credential {
	return Credential{
		AccessKeyID:   sc.accessKey,
		SecurityToken: sc.securityToken,
	}
}

// FederationToken contains Credential and Credential's expiration time
type FederationToken struct {
	Credential Credential
	Expiration time.Time
}

// FederationTokenProvider provides FederationToken
type FederationTokenProvider interface {
	FederationToken() (*FederationToken, error)
}

// FederationCredentials implements Credentials interfaces with flushing Credential periodically
type FederationCredentials struct {
	cachedToken   *FederationToken
	refreshing    uint32
	preFetch      time.Duration
	tokenProvider FederationTokenProvider
	flight        singleflight.Group
}

// NewFederationCredentials FederationCredentials implements Credentials interfaces with flushing Credential periodically
//
// use WithPreFetch set prefetch time
func NewFederationCredentials(tokenProvider FederationTokenProvider) (*FederationCredentials, error) {
	cred, err := tokenProvider.FederationToken()
	if err != nil {
		return nil, err
	}

	return &FederationCredentials{
		cachedToken:   cred,
		preFetch:      5 * time.Minute,
		tokenProvider: tokenProvider,
	}, nil
}

// WithPreFetch set prefetch time
func (fc *FederationCredentials) WithPreFetch(preFetch time.Duration) {
	fc.preFetch = preFetch
}

func (fc *FederationCredentials) token() *FederationToken {
	return (*FederationToken)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&fc.cachedToken))))
}

// Credential for Credentials interface
func (fc *FederationCredentials) Credential() Credential {
	now := time.Now()
	if token := fc.token(); now.After(token.Expiration) { // 已经过期
		_, _, _ = fc.flight.Do("flushing", func() (interface{}, error) {
			flushed, err := fc.tokenProvider.FederationToken()
			if err != nil {
				return nil, err
			}
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&fc.cachedToken)), unsafe.Pointer(flushed))
			return flushed, nil
		})
	} else if now.Add(fc.preFetch).After(token.Expiration) &&
		atomic.LoadUint32(&fc.refreshing) == 0 {
		// 将要过期, prefetch token
		if atomic.CompareAndSwapUint32(&fc.refreshing, 0, 1) {
			defer atomic.StoreUint32(&fc.refreshing, 0)
			if newToken, err := fc.tokenProvider.FederationToken(); err == nil {
				atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&fc.cachedToken)), unsafe.Pointer(newToken))
			}
		}
	}

	return fc.token().Credential
}
