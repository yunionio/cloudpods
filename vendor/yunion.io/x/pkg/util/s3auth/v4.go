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
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/streamutils"
	"yunion.io/x/pkg/util/timeutils"
)

// Signature and API related constants.
const (
	signV4Algorithm   = "AWS4-HMAC-SHA256"
	iso8601DateFormat = "20060102T150405Z"
	yyyymmdd          = "20060102"

	unsignedPayload = "UNSIGNED-PAYLOAD"
)

// getScope generate a string of a specific date, an AWS region, and a
// service.
func getScope(location string, t time.Time) string {
	scope := strings.Join([]string{
		t.Format(yyyymmdd),
		location,
		"s3",
		"aws4_request",
	}, "/")
	return scope
}

// sum256 calculate sha256 sum for an input byte array.
func sum256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

// getStringToSign a string based on selected query values.
func getStringToSignV4(t time.Time, location, canonicalRequest string) string {
	stringToSign := signV4Algorithm + "\n" + t.Format(iso8601DateFormat) + "\n"
	stringToSign += getScope(location, t) + "\n"
	stringToSign += hex.EncodeToString(sum256([]byte(canonicalRequest)))
	return stringToSign
}

// /
// / Excerpts from @lsegal -
// / https://github.com/aws/aws-sdk-js/issues/659#issuecomment-120477258.
// /
// /  User-Agent:
// /
// /      This is ignored from signing because signing this causes
// /      problems with generating pre-signed URLs (that are executed
// /      by other agents) or when customers pass requests through
// /      proxies, which may modify the user-agent.
// /
// /  Content-Length:
// /
// /      This is ignored from signing because generating a pre-signed
// /      URL should not provide a content-length constraint,
// /      specifically when vending a S3 pre-signed PUT URL. The
// /      corollary to this is that when sending regular requests
// /      (non-pre-signed), the signature contains a checksum of the
// /      body, which implicitly validates the payload length (since
// /      changing the number of bytes would change the checksum)
// /      and therefore this header is not valuable in the signature.
// /
// /  Content-Type:
// /
// /      Signing this header causes quite a number of problems in
// /      browser environments, where browsers like to modify and
// /      normalize the content-type header in different ways. There is
// /      more information on this in https://goo.gl/2E9gyy. Avoiding
// /      this field simplifies logic and reduces the possibility of
// /      future bugs.
// /
// /  Authorization:
// /
// /      Is skipped for obvious reasons
// /
var v4IgnoredHeaders = map[string]bool{
	"Authorization":  true,
	"Content-Type":   true,
	"Content-Length": true,
	"User-Agent":     true,
}

// sumHMAC calculate hmac between two input byte array.
func sumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

// getSigningKey hmac seed to calculate final signature.
func getSigningKey(secret, loc string, t time.Time) []byte {
	date := sumHMAC([]byte("AWS4"+secret), []byte(t.Format(yyyymmdd)))
	location := sumHMAC(date, []byte(loc))
	service := sumHMAC(location, []byte("s3"))
	signingKey := sumHMAC(service, []byte("aws4_request"))
	return signingKey
}

// getSignature final signature in hexadecimal form.
func getSignature(signingKey []byte, stringToSign string) string {
	return hex.EncodeToString(sumHMAC(signingKey, []byte(stringToSign)))
}

// getCanonicalRequest generate a canonical request of style.
//
// canonicalRequest =
//
//	<HTTPMethod>\n
//	<CanonicalURI>\n
//	<CanonicalQueryString>\n
//	<CanonicalHeaders>\n
//	<SignedHeaders>\n
//	<HashedPayload>
func getCanonicalRequest(req http.Request, signedHeaders []string) string {
	req.URL.RawQuery = strings.Replace(req.URL.Query().Encode(), "+", "%20", -1)
	canonicalRequest := strings.Join([]string{
		req.Method,
		encodePath(req.URL.Path),
		req.URL.RawQuery,
		getCanonicalHeaders(req, signedHeaders),
		strings.Join(signedHeaders, ";"),
		getHashedPayload(req),
	}, "\n")
	return canonicalRequest
}

// Trim leading and trailing spaces and replace sequential spaces with one space, following Trimall()
// in http://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html
func signV4TrimAll(input string) string {
	// Compress adjacent spaces (a space is determined by
	// unicode.IsSpace() internally here) to one space and return
	return strings.Join(strings.Fields(input), " ")
}

// getCanonicalHeaders generate a list of request headers for
// signature.
func getCanonicalHeaders(req http.Request, signedHeaders []string) string {
	var buf bytes.Buffer
	// Save all the headers in canonical form <header>:<value> newline
	// separated for each header.
	for _, k := range signedHeaders {
		buf.WriteString(k)
		buf.WriteByte(':')
		switch {
		case k == "host":
			buf.WriteString(getHostAddr(req))
			fallthrough
		default:
			for idx, v := range req.Header[http.CanonicalHeaderKey(k)] {
				if idx > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(signV4TrimAll(v))
			}
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

// getSignedHeaders generate all signed request headers.
// i.e lexically sorted, semicolon-separated list of lowercase
// request header names.
func getSignedHeaders(req http.Request, ignoredHeaders map[string]bool) []string {
	var headers []string
	hasHost := false
	hasContentHash := false
	for k := range req.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue // Ignored header found continue.
		}
		if strings.EqualFold(k, "host") {
			hasHost = true
		} else if strings.EqualFold(k, "x-amz-content-sha256") {
			hasContentHash = true
		}
		headers = append(headers, strings.ToLower(k))
	}
	if !hasHost {
		headers = append(headers, "host")
	}
	if !hasContentHash {
		headers = append(headers, "x-amz-content-sha256")
	}
	sort.Strings(headers)
	return headers
}

// getHashedPayload get the hexadecimal value of the SHA256 hash of
// the request payload.
func getHashedPayload(req http.Request) string {
	hashedPayload := req.Header.Get("X-Amz-Content-Sha256")
	if hashedPayload == "" {
		// Presign does not have a payload, use S3 recommended value.
		hashedPayload = unsignedPayload
	}
	return hashedPayload
}

type SAccessKeyRequestV4 struct {
	SAccessKeyRequest
	Location      string
	SignedHeaders []string
	SignDate      time.Time
}

func NewV4Request() SAccessKeyRequestV4 {
	req := SAccessKeyRequestV4{}
	req.Algorithm = signV4Algorithm
	return req
}

// AWS4-HMAC-SHA256
// Credential=xxxx/20190824/us-east-1/s3/aws4_request,SignedHeaders=date;host;x-amz-content-sha256;x-amz-date,Signature=27a135c6f51cc
func decodeAuthHeaderV4(authStr string) (*SAccessKeyRequestV4, error) {
	req := NewV4Request()
	parts := strings.Split(authStr, ",")
	if len(parts) != 3 ||
		!strings.HasPrefix(parts[0], "Credential=") ||
		!strings.HasPrefix(parts[1], "SignedHeaders=") ||
		!strings.HasPrefix(parts[2], "Signature=") {
		return nil, errors.Error("illegal v4 auth header")
	}
	credParts := strings.Split(parts[0][len("Credential="):], "/")
	if len(credParts) != 5 {
		return nil, errors.Error("illegal v4 auth header Credential")
	}
	req.AccessKey = credParts[0]
	req.Location = credParts[2]
	req.SignedHeaders = strings.Split(parts[1][len("SignedHeaders="):], ";")
	sort.Strings(req.SignedHeaders)
	req.Signature = parts[2][len("Signature="):]
	return &req, nil
}

func (aksk *SAccessKeyRequestV4) ParseRequest(req http.Request, virtualHost bool) error {
	dateStr := req.Header.Get(http.CanonicalHeaderKey("x-amz-date"))
	if len(dateStr) == 0 {
		return errors.Error("missing x-amz-date")
	}
	dateSign, err := time.Parse(iso8601DateFormat, dateStr)
	if err != nil {
		return errors.Wrap(err, "time.Parse")
	}
	canonicalReq := getCanonicalRequest(req, aksk.SignedHeaders)
	aksk.SignDate = dateSign
	aksk.Request = getStringToSignV4(dateSign, aksk.Location, canonicalReq)
	return nil
}

func (aksk SAccessKeyRequestV4) Verify(secret string) error {
	signingKey := getSigningKey(secret, aksk.Location, aksk.SignDate)
	signature := getSignature(signingKey, aksk.Request)
	if signature != aksk.Signature {
		return errors.Error("signature mismatch")
	}
	return nil
}

func (aksk SAccessKeyRequestV4) Encode() string {
	return jsonutils.Marshal(aksk).String()
}

// GetCredential generate a credential string.
func getCredential(accessKeyID, location string, t time.Time) string {
	scope := getScope(location, t)
	return accessKeyID + "/" + scope
}

// SignV4 sign the request before Do(), in accordance with
// http://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html.
func SignV4(req http.Request, accessKey, secretAccessKey, location string, body io.Reader) *http.Request {
	// Signature calculation is not needed for anonymous credentials.
	if accessKey == "" || secretAccessKey == "" {
		return &req
	}

	h := sha256.New()
	if body != nil {
		streamutils.StreamPipe(body, h, false, nil)
	}
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(h.Sum(nil)))

	// Initial time.
	t := time.Now().UTC()

	// Set x-amz-date.
	req.Header.Set("X-Amz-Date", t.Format(iso8601DateFormat))
	req.Header.Set("Date", timeutils.RFC2882Time(t))

	signedHeaders := getSignedHeaders(req, v4IgnoredHeaders)
	// Get canonical request.
	canonicalRequest := getCanonicalRequest(req, signedHeaders)

	// Get string to sign from canonical request.
	stringToSign := getStringToSignV4(t, location, canonicalRequest)

	// Get hmac signing key.
	signingKey := getSigningKey(secretAccessKey, location, t)

	// Get credential string.
	credential := getCredential(accessKey, location, t)

	// Calculate signature.
	signature := getSignature(signingKey, stringToSign)

	// If regular request, construct the final authorization header.
	parts := []string{
		signV4Algorithm + " Credential=" + credential,
		"SignedHeaders=" + strings.Join(signedHeaders, ";"),
		"Signature=" + signature,
	}

	// Set authorization header.
	auth := strings.Join(parts, ",")
	req.Header.Set("Authorization", auth)

	return &req
}
