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

package clientman

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwe"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/pquerna/otp/totp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	TotpEnable  = '1'
	TotpDisable = '0'
)

var (
	privateKey *rsa.PrivateKey

	cookieKeyMu sync.RWMutex
	cookieKey   []byte
)

func setPrivateKey(key *rsa.PrivateKey) {
	privateKey = key
}

func setCookieKey(key []byte) {
	cookieKeyMu.Lock()
	defer cookieKeyMu.Unlock()
	if len(key) == 0 {
		cookieKey = nil
		return
	}
	cookieKey = append([]byte(nil), key...)
}

func currentCookieKey() ([]byte, error) {
	cookieKeyMu.RLock()
	defer cookieKeyMu.RUnlock()
	if len(cookieKey) != 32 {
		return nil, errors.Error("auth cookie key is not initialized")
	}
	return append([]byte(nil), cookieKey...), nil
}

type SAuthToken struct {
	token      string
	verifyTotp bool
	enableTotp bool
	initTotp   bool
	isSsoLogin bool

	retryCount     int    // SSL 会话沿用 cookie 内的重试计数
	lockExpireTime uint32 // SSL 会话沿用 cookie 内的锁定时间
}

func (t SAuthToken) encodeBytes() []byte {
	msg := bytes.Buffer{}
	if t.verifyTotp {
		msg.WriteByte(TotpEnable)
	} else {
		msg.WriteByte(TotpDisable)
	}
	if t.enableTotp {
		msg.WriteByte(TotpEnable)
	} else {
		msg.WriteByte(TotpDisable)
	}
	if t.initTotp {
		msg.WriteByte(TotpEnable)
	} else {
		msg.WriteByte(TotpDisable)
	}
	if t.isSsoLogin {
		msg.WriteByte(TotpEnable)
	} else {
		msg.WriteByte(TotpDisable)
	}
	msg.WriteByte(byte(rand.Int()))
	msg.WriteByte(byte(t.retryCount))
	expBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(expBytes, t.lockExpireTime)
	msg.Write(expBytes)
	msg.WriteString(t.token)
	return msg.Bytes()
}

func (t SAuthToken) Encode() (string, error) {
	return EncryptString(t.encodeBytes())
}

func Decode(t string) (*SAuthToken, error) {
	tBytes, err := DecryptString(t)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt auth cookie")
	}
	return decodeBytes(tBytes)
}

func decodeBytes(tt []byte) (*SAuthToken, error) {
	ret := SAuthToken{}
	if len(tt) < 10 {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "too short")
	}
	if tt[0] == TotpEnable {
		ret.verifyTotp = true
	} else {
		ret.verifyTotp = false
	}
	if tt[1] == TotpEnable {
		ret.enableTotp = true
	} else {
		ret.enableTotp = false
	}
	if tt[2] == TotpEnable {
		ret.initTotp = true
	} else {
		ret.initTotp = false
	}
	if tt[3] == TotpEnable {
		ret.isSsoLogin = true
	} else {
		ret.isSsoLogin = false
	}
	// 4: skip rand number
	ret.retryCount = int(tt[5])
	ret.lockExpireTime = binary.LittleEndian.Uint32(tt[6:])
	ret.token = string(tt[10:])
	return &ret, nil
}

func EncryptString(in []byte) (string, error) {
	if privateKey != nil {
		enc, err := jwe.Encrypt(in, jwa.RSA1_5, &privateKey.PublicKey, jwa.A128GCM, jwa.Deflate)
		if err != nil {
			return "", errors.Wrap(err, "jwe.Encrypt")
		}
		return string(enc), nil
	}
	key, err := currentCookieKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.Wrap(err, "aes.NewCipher")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "cipher.NewGCM")
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := cryptorand.Read(nonce); err != nil {
		return "", errors.Wrap(err, "read nonce")
	}
	buf := make([]byte, len(nonce), len(nonce)+len(in)+aead.Overhead())
	copy(buf, nonce)
	sealed := aead.Seal(buf, nonce, in, nil)
	return base64.URLEncoding.EncodeToString(sealed), nil
}

func DecryptString(in string) ([]byte, error) {
	if privateKey != nil {
		plain, err := jwe.Decrypt([]byte(in), jwa.RSA1_5, privateKey)
		if err != nil {
			return nil, errors.Wrap(err, "jwe.Decrypt")
		}
		return plain, nil
	}
	key, err := currentCookieKey()
	if err != nil {
		return nil, err
	}
	raw, err := base64.URLEncoding.DecodeString(in)
	if err != nil {
		return nil, errors.Wrap(err, "base64.URLEncoding.DecodeString")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes.NewCipher")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "cipher.NewGCM")
	}
	ns := aead.NonceSize()
	if len(raw) < ns+aead.Overhead() {
		return nil, errors.Error("auth cookie truncated")
	}
	plain, err := aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return nil, errors.Wrap(err, "open auth cookie")
	}
	return plain, nil
}

func (t SAuthToken) GetToken(ctx context.Context) (mcclient.TokenCredential, error) {
	return auth.Verify(ctx, t.token)
}

func (t SAuthToken) GetAuthCookie(token mcclient.TokenCredential) (string, error) {
	sid, err := t.Encode()
	if err != nil {
		return "", errors.Wrap(err, "encode auth cookie")
	}
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewTimeString(token.GetExpires()), "exp")
	info.Add(jsonutils.NewString(sid), "session")
	info.Add(jsonutils.NewBool(t.verifyTotp), "totp_verified")                // 用户totp验证通过
	info.Add(jsonutils.NewBool(t.initTotp), "totp_init")                      // 是否初始化TOTP密钥
	info.Add(jsonutils.NewBool(t.enableTotp), "totp_on")                      // 用户totp 开启状态。 True（已开启）|False(未开启)
	info.Add(jsonutils.NewBool(t.isSsoLogin), "is_sso")                       // 用户是否通过SSO登录
	info.Add(jsonutils.NewBool(options.Options.EnableTotp), "system_totp_on") // 全局totp 开启状态。 True（已开启）|False(未开启)
	info.Add(jsonutils.NewString(token.GetUserId()), "user_id")
	info.Add(jsonutils.NewString(token.GetUserName()), "user")
	return info.String(), nil
}

func (t SAuthToken) IsTotpVerified() bool {
	if !options.Options.EnableTotp {
		return true
	}
	if !t.enableTotp {
		return true
	}
	return t.verifyTotp
}

func (t SAuthToken) IsTotpEnabled() bool {
	return t.enableTotp
}

func (t SAuthToken) IsTotpInitialized() bool {
	return t.initTotp
}

func (t *SAuthToken) SetTotpInitialized() {
	t.initTotp = true
}

func (t *SAuthToken) SetToken(tid string) {
	t.token = tid
}

func NewAuthToken(tid string, enableTotp bool, isTotpInit bool, isSsoLogin bool) *SAuthToken {
	return &SAuthToken{
		token:      tid,
		enableTotp: enableTotp,
		initTotp:   isTotpInit,
		isSsoLogin: isSsoLogin,
		verifyTotp: false,
	}
}

func (t *SAuthToken) updateRetryCount() {
	if t.retryCount < MAX_OTP_RETRY {
		t.retryCount += 1
		if t.retryCount >= MAX_OTP_RETRY {
			t.lockExpireTime = uint32(time.Now().Add(30 * time.Second).Unix())
		}
	} else if t.lockExpireTime < uint32(time.Now().Unix()) {
		t.lockExpireTime = 0
		t.retryCount = 0
	}
}

func (t *SAuthToken) VerifyTotpPasscode(s *mcclient.ClientSession, uid, passcode string) error {
	// TLS sessions keep the original cookie counter. Without TLS the counter
	// is server-side, because the login cookie can be replayed.
	if privateKey == nil {
		if err := checkOtpLock(uid); err != nil {
			return err
		}
	} else if t.lockExpireTime > uint32(time.Now().Unix()) {
		return errors.Wrapf(httperrors.ErrResourceBusy, "locked, retry after %d seconds", t.lockExpireTime-uint32(time.Now().Unix()))
	}

	secret, err := fetchUserTotpCredSecret(s, uid)
	if err != nil {
		return errors.Wrap(err, "fetch totp secrets error")
	}

	if totp.Validate(passcode, secret) {
		t.verifyTotp = true
		if privateKey == nil {
			clearOtpLock(uid)
		} else {
			t.lockExpireTime = 0
			t.retryCount = 0
		}
		return nil
	}

	if privateKey == nil {
		noteOtpFailure(uid)
	} else {
		t.updateRetryCount()
	}
	return errors.Wrap(httperrors.ErrInputParameter, "invalid passcode")
}

func SignJWT(t jwt.Token) (string, error) {
	//jwkKey, err := jwk.New(privateKey)
	//if err != nil {
	//	return "", errors.Wrap(err, "jwk.New")
	//}
	signed, err := jwt.Sign(t, jwa.RS256, privateKey)
	if err != nil {
		return "", errors.Wrap(err, "jwt.Sign")
	}
	return string(signed), nil
}

func GetJWKs(ctx context.Context) (jsonutils.JSONObject, error) {
	key := jsonutils.NewDict()
	key.Set("use", jsonutils.NewString("sig"))
	key.Set("kty", jsonutils.NewString("RSA"))
	key.Set("alg", jsonutils.NewString("RS256"))
	key.Set("e", jsonutils.NewString("AQAB"))
	key.Set("n", jsonutils.NewString(base64.URLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())))

	ret := jsonutils.NewDict()
	ret.Set("keys", jsonutils.NewArray(key))
	return ret, nil
}
