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
	"io"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func HistoryDataClean(s *mcclient.ClientSession, serviceType string, day *int) (jsonutils.JSONObject, error) {
	man := &BaseManager{serviceType: serviceType}
	input := jsonutils.NewDict()
	if day != nil {
		input.Set("day", jsonutils.NewInt(int64(*day)))
	}
	resp, err := man.rawRequest(s, "POST", "/history-data-clean", nil, strings.NewReader(input.String()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return jsonutils.Parse(body)
}
