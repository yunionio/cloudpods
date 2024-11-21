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

package baidu

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
)

func uriEncode(uri string, encodeSlash bool) string {
	var byte_buf bytes.Buffer
	for _, b := range []byte(uri) {
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') ||
			b == '-' || b == '_' || b == '.' || b == '~' || (b == '/' && !encodeSlash) {
			byte_buf.WriteByte(b)
		} else {
			byte_buf.WriteString(fmt.Sprintf("%%%02X", b))
		}
	}
	return byte_buf.String()
}

func getCanonicalURIPath(path string) string {
	if len(path) == 0 {
		return "/"
	}
	canonical_path := path
	if strings.HasPrefix(path, "/") {
		canonical_path = path[1:]
	}

	canonical_path = uriEncode(canonical_path, false)
	return "/" + canonical_path
}

func getCanonicalQueryString(params url.Values) string {
	if len(params) == 0 {
		return ""
	}
	result := make([]string, 0, len(params))
	for k, v := range params {
		if strings.EqualFold(k, "authorization") {
			continue
		}
		item := ""
		if len(v) == 0 {
			item = fmt.Sprintf("%s=", uriEncode(k, true))
		} else {
			item = fmt.Sprintf("%s=%s", uriEncode(k, true), uriEncode(params.Get(k), true))
		}
		result = append(result, item)
	}
	sort.Strings(result)
	return strings.Join(result, "&")
}

func getCanonicalHeaders(headers http.Header,
	headersToSign map[string]bool) (string, []string) {
	canonicalHeaders := make([]string, 0, len(headers))
	signHeaders := make([]string, 0, len(headersToSign))
	for k := range headers {
		headKey := strings.ToLower(k)
		if headKey == strings.ToLower("authorization") {
			continue
		}
		_, headExists := headersToSign[headKey]
		if headExists ||
			(strings.HasPrefix(headKey, "x-bce-") &&
				(headKey != "x-bce-request-id")) {

			headVal := strings.TrimSpace(headers.Get(k))
			encoded := uriEncode(headKey, true) + ":" + uriEncode(headVal, true)
			canonicalHeaders = append(canonicalHeaders, encoded)
			signHeaders = append(signHeaders, headKey)
		}
	}
	sort.Strings(canonicalHeaders)
	sort.Strings(signHeaders)
	return strings.Join(canonicalHeaders, "\n"), signHeaders
}

func (cli *SBaiduClient) sign(req *http.Request) (string, error) {
	signKeyInfo := fmt.Sprintf("%s/%s/%s/%d",
		"bce-auth-v1",
		cli.accessKeyId,
		time.Now().UTC().Format(ISO8601),
		1800)
	hasher := hmac.New(sha256.New, []byte(cli.accessKeySecret))
	hasher.Write([]byte(signKeyInfo))
	signKey := hex.EncodeToString(hasher.Sum(nil))
	canonicalUri := getCanonicalURIPath(req.URL.Path)
	params, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return "", errors.Wrapf(err, "ParseQuery")
	}
	canonicalQueryString := getCanonicalQueryString(params)
	canonicalHeaders, signedHeadersArr := getCanonicalHeaders(req.Header, map[string]bool{
		"host":           true,
		"Content-Length": true,
		"Content-Type":   true,
		"Content-Md5":    true,
	})

	signedHeaders := ""
	if len(signedHeadersArr) > 0 {
		sort.Strings(signedHeadersArr)
		signedHeaders = strings.Join(signedHeadersArr, ";")
	}

	canonicalParts := []string{req.Method, canonicalUri, canonicalQueryString, canonicalHeaders}
	canonicalReq := strings.Join(canonicalParts, "\n")
	hasher = hmac.New(sha256.New, []byte(signKey))
	hasher.Write([]byte(canonicalReq))
	signature := hex.EncodeToString(hasher.Sum(nil))

	return signKeyInfo + "/" + signedHeaders + "/" + signature, nil
}
