package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"io"
	"time"
)

type AppendObjectInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// The name of the object.
	Key *string `location:"uri" locationName:"Key" type:"string" required:"true"`

	// The starting position of the AppendObject operation.
	// When the AppendObject operation is successful, the x-kss-next-append-position header describes the starting position of the next operation.
	Position *int64 `location:"querystring" locationName:"position" type:"integer" required:"true"`

	// The readable body payload to send to KS3.
	Body io.Reader `type:"blob"`

	// The canned ACL to apply to the object.
	ACL *string `location:"header" locationName:"x-amz-acl" type:"string"`

	// Specifies caching behavior along the request/reply chain.
	CacheControl *string `location:"header" locationName:"Cache-Control" type:"string"`

	// Specifies presentational information for the object.
	ContentDisposition *string `location:"header" locationName:"Content-Disposition" type:"string"`

	// Specifies what content encodings have been applied to the object and thus
	// what decoding mechanisms must be applied to obtain the media-type referenced
	// by the Content-Type header field.
	ContentEncoding *string `location:"header" locationName:"Content-Encoding" type:"string"`

	// Size of the body in bytes. This parameter is useful when the size of the
	// body cannot be determined automatically.
	ContentLength *int64 `location:"header" locationName:"Content-Length" type:"integer"`

	// Calculate MD5 value for message content (excluding header)
	ContentMD5 *string `location:"header" locationName:"Content-MD5" type:"string"`

	// A standard MIME type describing the format of the object data.
	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// When using Expect: 100-continue, the client will only send the request body after receiving confirmation from the server.
	// If the information in the request header is rejected, the client will not send the request body.
	Expect *string `location:"header" locationName:"Expect" type:"string"`

	// The date and time at which the object is no longer cacheable.
	Expires *time.Time `location:"header" locationName:"Expires" type:"timestamp" timestampFormat:"rfc822"`

	// A map of metadata to store with the object in S3.
	Metadata map[string]*string `location:"headers" locationName:"x-amz-meta-" type:"map"`

	// The type of storage to use for the object. Defaults to 'STANDARD'.
	StorageClass *string `location:"header" locationName:"x-amz-storage-class" type:"string"`

	// Set the maximum allowed size for a single addition of content
	ContentMaxLength *int64 `location:"header" locationName:"x-amz-content-maxlength" type:"integer"`

	// Specifies the object tag of the object. Multiple tags can be set at the same time, such as: TagA=A&TagB=B.
	// Note: Key and Value need to be URL-encoded first. If an item does not have "=", the Value is considered to be an empty string.
	Tagging *string `location:"header" locationName:"x-amz-tagging" type:"string"`

	// Allows grantee to read the object data and its metadata.
	GrantRead *string `location:"header" locationName:"x-amz-grant-read" type:"string"`

	// Gives the grantee READ, READ_ACP, and WRITE_ACP permissions on the object.
	GrantFullControl *string `location:"header" locationName:"x-amz-grant-full-control" type:"string"`

	// The Server-side encryption algorithm used when storing this object in KS3, eg: AES256.
	ServerSideEncryption *string `location:"header" locationName:"x-amz-server-side-encryption" type:"string"`

	// Specifies the algorithm to use to when encrypting the object, eg: AES256.
	SSECustomerAlgorithm *string `location:"header" locationName:"x-amz-server-side-encryption-customer-algorithm" type:"string"`

	// Specifies the customer-provided encryption key for KS3 to use in encrypting data.
	SSECustomerKey *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key" type:"string"`

	// Specifies the 128-bit MD5 digest of the encryption key according to RFC 1321.
	SSECustomerKeyMD5 *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key-MD5" type:"string"`

	// Progress callback function
	ProgressFn aws.ProgressFunc `location:"function"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataAppendObjectInput `json:"-" xml:"-"`
}

type metadataAppendObjectInput struct {
	SDKShapeTraits bool `type:"structure" payload:"Body"`
}

type AppendObjectOutput struct {
	// Entity tag for the uploaded object.
	ETag *string `location:"header" locationName:"ETag" type:"string"`

	// The position that should be provided for the next request, which is the size of the current object.
	NextAppendPosition *int64 `location:"header" locationName:"x-amz-next-append-position" type:"integer"`

	// The type of Object.
	ObjectType *string `location:"header" locationName:"x-amz-object-type" type:"string"`

	// The Server-side encryption algorithm used when storing this object in KS3, eg: AES256.
	ServerSideEncryption *string `location:"header" locationName:"x-amz-server-side-encryption" type:"string"`

	// If server-side encryption with a customer-provided encryption key was requested,
	// the response will include this header confirming the encryption algorithm used.
	SSECustomerAlgorithm *string `location:"header" locationName:"x-amz-server-side-encryption-customer-algorithm" type:"string"`

	// If server-side encryption with a customer-provided encryption key was requested,
	// the response will include this header to provide round trip message integrity
	// verification of the customer-provided encryption key.
	SSECustomerKeyMD5 *string `location:"header" locationName:"x-amz-server-side-encryption-customer-key-MD5" type:"string"`

	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataAppendObjectOutput `json:"-" xml:"-"`
}

type metadataAppendObjectOutput struct {
	SDKShapeTraits bool `type:"structure"`
}

// AppendObjectRequest generates a request for the AppendObject operation.
func (c *S3) AppendObjectRequest(input *AppendObjectInput) (req *aws.Request, output *AppendObjectOutput) {
	op := &aws.Operation{
		Name:       "AppendObject",
		HTTPMethod: "POST",
		HTTPPath:   "/{Bucket}/{Key+}?append",
	}

	if input == nil {
		input = &AppendObjectInput{}
	}

	req = c.newRequest(op, input, output)

	if input.ProgressFn != nil {
		req.ProgressFn = input.ProgressFn
	}

	output = &AppendObjectOutput{}
	req.Data = output
	return
}

// AppendObject is used to append data to an Appendable object.
func (c *S3) AppendObject(input *AppendObjectInput) (*AppendObjectOutput, error) {
	req, out := c.AppendObjectRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) AppendObjectWithContext(ctx aws.Context, input *AppendObjectInput) (*AppendObjectOutput, error) {
	req, out := c.AppendObjectRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
