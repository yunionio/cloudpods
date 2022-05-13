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
	"compress/flate"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"io"
	"math/rand"
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
)

func setPrivateKey(key *rsa.PrivateKey) {
	privateKey = key
}

type SAuthToken struct {
	token      string
	verifyTotp bool
	enableTotp bool
	initTotp   bool
	isSsoLogin bool

	retryCount     int    // 重试计数器
	lockExpireTime uint32 // 锁定时间
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

func (t SAuthToken) Encode() string {
	encBytes := t.encodeBytes()
	if privateKey != nil {
		return EncryptString(encBytes)
	} else {
		return compressString(encBytes)
	}
}

func Decode(t string) (*SAuthToken, error) {
	var tBytes []byte
	var err error
	if privateKey != nil {
		tBytes, err = DecryptString(t)
		if err != nil {
			return nil, errors.Wrap(err, "decryptString")
		}
	} else {
		tBytes, err = decompressString(t)
		if err != nil {
			return nil, errors.Wrap(err, "decompressString")
		}
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

func compressString(in []byte) string {
	buf := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(buf, 9)
	compressor.Write(in)
	compressor.Close()
	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

func EncryptString(in []byte) string {
	enc, _ := jwe.Encrypt(in, jwa.RSA1_5, &privateKey.PublicKey, jwa.A128GCM, jwa.Deflate)
	return string(enc)
}

func decompressString(in string) ([]byte, error) {
	inBytes, err := base64.URLEncoding.DecodeString(in)
	if err != nil {
		return nil, errors.Wrap(err, "base64.URLEncoding.DecodeString")
	}
	buf := new(bytes.Buffer)
	decompressor := flate.NewReader(bytes.NewReader(inBytes))
	_, err = io.Copy(buf, decompressor)
	if err != nil {
		return nil, errors.Wrap(err, "decompress")
	}
	decompressor.Close()
	return buf.Bytes(), nil
}

func DecryptString(in string) ([]byte, error) {
	return jwe.Decrypt([]byte(in), jwa.RSA1_5, privateKey)
}

func (t SAuthToken) GetToken(ctx context.Context) (mcclient.TokenCredential, error) {
	return auth.Verify(ctx, t.token)
}

func (t SAuthToken) GetAuthCookie(token mcclient.TokenCredential) string {
	sid := t.Encode()
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
	return info.String()
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

		// 锁定
		if t.retryCount >= MAX_OTP_RETRY {
			t.lockExpireTime = uint32(time.Now().Add(30 * time.Second).Unix())
		}
	} else {
		// 清零计数器，解除锁定
		if t.lockExpireTime < uint32(time.Now().Unix()) {
			t.lockExpireTime = 0
			t.retryCount = 0
		}
	}
}

func (t *SAuthToken) VerifyTotpPasscode(s *mcclient.ClientSession, uid, passcode string) error {
	if t.lockExpireTime > uint32(time.Now().Unix()) {
		return errors.Wrapf(httperrors.ErrResourceBusy, "locked, retry after %d seconds", t.lockExpireTime-uint32(time.Now().Unix()))
	}

	secret, err := fetchUserTotpCredSecret(s, uid)
	if err != nil {
		return errors.Wrap(err, "fetch totp secrets error")
	}

	if totp.Validate(passcode, secret) {
		t.verifyTotp = true
		t.lockExpireTime = 0
		t.retryCount = 0
		return nil
	}

	t.updateRetryCount()
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
