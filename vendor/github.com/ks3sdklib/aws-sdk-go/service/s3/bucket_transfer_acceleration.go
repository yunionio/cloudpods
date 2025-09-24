package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
)

type PutBucketTransferAccelerationInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Bucket transfer acceleration configuration container.
	TransferAccelerationConfiguration *TransferAccelerationConfiguration `locationName:"TransferAccelerationConfiguration" type:"structure" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketTransferAccelerationInput `json:"-" xml:"-"`
}

type metadataPutBucketTransferAccelerationInput struct {
	SDKShapeTraits bool `type:"structure" payload:"TransferAccelerationConfiguration"`

	AutoFillMD5 bool
}

type TransferAccelerationConfiguration struct {
	// Whether the target bucket has enabled transfer acceleration.
	Enabled *bool `locationName:"Enabled" type:"boolean" required:"true"`
}

type PutBucketTransferAccelerationOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutBucketTransferAccelerationRequest generates a request for the PutBucketTransferAcceleration operation.
func (c *S3) PutBucketTransferAccelerationRequest(input *PutBucketTransferAccelerationInput) (req *aws.Request, output *PutBucketTransferAccelerationOutput) {
	op := &aws.Operation{
		Name:       "PutBucketTransferAcceleration",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?transferAcceleration",
	}

	if input == nil {
		input = &PutBucketTransferAccelerationInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketTransferAccelerationOutput{}
	req.Data = output
	return
}

// PutBucketTransferAcceleration sets bucket transfer acceleration configuration.
func (c *S3) PutBucketTransferAcceleration(input *PutBucketTransferAccelerationInput) (*PutBucketTransferAccelerationOutput, error) {
	req, out := c.PutBucketTransferAccelerationRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketTransferAccelerationWithContext(ctx aws.Context, input *PutBucketTransferAccelerationInput) (*PutBucketTransferAccelerationOutput, error) {
	req, out := c.PutBucketTransferAccelerationRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketTransferAccelerationInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketTransferAccelerationOutput struct {
	// Bucket transfer acceleration configuration container.
	TransferAccelerationConfiguration *TransferAccelerationConfiguration `locationName:"TransferAccelerationConfiguration" type:"structure"`

	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketTransferAccelerationOutput `json:"-" xml:"-"`
}

type metadataGetBucketTransferAccelerationOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"TransferAccelerationConfiguration"`
}

// GetBucketTransferAccelerationRequest generates a request for the GetBucketTransferAcceleration operation.
func (c *S3) GetBucketTransferAccelerationRequest(input *GetBucketTransferAccelerationInput) (req *aws.Request, output *GetBucketTransferAccelerationOutput) {
	op := &aws.Operation{
		Name:       "GetBucketTransferAcceleration",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?transferAcceleration",
	}

	if input == nil {
		input = &GetBucketTransferAccelerationInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketTransferAccelerationOutput{}
	req.Data = output
	return
}

// GetBucketTransferAcceleration gets bucket transfer acceleration configuration.
func (c *S3) GetBucketTransferAcceleration(input *GetBucketTransferAccelerationInput) (*GetBucketTransferAccelerationOutput, error) {
	req, out := c.GetBucketTransferAccelerationRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketTransferAccelerationWithContext(ctx aws.Context, input *GetBucketTransferAccelerationInput) (*GetBucketTransferAccelerationOutput, error) {
	req, out := c.GetBucketTransferAccelerationRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
