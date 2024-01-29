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

package service

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/webconsole/models"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/server"
)

func StartService() {

	opts := &o.Options
	commonOpts := &o.Options.CommonOptions
	common_options.ParseOptions(opts, os.Args, "webconsole.conf", api.SERVICE_TYPE)

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, o.OnOptionsChange)

	if opts.ApiServer == "" {
		log.Fatalf("--api-server must specified")
	}
	_, err := url.Parse(opts.ApiServer)
	if err != nil {
		log.Fatalf("invalid --api-server %s", opts.ApiServer)
	}

	registerSigTraps()
	start()
}

func registerSigTraps() {
	signalutils.SetDumpStackSignal()
	signalutils.StartTrap()
}

func start() {
	baseOpts := &o.Options.BaseOptions

	// commonOpts := &o.Options.CommonOptions
	app := app_common.InitApp(baseOpts, true)
	dbOpts := &o.Options.DBOptions

	cloudcommon.InitDB(dbOpts)

	initHandlers(app)

	db.EnsureAppSyncDB(app, dbOpts, models.InitDB)

	root := mux.NewRouter()
	root.UseEncodedPath()

	// api handler
	root.PathPrefix(ApiPathPrefix).Handler(app)

	srv := server.NewConnectionServer()
	// websocket command text console handler
	root.Handle(ConnectPathPrefix, srv)

	// websockify graphic console handler
	root.Handle(WebsockifyPathPrefix, srv)

	// websocketproxy handler
	root.Handle(WebsocketProxyPathPrefix, srv)

	// misc handler
	addMiscHandlers(app, root)

	cron := cronman.InitCronJobManager(true, o.Options.CronJobWorkerCount)

	cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)

	cron.Start()
	defer cron.Stop()

	addr := net.JoinHostPort(o.Options.Address, strconv.Itoa(o.Options.Port))
	log.Infof("Start listen on %s", addr)
	if o.Options.EnableSsl {
		err := http.ListenAndServeTLS(addr,
			o.Options.SslCertfile,
			o.Options.SslKeyfile,
			root)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("%v", err)
		}
	} else {
		err := http.ListenAndServe(addr, root)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}

func addMiscHandlers(app *appsrv.Application, root *mux.Router) {
	adapterF := func(appHandleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			appHandleFunc(app.GetContext(), w, r)
		}
	}

	// ref: pkg/appsrv/appsrv:addDefaultHandlers
	root.HandleFunc("/version", adapterF(appsrv.VersionHandler))
	root.HandleFunc("/stats", adapterF(appsrv.StatisticHandler))
	root.HandleFunc("/ping", adapterF(appsrv.PingHandler))
	root.HandleFunc("/worker_stats", adapterF(appsrv.WorkerStatsHandler))

	// pprof handler
	root.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
}
