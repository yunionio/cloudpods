package service

import (
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"gopkg.in/gin-gonic/gin.v1"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/prometheus"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	schedhandler "yunion.io/x/onecloud/pkg/scheduler/handler"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
	"yunion.io/x/onecloud/pkg/util/gin/middleware"

	_ "yunion.io/x/onecloud/pkg/scheduler/algorithmprovider"
)

func StartService() error {
	o.Init()
	opts := o.GetOptions()
	dbOpts := &opts.DBOptions

	// gin http framework mode configuration
	gin.SetMode(opts.GinMode)

	startSched := func() {
		sqlDialect, sqlConn, err := utils.TransSQLAchemyURL(opts.SqlConnection)
		if err != nil {
			log.Fatalf("Invalid SqlConnection: %v", err)
		}
		if err := models.Init(sqlDialect, sqlConn); err != nil {
			log.Fatalf("DB init error: %v, dialect: %s, url: %s", err, sqlDialect, sqlConn)
		}

		stopEverything := make(chan struct{})
		schedman.InitAndStart(stopEverything)
	}

	opts.Port = opts.SchedulerPort
	// init region compute models
	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	db.InitAllManagers()

	if err := computemodels.InitDB(); err != nil {
		log.Fatalf("InitDB fail: %s", err)
	}

	commonOpts := &opts.CommonOptions
	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
		startSched()
	})

	//app := cloudcommon.InitApp(commonOpts, true)

	//InitHandlers(app)
	return startHTTP(opts)
}

func startHTTP(opt *o.SchedulerOptions) error {
	gin.DefaultWriter = ioutil.Discard

	router := gin.Default()
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler)
	router.Use(middleware.KeystoneTokenVerifyMiddleware())

	prometheus.InstallHandler(router)
	schedhandler.InstallHandler(router)

	server := &http.Server{
		Addr:    net.JoinHostPort(opt.Address, strconv.Itoa(int(opt.Port))),
		Handler: router,
	}

	log.Infof("Start server on: %s:%d", opt.Address, opt.Port)

	if o.GetOptions().EnableSsl {
		return server.ListenAndServeTLS(o.GetOptions().SslCertfile,
			o.GetOptions().SslKeyfile)
	} else {
		return server.ListenAndServe()
	}
}
