package service

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/webconsole"
	"yunion.io/x/onecloud/pkg/webconsole/command"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/server"
)

func StartService() {
	cloudcommon.ParseOptions(&o.Options, &o.Options.Options, os.Args, "webconsole.conf")

	if o.Options.FrontendUrl == "" {
		log.Fatalf("--frontend-url must specified")
	}
	_, err := url.Parse(o.Options.FrontendUrl)
	if err != nil {
		log.Fatalf("invalid --frontend-url %s", o.Options.FrontendUrl)
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

	p1 := fmt.Sprintf("/%s/", command.PROTOCOL_TTY)
	// static file handler
	root.PathPrefix(p1).Handler(http.FileServer(http.Dir(o.Options.TtyStaticPath)))

	addr := net.JoinHostPort(o.Options.Address, strconv.Itoa(o.Options.Port))
	log.Infof("Start listen on %s", addr)
	err := http.ListenAndServe(addr, root)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
