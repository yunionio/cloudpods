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

package ft

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func ResolveAiproxyURL(session *mcclient.ClientSession, override string) (string, error) {
	if override = strings.TrimRight(strings.TrimSpace(override), "/"); override != "" {
		return override, nil
	}
	if url := resolveAiproxyURLFromEnv(); url != "" {
		return url, nil
	}
	query := jsonutils.NewDict()
	query.Set("service", jsonutils.NewString("aiproxy"))
	query.Set("interface", jsonutils.NewString("public"))
	query.Set("limit", jsonutils.NewInt(1))
	result, err := modules.EndpointsV3.List(session, query)
	if err != nil {
		return "", errors.Wrap(err, "endpoint-list aiproxy public")
	}
	if len(result.Data) == 0 {
		return "", errors.Error("cannot resolve aiproxy public URL; set AIPROXY_URL")
	}
	url, _ := result.Data[0].GetString("url")
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if url == "" {
		return "", errors.Error("cannot resolve aiproxy public URL; set AIPROXY_URL")
	}
	return url, nil
}
