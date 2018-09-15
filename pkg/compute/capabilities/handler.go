package capabilities

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddCapabilityHandler(prefix string, app *appsrv.Application) {
	app.AddHandler2("GET", fmt.Sprintf("%s/capabilities", prefix), auth.Authenticate(capaHandler), nil, "get_capabilities", nil)
}

func capaHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx)
	query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	capa, err := models.GetCapabilities(ctx, userCred, query, nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(capa))
}
