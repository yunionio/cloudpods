package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
)

type PutBucketTaggingInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Bucket tagging configuration container.
	Tagging *Tagging `locationName:"Tagging" type:"structure" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketTaggingInput `json:"-" xml:"-"`
}

type metadataPutBucketTaggingInput struct {
	SDKShapeTraits bool `type:"structure" payload:"Tagging"`

	AutoFillMD5 bool
}

type PutBucketTaggingOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers"  type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutBucketTaggingRequest generates a request for the PutBucketTagging operation.
func (c *S3) PutBucketTaggingRequest(input *PutBucketTaggingInput) (req *aws.Request, output *PutBucketTaggingOutput) {
	op := &aws.Operation{
		Name:       "PutBucketTagging",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?tagging",
	}

	if input == nil {
		input = &PutBucketTaggingInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketTaggingOutput{}
	req.Data = output
	return
}

// PutBucketTagging sets bucket tagging configuration.
func (c *S3) PutBucketTagging(input *PutBucketTaggingInput) (*PutBucketTaggingOutput, error) {
	req, out := c.PutBucketTaggingRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketTaggingWithContext(ctx aws.Context, input *PutBucketTaggingInput) (*PutBucketTaggingOutput, error) {
	req, out := c.PutBucketTaggingRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketTaggingInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketTaggingOutput struct {
	// Bucket tagging configuration container.
	Tagging *Tagging `locationName:"Tagging" type:"structure"`

	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers"  type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketTaggingOutput `json:"-" xml:"-"`
}

type metadataGetBucketTaggingOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"Tagging"`
}

// GetBucketTaggingRequest generates a request for the GetBucketTagging operation.
func (c *S3) GetBucketTaggingRequest(input *GetBucketTaggingInput) (req *aws.Request, output *GetBucketTaggingOutput) {
	op := &aws.Operation{
		Name:       "GetBucketTagging",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?tagging",
	}

	if input == nil {
		input = &GetBucketTaggingInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketTaggingOutput{}
	req.Data = output
	return
}

// GetBucketTagging gets bucket tagging configuration.
func (c *S3) GetBucketTagging(input *GetBucketTaggingInput) (*GetBucketTaggingOutput, error) {
	req, out := c.GetBucketTaggingRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketTaggingWithContext(ctx aws.Context, input *GetBucketTaggingInput) (*GetBucketTaggingOutput, error) {
	req, out := c.GetBucketTaggingRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteBucketTaggingInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type DeleteBucketTaggingOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers"  type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// DeleteBucketTaggingRequest generates a request for the DeleteBucketTagging operation.
func (c *S3) DeleteBucketTaggingRequest(input *DeleteBucketTaggingInput) (req *aws.Request, output *DeleteBucketTaggingOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketTagging",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?tagging",
	}

	if input == nil {
		input = &DeleteBucketTaggingInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketTaggingOutput{}
	req.Data = output
	return
}

// DeleteBucketTagging deletes bucket tagging configuration.
func (c *S3) DeleteBucketTagging(input *DeleteBucketTaggingInput) (*DeleteBucketTaggingOutput, error) {
	req, out := c.DeleteBucketTaggingRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketTaggingWithContext(ctx aws.Context, input *DeleteBucketTaggingInput) (*DeleteBucketTaggingOutput, error) {
	req, out := c.DeleteBucketTaggingRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
