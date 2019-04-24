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
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	_ "yunion.io/x/onecloud/pkg/image/tasks"
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
	common_options.ParseOptions(opts, os.Args, "glance-api.conf", SERVICE_TYPE)

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

	app_common.InitAuth(commonOpts, func() {
		log.Infof("Auth complete!!")
	})

	trackers := torrent.GetTrackers()
	if len(trackers) == 0 {
		log.Errorf("no valid torrent-tracker")
		return
	}

	cloudcommon.InitDB(dbOpts)

	app := app_common.InitApp(commonOpts, true)
	initHandlers(app)

	if !db.CheckSync(opts.AutoSyncTable) {
		log.Fatalf("database schema not in sync!")
	}

	models.InitDB()

	go models.CheckImages()

	cron := cronman.GetCronJobManager(true)
	cron.AddJob1("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)

	cron.Start()

	cloudcommon.AppDBInit(app)
	app_common.ServeForeverWithCleanup(app, commonOpts, func() {
		cloudcommon.CloseDB()

		cron.Stop()

		if options.Options.EnableTorrentService {
			torrent.StopTorrents()
		}
	})
}
