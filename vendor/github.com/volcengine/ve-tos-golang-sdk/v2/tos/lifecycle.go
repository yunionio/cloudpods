package tos

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

func (cli *ClientV2) parseLifecycleInput(input *PutBucketLifecycleInput) putBucketLifecycleInput {
	lifecycleInput := make([]lifecycleRule, 0, len(input.Rules))
	for _, lifecycle := range input.Rules {
		var exp *expiration
		if lifecycle.Expiration != nil {
			exp = &expiration{
				Days: lifecycle.Expiration.Days,
			}
			if !lifecycle.Expiration.Date.IsZero() {
				exp.Date = lifecycle.Expiration.Date.Format(time.RFC3339)
			}
		}

		transitionList := make([]transition, 0, len(lifecycle.Transitions))

		for _, trans := range lifecycle.Transitions {
			t := transition{
				Days:         trans.Days,
				StorageClass: trans.StorageClass,
			}
			if !trans.Date.IsZero() {
				t.Date = trans.Date.Format(time.RFC3339)
			}
			transitionList = append(transitionList, t)
		}

		lifecycleInput = append(lifecycleInput, lifecycleRule{
			ID:                             lifecycle.ID,
			Prefix:                         lifecycle.Prefix,
			Status:                         lifecycle.Status,
			Transitions:                    transitionList,
			Expiration:                     exp,
			NonCurrentVersionTransition:    lifecycle.NonCurrentVersionTransition,
			NoCurrentVersionExpiration:     lifecycle.NoCurrentVersionExpiration,
			Tag:                            lifecycle.Tag,
			AbortInCompleteMultipartUpload: lifecycle.AbortInCompleteMultipartUpload,
		})

	}
	return putBucketLifecycleInput{Rules: lifecycleInput}
}
func (cli *ClientV2) PutBucketLifecycle(ctx context.Context, input *PutBucketLifecycleInput) (*PutLifecycleOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	data, contentMD5, err := marshalInput("PutBucketLifecycleInput", cli.parseLifecycleInput(input))
	if err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("lifecycle", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodPut, bytes.NewReader(data), cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutLifecycleOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}

func (cli *ClientV2) GetBucketLifecycle(ctx context.Context, input *GetBucketLifecycleInput) (*GetBucketLifecycleOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("lifecycle", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketLifecycleOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(output.RequestID, res.Body, &output); err != nil {
		return nil, err
	}
	return &output, nil
}

func (cli *ClientV2) DeleteBucketLifecycle(ctx context.Context, input *DeleteBucketLifecycleInput) (*DeleteBucketLifecycleOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("lifecycle", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodDelete, nil, cli.roundTripper(http.StatusNoContent))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	output := DeleteBucketLifecycleOutput{RequestInfo: res.RequestInfo()}
	return &output, nil

}
