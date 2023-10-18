package tos

import (
	"net/http"
	"time"
)

type Option func(*requestBuilder)

// WithContentType set Content-Type header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithContentType(contentType string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentType, contentType)
	}
}

// WithContentLength set Content-Length header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.UploadPart
//
// If the length of the content is known, it is better to add this option when Put, Append or Upload.
func WithContentLength(length int64) Option {
	return func(rb *requestBuilder) {
		rb.WithContentLength(length)
	}
}

// WithCacheControl set Cache-Control header
//   used in Bucket.PutObject Bucket.AppendObject
//     Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithCacheControl(cacheControl string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderCacheControl, cacheControl)
	}
}

// WithContentDisposition set Content-Disposition header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithContentDisposition(contentDisposition string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentDisposition, contentDisposition)
	}
}

// WithContentEncoding set Content-Encoding header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithContentEncoding(contentEncoding string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentEncoding, contentEncoding)
	}
}

// WithContentLanguage set Content-Language header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithContentLanguage(contentLanguage string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentLanguage, contentLanguage)
	}
}

// WithContentMD5 set Content-MD5 header
func WithContentMD5(contentMD5 string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentMD5, contentMD5)
	}
}

// WithContentSHA256 set X-Tos-Content-Sha256 header
func WithContentSHA256(contentSHA256 string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderContentSha256, contentSHA256)
	}
}

// WithExpires set Expires header
//   used in Bucket.PutObject Bucket.AppendObject Bucket.CreateMultipartUpload Bucket.SetObjectMeta
func WithExpires(expires time.Time) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderExpires, expires.Format(http.TimeFormat))
	}
}

// WithServerSideEncryptionCustomer set server-side-encryption parameters
//  used in Bucket.PutObject Bucket.CreateMultipartUpload
func WithServerSideEncryptionCustomer(ssecAlgorithm, ssecKey, ssecKeyMD5 string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderSSECustomerAlgorithm, ssecAlgorithm)
		rb.Header.Set(HeaderSSECustomerKey, ssecKey)
		rb.Header.Set(HeaderSSECustomerKeyMD5, ssecKeyMD5)
	}
}

// WithIfModifiedSince set If-Modified-Since header
//   used in Bucket.GetObject Bucket.HeadObject
func WithIfModifiedSince(since time.Time) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderIfModifiedSince, since.Format(http.TimeFormat))
	}
}

// WithIfUnmodifiedSince set If-Unmodified-Since header
//   used in Bucket.GetObject Bucket.HeadObject
func WithIfUnmodifiedSince(since time.Time) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderIfUnmodifiedSince, since.Format(http.TimeFormat))
	}
}

// WithIfMatch set If-Match header
func WithIfMatch(ifMatch string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderIfMatch, ifMatch)
	}
}

// WithIfNoneMatch set If-None-Match header
func WithIfNoneMatch(ifNoneMatch string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderIfNoneMatch, ifNoneMatch)
	}
}

// WithCopySourceIfMatch set X-Tos-Copy-Source-If-Match header
//   used in Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom Bucket.UploadPartCopy
func WithCopySourceIfMatch(ifMatch string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderCopySourceIfMatch, ifMatch)
	}
}

// WithCopySourceIfNoneMatch set X-Tos-Copy-Source-If-None-Match
//   used in Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom Bucket.UploadPartCopy
func WithCopySourceIfNoneMatch(ifNoneMatch string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderCopySourceIfNoneMatch, ifNoneMatch)
	}
}

// WithCopySourceIfModifiedSince set X-Tos-Copy-Source-If-Modified-Since header
//   used in Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom Bucket.UploadPartCopy
func WithCopySourceIfModifiedSince(ifModifiedSince string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderCopySourceIfModifiedSince, ifModifiedSince)
	}
}

// WithCopySourceIfUnmodifiedSince set X-Tos-Copy-Source-If-Unmodified-Since header
//   used in Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom Bucket.UploadPartCopy
func WithCopySourceIfUnmodifiedSince(ifUnmodifiedSince string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderCopySourceIfUnmodifiedSince, ifUnmodifiedSince)
	}
}

// WithMeta set meta header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject Bucket.SetObjectMeta
func WithMeta(key, value string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderMetaPrefix+key, value)
	}
}

// WithRange set Range header
//   used in Bucket.GetObject Bucket.HeadObject
func WithRange(start, end int64) Option {
	return func(rb *requestBuilder) {
		rb.Range = &Range{Start: start, End: end}
		rb.Header.Set(HeaderRange, rb.Range.String())
	}
}

// WithVersionID set version parameter
//   used in Bucket.GetObject Bucket.HeadObject Bucket.DeleteObject
//     Bucket.GetObjectAcl Bucket.SetObjectMeta
//     Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom
//     Client.PreSignedURL
func WithVersionID(versionID string) Option {
	return func(rb *requestBuilder) {
		rb.Query.Add("versionId", versionID)
	}
}

// WithMetadataDirective set X-Tos-Metadata-Directive header
//  used in Bucket.CopyObject Bucket.CopyObjectTo Bucket.CopyObjectFrom
func WithMetadataDirective(directive string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Add(HeaderMetadataDirective, directive)
	}
}

// WithACL set X-Tos-Acl header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACL(acl string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderACL, acl)
	}
}

// WithACLGrantFullControl X-Tos-Grant-Full-Control header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACLGrantFullControl(grantFullControl string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderGrantFullControl, grantFullControl)
	}
}

// WithACLGrantRead set X-Tos-Grant-Read header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACLGrantRead(grantRead string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderGrantRead, grantRead)
	}
}

// WithACLGrantReadAcp set X-Tos-Grant-Read-Acp header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACLGrantReadAcp(grantReadAcp string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderGrantReadAcp, grantReadAcp)
	}
}

// WithACLGrantWrite set X-Tos-Grant-Write header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACLGrantWrite(grantWrite string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderGrantWrite, grantWrite)
	}
}

// WithACLGrantWriteAcp set X-Tos-Grant-Write-Acp header
//   used in Bucket.PutObject Bucket.CreateMultipartUpload Bucket.AppendObject
func WithACLGrantWriteAcp(grantWriteAcp string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderGrantWriteAcp, grantWriteAcp)
	}
}

// WithWebsiteRedirectLocation set X-Tos-Website-Redirect-Location header
func WithWebsiteRedirectLocation(redirectLocation string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(HeaderWebsiteRedirectLocation, redirectLocation)
	}
}

// WithPerRequestSigner set Signer for a request
//
// use this option when you need set request-level signature parameter(s).
// for example, use different ak and sk for each request.
//
// if 'signer' set to nil, the request will not be signed.
func WithPerRequestSigner(signer Signer) Option {
	return func(rb *requestBuilder) {
		rb.Signer = signer
	}
}

// WithHeader add request http header.
//
// NOTICE: use it carefully.
func WithHeader(key, value string) Option {
	return func(rb *requestBuilder) {
		rb.Header.Set(key, value)
	}
}

// WithQuery add request query parameter
//
// NOTICE: use it carefully.
func WithQuery(key, value string) Option {
	return func(rb *requestBuilder) {
		rb.Query.Set(key, value)
	}
}
