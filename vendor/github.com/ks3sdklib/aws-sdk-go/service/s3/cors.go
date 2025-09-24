package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
)

type PutBucketCORSInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	CORSConfiguration *CORSConfiguration `locationName:"CORSConfiguration" type:"structure" xmlURI:"http://s3.amazonaws.com/doc/2006-03-01/" `

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketCORSInput `json:"-" xml:"-"`
}

type metadataPutBucketCORSInput struct {
	SDKShapeTraits bool `type:"structure" payload:"CORSConfiguration"`

	AutoFillMD5 bool
}

type CORSConfiguration struct {
	Rules []*CORSRule `locationName:"CORSRule" type:"list" flattened:"true"`

	// Whether non-cross-domain requests return the Vary: Origin header.
	NonCrossOriginResponseVary *bool `locationName:"NonCrossOriginResponseVary" type:"boolean" xml:"NonCrossOriginResponseVary"`
}

type CORSRule struct {
	AllowedHeaders []string `locationName:"AllowedHeader" type:"list" flattened:"true"`
	AllowedMethods []string `locationName:"AllowedMethod" type:"list" flattened:"true"`
	AllowedOrigins []string `locationName:"AllowedOrigin" type:"list" flattened:"true"`
	ExposeHeaders  []string `locationName:"ExposeHeader" type:"list" flattened:"true"`
	MaxAgeSeconds  *int64   `locationName:"MaxAgeSeconds"`
}

type PutBucketCORSOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutBucketCORSRequest generates a request for the PutBucketCORS operation.
func (c *S3) PutBucketCORSRequest(input *PutBucketCORSInput) (req *aws.Request, output *PutBucketCORSOutput) {
	op := &aws.Operation{
		Name:       "PutBucketCors",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?cors",
	}

	if input == nil {
		input = &PutBucketCORSInput{}
	}

	//目前默认为true
	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketCORSOutput{}
	req.Data = output
	return
}

// PutBucketCORS Sets the cors configuration for a bucket.
func (c *S3) PutBucketCORS(input *PutBucketCORSInput) (*PutBucketCORSOutput, error) {
	req, out := c.PutBucketCORSRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketCORSWithContext(ctx aws.Context, input *PutBucketCORSInput) (*PutBucketCORSOutput, error) {
	req, out := c.PutBucketCORSRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketCORSInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketCORSOutput struct {
	CORSConfiguration *CORSConfiguration `locationName:"CORSConfiguration" type:"structure"`

	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketCORSOutput `json:"-" xml:"-"`
}

type metadataGetBucketCORSOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"CORSConfiguration"`
}

// GetBucketCORSRequest generates a request for the GetBucketCORS operation.
func (c *S3) GetBucketCORSRequest(input *GetBucketCORSInput) (req *aws.Request, output *GetBucketCORSOutput) {
	op := &aws.Operation{
		Name:       "GetBucketCors",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?cors",
	}

	if input == nil {
		input = &GetBucketCORSInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketCORSOutput{}
	req.Data = output
	return
}

// GetBucketCORS Returns the cors configuration for the bucket.
func (c *S3) GetBucketCORS(input *GetBucketCORSInput) (*GetBucketCORSOutput, error) {
	req, out := c.GetBucketCORSRequest(input)
	err := req.Send()
	if req.Data != nil {
		out = req.Data.(*GetBucketCORSOutput)
	}
	return out, err
}

func (c *S3) GetBucketCORSWithContext(ctx aws.Context, input *GetBucketCORSInput) (*GetBucketCORSOutput, error) {
	req, out := c.GetBucketCORSRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	if req.Data != nil {
		out = req.Data.(*GetBucketCORSOutput)
	}
	return out, err
}

type DeleteBucketCORSInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type DeleteBucketCORSOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// DeleteBucketCORSRequest generates a request for the DeleteBucketCORS operation.
func (c *S3) DeleteBucketCORSRequest(input *DeleteBucketCORSInput) (req *aws.Request, output *DeleteBucketCORSOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketCors",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?cors",
	}

	if input == nil {
		input = &DeleteBucketCORSInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketCORSOutput{}
	req.Data = output
	return
}

// DeleteBucketCORS Deletes the cors configuration information set for the bucket.
func (c *S3) DeleteBucketCORS(input *DeleteBucketCORSInput) (*DeleteBucketCORSOutput, error) {
	req, out := c.DeleteBucketCORSRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketCORSWithContext(ctx aws.Context, input *DeleteBucketCORSInput) (*DeleteBucketCORSOutput, error) {
	req, out := c.DeleteBucketCORSRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
