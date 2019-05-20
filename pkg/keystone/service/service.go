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

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-plus/uuid"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/keystone/keys"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/keystone/tokens"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func keystoneUUIDGenerator() string {
	id, _ := uuid.NewV4()
	return id.Format(uuid.StyleWithoutDash)
}

func StartService() {
	auth.DefaultTokenVerifier = tokens.FernetTokenVerifier
	db.DefaultUUIDGenerator = keystoneUUIDGenerator
	policy.DefaultPolicyFetcher = localPolicyFetcher

	opts := &options.Options
	commonOpts := &opts.BaseOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "keystone.conf", api.SERVICE_TYPE)

	if opts.Port == 0 {
		opts.Port = 5000 // keystone well-known port
	}

	err := keys.Init(opts.TokenKeyRepository, opts.CredentialKeyRepository)
	if err != nil {
		log.Fatalf("init fernet keys fail %s", err)
	}

	app := app_common.InitApp(commonOpts, true)
	initHandlers(app)

	cloudcommon.InitDB(dbOpts)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	app_common.InitBaseAuth(commonOpts)
	// register bootstrap default policy
	defaultAdminPolicy := rbacutils.SRbacPolicy{
		IsAdmin:  true,
		Projects: []string{options.Options.AdminProjectName},
		Roles:    []string{options.Options.AdminRoleName},
		Rules: []rbacutils.SRbacRule{
			{
				Service:  api.SERVICE_TYPE,
				Resource: "policies",
				Action:   policy.PolicyActionCreate,
				Result:   rbacutils.AdminAllow,
			},
			{
				Service:  api.SERVICE_TYPE,
				Resource: "policies",
				Action:   policy.PolicyActionList,
				Result:   rbacutils.AdminAllow,
			},
			{
				Service:  api.SERVICE_TYPE,
				Resource: "policies",
				Action:   policy.PolicyActionUpdate,
				Result:   rbacutils.AdminAllow,
			},
			{
				Service:  api.SERVICE_TYPE,
				Resource: "policies",
				Action:   policy.PolicyActionGet,
				Result:   rbacutils.AdminAllow,
			},
		},
	}
	policy.PolicyManager.RegisterDefaultAdminPolicy(&defaultAdminPolicy)

	// cron := cronman.GetCronJobManager(true)
	// cron.AddJob1("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)

	// cron.Start()

	cloudcommon.AppDBInit(app)

	go func() {
		app_common.ServeForeverExtended(app, commonOpts, options.Options.AdminPort, nil, false)
	}()

	app_common.ServeForeverWithCleanup(app, commonOpts, func() {
		cloudcommon.CloseDB()
		// cron.Stop()
	})
}
