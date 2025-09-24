package s3

import "github.com/ks3sdklib/aws-sdk-go/aws"

// PutBucketInventoryRequest generates a request for the PutBucketInventory operation.
func (c *S3) PutBucketInventoryRequest(input *PutBucketInventoryInput) (req *aws.Request, output *PutBucketInventoryOutput) {
	op := &aws.Operation{
		Name:       "PutBucketInventory",
		HTTPMethod: "PUT",
		HTTPPath:   "/{Bucket}?inventory",
	}

	if input == nil {
		input = &PutBucketInventoryInput{}
	}

	input.AutoFillMD5 = true
	req = c.newRequest(op, input, output)
	output = &PutBucketInventoryOutput{}
	req.Data = output
	return
}

// PutBucketInventory creates a new inventory configuration.
func (c *S3) PutBucketInventory(input *PutBucketInventoryInput) (*PutBucketInventoryOutput, error) {
	req, out := c.PutBucketInventoryRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) PutBucketInventoryWithContext(ctx aws.Context, input *PutBucketInventoryInput) (*PutBucketInventoryOutput, error) {
	req, out := c.PutBucketInventoryRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type PutBucketInventoryInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	Id *string `location:"querystring" locationName:"id" type:"string" required:"true"`

	InventoryConfiguration *InventoryConfiguration `locationName:"InventoryConfiguration" type:"structure" required:"true"`

	ContentType *string `location:"header" locationName:"Content-Type" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`

	metadataPutBucketInventoryInput `json:"-" xml:"-"`
}

type metadataPutBucketInventoryInput struct {
	SDKShapeTraits bool `type:"structure" payload:"InventoryConfiguration"`

	AutoFillMD5 bool
}

type PutBucketInventoryOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

type InventoryConfiguration struct {
	// The list name specified by the user is unique within a single bucket.
	Id *string `locationName:"Id" type:"string" required:"true"`

	// Is the inventory function enabled.
	IsEnabled *bool `locationName:"IsEnabled" type:"boolean" required:"true"`

	// Specify scanning prefix information.
	Filter *InventoryFilter `locationName:"Filter" type:"structure"`

	// Storage inventory results.
	Destination *Destination `locationName:"Destination" type:"structure" required:"true"`

	// Container for storing inventory export cycle information.
	Schedule *Schedule `locationName:"Schedule" type:"structure" required:"true"`

	// Set the configuration items included in the inventory results.
	OptionalFields *OptionalFields `locationName:"OptionalFields" type:"structure" required:"true"`
}

type InventoryFilter struct {
	// The storage path prefix of the inventory file.
	Prefix *string `locationName:"Prefix" type:"string" required:"true"`

	// The starting timestamp of the last modification time of the filtered file, in seconds.
	LastModifyBeginTimeStamp *string `locationName:"LastModifyBeginTimeStamp" type:"string"`

	// End timestamp of the last modification time of the filtered file, in seconds.
	LastModifyEndTimeStamp *string `locationName:"LastModifyEndTimeStamp" type:"string"`
}

type Destination struct {
	// Bucket information stored after exporting the inventory results.
	KS3BucketDestination *KS3BucketDestination `locationName:"KS3BucketDestination" type:"structure" required:"true"`
}

type KS3BucketDestination struct {
	// The file format of the inventory file is a CSV file compressed using GZIP after exporting the manifest file.
	Format *string `locationName:"Format" type:"string" required:"true"`

	// Bucket owner's account ID.
	AccountId *string `locationName:"AccountId" type:"string"`

	// Bucket for storing exported inventory files.
	Bucket *string `locationName:"Bucket" type:"string" required:"true"`

	// The storage path prefix of the inventory file.
	Prefix *string `locationName:"Prefix" type:"string"`
}

type Schedule struct {
	// Cycle of exporting inventory files.
	Frequency *string `locationName:"Frequency" type:"string" required:"true"`
}

type OptionalFields struct {
	// Configuration items included in the inventory results.
	// Valid values:
	// Size: The size of the object.
	// LastModifiedDate: The last modified time of an object.
	// ETag: The ETag value of an object, used to identify its contents.
	// StorageClass: The storage type of Object.
	// IsMultipartUploaded: Is it an object uploaded through shard upload method.
	// EncryptionStatus: Whether the object is encrypted. If the object is encrypted, the value of this field is True; otherwise, it is False.
	Field []*string `locationName:"Field" type:"list" flattened:"true"`
}

// GetBucketInventoryRequest generates a request for the GetBucketInventory operation.
func (c *S3) GetBucketInventoryRequest(input *GetBucketInventoryInput) (req *aws.Request, output *GetBucketInventoryOutput) {
	op := &aws.Operation{
		Name:       "GetBucketInventory",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?inventory",
	}

	if input == nil {
		input = &GetBucketInventoryInput{}
	}

	req = c.newRequest(op, input, output)
	output = &GetBucketInventoryOutput{}
	req.Data = output
	return
}

// GetBucketInventory gets the inventory configuration for the bucket.
func (c *S3) GetBucketInventory(input *GetBucketInventoryInput) (*GetBucketInventoryOutput, error) {
	req, out := c.GetBucketInventoryRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) GetBucketInventoryWithContext(ctx aws.Context, input *GetBucketInventoryInput) (*GetBucketInventoryOutput, error) {
	req, out := c.GetBucketInventoryRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type GetBucketInventoryInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	Id *string `location:"querystring" locationName:"id" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type GetBucketInventoryOutput struct {
	InventoryConfiguration *InventoryConfiguration `locationName:"Inventory" type:"structure"`

	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataGetBucketInventoryOutput `json:"-" xml:"-"`
}

type metadataGetBucketInventoryOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"InventoryConfiguration"`
}

// DeleteBucketInventoryRequest generates a request for the DeleteBucketInventory operation.
func (c *S3) DeleteBucketInventoryRequest(input *DeleteBucketInventoryInput) (req *aws.Request, output *DeleteBucketInventoryOutput) {
	op := &aws.Operation{
		Name:       "DeleteBucketInventory",
		HTTPMethod: "DELETE",
		HTTPPath:   "/{Bucket}?inventory",
	}

	if input == nil {
		input = &DeleteBucketInventoryInput{}
	}

	req = c.newRequest(op, input, output)
	output = &DeleteBucketInventoryOutput{}
	req.Data = output
	return
}

// DeleteBucketInventory deletes the inventory configuration for the bucket.
func (c *S3) DeleteBucketInventory(input *DeleteBucketInventoryInput) (*DeleteBucketInventoryOutput, error) {
	req, out := c.DeleteBucketInventoryRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) DeleteBucketInventoryWithContext(ctx aws.Context, input *DeleteBucketInventoryInput) (*DeleteBucketInventoryOutput, error) {
	req, out := c.DeleteBucketInventoryRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type DeleteBucketInventoryInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	Id *string `location:"querystring" locationName:"id" type:"string" required:"true"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}
type DeleteBucketInventoryOutput struct {
	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`
}

// ListBucketInventoryRequest generates a request for the ListBucketInventory operation.
func (c *S3) ListBucketInventoryRequest(input *ListBucketInventoryInput) (req *aws.Request, output *ListBucketInventoryOutput) {
	op := &aws.Operation{
		Name:       "ListBucketInventory",
		HTTPMethod: "GET",
		HTTPPath:   "/{Bucket}?inventory",
	}

	if input == nil {
		input = &ListBucketInventoryInput{}
	}

	req = c.newRequest(op, input, output)
	output = &ListBucketInventoryOutput{}
	req.Data = output
	return
}

// ListBucketInventory lists the inventory configurations for the bucket.
func (c *S3) ListBucketInventory(input *ListBucketInventoryInput) (*ListBucketInventoryOutput, error) {
	req, out := c.ListBucketInventoryRequest(input)
	err := req.Send()
	return out, err
}

func (c *S3) ListBucketInventoryWithContext(ctx aws.Context, input *ListBucketInventoryInput) (*ListBucketInventoryOutput, error) {
	req, out := c.ListBucketInventoryRequest(input)
	req.SetContext(ctx)
	err := req.Send()
	return out, err
}

type ListBucketInventoryInput struct {
	Bucket *string `location:"uri" locationName:"Bucket" type:"string" required:"true"`

	ContinuationToken *string `location:"querystring" locationName:"continuation-token" type:"string"`

	// Set extend request headers. If the existing fields do not support setting the request header you need, you can set it through this field.
	ExtendHeaders map[string]*string `location:"extendHeaders" type:"map"`

	// Set extend query params. If the existing fields do not support setting the query param you need, you can set it through this field.
	ExtendQueryParams map[string]*string `location:"extendQueryParams" type:"map"`
}

type ListInventoryConfigurationsResult struct {
	InventoryConfigurations []*InventoryConfiguration `locationName:"InventoryConfiguration" type:"list" flattened:"true"`

	IsTruncated *bool `locationName:"IsTruncated" type:"boolean"`

	NextContinuationToken *string `locationName:"NextContinuationToken" type:"string"`
}

type ListBucketInventoryOutput struct {
	InventoryConfigurationsResult *ListInventoryConfigurationsResult `locationName:"InventoryConfigurationsResult" type:"structure"`

	Metadata map[string]*string `location:"headers"  type:"map"`

	StatusCode *int64 `location:"statusCode" type:"integer"`

	metadataListBucketInventoryOutput `json:"-" xml:"-"`
}

type metadataListBucketInventoryOutput struct {
	SDKShapeTraits bool `type:"structure" payload:"InventoryConfigurationsResult"`
}
