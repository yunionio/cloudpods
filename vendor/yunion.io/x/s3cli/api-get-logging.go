/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package s3cli

import (
	"context"
	"encoding/xml"
	"net/http"
	"net/url"

	"github.com/minio/minio-go/v6/pkg/s3utils"
)

const (
	GRANTEE_TYPE_GROUP = "Group"
	GRANTEE_TYPE_EMAIL = "AmazonCustomerByEmail"
	GRANTEE_TYPE_USER  = "CanonicalUser"

	GRANTEE_GROUP_URI_ALL_USERS  = "http://acs.amazonaws.com/groups/global/AllUsers"
	GRANTEE_GROUP_URI_AUTH_USERS = "http://acs.amazonaws.com/groups/global/AuthenticatedUsers"

	PERMISSION_FULL_CONTROL = "FULL_CONTROL"
	PERMISSION_READ         = "READ"
	PERMISSION_WRITE        = "WRITE"
)

type BucketLoggingStatus struct {
	XMLName        xml.Name       `xml:"BucketLoggingStatus"`
	LoggingEnabled LoggingEnabled `xml:"LoggingEnabled"`
}

type LoggingEnabled struct {
	TargetBucket string       `xml:"TargetBucket"`
	TargetPrefix string       `xml:"TargetPrefix"`
	TargetGrants TargetGrants `xml:"TargetGrants"`
}

type TargetGrants struct {
	Grant []Grant `xml:"Grant"`
}

type Grant struct {
	Permission string  `xml:"Permission"`
	Grantee    Grantee `xml:"Grantee"`
}

type Grantee struct {
	Type         string `xml:"xsi:type,attr"`
	EmailAddress string `xml:"EmailAddress,omitempty"`
	ID           string `xml:"ID,omitempty"`
	DisplayName  string `xml:"DisplayName,omitempty"`
	URI          string `xml:"URI,omitempty"`
}

// GetBucketLogging - get bucket logging at a given path.
func (c Client) GetBucketLogging(bucketName string) (*BucketLoggingStatus, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return nil, err
	}
	bucketLogging, err := c.getBucketLogging(bucketName)
	if err != nil {
		errResponse := ToErrorResponse(err)
		if errResponse.Code == "NoSuchBucketPolicy" {
			return nil, nil
		}
		return nil, err
	}
	return bucketLogging, nil
}

// Request server for current bucket Logging.
func (c Client) getBucketLogging(bucketName string) (*BucketLoggingStatus, error) {
	// Get resources properly escaped and lined up before
	// using them in http request.
	urlValues := make(url.Values)
	urlValues.Set("logging", "")

	// Execute GET on bucket to list objects.
	resp, err := c.executeMethod(context.Background(), "GET", requestMetadata{
		bucketName:       bucketName,
		queryValues:      urlValues,
		contentSHA256Hex: emptySHA256Hex,
	})

	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, httpRespToErrorResponse(resp, bucketName, "")
		}
	}

	bucketLoggingStatus := BucketLoggingStatus{}
	err = xmlDecoder(resp.Body, &bucketLoggingStatus)
	if err != nil {
		return nil, err
	}
	return &bucketLoggingStatus, nil
}
