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

package misc

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddMiscHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/misc", prefix)
	addHandler("GET", fmt.Sprintf("%s/bm-agent-url", prefix), getBmAgentUrl, app)
	addHandler("GET", fmt.Sprintf("%s/bm-prepare-script", prefix), getBmPrepareScript, app)
}

func addHandler(method, prefix string, f appsrv.FilterHandler, app *appsrv.Application) {
	handler := auth.Authenticate(f)
	app.AddHandler(method, prefix, handler)
}

func getBmAgentUrl(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		httperrors.NewInternalServerError("Parse remote ip error %s", err)
		return
	}

	n, _ := models.NetworkManager.GetOnPremiseNetworkOfIP(ipAddr, "", tristate.None)
	if n == nil {
		httperrors.NotFoundError(w, "Network not found")
		return
	}

	zoneId := n.GetWire().ZoneId
	bmAgent := models.BaremetalagentManager.GetAgent(compute.AgentTypeBaremetal, zoneId)
	if bmAgent == nil {
		httperrors.InternalServerError(w, "Baremetal agent not found")
		return
	}

	fmt.Fprintf(w, bmAgent.ManagerUri)
}

func getBmPrepareScript(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if len(options.Options.BaremetalPreparePackageUrl) == 0 {
		httperrors.NotAcceptableError(w, "Baremetal package not prepared")
		return
	}
	regionUrl, err := auth.GetServiceURL("compute_v2", options.Options.Region, "", "")
	if err != nil {
		log.Errorln(err)
		httperrors.InternalServerError(w, err.Error())
		return
	}
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	var script string
	script += fmt.Sprintf("curl -fsSL -k -o ./baremetal_prepare.tar.gz %s;",
		options.Options.BaremetalPreparePackageUrl)
	script += "mkdir ./baremetal_prepare;"
	script += "tar -zxf ./baremetal_prepare.tar.gz -C ./baremetal_prepare;"
	script += fmt.Sprintf("./baremetal_prepare/prepare.sh %s %s",
		userCred.GetTokenString(), regionUrl)
	fmt.Fprintf(w, script)
}
