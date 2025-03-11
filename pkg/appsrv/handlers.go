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
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"

	"yunion.io/x/pkg/util/version"
)

type FilterHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request)

type TMiddleware func(handler FilterHandler) FilterHandler

func VersionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, version.GetShortString())
}

func PingHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

/*
func CORSHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    reqHdrs, enableCors := r.Header["Access-Control-Request-Headers"]
    if enableCors {
        w.Header().Set("Access-Control-Allow-Origin", getRequestOrigin(r))
        allowHdrs := strings.Join(reqHdrs, ",")
        allowHdrs = fmt.Sprintf("%s,%s", allowHdrs, "Authorization")
        w.Header().Set("Vary", "Origin,Access-Control-Request-Method,Access-Control-Request-Headers")
        w.Header().Set("Access-Control-Allow-Headers", allowHdrs)
        w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,PATCH,DELETE")
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Expose-Headers", allowHdrs)
        w.Header().Set("Access-Control-Max-Age", "86400")
    }
}*/

func addPProfHandler(prefix string, app *Application) {
	pp := "/debug/pprof"
	if prefix != "" {
		prefix = fmt.Sprintf("%s/%s", prefix, pp)
	} else {
		prefix = pp
	}
	app.AddHandler("GET", fmt.Sprintf("%s/", prefix), profIndex).SetProcessNoTimeout()
	app.AddHandler("GET", fmt.Sprintf("%s/cmdline", prefix), profCmdline).SetProcessNoTimeout()
	app.AddHandler("GET", fmt.Sprintf("%s/profile", prefix), profProfile).SetProcessNoTimeout()
	app.AddHandler("GET", fmt.Sprintf("%s/symbol", prefix), profSymbol).SetProcessNoTimeout()
	app.AddHandler("POST", fmt.Sprintf("%s/symbol", prefix), profSymbol).SetProcessNoTimeout()
	app.AddHandler("GET", fmt.Sprintf("%s/trace", prefix), profTrace).SetProcessNoTimeout()
}

func profIndex(_ context.Context, w http.ResponseWriter, r *http.Request) {
	pprof.Index(w, r)
}

func profCmdline(_ context.Context, w http.ResponseWriter, r *http.Request) {
	pprof.Cmdline(w, r)
}

func profProfile(_ context.Context, w http.ResponseWriter, r *http.Request) {
	pprof.Profile(w, r)
}

func profSymbol(_ context.Context, w http.ResponseWriter, r *http.Request) {
	pprof.Symbol(w, r)
}

func profTrace(_ context.Context, w http.ResponseWriter, r *http.Request) {
	pprof.Trace(w, r)
}

func AddMiscHandlersToMuxRouter(app *Application, root *mux.Router, enableProfiling bool) {
	adapterF := func(appHandleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			appHandleFunc(app.GetContext(), w, r)
		}
	}
	root.HandleFunc("/version", adapterF(VersionHandler))
	root.HandleFunc("/stats", adapterF(StatisticHandler))
	root.HandleFunc("/ping", adapterF(PingHandler))
	root.HandleFunc("/worker_stats", adapterF(WorkerStatsHandler))
	if enableProfiling {
		pp := "/debug/pprof"
		ppPath := func(sufix string) string {
			return fmt.Sprintf("%s/%s", pp, sufix)
		}
		root.HandleFunc(ppPath(""), pprof.Index)
		root.HandleFunc(ppPath("cmdline"), pprof.Cmdline)
		root.HandleFunc(ppPath("profile"), pprof.Profile)
		root.HandleFunc(ppPath("symbol"), pprof.Symbol)
		root.HandleFunc(ppPath("trace"), pprof.Trace)
	}
}
