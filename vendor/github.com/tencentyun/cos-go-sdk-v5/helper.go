package cos

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// 单次上传文件最大为5GB
const singleUploadMaxLength = 5 * 1024 * 1024 * 1024
const singleUploadThreshold = 32 * 1024 * 1024

// 计算 md5 或 sha1 时的分块大小
const calDigestBlockSize = 1024 * 1024 * 10

func calMD5Digest(msg []byte) []byte {
	// TODO: 分块计算,减少内存消耗
	m := md5.New()
	m.Write(msg)
	return m.Sum(nil)
}

func calSHA1Digest(msg []byte) []byte {
	// TODO: 分块计算,减少内存消耗
	m := sha1.New()
	m.Write(msg)
	return m.Sum(nil)
}

func calCRC64(fd io.Reader) (uint64, error) {
	tb := crc64.MakeTable(crc64.ECMA)
	hash := crc64.New(tb)
	_, err := io.Copy(hash, fd)
	if err != nil {
		return 0, err
	}
	sum := hash.Sum64()
	return sum, nil
}

// cloneRequest returns a clone of the provided *http.Request. The clone is a
// shallow copy of the struct and its Header map.
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

// encodeURIComponent like same function in javascript
//
// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent
//
// http://www.ecma-international.org/ecma-262/6.0/#sec-uri-syntax-and-semantics
func encodeURIComponent(s string, excluded ...[]byte) string {
	var b bytes.Buffer
	written := 0

	for i, n := 0, len(s); i < n; i++ {
		c := s[i]

		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if 'a' <= c && c <= 'z' {

				continue

			}
			if 'A' <= c && c <= 'Z' {

				continue

			}
			if '0' <= c && c <= '9' {

				continue
			}
			if len(excluded) > 0 {
				conti := false
				for _, ch := range excluded[0] {
					if ch == c {
						conti = true
						break
					}
				}
				if conti {
					continue
				}
			}
		}

		b.WriteString(s[written:i])
		fmt.Fprintf(&b, "%%%02X", c)
		written = i + 1
	}

	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}

func decodeURIComponent(s string) (string, error) {
	decodeStr, err := url.QueryUnescape(s)
	if err != nil {
		return s, err
	}
	return decodeStr, err
}

func DecodeURIComponent(s string) (string, error) {
	return decodeURIComponent(s)
}

func EncodeURIComponent(s string) string {
	return encodeURIComponent(s)
}

func GetReaderLen(reader io.Reader) (length int64, err error) {
	switch v := reader.(type) {
	case *bytes.Buffer:
		length = int64(v.Len())
	case *bytes.Reader:
		length = int64(v.Len())
	case *strings.Reader:
		length = int64(v.Len())
	case *os.File:
		stat, ferr := v.Stat()
		if ferr != nil {
			err = fmt.Errorf("can't get reader length: %s", ferr.Error())
		} else {
			length = stat.Size()
		}
	case *io.LimitedReader:
		length = int64(v.N)
	case *LimitedReadCloser:
		length = int64(v.N)
	case FixedLengthReader:
		length = v.Size()
	default:
		err = fmt.Errorf("can't get reader content length, unkown reader type")
	}
	return
}

func IsLenReader(reader io.Reader) bool {
	switch reader.(type) {
	case *bytes.Buffer:
		return true
	case *bytes.Reader:
		return true
	case *strings.Reader:
		return true
	default:
		return false
	}
	return false
}

func CheckReaderLen(reader io.Reader) error {
	nlen, err := GetReaderLen(reader)
	if err != nil || nlen < singleUploadMaxLength {
		return nil
	}
	return errors.New("The single object size you upload can not be larger than 5GB")
}

func CopyOptionsToMulti(opt *ObjectCopyOptions) *InitiateMultipartUploadOptions {
	if opt == nil {
		return nil
	}
	optini := &InitiateMultipartUploadOptions{
		opt.ACLHeaderOptions,
		&ObjectPutHeaderOptions{},
	}
	if opt.ObjectCopyHeaderOptions == nil {
		return optini
	}
	optini.ObjectPutHeaderOptions = &ObjectPutHeaderOptions{
		CacheControl:             opt.ObjectCopyHeaderOptions.CacheControl,
		ContentDisposition:       opt.ObjectCopyHeaderOptions.ContentDisposition,
		ContentEncoding:          opt.ObjectCopyHeaderOptions.ContentEncoding,
		ContentType:              opt.ObjectCopyHeaderOptions.ContentType,
		ContentLanguage:          opt.ObjectCopyHeaderOptions.ContentLanguage,
		Expect:                   opt.ObjectCopyHeaderOptions.Expect,
		Expires:                  opt.ObjectCopyHeaderOptions.Expires,
		XCosMetaXXX:              opt.ObjectCopyHeaderOptions.XCosMetaXXX,
		XCosStorageClass:         opt.ObjectCopyHeaderOptions.XCosStorageClass,
		XCosServerSideEncryption: opt.ObjectCopyHeaderOptions.XCosServerSideEncryption,
		XCosSSECustomerAglo:      opt.ObjectCopyHeaderOptions.XCosSSECustomerAglo,
		XCosSSECustomerKey:       opt.ObjectCopyHeaderOptions.XCosSSECustomerKey,
		XCosSSECustomerKeyMD5:    opt.ObjectCopyHeaderOptions.XCosSSECustomerKeyMD5,
		XOptionHeader:            opt.ObjectCopyHeaderOptions.XOptionHeader,
	}
	return optini
}

// 浅拷贝ObjectPutOptions
func cloneObjectPutOptions(opt *ObjectPutOptions) *ObjectPutOptions {
	res := &ObjectPutOptions{
		&ACLHeaderOptions{},
		&ObjectPutHeaderOptions{},
	}
	if opt != nil {
		if opt.ACLHeaderOptions != nil {
			*res.ACLHeaderOptions = *opt.ACLHeaderOptions
		}
		if opt.ObjectPutHeaderOptions != nil {
			*res.ObjectPutHeaderOptions = *opt.ObjectPutHeaderOptions
		}
	}
	return res
}

// 浅拷贝ObjectUploadPartOptions
func cloneObjectUploadPartOptions(opt *ObjectUploadPartOptions) *ObjectUploadPartOptions {
	var res ObjectUploadPartOptions
	if opt != nil {
		res = *opt
	}
	return &res
}
