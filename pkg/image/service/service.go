package service

import (
	"os"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	_ "yunion.io/x/onecloud/pkg/image/tasks"

	"time"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/image/torrent"
)

const (
	SERVICE_TYPE = "image"
)

func StartService() {
	cloudcommon.ParseOptions(&options.Options, os.Args, "glance-api.conf", SERVICE_TYPE)

	if options.Options.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", options.Options.PortV2)
		options.Options.Port = options.Options.PortV2
	}

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(&options.Options.Options, true)
	initHandlers(app)

	err := torrent.InitTorrentClient()
	if err != nil {
		log.Errorf("fail to initialize torrent client: %s", err)
		return
	}
	defer torrent.CloseTorrentClient()

	if !db.CheckSync(options.Options.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	models.SeedTorrents()

	cron := cronman.GetCronJobManager()
	cron.AddJob1("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)

	cron.Start()
	defer cron.Stop()

	cloudcommon.ServeForever(app, &options.Options.Options)
}
