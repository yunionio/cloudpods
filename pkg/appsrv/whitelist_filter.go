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

package appsrv

import (
	"context"
	"net/http"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
)

var (
	whitelisUserAgents []string
)

const kubeProbeUserAgent = "kube-probe"

func SetDefaultHandlersWhitelistUserAgents(userAgents []string) {
	for _, userAgent := range userAgents {
		whitelisUserAgents = append(whitelisUserAgents, strings.ToLower(userAgent))
	}
	if !utils.IsInArray(httputils.USER_AGENT, whitelisUserAgents) {
		whitelisUserAgents = append(whitelisUserAgents, httputils.USER_AGENT)
	}
	if !utils.IsInArray(kubeProbeUserAgent, whitelisUserAgents) {
		whitelisUserAgents = append(whitelisUserAgents, kubeProbeUserAgent)
	}
}

func WhitelistFilter(handler FilterHandler) FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		for _, userAgent := range whitelisUserAgents {
			if strings.Contains(strings.ToLower(r.UserAgent()), userAgent) {
				handler(ctx, w, r)
				return
			}
		}
		log.Errorf("Forbidden default handler request: %s allow: %s", r.UserAgent(), strings.Join(whitelisUserAgents, ","))
		httperrors.ForbiddenError(ctx, w, "Forbidden")
	}
}
