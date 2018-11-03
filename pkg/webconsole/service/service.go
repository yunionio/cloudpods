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
	cloudcommon.ParseOptions(&o.Options, &o.Options.Options, os.Args, "webconsole.conf")

	if o.Options.ApiServer == "" {
		log.Fatalf("--api-server must specified")
	}
	_, err := url.Parse(o.Options.ApiServer)
	if err != nil {
		log.Fatalf("invalid --api-server %s", o.Options.ApiServer)
	}

	for _, binPath := range []string{o.Options.KubectlPath, o.Options.IpmitoolPath, o.Options.SshToolPath, o.Options.SshpassToolPath} {
		ensureBinExists(binPath)
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

	// websocket command text console handler
	root.Handle(webconsole.ConnectPathPrefix, server.NewConnectionServer())

	// websockify graphic console handler
	root.Handle(webconsole.WebsockifyPathPrefix, server.NewConnectionServer())

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
