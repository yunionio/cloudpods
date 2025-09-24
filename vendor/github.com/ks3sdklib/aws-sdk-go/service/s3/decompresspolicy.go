package s3

import "github.com/ks3sdklib/aws-sdk-go/aws"

// PutBucketDecompressPolicyRequest generates a request for the PutBucketDecompressPolicy operation.
func (c *S3) PutBucketDecompressPolicyRequest(input *PutBucketDecompressPolicyInput) (req *aws.Request, output *PutBucketDecompressPolicyOutput) {
	op := &aws.Operation{
		Name:       "PutBucketDecompressPolicy",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?decompresspolicy",
	}

	if input == nil {
		input = &PutBucketDecompressPolicyInput{}
	}

	req = c.newRequest(op, input, output)
	req.ContentType = "application/json"
	output = &PutBucketDecompressPolicyOutput{}
	req.Data = output
	return
}

// PutBucketDecompressPolicy sets the decompression policy for the bucket.
func (c *S3) PutBucketDecompressPolicy(input *PutBucketDecompressPolicyInput) (*PutBucketDecompressPolicyOutput, error) {
	req, out := c.PutBucketDecompressPolicyRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketDecompressPolicyWithContext(ctx aws.Context, input *PutBucketDecompressPolicyInput) (*PutBucketDecompressPolicyOutput, error) {
	req, out := c.PutBucketDecompressPolicyRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type PutBucketDecompressPolicyInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	BucketDecompressPolicy *BucketDecompressPolicy `locationName:"BucketDecompressPolicy" type:"structure"`

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketDecompressPolicyInput `json:"-" xml:"-"`
}

type metadataPutBucketDecompressPolicyInput struct {
	SDKShapeTraits bool `type:"structure" payload:"BucketDecompressPolicy"`
}

type BucketDecompressPolicy struct {
	Rules []*DecompressPolicyRule `json:"rules,omitempty" type:"list" locationName:"rules" required:"true"`
}

type DecompressPolicyRule struct {
	// The name of the decompression strategy and the unique identifier of the decompression rule configured by the bucket. Value range: [1, 256].
	// Description: The length is 1-256 characters and can only be composed of uppercase or lowercase English letters, numbers, underscores (_), and hyphens (-).
	Id *string `json:"id,omitempty" type:"string" locationName:"id" required:"true"`

	// ZIP online decompression trigger event, currently supports the following operations:
	// "ObjectCreated:*": represents all operations for creating objects, including Put, Post, Copy objects, and merging segmentation tasks;
	// "ObjectCreated:Put": Use the Put method to upload a ZIP package;
	// "ObjectCreated:Post": Use the Post method to upload a ZIP package;
	// "ObjectCreated:Copy": Use the Copy method to copy a ZIP package;
	// "ObjectCreated:CompleteMultipartUpload": Use merge to upload ZIP packages in chunks.
	Events *string `json:"events,omitempty" type:"string" locationName:"events" required:"true"`

	// Match rule prefix (ZIP package that matches the prefix).
	// If no prefix is specified, all ZIP packages uploaded will be matched by default.
	Prefix *string `json:"prefix,omitempty" type:"string" locationName:"prefix"`

	// Match rule suffix.
	// The default is. zip, and currently only supports ZIP package format.
	Suffix []*string `json:"suffix,omitempty" type:"list" locationName:"suffix" required:"true"`

	// The processing method for files with the same name after decompression is not to overwrite them by default
	// The parameter values are as follows:
	// 0 (default value): Do not overwrite skip, keep existing objects in the bucket, skip objects with the same name, do not decompress;
	// 1: Overwrite, preserve the extracted object, and delete any existing objects with the same name in the bucket.
	Overwrite *int64 `json:"overwrite,omitempty" type:"integer" locationName:"overwrite" required:"true"`

	// The address for task callback, URL address.
	Callback *string `json:"callback,omitempty" type:"string" locationName:"callback"`

	// Task callback format, JSON format (required if callback address is set).
	CallbackFormat *string `json:"callback_format,omitempty" type:"string" locationName:"callback_format"`

	// Specify the prefix of the output file in the target bucket after decompression. If it is not empty, it must end with a '/'.
	// If left blank, it will be saved by default in the root path of the storage bucket.
	PathPrefix *string `json:"path_prefix,omitempty" type:"string" locationName:"path_prefix"`

	// Specify whether the compressed file path requires a compressed file name, with the following parameter values:
	// 0 (default): Keep compressed file name
	// 1: Extract directly to the target directory
	PathPrefixReplaced *int64 `json:"path_prefix_replaced,omitempty" type:"integer" locationName:"path_prefix_replaced"`

	// Type of file decompression strategy.
	// Fixed value: decompress.
	PolicyType *string `json:"policy_type,omitempty" type:"string" locationName:"policy_type" required:"true"`
}

type PutBucketDecompressPolicyOutput struct {
	Metadata map[string]*string `location:"headers" type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// GetBucketDecompressPolicyRequest generates a request for the GetBucketDecompressPolicy operation.
func (c *S3) GetBucketDecompressPolicyRequest(input *GetBucketDecompressPolicyInput) (req *aws.Request, output *GetBucketDecompressPolicyOutput) {
	op := &aws.Operation{
		Name:       "GetBucketDecompressPolicy",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?decompresspolicy",
	}

	if input == nil {
		input = &GetBucketDecompressPolicyInput{}
	}

	req = c.newRequest(op, input, output)
	req.ContentType = "application/json"
	output = &GetBucketDecompressPolicyOutput{
		BucketDecompressPolicy: &BucketDecompressPolicy{},
	}
	req.Data = output
	return
}

// GetBucketDecompressPolicy gets the decompression policy for the bucket.
func (c *S3) GetBucketDecompressPolicy(input *GetBucketDecompressPolicyInput) (*GetBucketDecompressPolicyOutput, error) {
	req, out := c.GetBucketDecompressPolicyRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketDecompressPolicyWithContext(ctx aws.Context, input *GetBucketDecompressPolicyInput) (*GetBucketDecompressPolicyOutput, error) {
	req, out := c.GetBucketDecompressPolicyRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketDecompressPolicyInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketDecompressPolicyOutput struct {
	BucketDecompressPolicy *BucketDecompressPolicy `locationName:"BucketDecompressPolicy" type:"structure"`

	Metadata map[string]*string `location:"headers" type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketDecompressPolicyOutput `json:"-" xml:"-"`
}

type metadataGetBucketDecompressPolicyOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"BucketDecompressPolicy"`
}

// DeleteBucketDecompressPolicyRequest generates a request for the DeleteBucketDecompressPolicy operation.
func (c *S3) DeleteBucketDecompressPolicyRequest(input *DeleteBucketDecompressPolicyInput) (req *aws.Request, output *DeleteBucketDecompressPolicyOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketDecompressPolicy",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?decompresspolicy",
	}

	if input == nil {
		input = &DeleteBucketDecompressPolicyInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketDecompressPolicyOutput{}
	req.Data = output
	return
}

// DeleteBucketDecompressPolicy deletes the decompression policy for the bucket.
func (c *S3) DeleteBucketDecompressPolicy(input *DeleteBucketDecompressPolicyInput) (*DeleteBucketDecompressPolicyOutput, error) {
	req, out := c.DeleteBucketDecompressPolicyRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketDecompressPolicyWithContext(ctx aws.Context, input *DeleteBucketDecompressPolicyInput) (*DeleteBucketDecompressPolicyOutput, error) {
	req, out := c.DeleteBucketDecompressPolicyRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteBucketDecompressPolicyInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}
type DeleteBucketDecompressPolicyOutput struct {
	Metadata map[string]*string `location:"headers" type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}
