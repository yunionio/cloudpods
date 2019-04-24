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

package app

import (
	"net"
	"os"
	"strconv"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func InitApp(options *common_options.CommonOptions, dbAccess bool) *appsrv.Application {
	// cache := appsrv.NewCache(options.AuthTokenCacheSize)
	app := appsrv.NewApplication(options.ApplicationID, options.RequestWorkerCount, dbAccess)
	app.CORSAllowHosts(options.CorsHosts)

	// app.SetContext(appsrv.APP_CONTEXT_KEY_CACHE, cache)
	// if dbConn != nil {
	//	app.SetContext(appsrv.APP_CONTEXT_KEY_DB, dbConn)
	//}
	return app
}

func ServeForever(app *appsrv.Application, options *common_options.CommonOptions) {
	ServeForeverWithCleanup(app, options, nil)
}

func ServeForeverWithCleanup(app *appsrv.Application, options *common_options.CommonOptions, onStop func()) {
	addr := net.JoinHostPort(options.Address, strconv.Itoa(options.Port))
	proto := "http"
	if options.EnableSsl {
		proto = "https"
	}
	log.Infof("Start listen on %s://%s", proto, addr)
	if options.EnableSsl {
		certfile := options.SslCertfile
		if len(options.SslCaCerts) > 0 {
			var err error
			certfile, err = seclib2.MergeCaCertFiles(options.SslCaCerts, options.SslCertfile)
			if err != nil {
				log.Fatalf("fail to merge ca+cert content: %s", err)
			}
			defer os.Remove(certfile)
		}
		if len(certfile) == 0 {
			log.Fatalf("Missing ssl-certfile")
		}
		if len(options.SslKeyfile) == 0 {
			log.Fatalf("Missing ssl-keyfile")
		}
		app.ListenAndServeTLSWithCleanup(addr, certfile, options.SslKeyfile, onStop)
	} else {
		app.ListenAndServeWithCleanup(addr, onStop)
	}
}
