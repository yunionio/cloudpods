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

package hostman

import (
	"io/ioutil"
	"os"
	"path/filepath"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/downloader"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/guestman/guesthandlers"
	"yunion.io/x/onecloud/pkg/hostman/host_health"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hosthandler"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/hostmetrics"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/kubehandlers"
	"yunion.io/x/onecloud/pkg/hostman/metadata"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/hostman/storageman/diskhandlers"
	"yunion.io/x/onecloud/pkg/hostman/storageman/storagehandler"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SHostService struct {
	*service.SServiceBase
}

func (host *SHostService) InitService() {
	options.Init()
	isRoot := sysutils.IsRootPermission()
	if !isRoot {
		log.Fatalf("host service must running with root permissions")
	}

	if len(options.HostOptions.DeployServerSocketPath) == 0 {
		log.Fatalf("missing deploy server socket path")
	}

	// options.HostOptions.EnableRbac = false // disable rbac
	// init base option for pid file
	host.SServiceBase.O = &options.HostOptions.BaseOptions

	log.Infof("exec socket path: %s", options.HostOptions.ExecutorSocketPath)
	if options.HostOptions.EnableRemoteExecutor {
		execlient.Init(options.HostOptions.ExecutorSocketPath)
		procutils.SetRemoteExecutor()
	}
}

func (host *SHostService) OnExitService() {}

func (host *SHostService) RunService() {
	hn, err := os.Hostname()
	if err != nil {
		log.Fatalf("fail to get hostname %s", err)
	}

	app := app_common.InitApp(&options.HostOptions.BaseOptions, false)
	cronManager := cronman.InitCronJobManager(false, options.HostOptions.CronJobWorkerCount)
	hostutils.Init()

	app_common.InitAuth(&options.HostOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")

		if err := host.initEtcdConfig(); err != nil {
			log.Fatalln("Init etcd config:", err)
		}

		if len(options.HostOptions.EtcdEndpoints) > 0 {
			_, err := host_health.InitHostHealthManager(hn, "")
			if err != nil {
				log.Fatalf("Init host health manager failed %s", err)
			}
		}
	})

	hostInstance := hostinfo.Instance()
	if err := hostInstance.Init(); err != nil {
		log.Fatalf("Host instance init error: %v", err)
	}

	deployclient.Init(options.HostOptions.DeployServerSocketPath)
	if err := storageman.Init(hostInstance); err != nil {
		log.Fatalf("Storage manager init error: %v", err)
	}

	var guestChan chan struct{}
	guestman.Init(hostInstance, options.HostOptions.ServersPath)

	hostInstance.StartRegister(2, func() {
		guestChan = guestman.GetGuestManager().Bootstrap()
		// hostmetrics after guestmanager bootstrap
		hostmetrics.Init()
		hostmetrics.Start()
	})
	<-hostinfo.Instance().IsRegistered // wait host and guest init

	host.initHandlers(app)

	// Init Metadata handler
	go metadata.Start(
		app_common.InitApp(&options.HostOptions.BaseOptions, false),
		&metadata.Service{
			Address: options.HostOptions.Address,
			Port:    options.HostOptions.Port + 1000,
			DescGetter: metadata.DescGetterFunc(func(ip string) *desc.SGuestDesc {
				guestDesc, _ := guestman.GetGuestManager().GetGuestNicDesc("", ip, "", "", false)
				return guestDesc
			}),
		},
	)

	cronManager.AddJobEveryFewDays(
		"CleanRecycleDiskFiles", 1, 3, 0, 0, storageman.CleanRecycleDiskfiles, false)
	cronManager.Start()

	close(guestChan)
	app_common.ServeForeverWithCleanup(app, &options.HostOptions.BaseOptions, func() {
		hostinfo.Stop()
		storageman.Stop()
		hostmetrics.Stop()
		guestman.Stop()
		hostutils.GetWorkManager().Stop()
	})
}

func (host *SHostService) initHandlers(app *appsrv.Application) {
	guesthandlers.AddGuestTaskHandler("", app)
	storagehandler.AddStorageHandler("", app)
	diskhandlers.AddDiskHandler("", app)
	downloader.AddDownloadHandler("", app)
	kubehandlers.AddKubeAgentHandler("", app)
	hosthandler.AddHostHandler("", app)

	app_common.ExportOptionsHandler(app, &options.HostOptions)
}

func (host *SHostService) initEtcdConfig() error {
	etcdEndpoint, err := app_common.FetchEtcdServiceInfo()
	if err != nil {
		if errors.Cause(err) == httperrors.ErrNotFound {
			return nil
		}
		return errors.Wrap(err, "fetch etcd service info")
	}
	if etcdEndpoint == nil {
		return nil
	}
	if len(options.HostOptions.EtcdEndpoints) == 0 {
		options.HostOptions.EtcdEndpoints = []string{etcdEndpoint.Url}
		if len(etcdEndpoint.CertId) > 0 {
			dir, err := ioutil.TempDir("", "etcd-cluster-tls")
			if err != nil {
				return errors.Wrap(err, "create dir etcd cluster tls")
			}
			options.HostOptions.EtcdCert, err = writeFile(dir, "etcd.crt", []byte(etcdEndpoint.Certificate))
			if err != nil {
				return errors.Wrap(err, "write file certificate")
			}
			options.HostOptions.EtcdKey, err = writeFile(dir, "etcd.key", []byte(etcdEndpoint.PrivateKey))
			if err != nil {
				return errors.Wrap(err, "write file private key")
			}
			options.HostOptions.EtcdCacert, err = writeFile(dir, "etcd-ca.crt", []byte(etcdEndpoint.CaCertificate))
			if err != nil {
				return errors.Wrap(err, "write file  cacert")
			}
			options.HostOptions.EtcdUseTLS = true
		}
	}
	return nil
}

func writeFile(dir, file string, data []byte) (string, error) {
	p := filepath.Join(dir, file)
	return p, ioutil.WriteFile(p, data, 0600)
}
