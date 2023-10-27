package tos

import (
	"context"
	"net/http"
)

const (
	BucketVersioningEnable    = "Enabled"
	BucketVersioningSuspended = "Suspended"
)

type GetBucketVersioningOutput struct {
	RequestInfo `json:"-"`
	Status      string `json:"Status"`
}

// GetBucketVersioning get the multi-version status of a bucket
func (cli *Client) GetBucketVersioning(ctx context.Context, bucket string) (*GetBucketVersioningOutput, error) {
	if err := isValidBucketName(bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	res, err := cli.newBuilder(bucket, "").
		WithQuery("versioning", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := GetBucketVersioningOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}
