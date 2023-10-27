package tos

import (
	"bytes"
	"context"
	"net/http"
)

const (
	FetchTaskStateFailed  = "Failed"
	FetchTaskStateSucceed = "Succeed"
	FetchTaskStateExpired = "Expired"
	FetchTaskStateRunning = "Running"
)

type FetchObjectInput struct {
	URL           string `json:"URL,omitempty"`           // required
	Key           string `json:"Key,omitempty"`           // required
	IgnoreSameKey bool   `json:"IgnoreSameKey,omitempty"` // optional, default value is false
	ContentMD5    string `json:"ContentMD5,omitempty"`    // hex-encoded md5, optional
}

type FetchObjectOutput struct {
	RequestInfo `json:"-"`
	VersionID   string `json:"VersionId,omitempty"` // may be empty
	ETag        string `json:"ETag,omitempty"`
}

type fetchObjectInput struct {
	URL           string `json:"URL,omitempty"`           // required
	IgnoreSameKey bool   `json:"IgnoreSameKey,omitempty"` // optional, default value is false
	ContentMD5    string `json:"ContentMD5,omitempty"`    // base64-encoded md5, optional
}

// FetchObject fetch an object from specified URL
// options:
//    WithMeta set meta header(s)
//    WithServerSideEncryptionCustomer set server side encryption options
//    WithACL WithACLGrantFullControl WithACLGrantRead WithACLGrantReadAcp WithACLGrantWrite WithACLGrantWriteAcp set object acl
// Calling FetchObject will be blocked util fetch operation is finished
func (bkt *Bucket) FetchObject(ctx context.Context, input *FetchObjectInput, options ...Option) (*FetchObjectOutput, error) {
	if err := isValidKey(input.Key); err != nil {
		return nil, err
	}

	data, contentMD5, err := marshalInput("FetchObjectInput", &fetchObjectInput{
		URL:           input.URL,
		IgnoreSameKey: input.IgnoreSameKey,
		ContentMD5:    input.ContentMD5,
	})
	if err != nil {
		return nil, err
	}

	res, err := bkt.client.newBuilder(bkt.name, input.Key, options...).
		WithQuery("fetch", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, ServerErrorClassifier{}).
		Request(ctx, http.MethodPost, bytes.NewReader(data), bkt.client.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	out := FetchObjectOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(out.RequestID, res.Body, &out); err != nil {
		return nil, err
	}

	out.VersionID = res.Header.Get(HeaderVersionID)
	return &out, nil
}

type PutFetchTaskInput struct {
	URL           string `json:"URL,omitempty"`           // required
	Object        string `json:"Object,omitempty"`        // object key, required
	IgnoreSameKey bool   `json:"IgnoreSameKey,omitempty"` // optional, default value is false
	ContentMD5    string `json:"ContentMD5,omitempty"`    // hex-encoded md5, optional
}

type PutFetchTaskOutput struct {
	RequestInfo `json:"-"`
	TaskID      string `json:"TaskId,omitempty"`
}

// PutFetchTask put a fetch task to a bucket
// options:
//    WithMeta set meta header(s)
//    WithServerSideEncryptionCustomer set server side encryption options
//    WithACL WithACLGrantFullControl WithACLGrantRead WithACLGrantReadAcp WithACLGrantWrite WithACLGrantWriteAcp set object acl
// Calling PutFetchTask will return immediately after the task created.
func (bkt *Bucket) PutFetchTask(ctx context.Context, input *PutFetchTaskInput, options ...Option) (*PutFetchTaskOutput, error) {
	if err := isValidKey(input.Object); err != nil {
		return nil, err
	}

	data, contentMD5, err := marshalInput("PutFetchTaskInput", input)
	if err != nil {
		return nil, err
	}

	res, err := bkt.client.newBuilder(bkt.name, "", options...).
		WithQuery("fetchTask", "").
		WithHeader(HeaderContentMD5, contentMD5).
		WithRetry(OnRetryFromStart, ServerErrorClassifier{}).
		Request(ctx, http.MethodPost, bytes.NewReader(data), bkt.client.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	out := PutFetchTaskOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(out.RequestID, res.Body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type GetFetchTaskInput struct {
	TaskID string `json:"taskID,omitempty"`
}

type GetFetchTaskOutput struct {
	RequestInfo `json:"-"`
	State       string `json:"State,omitempty"`
	// Cause       string `json:"Cause,omitempty"`
}

// GetFetchTask query the task state by the TaskID
// Task state:
//  FetchTaskStateFailed  = "Failed"
//  FetchTaskStateSucceed = "Succeed"
//  FetchTaskStateExpired = "Expired"
//  FetchTaskStateRunning = "Running"
func (bkt *Bucket) GetFetchTask(ctx context.Context, input *GetFetchTaskInput, options ...Option) (*GetFetchTaskOutput, error) {
	res, err := bkt.client.newBuilder(bkt.name, "", options...).
		WithQuery("fetchTask", "").
		WithQuery("taskId", input.TaskID).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, bkt.client.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	out := GetFetchTaskOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(out.RequestID, res.Body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
