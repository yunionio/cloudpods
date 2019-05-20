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

package tokens

import (
	api "yunion.io/x/onecloud/pkg/apis/identity"
)

func authMethodStr2Id(method string) byte {
	for i := range api.AUTH_METHODS {
		if api.AUTH_METHODS[i] == method {
			return byte(i) + 1
		}
	}
	return 0
}

func authMethodId2Str(mid byte) string {
	if mid >= 1 && int(mid) <= len(api.AUTH_METHODS) {
		return api.AUTH_METHODS[mid-1]
	}
	return ""
}

func authMethodsStr2Id(methods []string) []byte {
	ret := make([]byte, len(methods))
	for i := range methods {
		ret[i] = authMethodStr2Id(methods[i])
	}
	return ret
}

func authMethodsId2Str(mids []byte) []string {
	ret := make([]string, len(mids))
	for i := range mids {
		ret[i] = authMethodId2Str(mids[i])
	}
	return ret
}
