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

package modulebase

import (
	"fmt"
	"io"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func GetPProfByType(s *mcclient.ClientSession, serviceType string, profileType string, seconds int) (io.Reader, error) {
	man := &BaseManager{serviceType: serviceType}
	if seconds <= 0 {
		seconds = 15
	}
	resp, err := man.rawBaseUrlRequest(s, "GET", fmt.Sprintf("/debug/pprof/%s?seconds=%d", profileType, seconds), nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
