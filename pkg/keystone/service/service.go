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
