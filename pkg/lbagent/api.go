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

package lbagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/onecloud/pkg/apihelper"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type ApiHelper struct {
	opts *Options

	dataDirMan  *agentutils.ConfigDirManager
	apih        *apihelper.APIHelper
	corpus      *agentmodels.LoadbalancerCorpus
	agentParams *agentmodels.AgentParams

	haState         string
	haStateProvider HaStateProvider

	mcclientSession *mcclient.ClientSession

	ovn *OvnWorker
}

func NewApiHelper(opts *Options) (*ApiHelper, error) {
	corpus := agentmodels.NewEmptyLoadbalancerCorpus()
	apiOpts := &apihelper.Options{
		CommonOptions:        opts.CommonOptions,
		SyncIntervalSeconds:  opts.ApiSyncIntervalSeconds,
		RunDelayMilliseconds: opts.ApiRunDelayMilliseconds,
		ListBatchSize:        opts.ApiListBatchSize,
	}
	apih, err := apihelper.NewAPIHelper(apiOpts, corpus.ModelSets)
	if err != nil {
		return nil, errors.Wrap(err, "new apihelper")
	}
	helper := &ApiHelper{
		opts: opts,

		dataDirMan: agentutils.NewConfigDirManager(opts.apiDataStoreDir),
		apih:       apih,
		corpus:     corpus,

		haState: api.LB_HA_STATE_UNKNOWN,
	}
	return helper, nil
}

func (h *ApiHelper) Run(ctx context.Context) {
	wg := ctx.Value("wg").(*sync.WaitGroup)
	defer func() {
		wg.Done()
		log.Infof("api helper bye")
	}()
	h.startOvnWorker(ctx)
	defer h.stopOvnWorker()

	wg.Add(1)
	go h.apih.Start(ctx, nil, "")

	hbTicker := time.NewTicker(time.Duration(h.opts.ApiLbagentHbInterval) * time.Second)
	agentParamsSyncTicker := time.NewTicker(time.Duration(h.opts.ApiSyncIntervalSeconds) * time.Second)
	defer hbTicker.Stop()
	defer agentParamsSyncTicker.Stop()

	h.haState = <-h.haStateProvider.StateChannel()
	for {
		select {
		case <-hbTicker.C:
			_, err := h.doHb(ctx)
			if err != nil {
				log.Errorf("heartbeat: %s", err)
			}
		case imss := <-h.apih.ModelSets():
			log.Infof("got new data from api helper")
			mss := imss.(*agentmodels.ModelSets)
			h.corpus.ModelSets = mss
			h.doUseCorpus(ctx)
			h.agentUpdateSeen(ctx)

			err := h.saveCorpus(ctx)
			if err != nil {
				log.Errorf("save corpus failed: %s", err)
			} else {
				if err := h.dataDirMan.Prune(h.opts.DataPreserveN); err != nil {
					log.Errorf("prune corpus data dir failed: %s", err)
				}
			}
		case <-agentParamsSyncTicker.C:
			changed := h.doSyncAgentParams(ctx)
			if changed {
				log.Infof("agent params changed")
				h.doUseCorpus(ctx)
			}
		case state := <-h.haStateProvider.StateChannel():
			switch state {
			case api.LB_HA_STATE_BACKUP:
				h.stopOvnWorker()
				h.doStopDaemons(ctx)
			default:
				if state != h.haState {
					// try your best to make things up
					h.startOvnWorker(ctx)
					h.doUseCorpus(ctx)
				}
			}
			h.haState = state
		case <-ctx.Done():
			return
		}
	}
}

func (h *ApiHelper) SetHaStateProvider(hsp HaStateProvider) {
	h.haStateProvider = hsp
}

func (h *ApiHelper) startOvnWorker(ctx context.Context) {
	if h.ovn == nil {
		h.ovn = NewOvnWorker()
		go h.ovn.Start(ctx)
	}
}

func (h *ApiHelper) stopOvnWorker() {
	if h.ovn != nil {
		h.ovn.Stop()
		h.ovn = nil
	}
}

func (h *ApiHelper) adminClientSession(ctx context.Context) *mcclient.ClientSession {
	s := h.mcclientSession
	if s != nil {
		token := s.GetToken()
		expires := token.GetExpires()
		if time.Now().Add(time.Hour).After(expires) {
			return s
		}
	}

	region := h.opts.CommonOptions.Region
	// apiVersion := "v2"
	h.mcclientSession = auth.GetAdminSession(ctx, region)
	return h.mcclientSession
}

func (h *ApiHelper) agentPeekOnce(ctx context.Context) (*computemodels.SLoadbalancerAgent, error) {
	s := h.adminClientSession(ctx)
	params := jsonutils.NewDict()
	params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))
	data, err := modules.LoadbalancerAgents.Get(s, h.opts.ApiLbagentId, params)
	if err != nil {
		err := fmt.Errorf("agent get error: %s", err)
		return nil, err
	}
	agent := &computemodels.SLoadbalancerAgent{}
	err = data.Unmarshal(agent)
	if err != nil {
		err := fmt.Errorf("agent data unmarshal error: %s", err)
		return nil, err
	}
	return agent, nil
}

func (h *ApiHelper) agentPeekPeers(ctx context.Context, agent *computemodels.SLoadbalancerAgent) ([]*computemodels.SLoadbalancerAgent, error) {
	vri := agent.Params.Vrrp.VirtualRouterId
	clusterId := agent.ClusterId
	s := h.adminClientSession(ctx)
	params := jsonutils.NewDict()
	params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))
	params.Set("cluster_id", jsonutils.NewString(clusterId))
	listResult, err := modules.LoadbalancerAgents.List(s, params)
	if err != nil {
		err := fmt.Errorf("agent listing error: %s", err)
		return nil, err
	}
	peers := []*computemodels.SLoadbalancerAgent{}
	for _, data := range listResult.Data {
		peerAgent := &computemodels.SLoadbalancerAgent{}
		err := data.Unmarshal(peerAgent)
		if err != nil {
			err := fmt.Errorf("agent data unmarshal error: %s", err)
			return nil, err
		}
		// just in case
		if peerAgent.ClusterId != clusterId {
			continue
		}
		if peerAgent.Params.Vrrp.VirtualRouterId != vri {
			continue
		}
		peers = append(peers, peerAgent)
	}
	return peers, nil
}

type agentPeekResult computemodels.SLoadbalancerAgent

func (r *agentPeekResult) staleInFuture(s int) bool {
	if r.HbLastSeen.IsZero() {
		return true
	}
	duration := time.Since(r.HbLastSeen).Seconds()
	if int(duration) < s {
		return true
	}
	return false
}

func (h *ApiHelper) agentPeek(ctx context.Context) *agentPeekResult {
	doPeekWithLog := func() *computemodels.SLoadbalancerAgent {
		agent, err := h.agentPeekOnce(ctx)
		if err != nil {
			log.Errorf("agent peek failed: %s", err)
		}
		return agent
	}
	agent := doPeekWithLog()
	if agent == nil {
		initHbTicker := time.NewTicker(time.Duration(3) * time.Second)
		defer initHbTicker.Stop()
	initHbDone:
		for {
			select {
			case <-initHbTicker.C:
				agent = doPeekWithLog()
				if agent != nil {
					break initHbDone
				}
			case <-ctx.Done():
				return nil
			}
		}
	}
	return (*agentPeekResult)(agent)
}

func (h *ApiHelper) agentUpdateSeen(ctx context.Context) *computemodels.SLoadbalancerAgent {
	s := h.adminClientSession(ctx)
	params := h.corpus.MaxSeenUpdatedAtParams()
	data, err := modules.LoadbalancerAgents.Update(s, h.opts.ApiLbagentId, params)
	if err != nil {
		log.Errorf("agent get error: %s", err)
		return nil
	}
	agent := &computemodels.SLoadbalancerAgent{}
	err = data.Unmarshal(agent)
	if err != nil {
		log.Errorf("agent data unmarshal error: %s", err)
		return nil
	}
	return agent
}

func (h *ApiHelper) newAgentHbParams(ctx context.Context) (*jsonutils.JSONDict, error) {
	ip, err := netutils2.MyIP()
	if err != nil {
		return nil, err
	}
	state := h.haState
	version := version.Get().GitVersion
	opts := &options.LoadbalancerAgentActionHbOptions{
		IP:      ip,
		HaState: state,
		Version: version,
	}
	params, err := opts.Params()
	if err != nil {
		return nil, err
	}
	return params, nil
}

func (h *ApiHelper) doHb(ctx context.Context) (*computemodels.SLoadbalancerAgent, error) {
	// TODO check if things changed recently
	s := h.adminClientSession(ctx)
	params, err := h.newAgentHbParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("heartbeat: making params: %s", err)
	}
	data, err := modules.LoadbalancerAgents.PerformAction(s, h.opts.ApiLbagentId, "hb", params)
	if err != nil {
		err := fmt.Errorf("heartbeat api error: %s", err)
		return nil, err
	}
	agent := &computemodels.SLoadbalancerAgent{}
	err = data.Unmarshal(agent)
	if err != nil {
		err := fmt.Errorf("heartbeat data unmarshal error: %s", err)
		return nil, err
	}
	return agent, nil
}

func (h *ApiHelper) saveCorpus(ctx context.Context) error {
	_, err := h.dataDirMan.NewDir(func(dir string) error {
		err := h.corpus.SaveDir(dir)
		if err != nil {
			return fmt.Errorf("save to dir %s: %s", dir, err)
		}
		return nil
	})
	return err
}

func (h *ApiHelper) doSyncAgentParams(ctx context.Context) bool {
	agent, err := h.agentPeekOnce(ctx)
	if err != nil {
		log.Errorf("agent params get failure: %s", err)
		return false
	}
	peers, err := h.agentPeekPeers(ctx, agent)
	if err != nil {
		log.Errorf("agent get peers failure: %s", err)
		return false
	}
	unicastPeer := []string{}
	for _, peer := range peers {
		if peer.Id == agent.Id {
			continue
		}
		if peer.IP == "" {
			log.Warningf("agent %s(%s) has no ip, use multicast vrrp", peer.Name, peer.Id)
			break
		}
		unicastPeer = append(unicastPeer, peer.IP)
	}
	useUnicast := len(unicastPeer) == len(peers)-1

	agentParams, err := agentmodels.NewAgentParams(agent)
	if err != nil {
		log.Errorf("agent params prepare failure: %s", err)
		return false
	}
	agentParams.SetVrrpParams("notify_script", h.haStateProvider.StateScript())
	if useUnicast {
		agentParams.SetVrrpParams("unicast_peer", unicastPeer)
	}
	if !agentParams.Equals(h.agentParams) {
		if useUnicast {
			log.Infof("use unicast vrrp from %s to %s", agent.IP, strings.Join(unicastPeer, ","))
		}
		h.agentParams = agentParams
		return true
	}
	return false
}

func (h *ApiHelper) doUseCorpus(ctx context.Context) {
	if h.corpus == nil || h.corpus.ModelSets == nil {
		log.Warningf("agent corpus nil")
		return
	}
	if h.agentParams == nil {
		log.Warningf("agent params nil")
		return
	}
	if err := h.ovn.Refresh(ctx, h.corpus.ModelSets.Loadbalancers); err != nil {
		log.Errorf("ovn refresh: %v", err)
	}
	log.Infof("make effect new corpus and params")
	cmdData := &LbagentCmdUseCorpusData{
		Corpus:      h.corpus,
		AgentParams: h.agentParams,
	}
	cmdData.Wg.Add(1)
	cmd := &LbagentCmd{
		Type: LbagentCmdUseCorpus,
		Data: cmdData,
	}
	cmdChan := ctx.Value("cmdChan").(chan *LbagentCmd)
	select {
	case cmdChan <- cmd:
		cmdData.Wg.Wait()
	case <-ctx.Done():
		return
	}
}

func (h *ApiHelper) doStopDaemons(ctx context.Context) {
	cmd := &LbagentCmd{
		Type: LbagentCmdStopDaemons,
	}
	cmdChan := ctx.Value("cmdChan").(chan *LbagentCmd)
	select {
	case cmdChan <- cmd:
	case <-ctx.Done():
	}
}
