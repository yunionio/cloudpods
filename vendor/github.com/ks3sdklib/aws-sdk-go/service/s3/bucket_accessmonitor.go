package s3

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
)

type PutBucketAccessMonitorInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Bucket access monitor configuration.
	AccessMonitorConfiguration *AccessMonitorConfiguration `locationName:"AccessMonitorConfiguration" type:"structure" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketAccessMonitorInput `json:"-" xml:"-"`
}

type metadataPutBucketAccessMonitorInput struct {
	SDKShapeTraits bool `type:"structure" payload:"AccessMonitorConfiguration"`
}

type AccessMonitorConfiguration struct {
	// Specifies whether to enable access tracking for the bucket. The value range is as follows:
	// Enabled: After the bucket enables access tracking, the access tracking enable time is used as the default
	// last access time for all objects in the bucket.
	// Disabled: The access tracking status of the bucket can be changed to Disabled only when the bucket does not
	// have a lifecycle rule based on the last access time matching rule.
	Status *string `locationName:"Status" type:"string" required:"true"`
}

type PutBucketAccessMonitorOutput struct {
	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// PutBucketAccessMonitorRequest generates a request for the PutBucketAccessMonitor operation.
func (c *S3) PutBucketAccessMonitorRequest(input *PutBucketAccessMonitorInput) (req *aws.Request, output *PutBucketAccessMonitorOutput) {
	op := &aws.Operation{
		Name:       "PutBucketAccessMonitor",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?accessmonitor",
	}

	if input == nil {
		input = &PutBucketAccessMonitorInput{}
	}

	req = c.newRequest(op, input, output)
	output = &PutBucketAccessMonitorOutput{}
	req.Data = output
	return
}

// PutBucketAccessMonitor sets bucket access monitor configuration.
func (c *S3) PutBucketAccessMonitor(input *PutBucketAccessMonitorInput) (*PutBucketAccessMonitorOutput, error) {
	req, out := c.PutBucketAccessMonitorRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketAccessMonitorWithContext(ctx aws.Context, input *PutBucketAccessMonitorInput) (*PutBucketAccessMonitorOutput, error) {
	req, out := c.PutBucketAccessMonitorRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketAccessMonitorInput struct {
	// The name of the bucket.
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketAccessMonitorOutput struct {
	// Bucket access monitor configuration.
	AccessMonitorConfiguration *AccessMonitorConfiguration `locationName:"AccessMonitorConfiguration" type:"structure"`

	// The HTTP headers of the response.
	Metadata map[string]*string `location:"headers" type:"map"`

	// The HTTP status code of the response.
	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketAccessMonitorOutput `json:"-" xml:"-"`
}

type metadataGetBucketAccessMonitorOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"AccessMonitorConfiguration"`
}

// GetBucketAccessMonitorRequest generates a request for the GetBucketAccessMonitor operation.
func (c *S3) GetBucketAccessMonitorRequest(input *GetBucketAccessMonitorInput) (req *aws.Request, output *GetBucketAccessMonitorOutput) {
	op := &aws.Operation{
		Name:       "GetBucketAccessMonitor",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?accessmonitor",
	}

	if input == nil {
		input = &GetBucketAccessMonitorInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketAccessMonitorOutput{}
	req.Data = output
	return
}

// GetBucketAccessMonitor gets bucket access monitor configuration.
func (c *S3) GetBucketAccessMonitor(input *GetBucketAccessMonitorInput) (*GetBucketAccessMonitorOutput, error) {
	req, out := c.GetBucketAccessMonitorRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketAccessMonitorWithContext(ctx aws.Context, input *GetBucketAccessMonitorInput) (*GetBucketAccessMonitorOutput, error) {
	req, out := c.GetBucketAccessMonitorRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}
