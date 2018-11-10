package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
)

type Signer interface {
	Sign(secretKey string) (string, error)
	SignOnce(secretKey string) (string, error)
}

// 生成签名, 腾讯移动服务通过签名来验证请求的合法性,
// 开发者通过将签名授权给客户端, 使其具备上传下载及管理指定资源的能力,
// 签名分为多次有效签名和单次有效签名.
// 生成签名所需信息包括项目 ID(AppId)，空间名称(Bucket,文件资源的组织管理单元)，项目的 Secret ID 和 Secret Key,
// 获取这些信息的方法如下：
// 1) 登录 云对象存储, 进入云对象存储空间；
// 2) 如开发者未创建空间，可添加空间，空间名称(Bucket)由用户自行输入
// 3) 点击“获取secretKey”，获取 Appid，Secret ID 和 Secret Key
type Signature struct {
	AppId       string
	Bucket      string
	SecretId    string
	ExpiredTime string
	CurrentTime string
	Rand        string
	FileId      string
}

// 构造签名类, 可使用该类的 Sign 和 SignOnce 接口分别进行多次有效签名和单次有效签名
func NewSignature(appId, bucket, secretId, expiredTime, currentTime, rand, fileId string) *Signature {
	return &Signature{
		AppId:       appId,
		Bucket:      bucket,
		SecretId:    secretId,
		ExpiredTime: expiredTime,
		CurrentTime: currentTime,
		Rand:        rand,
		FileId:      fileId,
	}
}

// 多次有效签名, secretKey 为项目的 Secret Key
func (s *Signature) Sign(secretKey string) string {
	stringToSign := fmt.Sprintf("a=%s&k=%s&e=%s&t=%s&r=%s&f=%s&b=%s",
		s.AppId,
		s.SecretId,
		s.ExpiredTime,
		s.CurrentTime,
		s.Rand,
		"",
		s.Bucket,
	)

	hmacSha1 := hmac.New(sha1.New, []byte(secretKey))
	hmacSha1.Write([]byte(stringToSign))
	bytesSign := hmacSha1.Sum(nil)
	bytesSign = append(bytesSign, []byte(stringToSign)...)
	signature := base64.StdEncoding.EncodeToString(bytesSign)
	return signature
}

// 单次有效签名, secretKey 为项目的 Secret Key
func (s *Signature) SignOnce(secretKey string) string {
	stringToSign := fmt.Sprintf("a=%s&k=%s&e=%s&t=%s&r=%s&f=%s&b=%s",
		s.AppId,
		s.SecretId,
		"0",
		s.CurrentTime,
		s.Rand,
		s.FileId,
		s.Bucket,
	)

	hmacSha1 := hmac.New(sha1.New, []byte(secretKey))
	hmacSha1.Write([]byte(stringToSign))
	bytesSign := hmacSha1.Sum(nil)
	bytesSign = append(bytesSign, []byte(stringToSign)...)
	signature := base64.StdEncoding.EncodeToString(bytesSign)
	return signature
}

// 字符串签名类
func (s *Signature) String() string {
	str := fmt.Sprintf("a=%s&b=%s&k=%s&e=%s&t=%s&r=%s&f=%s",
		s.AppId,
		s.SecretId,
		s.ExpiredTime,
		s.CurrentTime,
		s.Rand,
		s.FileId,
		s.Bucket,
	)

	return str
}
