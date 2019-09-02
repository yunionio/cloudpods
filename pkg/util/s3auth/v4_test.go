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
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSignV4(t *testing.T) {
	getReq, _ := http.NewRequest(http.MethodGet, "https://api.aliyun-cs.com/2018-06-23/DescribeInstances?limit=10&offset=0", nil)
	putBody := "<xml><123></123></xml>"
	putReq, _ := http.NewRequest(http.MethodPut, "https://api.aliyun-cs.com/2018-06-23/ModifyInstance?InstanceId=aabbccdd", strings.NewReader(putBody))

	cases := []struct {
		request   http.Request
		accessKey string
		secret    string
		location  string
		body      io.Reader
	}{
		{
			request:   *getReq,
			accessKey: "1234567890",
			secret:    "1234567890",
			location:  "cn-beijing",
			body:      nil,
		},
		{
			request:   *putReq,
			accessKey: "1234567890",
			secret:    "1234567890",
			location:  "cn-beijing",
			body:      strings.NewReader(putBody),
		},
	}

	for _, c := range cases {
		newreq := SignV4(c.request, c.accessKey, c.secret, c.location, c.body)
		t.Logf("%#v", newreq)

		req, err := DecodeAccessKeyRequest(*newreq, false)
		if err != nil {
			t.Errorf("DecodeAccessKeyRequest fail %s", err)
			continue
		}

		err = req.Verify(c.secret)
		if err != nil {
			t.Errorf("verify fail %s", err)
		}
	}

}
