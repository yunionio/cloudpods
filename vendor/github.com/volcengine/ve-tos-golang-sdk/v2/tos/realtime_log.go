package tos

import (
	"bytes"
	"context"
	"net/http"
)

func (cli *ClientV2) PutBucketRealTimeLog(ctx context.Context, input *PutBucketRealTimeLogInput) (*PutBucketRealTimeLogOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	body := putBucketRealTimeLogInput{
		Configuration: input.Configuration,
	}
	data, contentMD5, err := marshalInput("PutBucketRealTimeLogInput", body)
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("realtimeLog", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketRealTimeLogOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketRealTimeLog(ctx context.Context, input *GetBucketRealTimeLogInput) (*GetBucketRealTimeLogOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("realtimeLog", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketRealTimeLogOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketRealTimeLog(ctx context.Context, input *DeleteBucketRealTimeLogInput) (*DeleteBucketRealTimeLogOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("realtimeLog", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketRealTimeLogOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}
