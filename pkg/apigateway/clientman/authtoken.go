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
	token          string
	verifyTotp     bool
	enableTotp     bool
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
	msg.WriteByte(byte(rand.Int()))
	msg.WriteByte(byte(t.retryCount))
	expBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(expBytes, t.lockExpireTime)
	msg.Write(expBytes)
	msg.WriteString(t.token)
	return msg.Bytes()
}

func (t SAuthToken) encode() string {
	encBytes := t.encodeBytes()
	if privateKey != nil {
		return encryptString(encBytes)
	} else {
		return compressString(encBytes)
	}
}

func Decode(t string) (*SAuthToken, error) {
	var tBytes []byte
	var err error
	if privateKey != nil {
		tBytes, err = decryptString(t)
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
	if len(tt) < 3 {
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
	ret.retryCount = int(tt[3])
	ret.lockExpireTime = binary.LittleEndian.Uint32(tt[4:])
	ret.token = string(tt[8:])
	return &ret, nil
}

func compressString(in []byte) string {
	buf := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(buf, 9)
	compressor.Write(in)
	compressor.Close()
	return base64.URLEncoding.EncodeToString(buf.Bytes())
}

func encryptString(in []byte) string {
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

func decryptString(in string) ([]byte, error) {
	return jwe.Decrypt([]byte(in), jwa.RSA1_5, privateKey)
}

func (t SAuthToken) GetToken(ctx context.Context) (mcclient.TokenCredential, error) {
	return auth.Verify(ctx, t.token)
}

func (t SAuthToken) GetAuthCookie(token mcclient.TokenCredential) string {
	sid := t.encode()
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewTimeString(token.GetExpires()), "exp")
	info.Add(jsonutils.NewString(sid), "session")
	info.Add(jsonutils.NewBool(t.enableTotp), "totp_on")                      // 用户totp 开启状态。 True（已开启）|False(未开启)
	info.Add(jsonutils.NewBool(options.Options.EnableTotp), "system_totp_on") // 全局totp 开启状态。 True（已开启）|False(未开启)
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

func (t *SAuthToken) SetToken(tid string) {
	t.token = tid
}

func NewAuthToken(tid string, enableTotp bool) *SAuthToken {
	return &SAuthToken{
		token:      tid,
		enableTotp: enableTotp,
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
	return errors.Wrap(httperrors.ErrInvalidCredential, "invalid passcode")
}
