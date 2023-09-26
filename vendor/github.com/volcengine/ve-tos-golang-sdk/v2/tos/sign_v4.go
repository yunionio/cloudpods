package tos

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	emptySHA256      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	unsignedPayload  = "UNSIGNED-PAYLOAD"
	signPrefix       = "TOS4-HMAC-SHA256"
	iso8601Layout    = "20060102T150405Z"
	yyMMdd           = "20060102"
	serverTimeFormat = "2006-01-02T15:04:05Z"
	authorization    = "Authorization"

	v4Algorithm      = "X-Tos-Algorithm"
	v4Credential     = "X-Tos-Credential"
	v4Date           = "X-Tos-Date"
	v4Expires        = "X-Tos-Expires"
	v4SignedHeaders  = "X-Tos-SignedHeaders"
	v4Signature      = "X-Tos-Signature"
	v4SignatureLower = "x-tos-signature"
	v4ContentSHA256  = "X-Tos-Content-Sha256"
	v4SecurityToken  = "X-Tos-Security-Token"

	v4Prefix = "x-tos"
)

func defaultSigningQueryV4(key string) bool {
	return key != v4SignatureLower
}

func defaultSigningHeaderV4(key string, isSigningQuery bool) bool {
	return (key == "content-type" && !isSigningQuery) || strings.HasPrefix(key, v4Prefix)
}

func UTCNow() time.Time {
	return time.Now().UTC()
}

type Signer interface {
	SignHeader(req *Request) http.Header

	SignQuery(req *Request, ttl time.Duration) url.Values
}

type SigningKeyInfo struct {
	Date       string
	Region     string
	Credential *Credential
}

type SignV4 struct {
	credentials   Credentials
	region        string
	signingHeader func(key string, isSigningQuery bool) bool
	signingQuery  func(key string) bool
	now           func() time.Time
	signingKey    func(*SigningKeyInfo) []byte
	logger        Logger
}

type signedRes struct {
	CanonicalString string
	StringToSign    string
	Sign            string
}

type signedHeader struct {
	CanonicalString string
	StringToSign    string
	Header          http.Header
}

type signedQuery struct {
	CanonicalString string
	StringToSign    string
	Query           url.Values
}

type SignV4Option func(*SignV4)

func (sv *SignV4) WithSignLogger(logger Logger) {
	sv.logger = logger
}

// NewSignV4 create SignV4
// use WithSignKey to set self-defined sign-key generator
// use WithSignLogger to set logger
func NewSignV4(credentials Credentials, region string) *SignV4 {
	signV4 := &SignV4{
		credentials:   credentials,
		region:        region,
		signingHeader: defaultSigningHeaderV4,
		signingQuery:  defaultSigningQueryV4,
		now:           UTCNow,
		signingKey:    SigningKey,
	}
	return signV4
}

// WithSigningKey for self-defined sign-key generator
func (sv *SignV4) WithSigningKey(signingKey func(*SigningKeyInfo) []byte) {
	sv.signingKey = signingKey
}

func (sv *SignV4) signedHeader(header http.Header, isSignedQuery bool) KVs {
	var signed = make(KVs, 0, 10)
	for key, values := range header {
		kk := strings.ToLower(key)
		if sv.signingHeader(kk, isSignedQuery) {
			vv := make([]string, 0, len(values))
			for _, value := range values {
				vv = append(vv, strings.Join(strings.Fields(value), " "))
			}
			signed = append(signed, KV{Key: kk, Values: vv})
		}
	}
	return signed
}

func (sv *SignV4) signedQuery(query url.Values, extra url.Values) KVs {
	var signed = make(KVs, 0, len(query)+len(extra))
	for key, values := range query {
		if sv.signingQuery(strings.ToLower(key)) {
			signed = append(signed, KV{Key: key, Values: values})
		}
	}
	for key, values := range extra {
		if sv.signingQuery(strings.ToLower(key)) {
			signed = append(signed, KV{Key: key, Values: values})
		}
	}
	return signed
}

func (sv *SignV4) canonicalRequest(method, path, contentSha256 string, header, query KVs) string {
	const split = byte('\n')
	var buf bytes.Buffer
	buf.Grow(512)

	// Method
	buf.WriteString(method)
	buf.WriteByte(split)

	// URI
	buf.Write(encodePath(path))
	buf.WriteByte(split)

	// query
	buf.Write(encodeQuery(query))
	buf.WriteByte(split)

	// canonical headers
	keys := make([]string, 0, len(header))
	for _, kv := range header {
		keys = append(keys, kv.Key)
		buf.WriteString(kv.Key)
		buf.WriteByte(':')
		buf.WriteString(strings.Join(kv.Values, ","))
		buf.WriteByte('\n')
	}
	buf.WriteByte(split)

	// signed headers
	buf.WriteString(strings.Join(keys, ";"))
	buf.WriteByte(split)

	if len(contentSha256) > 0 {
		buf.WriteString(contentSha256)
	} else {
		buf.WriteString(emptySHA256)
	}
	return buf.String()
}

func SigningKey(info *SigningKeyInfo) []byte {
	date := hmacSHA256([]byte(info.Credential.AccessKeySecret), []byte(info.Date))
	region := hmacSHA256(date, []byte(info.Region))
	service := hmacSHA256(region, []byte("tos"))
	return hmacSHA256(service, []byte("request"))
}

func (sv *SignV4) doSign(method, path, contentSha256 string, header, query KVs, now time.Time, cred *Credential) signedRes {
	const split = byte('\n')

	canonicalStr := sv.canonicalRequest(method, path, contentSha256, header, query)

	var buf bytes.Buffer
	buf.Grow(len(signPrefix) + 128)

	buf.WriteString(signPrefix)
	buf.WriteByte(split)

	buf.WriteString(now.Format(iso8601Layout))
	buf.WriteByte(split)

	date := now.Format(yyMMdd)
	buf.WriteString(date) // yyMMdd + '/' + region + '/' + service + '/' + request
	buf.WriteByte('/')
	buf.WriteString(sv.region)

	buf.WriteString("/tos/request")
	buf.WriteByte(split)

	sum := sha256.Sum256([]byte(canonicalStr))
	buf.WriteString(hex.EncodeToString(sum[:]))

	signK := sv.signingKey(&SigningKeyInfo{Date: date, Region: sv.region, Credential: cred})
	sign := hmacSHA256(signK, buf.Bytes())
	return signedRes{
		CanonicalString: canonicalStr,
		StringToSign:    buf.String(),
		Sign:            hex.EncodeToString(sign),
	}

}

func (sv *SignV4) SignHeader(req *Request) http.Header {
	signed := make(http.Header, 4)
	now := sv.now()
	date := now.Format(iso8601Layout)
	contentSha256 := req.Header.Get(v4ContentSHA256)

	signedHeader := sv.signedHeader(req.Header, false)
	signedHeader = append(signedHeader, KV{Key: strings.ToLower(v4Date), Values: []string{date}})
	signedHeader = append(signedHeader, KV{Key: "date", Values: []string{date}})
	signedHeader = append(signedHeader, KV{Key: "host", Values: []string{req.Host}})
	// if len(contentSha256) == 0 {
	//	signedHeader = append(signedHeader, KV{Key: strings.ToLower(v4ContentSHA256), Values: []string{unsignedPayload}})
	//	signed.Set(v4ContentSHA256, unsignedPayload)
	// }

	cred := sv.credentials.Credential()
	if sts := cred.SecurityToken; len(sts) > 0 {
		signedHeader = append(signedHeader, KV{Key: strings.ToLower(v4SecurityToken), Values: []string{sts}})
		signed.Set(v4SecurityToken, sts)
	}

	sort.Sort(signedHeader)
	signedQuery := sv.signedQuery(req.Query, nil)

	signRes := sv.doSign(req.Method, req.Path, contentSha256, signedHeader, signedQuery, now, &cred)
	credential := fmt.Sprintf("%s/%s/%s/tos/request", cred.AccessKeyID, now.Format(yyMMdd), sv.region)
	auth := fmt.Sprintf("TOS4-HMAC-SHA256 Credential=%s,SignedHeaders=%s,Signature=%s", credential, joinKeys(signedHeader), signRes.Sign)

	signed.Set(authorization, auth)
	signed.Set(v4Date, date)
	signed.Set("Date", date)
	if sv.logger != nil {
		sv.logger.Debug("[tos] CanonicalString:" + "\n" + signRes.CanonicalString + "\n")
		sv.logger.Debug("[tos] StringToSign:" + "\n" + signRes.StringToSign + "\n")
	}
	return signed
}

func (sv *SignV4) SignQuery(req *Request, ttl time.Duration) url.Values {
	now := sv.now()
	date := now.Format(iso8601Layout)
	query := req.Query
	extra := make(url.Values)

	cred := sv.credentials.Credential()
	credential := fmt.Sprintf("%s/%s/%s/tos/request", cred.AccessKeyID, now.Format(yyMMdd), sv.region)
	extra.Add(v4Algorithm, signPrefix)
	extra.Add(v4Credential, credential)

	extra.Add(v4Date, date)
	extra.Add(v4Expires, strconv.FormatInt(ttl.Milliseconds()/1000, 10))
	if sts := cred.SecurityToken; len(sts) > 0 {
		extra.Add(v4SecurityToken, sts)
	}

	signedHeader := sv.signedHeader(req.Header, true)
	signedHeader = append(signedHeader, KV{Key: "host", Values: []string{req.Host}})
	sort.Sort(signedHeader)

	extra.Add(v4SignedHeaders, joinKeys(signedHeader))
	signedQuery := sv.signedQuery(query, extra)

	signRes := sv.doSign(req.Method, req.Path, unsignedPayload, signedHeader, signedQuery, now, &cred)
	extra.Add(v4Signature, signRes.Sign)
	if sv.logger != nil {
		sv.logger.Debug("[tos] CanonicalString:" + "\n" + signRes.CanonicalString + "\n")
		sv.logger.Debug("[tos] StringToSign:" + "\n" + signRes.StringToSign + "\n")
	}
	return extra
}

type KV struct {
	Key    string
	Values []string
}

type KVs []KV

func (kvs KVs) Len() int           { return len(kvs) }
func (kvs KVs) Swap(i, j int)      { kvs[i], kvs[j] = kvs[j], kvs[i] }
func (kvs KVs) Less(i, j int) bool { return kvs[i].Key < kvs[j].Key }

func joinKeys(kvs KVs) string {
	keys := make([]string, 0, len(kvs))
	for i := range kvs {
		keys = append(keys, kvs[i].Key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func hmacSHA256(key []byte, value []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(value)
	return h.Sum(nil)
}

var (
	nonEscape [256]bool
)

// ((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' || ch == '~' || ch == '.')
func init() {
	for i := 'a'; i <= 'z'; i++ {
		nonEscape[i] = true
	}
	for i := 'A'; i <= 'Z'; i++ {
		nonEscape[i] = true
	}
	for i := '0'; i <= '9'; i++ {
		nonEscape[i] = true
	}
	nonEscape['-'] = true
	nonEscape['_'] = true
	nonEscape['.'] = true
	nonEscape['~'] = true
}

func encodePath(path string) []byte {
	if len(path) == 0 {
		return []byte{'/'}
	}

	return URIEncode(path, false)
}

func encodeQuery(query KVs) []byte {
	if len(query) == 0 {
		return make([]byte, 0)
	}

	var buf bytes.Buffer
	buf.Grow(512)

	sort.Sort(query)
	for _, kv := range query {
		keyEscaped := URIEncode(kv.Key, true)
		for _, v := range kv.Values {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.Write(keyEscaped)
			buf.WriteByte('=')
			buf.Write(URIEncode(v, true))
		}
	}
	return buf.Bytes()
}

func URIEncode(in string, encodeSlash bool) []byte {
	hexCount := 0
	for i := 0; i < len(in); i++ {
		c := uint8(in[i])
		if c == '/' {
			if encodeSlash {
				hexCount++
			}
		} else if !nonEscape[c] {
			hexCount++
		}
	}
	encoded := make([]byte, len(in)+2*hexCount)
	for i, j := 0, 0; i < len(in); i++ {
		c := uint8(in[i])
		if c == '/' {
			if encodeSlash {
				encoded[j] = '%'
				encoded[j+1] = '2'
				encoded[j+2] = 'F'
				j += 3
			} else {
				encoded[j] = c
				j++
			}
		} else if !nonEscape[c] {
			encoded[j] = '%'
			encoded[j+1] = "0123456789ABCDEF"[c>>4]
			encoded[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		} else {
			encoded[j] = c
			j++
		}
	}
	return encoded
}
