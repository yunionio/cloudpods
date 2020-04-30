package hosthandler

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	keyWords = []string{"hosts"}
)

func AddHostHandler(prefix string, app *appsrv.Application) {
	for _, keyword := range keyWords {
		app.AddHandler("POST", fmt.Sprintf("%s/%s/shutdown-servers-on-host-down", prefix, keyword),
			auth.Authenticate(setOnHostDown))
	}
}

func setOnHostDown(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if err := host_health.SetOnHostDown(host_health.SHUTDOWN_SERVERS); err != nil {
		hostutils.Response(ctx, w, err)
		return
	}
	hostutils.ResponseOk(ctx, w)
}
