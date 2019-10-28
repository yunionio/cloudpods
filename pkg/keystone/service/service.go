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
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-plus/uuid"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy" // "yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/keystone/cronjobs"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/cas"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/ldap"
	_ "yunion.io/x/onecloud/pkg/keystone/driver/sql"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	_ "yunion.io/x/onecloud/pkg/keystone/tasks"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func keystoneUUIDGenerator() string {
	id, _ := uuid.NewV4()
	return id.Format(uuid.StyleWithoutDash)
}

func StartService() {
	auth.DefaultTokenVerifier = tokens.FernetTokenVerifier
	db.DefaultUUIDGenerator = keystoneUUIDGenerator
	policy.DefaultPolicyFetcher = localPolicyFetcher
	logclient.DefaultSessionGenerator = models.GetDefaultClientSession
	cronman.DefaultAdminSessionGenerator = models.GetDefaultAdminCred

	models.InitSyncWorkers()

	opts := &options.Options
	common_options.ParseOptions(opts, os.Args, "keystone.conf", api.SERVICE_TYPE)

	if opts.Port == 0 {
		opts.Port = 5000 // keystone well-known port
	}

	/* err := keys.Init(opts.FernetKeyRepository, opts.SetupCredentialKey)
	if err != nil {
		log.Fatalf("init fernet keys fail %s", err)
	}
	*/

	app := app_common.InitApp(&opts.BaseOptions, true)
	initHandlers(app)

	db.EnsureAppInitSyncDB(app, &opts.DBOptions, models.InitDB)

	app_common.InitBaseAuth(&opts.BaseOptions)

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, opts.CronJobWorkerCount)

		cron.AddJobAtIntervalsWithStartRun("AutoSyncIdentityProviderTask", time.Duration(opts.AutoSyncIntervalSeconds)*time.Second, models.AutoSyncIdentityProviderTask, true)
		cron.AddJobAtIntervals("FetchProjectResourceCount", time.Duration(opts.FetchProjectResourceCountIntervalSeconds)*time.Second, cronjobs.FetchProjectResourceCount)

		cron.Start()
		defer cron.Stop()
	}

	go func() {
		app_common.ServeForeverExtended(app, &opts.BaseOptions, options.Options.AdminPort, nil, false)
	}()

	app_common.ServeForeverWithCleanup(app, &opts.BaseOptions, func() {
		cloudcommon.CloseDB()
		// cron.Stop()
	})
}
