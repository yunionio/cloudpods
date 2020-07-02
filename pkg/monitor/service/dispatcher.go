package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	sub "yunion.io/x/onecloud/pkg/monitor/influxdbsubscribe"
	"yunion.io/x/onecloud/pkg/monitor/subscriptionmodel"
)

var (
	SubscriptionWorkerManager *appsrv.SWorkerManager
)

func init() {
	SubscriptionWorkerManager = appsrv.NewWorkerManager("SubscriptionWorkerManager", 4, 1024, false)
}

func addCommonAlertDispatcher(prefix string, app *appsrv.Application) {
	manager := db.NewModelHandler(subscriptionmodel.SubscriptionManager)

	metadata := map[string]interface{}{"manager": manager}
	tags := map[string]string{"resource": subscriptionmodel.SubscriptionManager.KeywordPlural()}
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<subscription>", prefix, manager.KeywordPlural()),
		performHandler, metadata, "perform_class_subscription", tags)
}

func performHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	SubscriptionWorkerManager.Run(func() {
		_, query, body := fetchEnv(ctx, w, r)
		ctx = context.WithValue(context.Background(), auth.AUTH_TOKEN, auth.AdminCredential())
		subscriptionmodel.SubscriptionManager.PerformWrite(ctx, auth.AdminCredential(), query, body)
		appsrv.SendJSON(w, wrap(jsonutils.NewDict(), "subscription"))
	}, nil, nil)

}

// fetchEnv fetch handler, params, query and body from ctx(context.Context)
func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string,
	jsonutils.JSONObject, []sub.Point) {
	params, query, body := FetchEnv(ctx, w, r)
	return params, query, body
}

func FetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (params map[string]string,
	query jsonutils.JSONObject, body []sub.Point) {
	var err error
	params = appctx.AppContextParams(ctx)
	query, err = jsonutils.ParseQueryString(r.URL.RawQuery)
	if err != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, err)
	}
	//var body jsonutils.JSONObject = nil
	if (r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH") && r.ContentLength > 0 {
		body, err = FetchRequest(r)
		if err != nil {
			log.Errorln(err)
		}
	}
	return params, query, body
}

func FetchRequest(req *http.Request) ([]sub.Point, error) {
	body, err := appsrv.Fetch(req)
	if err != nil {
		return nil, err
	}
	precision := req.FormValue("precision")
	if precision == "" {
		precision = "n"
	}
	points, err := sub.ParsePointsWithPrecision(body, time.Now().UTC(), precision)
	if err != nil {
		return nil, err
	}
	return points, nil
}

func mergeQueryParams(params map[string]string, query jsonutils.JSONObject, excludes ...string) jsonutils.JSONObject {
	if query == nil {
		query = jsonutils.NewDict()
	}
	queryDict := query.(*jsonutils.JSONDict)
	for k, v := range params {
		queryDict.Add(jsonutils.NewString(v), k[1:len(k)-1])
	}
	return queryDict
}

func wrap(data jsonutils.JSONObject, key string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(data, key)
	return ret
}
