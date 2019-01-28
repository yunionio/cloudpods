package service

import (
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	_ "yunion.io/x/onecloud/pkg/image/tasks"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/image/torrent"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	SERVICE_TYPE = "image"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	dbOpts := &opts.DBOptions
	cloudcommon.ParseOptions(opts, os.Args, "glance-api.conf", SERVICE_TYPE)

	if opts.PortV2 > 0 {
		log.Infof("Port V2 %d is specified, use v2 port", opts.PortV2)
		opts.Port = opts.PortV2
	}
	if len(opts.FilesystemStoreDatadir) == 0 {
		log.Errorf("missing FilesystemStoreDatadir")
		return
	}
	if !fileutils2.Exists(opts.FilesystemStoreDatadir) {
		err := os.MkdirAll(opts.FilesystemStoreDatadir, 0755)
		if err != nil {
			log.Errorf("fail to create %s: %s", opts.FilesystemStoreDatadir, err)
			return
		}
	}
	if len(opts.TorrentStoreDir) == 0 {
		opts.TorrentStoreDir = filepath.Join(filepath.Dir(opts.FilesystemStoreDatadir), "torrents")
		if !fileutils2.Exists(opts.TorrentStoreDir) {
			err := os.MkdirAll(opts.TorrentStoreDir, 0755)
			if err != nil {
				log.Errorf("fail to create %s: %s", opts.TorrentStoreDir, err)
				return
			}
		}
	}

	log.Infof("Target image formats %#v", opts.TargetImageFormats)

	cloudcommon.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	cloudcommon.InitDB(dbOpts)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(commonOpts, true)
	initHandlers(app)

	if opts.EnableTorrentService {
		err := torrent.InitTorrentClient()
		if err != nil {
			log.Errorf("fail to initialize torrent client: %s", err)
			return
		}
		torrent.InitTorrentHandler(app)
		defer torrent.CloseTorrentClient()
	}

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	models.CheckImages()

	cron := cronman.GetCronJobManager(true)
	cron.AddJob1("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)

	cron.Start()
	defer cron.Stop()

	cloudcommon.ServeForever(app, commonOpts)
}
