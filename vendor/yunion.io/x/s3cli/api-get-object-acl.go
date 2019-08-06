/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2018 MinIO, Inc.
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
	"net/http"
	"net/url"
	"encoding/xml"
)

const (
	CANNED_ACL_PRIVATE           = "private"
	CANNED_ACL_AUTH_READ         = "authenticated-read"
	CANNED_ACL_PUBLIC_READ       = "public-read"
	CANNED_ACL_PUBLIC_READ_WRITE = "public-read-write"
)

type AccessControlPolicy struct {
	XMLName xml.Name `xml:"AccessControlPolicy"`
	Owner struct {
		ID          string `xml:"ID"`
		DisplayName string `xml:"DisplayName"`
	} `xml:"Owner"`
	AccessControlList struct {
		Grant []Grant `xml:"Grant"`
	} `xml:"AccessControlList"`
}

//GetObjectACL get object ACLs
func (c Client) GetObjectACL(bucketName, objectName string) (*AccessControlPolicy, error) {

	resp, err := c.executeMethod(context.Background(), "GET", requestMetadata{
		bucketName: bucketName,
		objectName: objectName,
		queryValues: url.Values{
			"acl": []string{""},
		},
	})
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp, bucketName, objectName)
	}

	res := &AccessControlPolicy{}

	if err := xmlDecoder(resp.Body, res); err != nil {
		return nil, err
	}

	return res, nil
}

func CannedAcl(ownerId string, ownerName string, acl string) AccessControlPolicy {
	owner := Owner{ID: ownerId, DisplayName: ownerName}
	switch acl {
	case CANNED_ACL_PRIVATE:
		return PrivateAcl(owner)
	case CANNED_ACL_AUTH_READ:
		return AuthReadAcl(owner)
	case CANNED_ACL_PUBLIC_READ:
		return PublicReadAcl(owner)
	case CANNED_ACL_PUBLIC_READ_WRITE:
		return PublicReadWriteAcl(owner)
	default:
		return PrivateAcl(owner)
	}
}

func PrivateAcl(owner Owner) AccessControlPolicy {
	policy := AccessControlPolicy{}
	policy.Owner.ID = owner.ID
	policy.Owner.DisplayName = owner.DisplayName
	policy.AccessControlList.Grant = make([]Grant, 1)
	policy.AccessControlList.Grant[0].Grantee.Type = GRANTEE_TYPE_USER
	policy.AccessControlList.Grant[0].Grantee.ID = owner.ID
	policy.AccessControlList.Grant[0].Grantee.DisplayName = owner.DisplayName
	policy.AccessControlList.Grant[0].Permission = PERMISSION_FULL_CONTROL
	return policy
}

func AuthReadAcl(owner Owner) AccessControlPolicy {
	policy := PrivateAcl(owner)
	authRead := Grant{}
	authRead.Permission = PERMISSION_READ
	authRead.Grantee.Type = GRANTEE_TYPE_GROUP
	authRead.Grantee.URI = GRANTEE_GROUP_URI_AUTH_USERS
	policy.AccessControlList.Grant = append(policy.AccessControlList.Grant, authRead)
	return policy
}

func PublicReadAcl(owner Owner) AccessControlPolicy {
	policy := PrivateAcl(owner)
	publicRead := Grant{}
	publicRead.Permission = PERMISSION_READ
	publicRead.Grantee.Type = GRANTEE_TYPE_GROUP
	publicRead.Grantee.URI = GRANTEE_GROUP_URI_ALL_USERS
	policy.AccessControlList.Grant = append(policy.AccessControlList.Grant, publicRead)
	return policy
}

func PublicReadWriteAcl(owner Owner) AccessControlPolicy {
	policy := PublicReadAcl(owner)
	publicWrite := Grant{}
	publicWrite.Permission = PERMISSION_WRITE
	publicWrite.Grantee.Type = GRANTEE_TYPE_GROUP
	publicWrite.Grantee.URI = GRANTEE_GROUP_URI_ALL_USERS
	policy.AccessControlList.Grant = append(policy.AccessControlList.Grant, publicWrite)
	return policy
}

func (aCPolicy *AccessControlPolicy) GetCannedACL() string {
	grants := aCPolicy.AccessControlList.Grant

	switch {
	case len(grants) == 1:
		if grants[0].Grantee.URI == "" && grants[0].Permission == PERMISSION_FULL_CONTROL {
			return CANNED_ACL_PRIVATE
		}
	case len(grants) == 2:
		for _, g := range grants {
			if g.Grantee.URI == GRANTEE_GROUP_URI_AUTH_USERS && g.Permission == PERMISSION_READ {
				return CANNED_ACL_AUTH_READ
			}
			if g.Grantee.URI == GRANTEE_GROUP_URI_ALL_USERS && g.Permission == PERMISSION_READ {
				return CANNED_ACL_PUBLIC_READ
			}
			if g.Permission == "READ" && g.Grantee.ID == aCPolicy.Owner.ID {
				return "bucket-owner-read"
			}
		}
	case len(grants) == 3:
		for _, g := range grants {
			if g.Grantee.URI == GRANTEE_GROUP_URI_ALL_USERS && g.Permission == PERMISSION_WRITE {
				return CANNED_ACL_PUBLIC_READ_WRITE
			}
		}
	}
	return ""
}

func (aCPolicy *AccessControlPolicy) GetAmzGrantACL() map[string][]string {
	grants := aCPolicy.AccessControlList.Grant
	res := map[string][]string{}

	for _, g := range grants {
		switch {
		case g.Permission == "READ":
			res["X-Amz-Grant-Read"] = append(res["X-Amz-Grant-Read"], "id="+g.Grantee.ID)
		case g.Permission == "WRITE":
			res["X-Amz-Grant-Write"] = append(res["X-Amz-Grant-Write"], "id="+g.Grantee.ID)
		case g.Permission == "READ_ACP":
			res["X-Amz-Grant-Read-Acp"] = append(res["X-Amz-Grant-Read-Acp"], "id="+g.Grantee.ID)
		case g.Permission == "WRITE_ACP":
			res["X-Amz-Grant-Write-Acp"] = append(res["X-Amz-Grant-Write-Acp"], "id="+g.Grantee.ID)
		case g.Permission == "FULL_CONTROL":
			res["X-Amz-Grant-Full-Control"] = append(res["X-Amz-Grant-Full-Control"], "id="+g.Grantee.ID)
		}
	}
	return res
}
