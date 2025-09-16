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

package worker

import (
	"context"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/vishvananda/netlink"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apihelper"
	api "yunion.io/x/onecloud/pkg/apis/cloudproxy"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	agentmodels "yunion.io/x/onecloud/pkg/cloudproxy/agent/models"
	agentoptions "yunion.io/x/onecloud/pkg/cloudproxy/agent/options"
	agentssh "yunion.io/x/onecloud/pkg/cloudproxy/agent/ssh"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	cloudproxy_modules "yunion.io/x/onecloud/pkg/mcclient/modules/cloudproxy"
	"yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	ssh_util "yunion.io/x/onecloud/pkg/util/ssh"
)

type Worker struct {
	commonOpts *common_options.CommonOptions
	opts       *agentoptions.Options

	proxyAgentId string
	bindAddr     string

	apih         *apihelper.APIHelper
	clientSet    *agentssh.ClientSet
	sessionCache *auth.SessionCache
}

func NewWorker(commonOpts *common_options.CommonOptions, opts *agentoptions.Options) *Worker {
	modelSets := agentmodels.NewModelSets()
	apiOpts := &apihelper.Options{
		CommonOptions:       *commonOpts,
		SyncIntervalSeconds: opts.APISyncIntervalSeconds,
		ListBatchSize:       opts.APIListBatchSize,
	}
	apih, err := apihelper.NewAPIHelper(apiOpts, modelSets)
	if err != nil {
		return nil
	}
	w := &Worker{
		commonOpts:   commonOpts,
		opts:         opts,
		proxyAgentId: opts.ProxyAgentId,

		apih:      apih,
		clientSet: agentssh.NewClientSet(),
		sessionCache: &auth.SessionCache{
			Region:        commonOpts.Region,
			UseAdminToken: true,
			EarlyRefresh:  time.Hour,
		},
	}
	return w
}

func (w *Worker) initProxyAgent_(ctx context.Context) error {
	s := w.sessionCache.Get(ctx)

	var agentDetail api.ProxyAgentDetails
	{
		j, err := cloudproxy_modules.ProxyAgents.Get(s, w.proxyAgentId, nil)
		if err != nil {
			return errors.Wrapf(err, "fetch proxy agent %s", w.proxyAgentId)
		}
		if err := j.Unmarshal(&agentDetail); err != nil {
			return errors.Wrapf(err, "unmarshal proxy agent detail: %s", j.String())
		}
		if agentDetail.Id == "" {
			return errors.Error("proxy agent id is empty")
		}
		w.proxyAgentId = agentDetail.Id
	}

	bindAddrExist := func(addr string) bool {
		as, err := netlink.AddrList(nil, netlink.FAMILY_ALL)
		if err != nil {
			log.Fatalf("list system available addresses: %v", err)
		}
		for _, a := range as {
			ipstr := a.IPNet.IP.String()
			if addr == ipstr {
				return true
			}
		}
		return false
	}

	var (
		bindAddr            string
		advertiseAddr       string
		bindAddrUpdate      = false
		advertiseAddrUpdate = false
	)
	if agentDetail.BindAddr == "" || !bindAddrExist(bindAddr) {
		var err error
		bindAddr, err = netutils2.MyIP()
		if err != nil {
			return errors.Wrap(err, "find bind Addr")
		}
		bindAddrUpdate = true
	} else {
		bindAddr = agentDetail.BindAddr
	}
	w.bindAddr = bindAddr

	if agentDetail.AdvertiseAddr == "" || (bindAddrUpdate && agentDetail.AdvertiseAddr == agentDetail.BindAddr) {
		advertiseAddr = bindAddr
		advertiseAddrUpdate = true
	} else {
		advertiseAddr = agentDetail.AdvertiseAddr
	}

	if bindAddrUpdate || advertiseAddrUpdate {
		req := api.ProxyAgentUpdateInput{
			BindAddr:      bindAddr,
			AdvertiseAddr: advertiseAddr,
		}
		reqJ := req.JSON(req)
		if _, err := cloudproxy_modules.ProxyAgents.Put(s, w.proxyAgentId, reqJ); err != nil {
			return errors.Wrapf(err, "update proxy agent addr: %s", reqJ.String())
		}
	}

	return nil
}

func (w *Worker) initProxyAgent(ctx context.Context) error {
	done, err := utils.NewFibonacciRetrierMaxElapse(
		w.opts.GetProxyAgentInitWaitDuration(),
		func(retrier utils.FibonacciRetrier) (bool, error) {
			err := w.initProxyAgent_(ctx)
			if err != nil {
				return false, err
			}
			return true, nil
		}).Start(ctx)
	if done {
		return nil
	}
	return err
}

func (w *Worker) Start(ctx context.Context) {
	wg := ctx.Value("wg").(*sync.WaitGroup)
	wg.Add(1)
	defer func() {
		log.Infoln("agent: worker bye")
		wg.Done()
	}()

	if err := w.initProxyAgent(ctx); err != nil {
		log.Errorf("init proxy agent: %v", err)
		return
	}
	go w.apih.Start(ctx, nil, "")

	const tickDur = 11 * time.Second
	var (
		mss  *agentmodels.ModelSets
		tick = time.NewTicker(tickDur)
	)
	for {
		select {
		case imss := <-w.apih.ModelSets():
			log.Infof("agent: got new data from api helper")
			mss = imss.(*agentmodels.ModelSets)
			if err := w.run(ctx, mss); err != nil {
				log.Errorf("agent run: %v", err)
			}
		case <-tick.C:
			if mss != nil {
				if err := w.run(ctx, mss); err != nil {
					log.Errorf("agent refresh run: %v", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) run(ctx context.Context, mss *agentmodels.ModelSets) (err error) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			yunionconf.BugReport.SendBugReport(context.Background(), version.GetShortString(), string(debug.Stack()), errors.Errorf("%s", panicVal))
			if panicErr, ok := panicVal.(runtime.Error); ok {
				err = errors.Wrap(panicErr, string(debug.Stack()))
			} else if panicErr, ok := panicVal.(error); ok {
				err = panicErr
			} else {
				panic(panicVal)
			}
		}
	}()

	w.clientSet.ClearAllMark()
	for _, pep := range mss.ProxyEndpoints {
		cc := ssh_util.ClientConfig{
			Username:   pep.User,
			Host:       pep.Host,
			Port:       pep.Port,
			PrivateKey: pep.PrivateKey,
		}
		if reset := w.clientSet.ResetIfChanged(ctx, pep.Id, cc); reset {
			log.Warningf("proxy endpoint %s changed, connections reset", pep.Id)
		} else if added := w.clientSet.AddIfNotExist(ctx, pep.Id, cc); added {
			log.Infof("proxy endpoint %s added", pep.Id)
		}
	}
	w.clientSet.ResetUnmarked(ctx)

	removes := w.clientSet.ForwardKeySet()
	adds := agentssh.ForwardKeySet{}
	for _, pep := range mss.ProxyEndpoints {
		for _, forward := range pep.Forwards {
			if forward.ProxyAgentId != w.proxyAgentId {
				continue
			}
			if forward.ProxyEndpointId == "" {
				continue
			}
			var (
				typ  string
				addr string
				port int
			)
			switch forward.Type {
			case api.FORWARD_TYPE_LOCAL:
				addr = w.bindAddr
				port = forward.BindPort
				typ = agentssh.ForwardKeyTypeL
			case api.FORWARD_TYPE_REMOTE:
				addr = forward.ProxyEndpoint.IntranetIpAddr
				port = forward.BindPort
				typ = agentssh.ForwardKeyTypeR
			default:
				log.Warningf("unknown forward type %s", forward.Type)
				continue
			}
			fk := agentssh.ForwardKey{
				EpKey:   forward.ProxyEndpointId,
				Type:    typ,
				KeyAddr: addr,
				KeyPort: port,

				Value: forward,
			}
			if removes.Contains(fk) {
				removes.Remove(fk)
			} else {
				adds.Add(fk)
			}
		}
	}
	for _, fk := range removes {
		log.Infof("close forward %s", fk.Key())
		w.clientSet.CloseForward(ctx, fk)
	}
	for _, fk := range adds {
		log.Infof("open forward %s", fk.Key())
		forward := fk.Value.(*agentmodels.Forward)
		tick := tickDuration(forward.LastSeenTimeout)
		tickCb := heartbeatFunc(forward.Id, w.sessionCache)
		switch fk.Type {
		case agentssh.ForwardKeyTypeL:
			w.clientSet.LocalForward(ctx, fk.EpKey, agentssh.LocalForwardReq{
				LocalAddr:  fk.KeyAddr,
				LocalPort:  fk.KeyPort,
				RemoteAddr: forward.RemoteAddr,
				RemotePort: forward.RemotePort,
				Tick:       tick,
				TickCb:     tickCb,
			})
		case agentssh.ForwardKeyTypeR:
			w.clientSet.RemoteForward(ctx, fk.EpKey, agentssh.RemoteForwardReq{
				RemoteAddr: fk.KeyAddr,
				RemotePort: fk.KeyPort,
				LocalAddr:  forward.RemoteAddr,
				LocalPort:  forward.RemotePort,
				Tick:       tick,
				TickCb:     tickCb,
			})
		}
	}
	return nil
}
