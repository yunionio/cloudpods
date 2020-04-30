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
	"yunion.io/x/onecloud/pkg/hostman/guestman/guesthandlers"
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
	"yunion.io/x/onecloud/pkg/hostman/system_service"
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

	options.HostOptions.EnableRbac = false // disable rbac
	// init base option for pid file
	host.SServiceBase.O = &options.HostOptions.BaseOptions

	log.Infof("exec socket path: %s", options.HostOptions.ExecutorSocketPath)
	if options.HostOptions.EnableRemoteExecutor {
		execlient.Init(options.HostOptions.ExecutorSocketPath)
		procutils.SetRemoteExecutor()
	}

	system_service.Init()
}

func (host *SHostService) OnExitService() {}

func (host *SHostService) RunService() {
	app := app_common.InitApp(&options.HostOptions.BaseOptions, false)
	cronManager := cronman.InitCronJobManager(false, options.HostOptions.CronJobWorkerCount)
	hostutils.Init()

	hostInstance := hostinfo.Instance()
	if err := hostInstance.Init(); err != nil {
		log.Fatalf(err.Error())
	}

	deployclient.Init(options.HostOptions.DeployServerSocketPath)
	if err := storageman.Init(hostInstance); err != nil {
		log.Fatalf(err.Error())
	}

	guestman.Init(hostInstance, options.HostOptions.ServersPath)
	app_common.InitAuth(&options.HostOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")

		if err := host.initEtcdConfig(); err != nil {
			log.Fatalln(err)
		}

		hostInstance.StartRegister(2, func() {
			guestman.GetGuestManager().Bootstrap()
			// hostmetrics after guestmanager bootstrap
			hostmetrics.Init()
			hostmetrics.Start()
		})
	})

	<-hostinfo.Instance().IsRegistered // wait host and guest init
	host.initHandlers(app)

	// Init Metadata handler
	go metadata.StartService(
		app_common.InitApp(&options.HostOptions.BaseOptions, false),
		options.HostOptions.Address, options.HostOptions.Port+1000)

	cronManager.AddJobEveryFewDays(
		"CleanRecycleDiskFiles", 1, 3, 0, 0, storageman.CleanRecycleDiskfiles, false)
	cronManager.Start()

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
}

func (host *SHostService) initEtcdConfig() error {
	etcdEndpoint, err := app_common.FetchEtcdServiceInfo()
	if err != nil {
		return errors.Wrap(err, "fetch etcd service info")
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
