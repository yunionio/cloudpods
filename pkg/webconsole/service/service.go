package service

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/webconsole"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/server"
)

func StartService() {
	cloudcommon.ParseOptions(&o.Options, &o.Options.Options, os.Args, "webconsole.conf")

	if o.Options.ApiServer == "" {
		log.Fatalf("--api-server must specified")
	}
	_, err := url.Parse(o.Options.ApiServer)
	if err != nil {
		log.Fatalf("invalid --api-server %s", o.Options.ApiServer)
	}

	cloudcommon.InitAuth(&o.Options.Options, func() {
		log.Infof("Auth complete")
	})
	start()
}

func start() {
	app := cloudcommon.InitApp(&o.Options.Options)
	webconsole.InitHandlers(app)

	root := mux.NewRouter()
	root.UseEncodedPath()

	// api handler
	root.PathPrefix(webconsole.ApiPathPrefix).Handler(app)

	// websocket related console handler
	root.Handle(webconsole.ConnectPathPrefix, server.NewConnectionServer())

	addr := net.JoinHostPort(o.Options.Address, strconv.Itoa(o.Options.Port))
	log.Infof("Start listen on %s", addr)
	err := http.ListenAndServe(addr, root)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
