package open_api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"net/url"
	"sort"
	"time"
)

func Sign(req *aws.Request) {
	if req.Config.Credentials == credentials.AnonymousCredentials {
		return
	}

	CredValues, err := req.Config.Credentials.Get()
	if err != nil {
		req.Error = err
		return
	}

	params := map[string]string{
		//固定参数
		"Accesskey":        CredValues.AccessKeyID,
		"Service":          "ks3bill",
		"Version":          "v1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureMethod":  "HMAC-SHA256",
	}

	query := req.HTTPRequest.URL.Query()
	for k, v := range params {
		query.Set(k, v)
	}

	queryString := getCanonicalizedQueryString(query)
	cfg := req.Config
	cfg.LogDebug("%s", "---[ QUERY STRING ]--------------------------------")
	cfg.LogDebug("%s", queryString)
	cfg.LogDebug("-----------------------------------------------------")

	signature := getSignature(queryString, CredValues.SecretAccessKey)
	query.Set("Signature", signature)
	req.HTTPRequest.URL.RawQuery = query.Encode()

	return
}

// getCanonicalizedQueryString 构建规范化查询字符串
func getCanonicalizedQueryString(params url.Values) string {
	//对参数键进行排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	//构建待签名字符串
	var strEncode string
	for _, k := range keys {
		strEncode += url.QueryEscape(k) + "=" + url.QueryEscape(params.Get(k)) + "&"
	}
	strEncode = strEncode[:len(strEncode)-1]
	return strEncode
}

// getSignature 简易签名函数，使用HMAC-SHA256算法
func getSignature(queryString, secretKey string) string {
	//生成HMAC-SHA256签名
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}
