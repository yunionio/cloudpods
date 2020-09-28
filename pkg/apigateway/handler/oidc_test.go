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

package handler

import (
	"reflect"
	"testing"
	"time"

	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/apigateway/clientman"
)

func TestClientInfo(t *testing.T) {
	clientman.SetupTest()

	cases := []struct {
		ip      string
		user    string
		project string
	}{
		{
			ip:      "0.0.0.0",
			user:    "sysadmin",
			project: "system",
		},
		{
			ip:      "10.168.26.253",
			user:    "ab9502de-c6b6-4150-880b-d0e3e6ba8ec8",
			project: "a2049cfadf4c40888b9da136faba5cc8",
		},
	}
	for _, c := range cases {
		info := SOIDCClientInfo{}
		info.Timestamp = time.Now().UnixNano()
		info.Ip, _ = netutils.NewIPV4Addr(c.ip)
		info.UserId = c.user
		info.ProjectId = c.project

		msg := info.toBytes()
		if len(msg) != 14+len(info.UserId)+len(info.ProjectId)+len(info.Region) {
			t.Fatalf("incorrect msg size")
		}

		info2, err := decodeOIDCClientInfo(msg)
		if err != nil {
			t.Fatalf("decode error %s", err)
		}

		if info2.Timestamp != info.Timestamp {
			t.Fatalf("incorrect timestamp")
		}
		if info2.Ip.String() != info.Ip.String() {
			t.Fatalf("incorrect ip")
		}
		if info2.UserId != info.UserId {
			t.Fatalf("incorrect user id")
		}
		if info2.ProjectId != info.ProjectId {
			t.Fatalf("incorrect project id")
		}
		if info2.Region != info.Region {
			t.Fatalf("incorrect region id")
		}

		token := SOIDCClientToken{
			Info: info,
		}

		tokenStr := token.encode()
		token2, err := decodeOIDCClientToken(tokenStr)
		if err != nil {
			t.Fatalf("decodeOIDCClientToken fail %s", err)
		}

		if !reflect.DeepEqual(token2.Info, info2) {
			t.Fatalf("token2 info not equal to info2")
		}
	}
}
