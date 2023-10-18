package tos

import (
	"bytes"
	"context"
	"net/http"
)

func (cli *ClientV2) PutBucketReplication(ctx context.Context, input *PutBucketReplicationInput) (*PutBucketReplicationOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	data, contentMD5, err := marshalInput("PutBucketReplication", putBucketReplicationInput{Role: input.Role, Rules: input.Rules})
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("replication", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketReplicationOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketReplication(ctx context.Context, input *GetBucketReplicationInput) (*GetBucketReplicationOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	req := cli.newBuilder(input.Bucket, "").
		WithQuery("replication", "").
		WithQuery("progress", "").
		WithRetry(nil, StatusCodeClassifier{})

	if input.RuleID != "" {
		req.WithQuery("rule-id", input.RuleID)
	}
	res, err := req.Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketReplicationOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketReplication(ctx context.Context, input *DeleteBucketReplicationInput) (*DeleteBucketReplicationOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("replication", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketReplicationOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}
