package usages

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/onecloud/pkg/appsrv"
)

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	app.AddHandler2("GET", prefix, ReportGeneralUsage, nil, "get_usage", nil)
}

func ReportGeneralUsage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}
