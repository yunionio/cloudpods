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
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/webconsole"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/server"
)

func ensureBinExists(binPath string) {
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		log.Fatalf("Binary %s not exists", binPath)
	}
}

func StartService() {

	opts := &o.Options
	commonOpts := &o.Options.CommonOptions
	common_options.ParseOptions(opts, os.Args, "webconsole.conf", "webconsole")

	if opts.ApiServer == "" {
		log.Fatalf("--api-server must specified")
	}
	_, err := url.Parse(opts.ApiServer)
	if err != nil {
		log.Fatalf("invalid --api-server %s", opts.ApiServer)
	}

	for _, binPath := range []string{opts.IpmitoolPath, opts.SshToolPath, opts.SshpassToolPath} {
		ensureBinExists(binPath)
	}

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})
	start()
}

func start() {
	baseOpts := &o.Options.BaseOptions
	// commonOpts := &o.Options.CommonOptions
	app := app_common.InitApp(baseOpts, false)
	webconsole.InitHandlers(app)

	root := mux.NewRouter()
	root.UseEncodedPath()

	// api handler
	root.PathPrefix(webconsole.ApiPathPrefix).Handler(app)

	srv := server.NewConnectionServer()
	// websocket command text console handler
	root.Handle(webconsole.ConnectPathPrefix, srv)

	// websockify graphic console handler
	root.Handle(webconsole.WebsockifyPathPrefix, srv)

	// websocketproxy handler
	root.Handle(webconsole.WebsocketProxyPathPrefix, srv)

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
