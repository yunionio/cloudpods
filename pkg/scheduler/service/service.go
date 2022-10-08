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
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/prometheus"
	"yunion.io/x/pkg/utils"

	compute_api "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	_ "yunion.io/x/onecloud/pkg/compute/guestdrivers"
	_ "yunion.io/x/onecloud/pkg/compute/hostdrivers"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	_ "yunion.io/x/onecloud/pkg/scheduler/algorithmprovider"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/schedtag"
	skuman "yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	schedhandler "yunion.io/x/onecloud/pkg/scheduler/handler"
	schedman "yunion.io/x/onecloud/pkg/scheduler/manager"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
	"yunion.io/x/onecloud/pkg/util/gin/middleware"
)

func StartService() error {
	o.Init()
	dbOpts := o.Options.DBOptions

	o.Options.Port = o.Options.SchedulerPort
	// init region compute models
	cloudcommon.InitDB(&dbOpts)
	defer cloudcommon.CloseDB()

	db.InitAllManagers()

	checkDBSyncRetries := 5
	count := 1
	for {
		if count == checkDBSyncRetries {
			log.Fatalf("database schema not in sync!!!")
		}
		if !db.CheckSync(false) {
			log.Errorf("database schema not in sync, wait region sync database")
			time.Sleep(2 * time.Second)
		} else {
			break
		}
		count++
	}

	commonOpts := &o.Options.CommonOptions
	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	common_options.StartOptionManager(&o.Options, o.Options.ConfigSyncPeriodSeconds, compute_api.SERVICE_TYPE, compute_api.SERVICE_VERSION, o.OnOptionsChange)

	// gin http framework mode configuration
	ginMode := "release"
	if o.Options.LogLevel == "debug" {
		ginMode = "debug"
	}
	gin.SetMode(ginMode)

	if err := computemodels.InitDB(); err != nil {
		log.Fatalf("InitDB fail: %s", err)
	}

	app := app_common.InitApp(&o.Options.BaseOptions, true)
	cloudcommon.AppDBInit(app)

	startSched := func() {
		stopEverything := make(chan struct{})
		go skuman.Start(utils.ToDuration(o.Options.SkuRefreshInterval))
		go schedtag.Start(context.Background(), utils.ToDuration("30s"))
		schedman.InitAndStart(stopEverything)
	}
	startSched()
	//InitHandlers(app)
	return startHTTP(&o.Options)
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

	if opt.EnableSsl {
		return server.ListenAndServeTLS(opt.SslCertfile, opt.SslKeyfile)
	} else {
		return server.ListenAndServe()
	}
}
