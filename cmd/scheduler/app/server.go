// Package app implements a Server object for running the scheduler.
package app

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/prometheus"
	"yunion.io/x/pkg/utils"
	"gopkg.in/gin-gonic/gin.v1"

	o "yunion.io/x/onecloud/cmd/scheduler/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	_ "yunion.io/x/onecloud/pkg/scheduler/algorithmprovider"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	schedhandler "yunion.io/x/onecloud/pkg/scheduler/handler"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
	"yunion.io/x/onecloud/pkg/util/gin/middleware"
)

type SchedulerServer struct {
	Address   string
	Port      int32
	SQLConn   string
	DBDialect string
	AuthInfo  *auth.AuthInfo
}

func NewServerFromConfig() *SchedulerServer {
	authURL := o.GetOptions().AuthURL
	adminUser := o.GetOptions().AdminUser
	adminPasswd := o.GetOptions().AdminPasswd
	adminTenant := o.GetOptions().AdminTenant
	a := auth.NewV2AuthInfo(authURL, adminUser, adminPasswd, adminTenant)

	s := &SchedulerServer{}
	s.Address = o.GetOptions().Address
	s.Port = int32(o.GetOptions().Port)
	sqlDialect, sqlConn, err := utils.TransSQLAchemyURL(o.GetOptions().SqlConnection)
	if err != nil {
		log.Fatalf("Backup python sql_connection config err: %v", err)
	}
	s.SQLConn = sqlConn
	s.DBDialect = sqlDialect
	s.AuthInfo = a

	return s
}

func Run(s *SchedulerServer) error {
	startSched := func() {
		err := models.Init(s.DBDialect, s.SQLConn)
		if err != nil {
			log.Fatalf("DB init error: %v, dialect: %s, url: %s", err, s.DBDialect, s.SQLConn)
		}

		stopEverything := make(chan struct{})
		schedman.InitAndStart(stopEverything)
	}

	debug := o.GetOptions().LogLevel == "debug"

	auth.AsyncInit(s.AuthInfo, debug, true, startSched)

	return startHTTP(s)
}

func startHTTP(s *SchedulerServer) error {
	gin.DefaultWriter = ioutil.Discard

	router := gin.Default()
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler)
	router.Use(middleware.KeystoneTokenVerifyMiddleware())

	prometheus.InstallHandler(router)
	schedhandler.InstallHandler(router)

	server := &http.Server{
		Addr:    net.JoinHostPort(s.Address, strconv.Itoa(int(s.Port))),
		Handler: router,
	}

	log.Infof("Start server on: %s:%d", s.Address, s.Port)
	return server.ListenAndServe()
}

func Execute() error {
	o.Parse()

	err := Run(NewServerFromConfig())
	if err != nil {
		err = fmt.Errorf("scheduler app failed to run: %v", err)
	}
	return err
}
