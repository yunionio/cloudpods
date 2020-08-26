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

package netutils2

import (
	"net/http"
	"testing"
)

func TestGetHttpRequestIp(t *testing.T) {
	cases := []struct {
		request *http.Request
		want    string
	}{
		{
			request: &http.Request{
				Header: map[string][]string{
					"X-Forwarded-For": []string{
						"10.168.26.232",
					},
				},
				RemoteAddr: "10.168.26.2:32322",
			},
			want: "10.168.26.232",
		},
		{
			request: &http.Request{
				RemoteAddr: "192.168.222.23:35533",
			},
			want: "192.168.222.23",
		},
		{
			request: &http.Request{
				Header: map[string][]string{
					"X-Real-Ip": []string{
						"10.40.23.3",
					},
				},
				RemoteAddr: "192.168.222.23:34343",
			},
			want: "10.40.23.3",
		},
		{
			request: &http.Request{
				Header: map[string][]string{
					"X-Real-Ip": []string{
						"10.40.23.3",
					},
					"X-Forwarded-For": []string{
						"211.25.23.2, 10.34.3.1, 192.168.222.3",
					},
				},
				RemoteAddr: "192.168.222.23:34343",
			},
			want: "211.25.23.2",
		},
	}
	for _, c := range cases {
		got := GetHttpRequestIp(c.request)
		if got != c.want {
			t.Errorf("got: %s want: %s", got, c.want)
		}
	}
}
