package cos

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (conn *Conn) signHeader(req *http.Request, params map[string]interface{}, headers map[string]string) {
	signTime := getSignTime()
	signature := conn.getSignature(req, params, headers, signTime)
	authStr := fmt.Sprintf("q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=%s&q-url-param-list=%s&q-signature=%s",
		conn.conf.SecretID, signTime, signTime, getHeadKeys(headers), getParamKeys(params), signature)

	req.Header.Set("Authorization", authStr)
}

func getSignTime() string {
	now := time.Now()
	expired := now.Add(time.Second * 1800)
	return fmt.Sprintf("%d;%d", now.Unix(), expired.Unix())
}

func getHeadKeys(headers map[string]string) string {
	if headers == nil || len(headers) == 0 {
		return ""
	}

	tmp := []string{}
	for k := range headers {
		tmp = append(tmp, strings.ToLower(k))
	}
	sort.Strings(tmp)

	return strings.Join(tmp, ";")
}

func getParamKeys(params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return ""
	}

	tmp := []string{}
	for k := range params {
		tmp = append(tmp, strings.ToLower(k))
	}
	sort.Strings(tmp)

	return strings.Join(tmp, ";")
}

func (conn *Conn) getSignature(req *http.Request, params map[string]interface{}, headers map[string]string, signTime string) string {
	httpString := fmt.Sprintf("%s\n%s\n%s\n%s\n", strings.ToLower(req.Method),
		req.URL.Path, getParamStr(params), getHeadStr(headers))

	httpString = sha(httpString)
	signKey := hmacSha(conn.conf.SecretKey, signTime)
	signStr := fmt.Sprintf("sha1\n%s\n%s\n", signTime, httpString)

	return hmacSha(signKey, signStr)
}

func interfaceToString(i interface{}) string {
	switch x := i.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	default:
		return ""
	}
}

func getParamStr(params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return ""
	}

	tmp := []string{}
	for k, v := range params {
		str := strings.ToLower(fmt.Sprintf("%s=%s", k, interfaceToString(v)))
		tmp = append(tmp, str)
	}
	sort.Strings(tmp)

	return strings.Join(tmp, "&")
}

func getHeadStr(headers map[string]string) string {
	if headers == nil || len(headers) == 0 {
		return ""
	}

	tmp := []string{}
	for k, v := range headers {
		str := fmt.Sprintf("%s=%s", strings.ToLower(k), escape(v))
		tmp = append(tmp, str)
	}
	sort.Strings(tmp)

	return strings.Join(tmp, "&")
}

func sha(s string) string {
	sha := sha1.New()
	sha.Write([]byte(s))
	b := sha.Sum(nil)

	return hex.EncodeToString(b)
}

func hmacSha(k, s string) string {
	enc := hmac.New(sha1.New, []byte(k))
	enc.Write([]byte(s))
	b := enc.Sum(nil)

	return hex.EncodeToString(b)
}
