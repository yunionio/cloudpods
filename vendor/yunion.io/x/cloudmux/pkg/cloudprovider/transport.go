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

package cloudprovider

import "net/http"

type transport struct {
	check func(*http.Request) (func(resp *http.Response) error, error)
	ts    *http.Transport
}

func (self *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	var respCheck func(resp *http.Response) error = nil
	var err error
	if self.check != nil {
		respCheck, err = self.check(req)
		if err != nil {
			return nil, err
		}
	}
	resp, err := self.ts.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if respCheck != nil {
		err = respCheck(resp)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func GetCheckTransport(ts *http.Transport, check func(*http.Request) (func(resp *http.Response) error, error)) http.RoundTripper {
	ret := &transport{ts: ts, check: check}
	return ret
}
