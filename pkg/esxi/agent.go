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

package esxi

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/agent"
	"yunion.io/x/onecloud/pkg/cloudcommon/workmanager"
	"yunion.io/x/onecloud/pkg/esxi/options"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

const (
	_ = compute.AgentTypeEsxi
)

var (
	EsxiAgent *SEsxiAgent
)

type SEsxiAgent struct {
	agent.SBaseAgent

	agentImageCache *storageman.SAgentImageCacheManager
	AgentStorage    *storageman.SAgentStorage
	ListenNic       netutils2.SNetInterface
}

func NewEsxiAgent() (*SEsxiAgent, error) {
	agent := &SEsxiAgent{}
	err := agent.Init(agent, options.Options.ListenInterface, options.Options.ImageCachePath)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (ea *SEsxiAgent) GetAgentType() string {
	return string(compute.AgentTypeEsxi)
}

func (ea *SEsxiAgent) GetAccessIP() (net.IP, error) {
	return ea.GetListenIP()
}

func (ea *SEsxiAgent) GetListenIP() (net.IP, error) {
	return ea.FindListenIP(options.Options.ListenAddress)
}

func (ea *SEsxiAgent) GetPort() int {
	return options.Options.Port
}

func (ea *SEsxiAgent) GetEnableSsl() bool {
	return options.Options.EnableSsl
}

func (ea *SEsxiAgent) GetZoneName() string {
	return options.Options.Zone
}

func (ea *SEsxiAgent) GetAdminSession() *mcclient.ClientSession {
	return auth.GetAdminSession(context.TODO(), options.Options.Region, "v2")
}

func (ea *SEsxiAgent) TuneSystem() error {
	return nil
}

func (ea *SEsxiAgent) StartService() error {

	ea.DoOnline(ea.GetAdminSession())
	return nil
}

func (ea *SEsxiAgent) StopService() error {
	return nil
}

func (ea *SEsxiAgent) Start() error {
	err := ea.SBaseAgent.Start()
	if err != nil {
		return err
	}
	// add agent image cache
	ea.agentImageCache = storageman.NewAgentImageCacheManager(ea.CacheManager)
	ea.AgentStorage = storageman.NewAgentStorage(&storageman.SStorageManager{LocalStorageImagecacheManager: ea.CacheManager},
		ea, options.Options.AgentTempPath)
	return nil
}

func Start(app *appsrv.Application) error {
	var err error
	if EsxiAgent != nil {
		log.Warningf("Global EsxiAgent already start")
		return nil
	}
	EsxiAgent, err = NewEsxiAgent()
	if err != nil {
		return err
	}
	err = EsxiAgent.Start()
	if err != nil {
		return err
	}
	EsxiAgent.AddImageCacheHandler("", app)
	return nil
}

func (agent *SEsxiAgent) AddImageCacheHandler(prefix string, app *appsrv.Application) {
	hostutils.InitWorkerManager()
	app.AddHandler("POST",
		fmt.Sprintf("%s/disks/image_cache", prefix),
		auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			performImageCache(ctx, w, r, agent.agentImageCache.PrefetchImageCache)
		}))
	app.AddHandler("DELETE",
		fmt.Sprintf("%s/disks/image_cache", prefix),
		auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			performImageCache(ctx, w, r, agent.agentImageCache.DeleteImageCache)
		}))
}

func performImageCache(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	performTask workmanager.DelayTaskFunc,
) {
	_, _, body := appsrv.FetchEnv(ctx, w, r)

	disk, err := body.Get("disk")
	if err != nil {
		httperrors.MissingParameterError(w, "disk")
		return
	}

	hostutils.DelayTask(ctx, performTask, disk)
	hostutils.ResponseOk(ctx, w)
}

func Stop() error {
	if EsxiAgent != nil {
		log.Infof("EsxiAgent stop...")
		tmpAgent := EsxiAgent
		EsxiAgent = nil
		tmpAgent.Stop()
	}
	return nil
}
