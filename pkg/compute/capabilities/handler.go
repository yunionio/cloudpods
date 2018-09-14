package capabilities

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddCapabilityHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/capabilities", prefix), auth.Authenticate(capaHandler), nil, "get_capabilities", nil)
}

func capaHandler(context context.Context, w http.ResponseWriter, r *http.Request) {
	capa := models.GetCapabilities(nil)
	appsrv.SendJSON(w, jsonutils.Marshal(capa))
}
