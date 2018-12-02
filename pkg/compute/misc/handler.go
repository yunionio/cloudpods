package misc

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/tristate"
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
	body, _ := appsrv.FetchJSON(r)
	if body == nil || !body.Contains("ip") {
		httperrors.InputParameterError(w, "Missing ip ?")
		return
	}
	ipAddr, _ := body.GetString("ip")
	n, _ := models.NetworkManager.GetNetworkOfIP(ipAddr, "baremetal", tristate.None)
	if n == nil {
		httperrors.NotFoundError(w, "Network not found")
		return
	}
	zoneId := n.GetWire().ZoneId
	bmAgentUrl, err := auth.GetServiceURL("baremetal", options.Options.Region, zoneId, "")
	if err != nil {
		log.Errorln(err)
		httperrors.InternalServerError(w, err.Error())
		return
	}
	fmt.Fprintf(w, bmAgentUrl)
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
