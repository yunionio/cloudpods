package cloudcommon

import (
	"net"
	"strconv"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
)

func InitApp(options *Options) *appsrv.Application {
	// cache := appsrv.NewCache(options.AuthTokenCacheSize)
	app := appsrv.NewApplication(options.ApplicationID, options.RequestWorkerCount)
	app.CORSAllowHosts(options.CorsHosts)

	// app.SetContext(appsrv.APP_CONTEXT_KEY_CACHE, cache)
	// if dbConn != nil {
	//	app.SetContext(appsrv.APP_CONTEXT_KEY_DB, dbConn)
	//}
	return app
}

func ServeForever(app *appsrv.Application, options *Options) {
	addr := net.JoinHostPort(options.Address, strconv.Itoa(options.Port))
	proto := "http"
	if options.EnableSsl {
		proto = "https"
	}
	log.Infof("Start listen on %s://%s", proto, addr)
	if options.EnableSsl {
		app.ListenAndServeTLS(addr, options.SslCertfile, options.SslKeyfile)
	} else {
		app.ListenAndServe(addr)
	}
}
