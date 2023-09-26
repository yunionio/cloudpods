package tos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// PutObjectAcl AclGrant, AclRules can not set both.
//
// Deprecated: ues PutObjectACL of ClientV2 instead
func (bkt *Bucket) PutObjectAcl(ctx context.Context, input *PutObjectAclInput, options ...Option) (*PutObjectAclOutput, error) {
	if err := isValidKey(input.Key); err != nil {
		return nil, err
	}

	var content io.Reader
	if input.AclRules != nil {
		data, err := json.Marshal(input.AclRules)
		if err != nil {
			return nil, fmt.Errorf("tos: marshal BucketAcl Ruels err: %s", err.Error())
		}
		content = bytes.NewReader(data)
	}

	builder := bkt.client.newBuilder(bkt.name, input.Key, options...).
		WithQuery("acl", "").
		WithQuery("versionId", input.VersionID)
	if grant := input.AclGrant; grant != nil {
		builder.WithHeader(HeaderACL, grant.ACL).
			WithHeader(HeaderGrantFullControl, grant.GrantFullControl).
			WithHeader(HeaderGrantRead, grant.GrantRead).
			WithHeader(HeaderGrantReadAcp, grant.GrantReadAcp).
			WithHeader(HeaderGrantWriteAcp, grant.GrantWriteAcp)
	}

	res, err := builder.WithRetry(OnRetryFromStart, StatusCodeClassifier{}).Request(ctx, http.MethodPut, content, bkt.client.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	return &PutObjectAclOutput{RequestInfo: res.RequestInfo()}, nil
}

// PutObjectACL put object ACL
func (cli *ClientV2) PutObjectACL(ctx context.Context, input *PutObjectACLInput) (*PutObjectACLOutput, error) {
	if err := isValidKey(input.Key); err != nil {
		return nil, err
	}

	if len(input.ACL) != 0 {
		if err := isValidACL(input.ACL); err != nil {
			return nil, err
		}
	}

	var content io.Reader
	if len(input.Grants) != 0 {

		for _, grants := range input.Grants {
			if err := isValidGrantee(grants.GranteeV2.Type); len(grants.GranteeV2.Type) != 0 && err != nil {
				return nil, err
			}
			if err := isValidCannedType(grants.GranteeV2.Canned); len(grants.GranteeV2.Canned) != 0 && err != nil {
				return nil, err
			}

			if err := isValidPermission(grants.Permission); len(grants.Permission) != 0 && err != nil {
				return nil, err
			}
		}

		data, err := json.Marshal(&accessControlList{
			Owner:                input.Owner,
			Grants:               input.Grants,
			BucketOwnerEntrusted: input.BucketOwnerEntrusted,
		})
		if err != nil {
			return nil, InvalidMarshal
		}
		content = bytes.NewReader(data)
	}
	builder := cli.newBuilder(input.Bucket, input.Key).
		WithQuery("acl", "").
		WithParams(*input)
	res, err := builder.WithRetry(OnRetryFromStart, StatusCodeClassifier{}).Request(ctx, http.MethodPut, content, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	return &PutObjectACLOutput{
		PutObjectAclOutput{RequestInfo: res.RequestInfo()},
	}, nil
}

// GetObjectAcl get object ACL
//   objectKey: the name of object
//   Options: WithVersionID the version of the object
//
// Deprecated: use GetObjectACL of ClientV2 instead
func (bkt *Bucket) GetObjectAcl(ctx context.Context, objectKey string, options ...Option) (*GetObjectAclOutput, error) {
	if err := isValidKey(objectKey); err != nil {
		return nil, err
	}

	res, err := bkt.client.newBuilder(bkt.name, objectKey, options...).
		WithQuery("acl", "").
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, bkt.client.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	out := GetObjectAclOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(out.RequestID, res.Body, &out); err != nil {
		return nil, err
	}

	out.VersionID = res.Header.Get(HeaderVersionID)
	return &out, nil
}

// GetObjectACL get object ACL
func (cli *ClientV2) GetObjectACL(ctx context.Context, input *GetObjectACLInput) (*GetObjectACLOutput, error) {
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	if err := isValidKey(input.Key); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, input.Key).
		WithQuery("acl", "").
		WithParams(*input).
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	out := GetObjectACLOutput{RequestInfo: res.RequestInfo()}
	if err = marshalOutput(out.RequestID, res.Body, &out); err != nil {
		return nil, err
	}
	out.VersionID = res.Header.Get(HeaderVersionID)
	return &out, nil
}

func (cli *ClientV2) GetBucketACL(ctx context.Context, input *GetBucketACLInput) (*GetBucketACLOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}
	res, err := cli.newBuilder(input.Bucket, "").
		WithQuery("acl", "").
		WithRetry(nil, StatusCodeClassifier{}).
		Request(ctx, http.MethodGet, nil, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := GetBucketACLOutput{RequestInfo: res.RequestInfo()}
	marshalRes := bucketACL{}
	if err = marshalOutput(output.RequestID, res.Body, &marshalRes); err != nil {
		return nil, err
	}
	output.Grants = marshalRes.GrantList
	output.Owner = marshalRes.Owner
	return &output, nil
}

func (cli *ClientV2) PutBucketACL(ctx context.Context, input *PutBucketACLInput) (*PutBucketACLOutput, error) {
	if input == nil {
		return nil, InputIsNilClientError
	}
	if err := isValidBucketName(input.Bucket, cli.isCustomDomain); err != nil {
		return nil, err
	}

	reqBuilder := cli.newBuilder(input.Bucket, "").
		WithQuery("acl", "").
		WithRetry(OnRetryFromStart, StatusCodeClassifier{}).
		WithParams(*input)
	var reqData io.Reader

	if input.Owner.ID != "" && len(input.Grants) != 0 {
		data, contentMD5, err := marshalInput("PutBucketACLInput", bucketACL{
			Owner:     input.Owner,
			GrantList: input.Grants,
		})
		if err != nil {
			return nil, err
		}
		_ = reqBuilder.WithHeader(HeaderContentMD5, contentMD5)
		reqData = bytes.NewReader(data)
	}

	res, err := reqBuilder.Request(ctx, http.MethodPut, reqData, cli.roundTripper(http.StatusOK))
	if err != nil {
		return nil, err
	}
	defer res.Close()
	output := PutBucketACLOutput{RequestInfo: res.RequestInfo()}
	return &output, nil
}
