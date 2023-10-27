package tos

import (
	"bytes"
	"context"
	"net/http"
)

func newBaseClient(c *Client) *baseClient {
	return &baseClient{Client: c}
}

type baseClient struct {
	*Client
}

func (cli *baseClient) PutObjectTagging(ctx context.Context, input *PutObjectTaggingInput, option ...Option) (*PutObjectTaggingOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	data, contentMD5, err := marshalInput("PutObjectTaggingInput", putObjectTaggingInput{
		TagSet: input.TagSet,
	})
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, input.Key, option...).
		WithQuery("tagging", "").
		WithParams(*input).
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutObjectTaggingOutput{RequestInfo: res.RequestInfo()}
	output.VersionID = res.Header.Get(HeaderVersionID)
	return &output, nil
}

func (cli *baseClient) GetObjectTagging(ctx context.Context, input *GetObjectTaggingInput, option ...Option) (*GetObjectTaggingOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, input.Key, option...).
		WithQuery("tagging", "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetObjectTaggingOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	output.VersionID = res.Header.Get(HeaderVersionID)
	return &output, nil
}

func (cli *baseClient) DeleteObjectTagging(ctx context.Context, input *DeleteObjectTaggingInput, option ...Option) (*DeleteObjectTaggingOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, input.Key, option...).
		WithQuery("tagging", "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteObjectTaggingOutput{RequestInfo: res.RequestInfo()}
	output.VersionID = res.Header.Get(HeaderVersionID)

	return &output, nil

}

func (cli *baseClient) RestoreObject(ctx context.Context, input *RestoreObjectInput, option ...Option) (*RestoreObjectOutput, error) {

	if input == nil {
		return nil, InputIsNilClientError
	}

	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	data, contentMD5, err := marshalInput("RestoreObjectInput", restoreObjectInput{
		Days:                 input.Days,
		RestoreJobParameters: input.RestoreJobParameters,
	})
	if err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, input.Key, option...).
		WithParams(*input).
		WithQuery("restore", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPost, bytes.NewReader(data), cli.roundTripper(http.StatusOK, http.StatusAccepted))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := RestoreObjectOutput{RequestInfo: res.RequestInfo()}
	return &output, nil

}
