package tos

import (
	"bytes"
	"context"
	"net/http"
)

func (cli *ClientV2) PutBucketMirrorBack(ctx context.Context, input *PutBucketMirrorBackInput) (*PutBucketMirrorBackOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	data, contentMD5, err := marshalInput("PutBucketMirrorBackInput", putBucketMirrorBackInput{
		Rules: input.Rules,
	})
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("mirror", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketMirrorBackOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketMirrorBack(ctx context.Context, input *GetBucketMirrorBackInput) (*GetBucketMirrorBackOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("mirror", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketMirrorBackOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketMirrorBack(ctx context.Context, input *DeleteBucketMirrorBackInput) (*DeleteBucketMirrorBackOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("mirror", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketMirrorBackOutput{RequestInfo: res.RequestInfo()}
	return &output, nil

}
