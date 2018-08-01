package cloudcommon

import (
	"net"
	"strconv"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/appsrv"
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
	log.Infof("Start listen on %s", addr)
	app.ListenAndServe(addr)
}
