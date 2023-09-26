package tos

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
)

type BucketPolicy struct {
	Policy string `json:"Policy,omitempty"`
}

type GetBucketPolicyOutput struct {
	RequestInfo `json:"-"`
	Policy      string `json:"Policy,omitempty"`
}

type PutBucketPolicyOutput struct {
	RequestInfo `json:"-"`
}

type DeleteBucketPolicyOutput struct {
	RequestInfo `json:"-"`
}

type GetBucketPolicyV2Input struct {
	Bucket string `json:"-"`
}

type GetBucketPolicyV2Output struct {
	RequestInfo `json:"-"`
	Policy      string `json:"Policy,omitempty"`
}

type putBucketPolicyV2Input struct {
	Policy string `json:"Policy,omitempty"`
}

type PutBucketPolicyV2Input struct {
	Bucket string `json:"-"`
	Policy string `json:"Policy,omitempty"`
}

type PutBucketPolicyV2Output struct {
	RequestInfo `json:"-"`
}

type DeleteBucketPolicyV2Input struct {
	Bucket string `json:"-"`
}
type DeleteBucketPolicyV2Output struct {
	RequestInfo
}

// GetBucketPolicy get bucket access policy
func (cli *Client) GetBucketPolicy(ctx context.Context, bucket string) (*GetBucketPolicyOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(bucket, "").
		WithQuery("policy", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return &GetBucketPolicyOutput{
		RequestInfo: res.RequestInfo(),
		Policy:      string(data),
	}, nil
}

// PutBucketPolicy set bucket access policy
func (cli *Client) PutBucketPolicy(ctx context.Context, bucket string, policy *BucketPolicy) (*PutBucketPolicyOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(bucket, "").
		WithQuery("policy", "").
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, strings.NewReader(policy.Policy), cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	return &PutBucketPolicyOutput{RequestInfo: res.RequestInfo()}, nil
}

// DeleteBucketPolicy delete bucket access policy
func (cli *Client) DeleteBucketPolicy(ctx context.Context, bucket string) (*DeleteBucketPolicyOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(bucket, "").
		WithQuery("policy", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	return &DeleteBucketPolicyOutput{RequestInfo: res.RequestInfo()}, nil
}

func (cli *ClientV2) PutBucketPolicyV2(ctx context.Context, input *PutBucketPolicyV2Input) (*PutBucketPolicyV2Output, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("policy", "").
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader([]byte(input.Policy)), cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketPolicyV2Output{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketPolicyV2(ctx context.Context, input *GetBucketPolicyV2Input) (*GetBucketPolicyV2Output, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("policy", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketPolicyV2Output{RequestInfo: res.RequestInfo()}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	output.Policy = string(data)
	return &output, nil
}

func (cli *ClientV2) DeleteBucketPolicyV2(ctx context.Context, input *DeleteBucketPolicyV2Input) (*DeleteBucketPolicyV2Output, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("policy", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketPolicyV2Output{RequestInfo: res.RequestInfo()}
	return &output, nil

}
