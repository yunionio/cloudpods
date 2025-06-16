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
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func InitApp(options *common_options.BaseOptions, dbAccess bool) *appsrv.Application {
	// cache := appsrv.NewCache(options.AuthTokenCacheSize)
	log.Infof("RequestWorkerCount: %d", options.RequestWorkerCount)
	app := appsrv.NewApplication(options.ApplicationID, options.RequestWorkerCount, options.RequestWorkerQueueSize, dbAccess)
	app.CORSAllowHosts(options.CorsHosts)
	app.SetDefaultTimeout(time.Duration(options.DefaultProcessTimeoutSeconds) * time.Second)
	// app.SetContext(appsrv.APP_CONTEXT_KEY_CACHE, cache)
	// if dbConn != nil {
	//	app.SetContext(appsrv.APP_CONTEXT_KEY_DB, dbConn)
	//}
	if options.EnableAppProfiling {
		app.EnableProfiling()
	}
	if options.AllowTLS1x {
		app.AllowTLS1x()
	}
	return app
}

func ServeForever(app *appsrv.Application, options *common_options.BaseOptions) {
	ServeForeverWithCleanup(app, options, nil)
}

func ServeForeverWithCleanup(app *appsrv.Application, options *common_options.BaseOptions, onStop func()) {
	ServeForeverExtended(app, options, options.Port, onStop, true)
}

func ServeForeverExtended(app *appsrv.Application, options *common_options.BaseOptions, port int, onStop func(), isMaster bool) {
	addr := net.JoinHostPort(options.Address, strconv.Itoa(port))
	proto := "http"
	if options.EnableSsl {
		proto = "https"
	}
	log.Infof("Start listen on %s://%s, isMaster: %v", proto, addr, isMaster)
	var certfile string
	var sslfile string
	if options.EnableSsl {
		certfile = options.SslCertfile
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
		sslfile = options.SslKeyfile
	}
	app.ListenAndServeTLSWithCleanup2(addr, certfile, sslfile, onStop, isMaster)
}
