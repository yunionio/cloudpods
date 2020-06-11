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

package modules

import (
	"context"
	"fmt"
	"io"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func GetPProfByType(s *mcclient.ClientSession, serviceType string, profileType string, seconds int) (io.Reader, error) {
	return modulebase.GetPProfByType(s, serviceType, profileType, seconds)
}

func GetNamedAddressPProfByType(s *mcclient.ClientSession, address string, profileType string, params *jsonutils.JSONDict) (io.Reader, error) {
	urlStr := fmt.Sprintf("%s%s", address, fmt.Sprintf("/debug/pprof/%s", profileType))
	if params != nil {
		if queryStr := params.QueryString(); queryStr != "" {
			urlStr = fmt.Sprintf("%s?%s", urlStr, queryStr)
		}
	}
	cli := s.GetClient()
	resp, err := httputils.Request(cli.HttpClient(), context.Background(), "GET", urlStr, s.Header, nil, cli.GetDebug())
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
