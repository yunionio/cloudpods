package tos

import (
	"bytes"
	"context"
	"net/http"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
)

// Bucket create a Bucket handle
//
// Deprecated: request with bucket handle is deprecated, use ClientV2 instead
func (cli *Client) Bucket(bucket string) (*Bucket, error) {
	if err := isValidBucketName(bucket, false); err != nil {
		return nil, err
	}
	return &Bucket{name: bucket, client: cli, baseClient: newBaseClient(cli)}, nil
}

// CreateBucket create a bucket
//
// Deprecated: use CreateBucket of ClientV2 instead
func (cli *Client) CreateBucket(ctx context.Context, input *CreateBucketInput) (*CreateBucketOutput, error) {
	if err := isValidBucketName(input.Bucket, false); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, "").
		WithHeader(HeaderACL, input.ACL).
		WithHeader(HeaderGrantFullControl, input.GrantFullControl).
		WithHeader(HeaderGrantRead, input.GrantRead).
		WithHeader(HeaderGrantReadAcp, input.GrantReadAcp).
		WithHeader(HeaderGrantWrite, input.GrantWrite).
		WithHeader(HeaderGrantWriteAcp, input.GrantWriteAcp).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	return &CreateBucketOutput{
		RequestInfo: res.RequestInfo(),
		Location:    res.Header.Get(HeaderLocation),
	}, nil
}

// CreateBucketV2 create a bucket
func (cli *ClientV2) CreateBucketV2(ctx context.Context, input *CreateBucketV2Input) (*CreateBucketV2Output, error) {
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	// TODO: ACL和Grant不能同时设置，可以在sdk校验
	if err := isValidACL(input.ACL); len(input.ACL) != 0 && err != nil {
		return nil, err
	}

	if err := isValidStorageClass(input.StorageClass); len(input.StorageClass) != 0 && err != nil {
		return nil, err
	}

	if err := isValidAzRedundancy(input.AzRedundancy); len(input.AzRedundancy) != 0 && err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	return &CreateBucketV2Output{
		CreateBucketOutput: CreateBucketOutput{
			RequestInfo: res.RequestInfo(),
			Location:    res.Header.Get(HeaderLocation)}}, nil
}

// HeadBucket get some info of a bucket
//
// Deprecated: use HeadBucket of ClientV2 instead
func (cli *Client) HeadBucket(ctx context.Context, bucket string) (*HeadBucketOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(bucket, "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodHead, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	return &HeadBucketOutput{
		RequestInfo:  res.RequestInfo(),
		Region:       res.Header.Get(HeaderBucketRegion),
		StorageClass: enum.StorageClassType(res.Header.Get(HeaderStorageClass)),
		AzRedundancy: enum.AzRedundancyType(res.Header.Get(HeaderAzRedundancy)),
	}, nil
}

// HeadBucket get some info of a bucket
func (cli *ClientV2) HeadBucket(ctx context.Context, input *HeadBucketInput) (*HeadBucketOutput, error) {
	return cli.Client.HeadBucket(ctx, input.Bucket)
}

// DeleteBucket delete a bucket
//
// Deprecated: use DeleteBucket of ClientV2 instead
func (cli *Client) DeleteBucket(ctx context.Context, bucket string) (*DeleteBucketOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(bucket, "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	return &DeleteBucketOutput{RequestInfo: res.RequestInfo()}, nil
}

// DeleteBucket delete a bucket.Deleting a non-empty bucket is not allowed.
// A bucket is empty only if there is no exist object and uncanceled segmented tasks.
func (cli *ClientV2) DeleteBucket(ctx context.Context, input *DeleteBucketInput) (*DeleteBucketOutput, error) {
	return cli.Client.DeleteBucket(ctx, input.Bucket)
}

// ListBuckets list the buckets that the AK can access
//
// Deprecated: use ListBuckets of ClientV2 instead
func (cli *Client) ListBuckets(ctx context.Context, _ *ListBucketsInput) (*ListBucketsOutput, error) {
	res, err := cli.newBuilder("", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := ListBucketsOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

// ListBuckets list the buckets that the AK can access
func (cli *ClientV2) ListBuckets(ctx context.Context, _ *ListBucketsInput) (*ListBucketsOutput, error) {
	res, err := cli.newBuilder("", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := ListBucketsOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) PutBucketStorageClass(ctx context.Context, input *PutBucketStorageClassInput) (*PutBucketStorageClassOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	if err := isValidStorageClass(input.StorageClass); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("storageClass", "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketStorageClassOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketLocation(ctx context.Context, input *GetBucketLocationInput) (*GetBucketLocationOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("location", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketLocationOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) PutBucketVersioning(ctx context.Context, input *PutBucketVersioningInput) (*PutBucketVersioningOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	data, contentMD5, err := marshalInput("PutBucketVersioning", putBucketVersioningInput{
		Status: input.Status,
	})
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("versioning", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))

	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketVersioningOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketVersioning(ctx context.Context, input *GetBucketVersioningInput) (*GetBucketVersioningOutputV2, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("versioning", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketVersioningOutputV2{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}
