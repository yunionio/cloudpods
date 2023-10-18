package tos

import (
	"bytes"
	"context"
	"net/http"
)

func (cli *ClientV2) PutBucketCustomDomain(ctx context.Context, input *PutBucketCustomDomainInput) (*PutBucketCustomDomainOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	body := putBucketCustomDomainInput{
		Rule: input.Rule,
	}
	data, contentMD5, err := marshalInput("PutBucketCustomDomainInput", body)
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("customdomain", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketCustomDomainOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) ListBucketCustomDomain(ctx context.Context, input *ListBucketCustomDomainInput) (*ListBucketCustomDomainOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("customdomain", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := ListBucketCustomDomainOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketCustomDomain(ctx context.Context, input *DeleteBucketCustomDomainInput) (*DeleteBucketCustomDomainOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("customdomain", input.Domain).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketCustomDomainOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}
