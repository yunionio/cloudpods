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

package s3auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

// Signature and API related constants.
const (
	signV2Algorithm = "AWS"
)

// From the Amazon docs:
//
// StringToSign = HTTP-Verb + "\n" +
//
//	Content-Md5 + "\n" +
//	Content-Type + "\n" +
//	Date + "\n" +
//	CanonicalizedProtocolHeaders +
//	CanonicalizedResource;
func stringToSignV2(req http.Request, virtualHost bool) string {
	buf := new(bytes.Buffer)
	// Write standard headers.
	writeSignV2Headers(buf, req)
	// Write canonicalized protocol headers if any.
	writeCanonicalizedHeaders(buf, req)
	// Write canonicalized Query resources if any.
	writeCanonicalizedResource(buf, req, virtualHost)
	return buf.String()
}

// writeSignV2Headers - write signV2 required headers.
func writeSignV2Headers(buf *bytes.Buffer, req http.Request) {
	buf.WriteString(req.Method + "\n")
	buf.WriteString(req.Header.Get("Content-Md5") + "\n")
	buf.WriteString(req.Header.Get("Content-Type") + "\n")
	buf.WriteString(req.Header.Get("Date") + "\n")
}

// writeCanonicalizedHeaders - write canonicalized headers.
func writeCanonicalizedHeaders(buf *bytes.Buffer, req http.Request) {
	var protoHeaders []string
	vals := make(map[string][]string)
	for k, vv := range req.Header {
		// All the AMZ headers should be lowercase
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz") {
			protoHeaders = append(protoHeaders, lk)
			vals[lk] = vv
		}
	}
	sort.Strings(protoHeaders)
	for _, k := range protoHeaders {
		buf.WriteString(k)
		buf.WriteByte(':')
		for idx, v := range vals[k] {
			if idx > 0 {
				buf.WriteByte(',')
			}
			if strings.Contains(v, "\n") {
				// TODO: "Unfold" long headers that
				// span multiple lines (as allowed by
				// RFC 2616, section 4.2) by replacing
				// the folding white-space (including
				// new-line) by a single space.
				buf.WriteString(v)
			} else {
				buf.WriteString(v)
			}
		}
		buf.WriteByte('\n')
	}
}

// AWS S3 Signature V2 calculation rule is give here:
// http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html#RESTAuthenticationStringToSign

// Whitelist resource list that will be used in query string for signature-V2 calculation.
// The list should be alphabetically sorted
var resourceList = []string{
	"acl",
	"delete",
	"lifecycle",
	"location",
	"logging",
	"notification",
	"partNumber",
	"policy",
	"requestPayment",
	"response-cache-control",
	"response-content-disposition",
	"response-content-encoding",
	"response-content-language",
	"response-content-type",
	"response-expires",
	"torrent",
	"uploadId",
	"uploads",
	"versionId",
	"versioning",
	"versions",
	"website",
}

// From the Amazon docs:
//
// CanonicalizedResource = [ "/" + Bucket ] +
//
//	<HTTP-Request-URI, from the protocol name up to the query string> +
//	[ sub-resource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
func writeCanonicalizedResource(buf *bytes.Buffer, req http.Request, virtualHost bool) {
	// Save request URL.
	requestURL := req.URL
	// Get encoded URL path.
	buf.WriteString(encodeURL2Path(req, virtualHost))
	if requestURL.RawQuery != "" {
		var n int
		vals, _ := url.ParseQuery(requestURL.RawQuery)
		// Verify if any sub resource queries are present, if yes
		// canonicallize them.
		for _, resource := range resourceList {
			if vv, ok := vals[resource]; ok && len(vv) > 0 {
				n++
				// First element
				switch n {
				case 1:
					buf.WriteByte('?')
					// The rest
				default:
					buf.WriteByte('&')
				}
				buf.WriteString(resource)
				// Request parameters
				if len(vv[0]) > 0 {
					buf.WriteByte('=')
					buf.WriteString(vv[0])
				}
			}
		}
	}
}

// Authorization = "AWS" + " " + AWSAccessKeyId + ":" + Signature;
// Signature = Base64( HMAC-SHA1( YourSecretAccessKeyID, UTF-8-Encoding-Of( StringToSign ) ) );
//
// StringToSign = HTTP-Verb + "\n" +
//  	Content-Md5 + "\n" +
//  	Content-Type + "\n" +
//  	Date + "\n" +
//  	CanonicalizedProtocolHeaders +
//  	CanonicalizedResource;
//
// CanonicalizedResource = [ "/" + Bucket ] +
//  	<HTTP-Request-URI, from the protocol name up to the query string> +
//  	[ subresource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
//
// CanonicalizedProtocolHeaders = <described below>
// https://${S3_BUCKET}.s3.amazonaws.com/${S3_OBJECT}?AWSAccessKeyId=${S3_ACCESS_KEY}&Expires=${TIMESTAMP}&Signature=${SIGNATURE}.
/*func verifyV2(ctx context.Context, req http.Request, virtualHost bool) error {
	aksk, err := DecodeAccessKeyRequestV2(req, virtualHost)
	if err != nil {
		return errors.Wrap(err, "DecodeAccessKeyRequestV2")
	}

	authSession := session.GetAdminSession(ctx)
	result, err := modules.Credentials.Get(authSession, aksk.AccessKey, nil)
	if err != nil {
		return errors.Wrap(err, "modules.Credentials.Get")
	}
	secret, err := modules.DecodeAccessKeySecret(result)
	if err != nil {
		return errors.Wrap(err, "modules.DecodeAccessKeySecret")
	}

	hm := hmac.New(sha1.New, []byte(secret.Secret))
	hm.Write([]byte(aksk.RequestString))

	signature := base64.StdEncoding.EncodeToString(hm.Sum(nil))

	if aksk.Signature != signature {
		return errors.Error("signature mismatch")
	}

	return nil
}*/

type SAccessKeyRequestV2 struct {
	SAccessKeyRequest
}

func (aksk *SAccessKeyRequestV2) ParseRequest(req http.Request, virtualHost bool) error {
	aksk.Request = stringToSignV2(req, virtualHost)
	return nil
}

func (aksk SAccessKeyRequestV2) Verify(secret string) error {
	hm := hmac.New(sha1.New, []byte(secret))
	hm.Write([]byte(aksk.Request))

	signature := base64.StdEncoding.EncodeToString(hm.Sum(nil))
	if signature != aksk.Signature {
		return errors.Error("signature mismatch")
	}

	return nil
}

func (aksk SAccessKeyRequestV2) Encode() string {
	return jsonutils.Marshal(aksk).String()
}

func NewV2Request() SAccessKeyRequestV2 {
	req := SAccessKeyRequestV2{}
	req.Algorithm = signV2Algorithm
	return req
}

func decodeAuthHeaderV2(authStr string) (*SAccessKeyRequestV2, error) {
	akskReq := NewV2Request()
	pos := strings.IndexByte(authStr, ':')
	if pos <= 0 {
		return nil, errors.Error("illegal authorization header")
	}
	akskReq.AccessKey = authStr[:pos]
	akskReq.Signature = authStr[pos+1:]
	return &akskReq, nil
}
