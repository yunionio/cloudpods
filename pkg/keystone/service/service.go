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
	"os"
	"time"

	"yunion.io/x/jsonutils"
	_ "yunion.io/x/sqlchemy/backends"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	common_app "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/keystone/cronjobs"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	_ "yunion.io/x/onecloud/pkg/keystone/policy"
	"yunion.io/x/onecloud/pkg/keystone/saml"
	_ "yunion.io/x/onecloud/pkg/keystone/tasks"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/keystone/util"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func StartService() {
	auth.DefaultTokenVerifier = tokens.FernetTokenVerifier
	db.DefaultUUIDGenerator = keystoneUUIDGenerator
	db.DefaultProjectFetcher = keystoneProjectFetcher
	db.DefaultDomainFetcher = keystoneDomainFetcher
	db.DefaultUserFetcher = keystoneUserFetcher
	db.DefaultDomainQuery = keystoneDomainQuery
	db.DefaultProjectQuery = keystoneProjectQuery
	db.DefaultProjectsFetcher = keystoneProjectsFetcher
	policy.DefaultPolicyFetcher = localPolicyFetcher
	logclient.DefaultSessionGenerator = models.GetDefaultClientSession
	cronman.DefaultAdminSessionGenerator = tokens.GetDefaultAdminCredToken
	notifyclient.AdminSessionGenerator = util.GetDefaulAdminSession
	notifyclient.UserLangFetcher = models.GetUserLangForKeyStone

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

	app := common_app.InitApp(&opts.BaseOptions, true).
		OnException(func(method, path string, body jsonutils.JSONObject, err error) {
			ctx := context.Background()
			token := tokens.GetDefaultAdminCredToken()
			notifyclient.EventNotifyServiceAbnormal(ctx, token, consts.GetServiceType(), method, path, body, err)
		})

	cloudcommon.InitDB(&opts.DBOptions)

	InitHandlers(app)

	db.EnsureAppSyncDB(app, &opts.DBOptions, models.InitDB)

	common_app.InitBaseAuth(&opts.BaseOptions)

	common_options.StartOptionManagerWithSessionDriver(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, "", options.OnOptionsChange, models.NewServiceConfigSession())

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, opts.CronJobWorkerCount)

		cron.AddJobAtIntervalsWithStartRun("AutoSyncIdentityProviderTask", time.Duration(opts.AutoSyncIntervalSeconds)*time.Second, models.AutoSyncIdentityProviderTask, true)
		cron.AddJobAtIntervalsWithStartRun("FetchScopeResourceCount", time.Duration(opts.FetchScopeResourceCountIntervalSeconds)*time.Second, cronjobs.FetchScopeResourceCount, false)
		cron.AddJobAtIntervalsWithStartRun("CalculateIdentityQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.IdentityQuotaManager.CalculateQuotaUsages, true)

		cron.AddJobEveryFewHour("AutoPurgeSplitable", 4, 30, 0, db.AutoPurgeSplitable, false)
		cron.AddJobEveryFewDays("CheckAllUserPasswordIsExpired", 1, 8, 0, 0, models.CheckAllUserPasswordIsExpired, true)

		cron.Start()
		defer cron.Stop()
	}

	if options.Options.EnableSsl {
		// enable SAML support only if ssl is enabled
		err := saml.InitSAMLInstance()
		if err != nil {
			panic(err)
		}
	}

	go func() {
		common_app.ServeForeverExtended(app, &opts.BaseOptions, options.Options.AdminPort, nil, false)
	}()

	common_app.ServeForeverWithCleanup(app, &opts.BaseOptions, func() {
		cloudcommon.CloseDB()
		// cron.Stop()
	})
}
