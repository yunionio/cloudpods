package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"time"
)

type PutVpcAccessBlockInput struct {
	// Vpc access block configuration container.
	VpcAccessBlockConfiguration *VpcAccessBlockConfiguration `locationName:"VpcAccessBlockConfiguration" type:"structure" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutVpcAccessBlockInput `json:"-" xml:"-"`
}

type metadataPutVpcAccessBlockInput struct {
	SDKShapeTraits bool `type:"structure" payload:"VpcAccessBlockConfiguration"`

	AutoFillMD5 bool
}

type VpcAccessBlockConfiguration struct {
	// Set up VPC to access the KS3 public Region rules container.
	Rules []*VpcAccessBlockRule `locationName:"Rule" type:"list" flattened:"true" required:"true"`
}

type VpcAccessBlockRule struct {
	// The unique identifier of a Rule. The ID cannot be repeated in a rule.
	RuleID *string `locationName:"RuleID" type:"string" required:"true"`

	// Region to which the VPC belongs.
	Region *string `locationName:"Region" type:"string" required:"true"`

	// Set the VPC ID of the container.
	VPC *VPC `locationName:"VPC" type:"structure"`

	// Set the bucket's container.
	BucketAllowAccess *BucketAllowAccess `locationName:"BucketAllowAccess" type:"structure"`

	// Whether to enable this rule.
	Status *string `locationName:"Status" type:"string" required:"true"`

	// Creation time.
	CreationDate *time.Time `locationName:"CreationDate" type:"timestamp" timestampFormat:"iso8601"`
}

type VPC struct {
	// List of VPC IDs that are not allowed to access resources in this Region.
	IDs []string `locationName:"ID" type:"list" flattened:"true"`
}

type BucketAllowAccess struct {
	// List of Bucket names that are allowed to be accessed.
	Names []string `locationName:"Name" type:"list" flattened:"true"`
}

type PutVpcAccessBlockOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutVpcAccessBlockRequest generates a request for the PutVpcAccessBlock operation.
func (c *S3) PutVpcAccessBlockRequest(input *PutVpcAccessBlockInput) (req *aws.Request, output *PutVpcAccessBlockOutput) {
	op := &aws.Operation{
		Name:       "PutVpcAccessBlock",
		HTTPMethod: "PUT",
		HTTPPath:   "/?VpcAccessBlock",
	}

	if input == nil {
		input = &PutVpcAccessBlockInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutVpcAccessBlockOutput{}
	req.Data = output
	return
}

// PutVpcAccessBlock sets vpc access block configuration.
func (c *S3) PutVpcAccessBlock(input *PutVpcAccessBlockInput) (*PutVpcAccessBlockOutput, error) {
	req, out := c.PutVpcAccessBlockRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutVpcAccessBlockWithContext(ctx aws.Context, input *PutVpcAccessBlockInput) (*PutVpcAccessBlockOutput, error) {
	req, out := c.PutVpcAccessBlockRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetVpcAccessBlockInput struct {
	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetVpcAccessBlockOutput struct {
	// Vpc access block configuration container.
	VpcAccessBlockConfiguration *VpcAccessBlockConfiguration `locationName:"VpcAccessBlockConfiguration" type:"structure"`

	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetVpcAccessBlockOutput `json:"-" xml:"-"`
}

type metadataGetVpcAccessBlockOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"VpcAccessBlockConfiguration"`
}

// GetVpcAccessBlockRequest generates a request for the GetVpcAccessBlock operation.
func (c *S3) GetVpcAccessBlockRequest(input *GetVpcAccessBlockInput) (req *aws.Request, output *GetVpcAccessBlockOutput) {
	op := &aws.Operation{
		Name:       "GetVpcAccessBlock",
		HTTPMethod: "GET",
		HTTPPath:   "/?VpcAccessBlock",
	}

	if input == nil {
		input = &GetVpcAccessBlockInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetVpcAccessBlockOutput{}
	req.Data = output
	return
}

// GetVpcAccessBlock gets vpc access block configuration.
func (c *S3) GetVpcAccessBlock(input *GetVpcAccessBlockInput) (*GetVpcAccessBlockOutput, error) {
	req, out := c.GetVpcAccessBlockRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetVpcAccessBlockWithContext(ctx aws.Context, input *GetVpcAccessBlockInput) (*GetVpcAccessBlockOutput, error) {
	req, out := c.GetVpcAccessBlockRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteVpcAccessBlockInput struct {
	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type DeleteVpcAccessBlockOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// DeleteVpcAccessBlockRequest generates a request for the DeleteVpcAccessBlock operation.
func (c *S3) DeleteVpcAccessBlockRequest(input *DeleteVpcAccessBlockInput) (req *aws.Request, output *DeleteVpcAccessBlockOutput) {
	op := &aws.Operation{
		Name:       "DeleteVpcAccessBlock",
		HTTPMethod: "DELETE",
		HTTPPath:   "/?VpcAccessBlock",
	}

	if input == nil {
		input = &DeleteVpcAccessBlockInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteVpcAccessBlockOutput{}
	req.Data = output
	return
}

// DeleteVpcAccessBlock deletes vpc access block configuration.
func (c *S3) DeleteVpcAccessBlock(input *DeleteVpcAccessBlockInput) (*DeleteVpcAccessBlockOutput, error) {
	req, out := c.DeleteVpcAccessBlockRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteVpcAccessBlockWithContext(ctx aws.Context, input *DeleteVpcAccessBlockInput) (*DeleteVpcAccessBlockOutput, error) {
	req, out := c.DeleteVpcAccessBlockRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
