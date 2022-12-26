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

package mcclient

import (
	"context"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/s3auth"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type SAkskTokenCredential struct {
	AccessKeySecret api.SAccessKeySecretInfo
	Token           TokenCredential
}

func (this *Client) _verifyKeySecret(aksk s3auth.IAccessKeySecretRequest, aCtx SAuthContext) (*SAkskTokenCredential, error) {
	input := SAuthenticationInputV3{}
	input.Auth.Identity.Methods = []string{api.AUTH_METHOD_AKSK}
	input.Auth.Identity.AccessKeyRequest = aksk.Encode()
	input.Auth.Context = aCtx

	hdr, rbody, err := this.jsonRequest(context.Background(), this.authUrl, "", "POST", "/auth/tokens", nil, jsonutils.Marshal(&input))
	if err != nil {
		return nil, err
	}

	tokenId := hdr.Get("X-Subject-Token")
	if len(tokenId) == 0 {
		return nil, errors.Error("No X-Subject-Token in header")
	}

	ret := SAkskTokenCredential{}
	ret.Token, err = this.unmarshalV3Token(rbody, tokenId)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalV3Token")
	}
	ret.AccessKeySecret = ret.Token.(*TokenCredentialV3).Token.AccessKey
	return &ret, nil
}

func (this *Client) VerifyRequest(req http.Request, aksk s3auth.IAccessKeySecretRequest, virtualHost bool) (*SAkskTokenCredential, error) {
	cliIp := netutils2.GetHttpRequestIp(&req)
	aCtx := SAuthContext{
		Source: AuthSourceSrv,
		Ip:     cliIp,
	}

	token, err := this._verifyKeySecret(aksk, aCtx)
	if err != nil {
		return nil, errors.Wrap(err, "this._verifyKeySecret")
	}

	return token, nil
}

func jsonReader(input interface{}) io.Reader {
	return strings.NewReader(jsonutils.Marshal(input).String())
}

func (this *Client) AuthenticateByAccessKey(accessKey string, secret string, source string) (TokenCredential, error) {
	aCtx := SAuthContext{Source: source}

	seedAksk := s3auth.NewV4Request()
	seedAksk.AccessKey = accessKey
	seedAksk.Location = "cn-beijing"
	input := SAuthenticationInputV3{}
	input.Auth.Identity.Methods = []string{api.AUTH_METHOD_AKSK}
	input.Auth.Identity.AccessKeyRequest = seedAksk.Encode()
	input.Auth.Context = aCtx

	urlStr := joinUrl(this.authUrl, "/auth/tokens")
	req, err := http.NewRequest(http.MethodPost, urlStr, jsonReader(input))
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequest")
	}
	newreq := s3auth.SignV4(*req, seedAksk.AccessKey, secret, seedAksk.Location, jsonReader(input))

	aksk, err := s3auth.DecodeAccessKeyRequest(*newreq, false)
	if err != nil {
		return nil, errors.Wrap(err, "s3auth.DecodeAccessKeyRequest")
	}

	token, err := this._verifyKeySecret(aksk, aCtx)
	if err != nil {
		return nil, errors.Wrap(err, "this._verifyKeySecret")
	}

	return token.Token, nil
}
