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

package ecloud

import (
	"testing"
)

type signerMock struct {
	SRamRoleSigner
}

func (s *signerMock) GetNonce() string {
	return "9d81ffbeaaf7477390db5df577bb3299"
}

type requestMock struct {
	SBaseRequest
}

func (r *requestMock) GetTimestamp() string {
	return "2017-01-11T15:15:11Z"
}

func TestBuildStringToSing(t *testing.T) {
	signer := &signerMock{SRamRoleSigner: *NewRamRoleSigner("testid", "testsecret")}
	client, _ := NewEcloudClient(NewEcloudClientConfig(signer))
	request := &requestMock{
		SBaseRequest: SBaseRequest{
			Method:      "GET",
			ServerPath:  "/api/keypair",
			QueryParams: map[string]string{},
			Headers:     map[string]string{},
			Content:     []byte{},
		},
	}
	client.completeSingParams(request)
	stringToSign := client.buildStringToSign(request)
	want := `GET
%2Fapi%2Fkeypair
25fb697a06bc8a16a0a2549460f5cba35aded3818e15f088e4e17cd394aa07af`
	if stringToSign != want {
		t.Fatalf("want:\n%s, get:\n%s", want, stringToSign)
	}
}
