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

package cloudcommon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/informer"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

const (
	MIN_DB_CONN_MAX = 5
)

func InitDB(options *common_options.DBOptions) {
	if options.DebugSqlchemy {
		log.Warningf("debug Sqlchemy is turned on")
		sqlchemy.DEBUG_SQLCHEMY = true
	}

	consts.QueryOffsetOptimization = options.QueryOffsetOptimization

	if options.HistoricalUniqueName {
		consts.EnableHistoricalUniqueName()
	} else {
		consts.DisableHistoricalUniqueName()
	}

	dialect, sqlStr, err := options.GetDBConnection()
	if err != nil {
		log.Fatalf("Invalid SqlConnection string: %s error: %v", options.SqlConnection, err)
	}
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		panic(err)
	}
	sqlchemy.SetDB(dbConn)

	switch options.LockmanMethod {
	case common_options.LockMethodInMemory, "":
		log.Infof("using inmemory lockman")
		lm := lockman.NewInMemoryLockManager()
		lockman.Init(lm)
	case common_options.LockMethodEtcd:
		log.Infof("using etcd lockman")
		tlsCfg, err := options.GetEtcdTLSConfig()
		if err != nil {
			log.Fatalln(err.Error())
		}
		lm, err := lockman.NewEtcdLockManager(&lockman.SEtcdLockManagerConfig{
			Endpoints:  options.EtcdEndpoints,
			Username:   options.EtcdUsername,
			Password:   options.EtcdPassword,
			LockTTL:    options.EtcdLockTTL,
			LockPrefix: options.EtcdLockPrefix,
			TLS:        tlsCfg,
		})
		if err != nil {
			log.Fatalf("etcd lockman: %v", err)
		}
		lockman.Init(lm)
	}
	// lm := lockman.NewNoopLockManager()

	startInitInformer(options)
}

// startInitInformer starts goroutine init informer backend
func startInitInformer(options *common_options.DBOptions) {
	go func() {
		if len(options.EtcdEndpoints) == 0 {
			return
		}
		for {
			log.Infof("using etcd as resource informer backend")
			if err := initInformer(options); err != nil {
				log.Errorf("Init informer error: %v", err)
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
	}()
}

func initInformer(options *common_options.DBOptions) error {
	tlsCfg, err := options.GetEtcdTLSConfig()
	if err != nil {
		return errors.Wrap(err, "get etcd informer backend tls config")
	}
	informerBackend, err := informer.NewEtcdBackend(&etcd.SEtcdOptions{
		EtcdEndpoint:              options.EtcdEndpoints,
		EtcdTimeoutSeconds:        5,
		EtcdRequestTimeoutSeconds: 2,
		EtcdLeaseExpireSeconds:    5,
		EtcdEnabldSsl:             options.EtcdUseTLS,
		TLSConfig:                 tlsCfg,
	}, nil)
	if err != nil {
		return errors.Wrap(err, "new etcd informer backend")
	}
	informer.Init(informerBackend)
	return nil
}

func CloseDB() {
	sqlchemy.CloseDB()
}

func AppDBInit(app *appsrv.Application) {
	dbConn := sqlchemy.GetDB()
	if dbConn != nil {
		connMax := appsrv.GetDBConnectionCount()
		if connMax < MIN_DB_CONN_MAX {
			connMax = MIN_DB_CONN_MAX
		}
		log.Infof("Total %d db workers, set db connection max", connMax)
		dbConn.SetMaxIdleConns(connMax)
		dbConn.SetMaxOpenConns(connMax*2 + 1)
	}

	app.AddDefaultHandler("GET", "/db_stats", DBStatsHandler, "db_stats")
}

func DBStatsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	result := jsonutils.NewDict()
	dbConn := sqlchemy.GetDB()
	if dbConn != nil {
		stats := dbConn.Stats()
		result.Add(jsonutils.Marshal(&stats), "db_stats")
	}
	fmt.Fprintf(w, result.String())
}
