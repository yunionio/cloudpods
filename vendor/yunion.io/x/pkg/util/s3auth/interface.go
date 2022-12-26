// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3auth

import (
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type IAccessKeySecretRequest interface {
	GetAccessKey() string
	Validate() error
	ParseRequest(req http.Request, virtualHost bool) error
	Verify(secret string) error
	Encode() string
}

type SAccessKeyRequest struct {
	Algorithm string `json:"algorithm,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	Signature string `json:"signature,omitempty"`
	Request   string `json:"request,omitempty"`
}

func (aksk SAccessKeyRequest) GetAccessKey() string {
	return aksk.AccessKey
}

func (aksk SAccessKeyRequest) Validate() error {
	if len(aksk.AccessKey) == 0 {
		return errors.Error("Missing AWSAccessKeyId")
	}
	if len(aksk.Signature) == 0 {
		return errors.Error("Missing Signature")
	}
	return nil
}

func decodeAuthHeader(authHeader string) (IAccessKeySecretRequest, error) {
	pos := strings.IndexByte(authHeader, ' ')
	if pos <= 0 {
		return nil, errors.Error("illegal authorization header")
	}
	algo := authHeader[:pos]
	switch algo {
	case signV2Algorithm:
		req, err := decodeAuthHeaderV2(authHeader[pos+1:])
		if err != nil {
			return nil, errors.Wrap(err, "decodeAuthHeaderV2")
		}
		return req, nil
	case signV4Algorithm:
		req, err := decodeAuthHeaderV4(authHeader[pos+1:])
		if err != nil {
			return nil, errors.Wrap(err, "decodeAuthHeaderV4")
		}
		return req, nil
	default:
		return nil, errors.Error("unsupported signing algorithm")
	}
}

func DecodeAccessKeyRequest(req http.Request, virtualHost bool) (IAccessKeySecretRequest, error) {
	authHeader := req.Header.Get("Authorization")
	if len(authHeader) == 0 {
		return nil, errors.Error("missing authorization header")
	}
	akskReq, err := decodeAuthHeader(authHeader)
	if err != nil {
		return nil, errors.Wrap(err, "decodeAuthHeader")
	}
	err = akskReq.ParseRequest(req, virtualHost)
	if err != nil {
		return nil, errors.Wrap(err, "akskReq.ParseRequest")
	}

	return akskReq, akskReq.Validate()
}

func Decode(reqStr string) (IAccessKeySecretRequest, error) {
	rawReq := SAccessKeyRequest{}
	reqJson, err := jsonutils.ParseString(reqStr)
	if err != nil {
		return nil, errors.Wrap(err, "jsonutils.ParseString")
	}
	err = reqJson.Unmarshal(&rawReq)
	if err != nil {
		return nil, errors.Wrap(err, "reqJson.Unmarshal rawReq")
	}
	var ret IAccessKeySecretRequest
	switch rawReq.Algorithm {
	case signV2Algorithm:
		ret = &SAccessKeyRequestV2{}
	case signV4Algorithm:
		ret = &SAccessKeyRequestV4{}
	default:
		return nil, errors.Error("unsupported sign algorithm")
	}
	err = reqJson.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrap(err, "reqJson.Unmarshal")
	}
	return ret, nil
}
