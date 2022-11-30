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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/auth/credentials"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/auth/signers"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/requests"
)

type Signer interface {
	GetName() string                                 // 签名算法名称
	GetAccessKeyId() (accessKeyId string, err error) // 签名access key
	GetSecretKey() (secretKey string, err error)     // access secret
	Sign(stringToSign, secretSuffix string) string   // 生成签名结果
}

func NewSignerWithCredential(credential Credential) (signer Signer, err error) {
	switch instance := credential.(type) {
	case *credentials.AccessKeyCredential:
		return signers.NewAccessKeySigner(instance), nil
	default:
		return nil, fmt.Errorf("unsupported credential error")
	}
}

// 对request进行签名
func Sign(request requests.IRequest, signer Signer) (err error) {
	return signRequest(request, signer)
}

func signRequest(request requests.IRequest, signer Signer) error {
	// https://support.huaweicloud.com/api-dis/dis_02_0508.html
	// requestTime
	reqTime := time.Now()
	// 添加 必须的Headers
	fillRequiredHeaders(request, reqTime)
	// 计算CanonicalRequest
	canonicalRequest := canonicalRequest(request)
	// stringToSign
	credentialScope := strings.Join([]string{
		formattedSignTime(reqTime, "Date"),
		request.GetRegionId(),
		request.GetProduct(),
		"sdk_request",
	}, "/")
	stringToSign := strings.Join([]string{"SDK-HMAC-SHA256",
		formattedSignTime(reqTime, "DateTime"),
		credentialScope,
		hashSha256([]byte(canonicalRequest)),
	}, "\n")
	// 计算SigningKey
	secret, _ := signer.GetSecretKey()
	signKey := getSigningKey(secret, formattedSignTime(reqTime, "Date"),
		request.GetRegionId(), request.GetProduct())
	// 计算Signature
	signature := signer.Sign(stringToSign, signKey)
	accesskey, _ := signer.GetAccessKeyId()
	addAuthorizationHeader(request, accesskey, credentialScope, signature)
	return nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0132456728.html
// X-Project-Id 如果是专属云场景采用AK/SK 认证方式的接口请求或者多project场景采用AK/SK认证的接口请求则该字段必选。
func fillRequiredHeaders(request requests.IRequest, t time.Time) {
	request.AddHeaderParam("HOST", request.GetHost())
	request.AddHeaderParam("X-Sdk-Date", formattedSignTime(t, "DateTime"))
	if len(request.GetProjectId()) > 0 {
		request.AddHeaderParam("X-Project-Id", request.GetProjectId())
	}
	return
}

func buildRequestStringToSign(request requests.IRequest) string {
	return ""
}

func formattedSignTime(t time.Time, format string) string {
	switch format {
	case "Date":
		return t.UTC().Format("20060102")
	case "DateTime":
		return t.UTC().Format("20060102T150405Z")
	default:
		return t.UTC().Format("20060102T150405Z")
	}
}

func contentSha256(request requests.IRequest) string {
	method := strings.ToUpper(request.GetMethod())
	content := []byte{}
	body := request.GetBodyReader()
	content, _ = ioutil.ReadAll(body)
	if method == "POST" {
		if len(content) == 0 {
			// other http method use query as content
			content = []byte(request.BuildQueries())
		}
	}

	return hashSha256(content)
}

func hashSha256(msg []byte) string {
	sh256 := sha256.New()
	sh256.Write(msg)

	return hex.EncodeToString(sh256.Sum(nil))
}

func canonicalRequest(request requests.IRequest) string {
	sha256 := contentSha256(request)
	uri := request.GetURI()
	if !strings.HasSuffix(uri, "/") {
		uri = uri + "/"
	}

	return strings.Join([]string{
		request.GetMethod(),
		uri,
		canonicalQueryString(request),
		canonicalHeaders(request),
		canonicalHeaderNames(request),
		sha256,
	}, "\n")
}

func sortedHeaderNames(request requests.IRequest) []string {
	headers := request.GetHeaders()
	keys := make([]string, 0)
	for k := range headers {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	return keys
}

func canonicalQueryString(request requests.IRequest) string {
	if strings.ToUpper(request.GetMethod()) != "POST" {
		return request.BuildQueries()
	}

	return ""
}

func canonicalHeaders(request requests.IRequest) string {
	keys := sortedHeaderNames(request)
	headers := request.GetHeaders()
	ret := []string{}
	for _, k := range keys {
		ret = append(ret, strings.ToLower(k)+":"+strings.TrimSpace(headers[k]))
	}

	return strings.Join(ret, "\n") + "\n"
}

func canonicalHeaderNames(request requests.IRequest) string {
	keys := sortedHeaderNames(request)
	ret := strings.Join(keys, ";")
	return strings.ToLower(ret)
}

func getSigningKey(secretKey, date, regionId, service string) string {
	ret := []byte("SDK" + secretKey)
	for _, k := range []string{date, regionId, service, "sdk_request"} {
		ret = signers.HmacSha256(k, ret)
	}
	return string(ret)
}

func addAuthorizationHeader(request requests.IRequest, accessKey, credentialScope, signature string) {
	auth := "SDK-HMAC-SHA256" + " " + strings.Join([]string{
		"Credential=" + accessKey + "/" + credentialScope,
		"SignedHeaders=" + canonicalHeaderNames(request),
		"Signature=" + signature,
	}, ", ")

	request.AddHeaderParam("Authorization", auth)
}
