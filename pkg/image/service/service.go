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
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/image/drivers/s3"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
	_ "yunion.io/x/onecloud/pkg/image/policy"
	_ "yunion.io/x/onecloud/pkg/image/tasks"
	"yunion.io/x/onecloud/pkg/image/torrent"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func StartService() {
	opts := &options.Options
	commonOpts := &opts.CommonOptions
	baseOpts := &opts.BaseOptions
	dbOpts := &opts.DBOptions
	common_options.ParseOptions(opts, os.Args, "glance-api.conf", api.SERVICE_TYPE)

	// no need to run glance as root any more
	// isRoot := sysutils.IsRootPermission()
	// if !isRoot {
	// 	log.Fatalf("glance service must running with root permissions")
	// }

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

	app := app_common.InitApp(baseOpts, true)
	InitHandlers(app)

	db.EnsureAppInitSyncDB(app, dbOpts, models.InitDB)

	common_options.StartOptionManager(opts, opts.ConfigSyncPeriodSeconds, api.SERVICE_TYPE, api.SERVICE_VERSION, options.OnOptionsChange)

	go models.CheckImages()
	models.Init(options.Options.StorageDriver)
	if options.Options.StorageDriver == api.IMAGE_STORAGE_DRIVER_S3 {
		initS3()
	}

	if len(options.Options.DeployServerSocketPath) > 0 {
		log.Infof("deploy server socket path: %s", options.Options.DeployServerSocketPath)
		deployclient.Init(options.Options.DeployServerSocketPath)
	}

	if !opts.IsSlaveNode {
		cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
		cron.AddJobAtIntervals("CleanPendingDeleteImages", time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.ImageManager.CleanPendingDeleteImages)
		cron.AddJobAtIntervals("CalculateQuotaUsages", time.Duration(opts.CalculateQuotaUsageIntervalSeconds)*time.Second, models.QuotaManager.CalculateQuotaUsages)
		cron.AddJobAtIntervals("CleanPendingDeleteGuestImages",
			time.Duration(options.Options.PendingDeleteCheckSeconds)*time.Second, models.GuestImageManager.CleanPendingDeleteImages)

		cron.Start()
	}

	app_common.ServeForeverWithCleanup(app, baseOpts, func() {
		cloudcommon.CloseDB()

		// cron.Stop()

		if options.Options.EnableTorrentService {
			torrent.StopTorrents()
		}
		if options.Options.StorageDriver == api.IMAGE_STORAGE_DRIVER_S3 {
			procutils.NewCommand("umount", options.Options.S3MountPoint).Run()
		}
	})
}

func initS3() {
	err := s3.Init(
		options.Options.S3Endpoint,
		options.Options.S3AccessKey,
		options.Options.S3SecretKey,
		options.Options.S3BucketName,
		options.Options.S3UseSSL,
	)
	if err != nil {
		log.Fatalf("failed init s3 client %s", err)
	}
	func() {
		fd, err := os.OpenFile("/tmp/s3-pass", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalf("failed open s3 pass file %s", err)
		}
		defer fd.Close()
		_, err = fd.WriteString(fmt.Sprintf("%s:%s", options.Options.S3AccessKey, options.Options.S3SecretKey))
		if err != nil {
			log.Fatalf("failed write s3 pass file")
		}
	}()
	if !fileutils2.Exists(options.Options.S3MountPoint) {
		err := os.MkdirAll(options.Options.S3MountPoint, 0755)
		if err != nil {
			log.Fatalf("fail to create %s: %s", options.Options.S3MountPoint, err)
		}
	}

	prefix := "http://"
	if options.Options.S3UseSSL {
		prefix = "https://"
	}
	url := prefix + options.Options.S3Endpoint
	out, err := procutils.NewCommand("s3fs",
		options.Options.S3BucketName, options.Options.S3MountPoint,
		"-o", fmt.Sprintf("passwd_file=/tmp/s3-pass,use_path_request_style,url=%s", url)).Output()
	if err != nil {
		log.Fatalf("failed mount s3fs %s %s", err, out)
	}
}
