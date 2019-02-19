package service

import (
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"

	"net/http"
	"yunion.io/x/onecloud/pkg/cloudcommon"
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
	cloudcommon.ParseOptions(opts, os.Args, "webconsole.conf", "webconsole")

	if opts.ApiServer == "" {
		log.Fatalf("--api-server must specified")
	}
	_, err := url.Parse(opts.ApiServer)
	if err != nil {
		log.Fatalf("invalid --api-server %s", opts.ApiServer)
	}

	for _, binPath := range []string{opts.KubectlPath, opts.IpmitoolPath, opts.SshToolPath, opts.SshpassToolPath} {
		ensureBinExists(binPath)
	}

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete")
	})
	start()
}

func start() {
	commonOpts := &o.Options.CommonOptions
	app := cloudcommon.InitApp(commonOpts, false)
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
