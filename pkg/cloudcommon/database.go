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
	"time"

	"github.com/mattn/go-sqlite3"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/informer"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/dbutils"
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

	if options.OpsLogMaxKeepMonths > 0 {
		consts.SetSplitableMaxKeepMonths(options.OpsLogMaxKeepMonths)
	}
	if options.SplitableMaxDurationHours > 0 {
		consts.SetSplitableMaxDurationHours(options.SplitableMaxDurationHours)
	}

	dialect, sqlStr, err := options.GetDBConnection()
	if err != nil {
		log.Fatalf("Invalid SqlConnection string: %s error: %v", options.SqlConnection, err)
	}
	backend := sqlchemy.MySQLBackend
	switch dialect {
	case "sqlite3":
		backend = sqlchemy.SQLiteBackend
		dialect = "sqlite3_with_extensions"
		sql.Register(dialect,
			&sqlite3.SQLiteDriver{
				Extensions: []string{
					"/opt/yunion/share/sqlite/inet",
				},
			},
		)
	case "clickhouse":
		log.Fatalf("cannot use clickhouse as primary database")
	}
	log.Infof("database dialect: %s sqlStr: %s", dialect, sqlStr)
	// save configuration to consts
	consts.SetDefaultDB(dialect, sqlStr)
	dbConn, err := sql.Open(dialect, sqlStr)
	if err != nil {
		panic(err)
	}
	sqlchemy.SetDBWithNameBackend(dbConn, sqlchemy.DefaultDB, backend)

	dialect, sqlStr, err = options.GetClickhouseConnStr()
	if err == nil {
		// connect to clickcloud
		// force convert sqlstr from clickhouse v2 to v1
		sqlStr, err = dbutils.ClickhouseSqlStrV2ToV1(sqlStr)
		if err != nil {
			log.Fatalf("fail to convert clickhouse sqlstr from v2 to v1: %s", err)
		}
		err = dbutils.ValidateClickhouseV1Str(sqlStr)
		if err != nil {
			log.Fatalf("invalid clickhouse sqlstr: %s", err)
		}
		click, err := sql.Open(dialect, sqlStr)
		if err != nil {
			panic(err)
		}
		sqlchemy.SetDBWithNameBackend(click, db.ClickhouseDB, sqlchemy.ClickhouseBackend)

		if options.OpsLogWithClickhouse {
			consts.OpsLogWithClickhouse = true
		}
	}

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

	initDBNotifier()
	startInitInformer(options)
}

func initDBNotifier() {
	db.SetChecksumTestFailedNotifier(func(obj *jsonutils.JSONDict) {
		notifyclient.SystemExceptionNotifyWithResult(context.TODO(), noapi.ActionChecksumTest, noapi.TOPIC_RESOURCE_DB_TABLE_RECORD, noapi.ResultFailed, obj)
	})
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
