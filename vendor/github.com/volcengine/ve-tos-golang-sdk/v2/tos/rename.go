package tos

import (
	"bytes"
	"context"
	"net/http"
)

type PutBucketRenameInput struct {
	Bucket       string `json:"-"`
	RenameEnable bool   `json:"RenameEnable"`
}

type PutBucketRenameOutput struct {
	RequestInfo
}

type GetBucketRenameInput struct {
	Bucket string
}

type GetBucketRenameOutput struct {
	RequestInfo
	RenameEnable bool
}

type DeleteBucketRenameInput struct {
	Bucket string
}

type DeleteBucketRenameOutput struct {
	RequestInfo
}

func (cli *ClientV2) PutBucketRename(ctx context.Context, input *PutBucketRenameInput) (*PutBucketRenameOutput, error) {

	if input == nil {
		return nil, InputIsNilClientError
	}

	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	data, contentMD5, err := marshalInput("PutBucketRename", input)
	if err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("rename", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketRenameOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketRename(ctx context.Context, input *GetBucketRenameInput) (*GetBucketRenameOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	req := cli.newBuilder(input.Bucket, "").
		WithQuery("rename", "").
		WithRetry(nil, StatusCodeClassifier{})

	res, err := req.Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketRenameOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketRename(ctx context.Context, input *DeleteBucketRenameInput) (*DeleteBucketRenameOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("rename", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketRenameOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) RenameObject(ctx context.Context, input *RenameObjectInput) (*RenameObjectOutput, error) {

	if input == nil {
		return nil, InputIsNilClientError
	}

	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, input.Key).
		WithQuery("rename", "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := RenameObjectOutput{RequestInfo: res.RequestInfo()}

	return &output, nil
}
