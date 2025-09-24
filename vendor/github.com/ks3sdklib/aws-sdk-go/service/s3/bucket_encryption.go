package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
)

type PutBucketEncryptionInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Bucket encryption configuration container.
	ServerSideEncryptionConfiguration *ServerSideEncryptionConfiguration `locationName:"ServerSideEncryptionConfiguration" type:"structure" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketEncryptionInput `json:"-" xml:"-"`
}

type metadataPutBucketEncryptionInput struct {
	SDKShapeTraits bool `type:"structure" payload:"ServerSideEncryptionConfiguration"`

	AutoFillMD5 bool
}

type ServerSideEncryptionConfiguration struct {
	// Default encryption rule for bucket.
	Rule *BucketEncryptionRule `locationName:"Rule" type:"structure" required:"true"`
}

type BucketEncryptionRule struct {
	// The child element of the default encryption configuration for the bucket.
	ApplyServerSideEncryptionByDefault *ApplyServerSideEncryptionByDefault `locationName:"ApplyServerSideEncryptionByDefault" type:"structure" required:"true"`
}

type ApplyServerSideEncryptionByDefault struct {
	// The server-side encryption algorithm to be used by the bucket's default encryption configuration.
	SSEAlgorithm *string `locationName:"SSEAlgorithm" type:"string" required:"true"`
}

type PutBucketEncryptionOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutBucketEncryptionRequest generates a request for the PutBucketEncryption operation.
func (c *S3) PutBucketEncryptionRequest(input *PutBucketEncryptionInput) (req *aws.Request, output *PutBucketEncryptionOutput) {
	op := &aws.Operation{
		Name:       "PutBucketEncryption",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?encryption",
	}

	if input == nil {
		input = &PutBucketEncryptionInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketEncryptionOutput{}
	req.Data = output
	return
}

// PutBucketEncryption sets bucket encryption configuration.
func (c *S3) PutBucketEncryption(input *PutBucketEncryptionInput) (*PutBucketEncryptionOutput, error) {
	req, out := c.PutBucketEncryptionRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketEncryptionWithContext(ctx aws.Context, input *PutBucketEncryptionInput) (*PutBucketEncryptionOutput, error) {
	req, out := c.PutBucketEncryptionRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketEncryptionInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketEncryptionOutput struct {
	// Bucket encryption configuration container.
	ServerSideEncryptionConfiguration *ServerSideEncryptionConfiguration `locationName:"ServerSideEncryptionConfiguration" type:"structure"`

	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketEncryptionOutput `json:"-" xml:"-"`
}

type metadataGetBucketEncryptionOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"ServerSideEncryptionConfiguration"`
}

// GetBucketEncryptionRequest generates a request for the GetBucketEncryption operation.
func (c *S3) GetBucketEncryptionRequest(input *GetBucketEncryptionInput) (req *aws.Request, output *GetBucketEncryptionOutput) {
	op := &aws.Operation{
		Name:       "GetBucketEncryption",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?encryption",
	}

	if input == nil {
		input = &GetBucketEncryptionInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketEncryptionOutput{}
	req.Data = output
	return
}

// GetBucketEncryption gets bucket encryption configuration.
func (c *S3) GetBucketEncryption(input *GetBucketEncryptionInput) (*GetBucketEncryptionOutput, error) {
	req, out := c.GetBucketEncryptionRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketEncryptionWithContext(ctx aws.Context, input *GetBucketEncryptionInput) (*GetBucketEncryptionOutput, error) {
	req, out := c.GetBucketEncryptionRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteBucketEncryptionInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type DeleteBucketEncryptionOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// DeleteBucketEncryptionRequest generates a request for the DeleteBucketEncryption operation.
func (c *S3) DeleteBucketEncryptionRequest(input *DeleteBucketEncryptionInput) (req *aws.Request, output *DeleteBucketEncryptionOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketEncryption",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?encryption",
	}

	if input == nil {
		input = &DeleteBucketEncryptionInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketEncryptionOutput{}
	req.Data = output
	return
}

// DeleteBucketEncryption deletes bucket encryption configuration.
func (c *S3) DeleteBucketEncryption(input *DeleteBucketEncryptionInput) (*DeleteBucketEncryptionOutput, error) {
	req, out := c.DeleteBucketEncryptionRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketEncryptionWithContext(ctx aws.Context, input *DeleteBucketEncryptionInput) (*DeleteBucketEncryptionOutput, error) {
	req, out := c.DeleteBucketEncryptionRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
