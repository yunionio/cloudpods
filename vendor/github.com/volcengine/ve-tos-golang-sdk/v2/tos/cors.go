package tos

import (
	"bytes"
	"context"
	"net/http"
)

// GetBucketCORS get the bucket's CORS settings.
func (cli *ClientV2) GetBucketCORS(ctx context.Context, input *GetBucketCORSInput) (*GetBucketCORSOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("cors", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := GetBucketCORSOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

// PutBucketCORS upsert the bucket's CORS settings.
func (cli *ClientV2) PutBucketCORS(ctx context.Context, input *PutBucketCORSInput) (*PutBucketCORSOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	data, contentMD5, err := marshalInput("PutBucketCORSInput", input)
	if err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("cors", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := PutBucketCORSOutput{RequestInfo: res.RequestInfo()}
	return &output, nil

}

// DeleteBucketCORS delete the bucket's all CORS settings.
func (cli *ClientV2) DeleteBucketCORS(ctx context.Context, input *DeleteBucketCORSInput) (*DeleteBucketCORSOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("cors", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketCORSOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}
