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

package host_health

import (
	"io/ioutil"
	"os"
	"path/filepath"

	execlient "yunion.io/x/executor/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type SHostHealthService struct {
	*service.SServiceBase
}

func (host *SHostHealthService) InitService() {
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
		execlient.SetTimeoutSeconds(options.HostOptions.ExecutorConnectTimeoutSeconds)
		procutils.SetRemoteExecutor()
	}
}

func (host *SHostHealthService) RunService() {
	hn, err := os.Hostname()
	if err != nil {
		log.Fatalf("fail to get hostname %s", err)
	}
	app_common.InitAuth(&options.HostOptions.CommonOptions, func() {
		log.Infof("Auth complete!!")

		if err := host.initEtcdConfig(); err != nil {
			log.Fatalln("Init etcd config:", err)
		}

		if len(options.HostOptions.EtcdEndpoints) > 0 {
			_, err := InitHostHealthManager(hn)
			if err != nil {
				log.Fatalf("Init host health manager failed %s", err)
			}
		}
	})
	select {}
}

func writeFile(dir, file string, data []byte) (string, error) {
	p := filepath.Join(dir, file)
	return p, ioutil.WriteFile(p, data, 0600)
}

func (host *SHostHealthService) initEtcdConfig() error {
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

func (host *SHostHealthService) OnExitService() {}

func StartService() {
	var srv = &SHostHealthService{}
	srv.SServiceBase = &service.SServiceBase{
		Service: srv,
	}
	srv.StartService()
}
