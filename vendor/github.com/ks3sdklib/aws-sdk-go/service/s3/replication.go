package s3

import "github.com/ks3sdklib/aws-sdk-go/aws"

// PutBucketReplicationRequest generates a request for the PutBucketReplication operation.
func (c *S3) PutBucketReplicationRequest(input *PutBucketReplicationInput) (req *aws.Request, output *PutBucketReplicationOutput) {
	op := &aws.Operation{
		Name:       "PutBucketReplication",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?crr",
	}

	if input == nil {
		input = &PutBucketReplicationInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketReplicationOutput{}
	req.Data = output
	return
}

// PutBucketReplication creates a new replication configuration.
func (c *S3) PutBucketReplication(input *PutBucketReplicationInput) (*PutBucketReplicationOutput, error) {
	req, out := c.PutBucketReplicationRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketReplicationWithContext(ctx aws.Context, input *PutBucketReplicationInput) (*PutBucketReplicationOutput, error) {
	req, out := c.PutBucketReplicationRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type PutBucketReplicationInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	ReplicationConfiguration *ReplicationConfiguration `locationName:"Replication" type:"structure" required:"true" xmlURI:"http://s3.amazonaws.com/doc/2006-03-01/"`

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketReplicationInput `json:"-" xml:"-"`
}

type metadataPutBucketReplicationInput struct {
	SDKShapeTraits bool `type:"structure" payload:"ReplicationConfiguration"`

	AutoFillMD5 bool
}

type PutBucketReplicationOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

type ReplicationConfiguration struct {
	// Prefix matching, only objects that match prefix rules will be copied. Each copying rule
	// can add up to 10 prefix matching rules, and prefixes cannot overlap with each other.
	Prefix []*string `locationName:"prefix" type:"list" flattened:"true"`

	// Indicate whether to enable delete replication. If set to Enabled, it means enabled; if set to
	// Disabled or not, it means disabled. If set to delete replication, when the source bucket deletes
	// an object, the replica of that object in the target bucket will also be deleted.
	DeleteMarkerStatus *string `locationName:"DeleteMarkerStatus" type:"string" required:"true"`

	// Target bucket for copying rules.
	TargetBucket *string `locationName:"targetBucket" type:"string" required:"true"`

	// Specify whether to copy historical data. Whether to copy the data from the source bucket
	// to the target bucket before enabling data replication.
	// Enabled: Copy historical data to the target bucket (default value)
	// Disabled: Do not copy historical data, only copy new data after enabling the rule to the target bucket.
	HistoricalObjectReplication *string `locationName:"HistoricalObjectReplication" type:"string"`

	// Region of the target bucket.
	Region *string `locationName:"region" type:"string"`
}

// GetBucketReplicationRequest generates a request for the GetBucketReplication operation.
func (c *S3) GetBucketReplicationRequest(input *GetBucketReplicationInput) (req *aws.Request, output *GetBucketReplicationOutput) {
	op := &aws.Operation{
		Name:       "GetBucketReplication",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?crr",
	}

	if input == nil {
		input = &GetBucketReplicationInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketReplicationOutput{}
	req.Data = output
	return
}

// GetBucketReplication gets the replication configuration for the bucket.
func (c *S3) GetBucketReplication(input *GetBucketReplicationInput) (*GetBucketReplicationOutput, error) {
	req, out := c.GetBucketReplicationRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketReplicationWithContext(ctx aws.Context, input *GetBucketReplicationInput) (*GetBucketReplicationOutput, error) {
	req, out := c.GetBucketReplicationRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketReplicationInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketReplicationOutput struct {
	ReplicationConfiguration *ReplicationConfiguration `locationName:"Replication" type:"structure"`

	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketReplicationOutput `json:"-" xml:"-"`
}

type metadataGetBucketReplicationOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"ReplicationConfiguration"`
}

// DeleteBucketReplicationRequest generates a request for the DeleteBucketReplication operation.
func (c *S3) DeleteBucketReplicationRequest(input *DeleteBucketReplicationInput) (req *aws.Request, output *DeleteBucketReplicationOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketReplication",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?crr",
	}

	if input == nil {
		input = &DeleteBucketReplicationInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketReplicationOutput{}
	req.Data = output
	return
}

// DeleteBucketReplication deletes the replication configuration for the bucket.
func (c *S3) DeleteBucketReplication(input *DeleteBucketReplicationInput) (*DeleteBucketReplicationOutput, error) {
	req, out := c.DeleteBucketReplicationRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketReplicationWithContext(ctx aws.Context, input *DeleteBucketReplicationInput) (*DeleteBucketReplicationOutput, error) {
	req, out := c.DeleteBucketReplicationRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteBucketReplicationInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}
type DeleteBucketReplicationOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}
