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
	"yunion.io/x/pkg/util/version"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/netutils2"
)

type ApiHelper struct {
	opts *Options

	dataDirMan  *agentutils.ConfigDirManager
	corpus      *agentmodels.LoadbalancerCorpus
	agentParams *agentmodels.AgentParams

	haState         string
	haStateProvider HaStateProvider
}

func NewApiHelper(opts *Options) (*ApiHelper, error) {
	helper := &ApiHelper{
		opts:       opts,
		dataDirMan: agentutils.NewConfigDirManager(opts.apiDataStoreDir),
		haState:    api.LB_HA_STATE_UNKNOWN,
	}
	return helper, nil
}

func (h *ApiHelper) Run(ctx context.Context) {
	defer func() {
		log.Infof("api helper bye")
		wg := ctx.Value("wg").(*sync.WaitGroup)
		wg.Done()
	}()
	h.runInit(ctx)
	apiSyncTicker := time.NewTicker(time.Duration(h.opts.ApiSyncInterval) * time.Second)
	hbTicker := time.NewTicker(time.Duration(h.opts.ApiLbagentHbInterval) * time.Second)
	defer hbTicker.Stop()
	defer apiSyncTicker.Stop()
	for {
		select {
		case <-hbTicker.C:
			_, err := h.doHb(ctx)
			if err != nil {
				log.Errorf("heartbeat: %s", err)
			}
		case <-apiSyncTicker.C:
			apiDataChanged := h.doSyncApiData(ctx)
			agentParamsChanged := h.doSyncAgentParams(ctx)
			if apiDataChanged || agentParamsChanged {
				log.Infof("things changed: params: %v, data %v", agentParamsChanged, apiDataChanged)
				h.doUseCorpus(ctx)
			}
		case state := <-h.haStateProvider.StateChannel():
			switch state {
			case api.LB_HA_STATE_BACKUP:
				h.doStopDaemons(ctx)
			default:
				if state != h.haState {
					// try your best to make things up
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

func (h *ApiHelper) adminClientSession(ctx context.Context) *mcclient.ClientSession {
	region := h.opts.CommonOptions.Region
	apiVersion := "v2"
	s := auth.GetAdminSession(ctx, region, apiVersion)
	return s
}

func (h *ApiHelper) agentPeekOnce(ctx context.Context) (*models.LoadbalancerAgent, error) {
	s := h.adminClientSession(ctx)
	params := jsonutils.NewDict()
	params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))
	data, err := modules.LoadbalancerAgents.Get(s, h.opts.ApiLbagentId, params)
	if err != nil {
		err := fmt.Errorf("agent get error: %s", err)
		return nil, err
	}
	agent := &models.LoadbalancerAgent{}
	err = data.Unmarshal(agent)
	if err != nil {
		err := fmt.Errorf("agent data unmarshal error: %s", err)
		return nil, err
	}
	return agent, nil
}

func (h *ApiHelper) agentPeekPeers(ctx context.Context, agent *models.LoadbalancerAgent) ([]*models.LoadbalancerAgent, error) {
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
	peers := []*models.LoadbalancerAgent{}
	for _, data := range listResult.Data {
		peerAgent := &models.LoadbalancerAgent{}
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

type agentPeekResult models.LoadbalancerAgent

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
	doPeekWithLog := func() *models.LoadbalancerAgent {
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

func (h *ApiHelper) runInit(ctx context.Context) {
	h.haState = <-h.haStateProvider.StateChannel()
	if false {
		r := h.agentPeek(ctx)
		if r == nil {
			return
		}
		if !r.staleInFuture(h.opts.ApiLbagentHbTimeoutRelaxation) {
			log.Warningf("agent will stale in %d seconds, ignore old corpus",
				h.opts.ApiLbagentHbTimeoutRelaxation)
		} else {
			h.doHb(ctx)
			corpus, err := h.loadLocalData(ctx)
			if err == nil {
				h.corpus = corpus
			} else {
				log.Errorf("load local api data failed: %s", err)
			}
		}
	}
	// better reload now because agent data is not in corpus yet
	h.doSyncApiData(ctx)
	h.doSyncAgentParams(ctx)
	h.doUseCorpus(ctx)
}

func (h *ApiHelper) loadLocalData(ctx context.Context) (*agentmodels.LoadbalancerCorpus, error) {
	corpus := agentmodels.NewEmptyLoadbalancerCorpus()
	dataDir := h.dataDirMan.MostRecentSubdir()
	err := corpus.LoadDir(dataDir)
	if err != nil {
		return nil, err
	}
	return corpus, nil
}

func (h *ApiHelper) agentUpdateSeen(ctx context.Context) *models.LoadbalancerAgent {
	s := h.adminClientSession(ctx)
	params := h.corpus.MaxSeenUpdatedAtParams()
	data, err := modules.LoadbalancerAgents.Update(s, h.opts.ApiLbagentId, params)
	if err != nil {
		log.Errorf("agent get error: %s", err)
		return nil
	}
	agent := &models.LoadbalancerAgent{}
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
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	return params, nil
}

func (h *ApiHelper) doHb(ctx context.Context) (*models.LoadbalancerAgent, error) {
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
	agent := &models.LoadbalancerAgent{}
	err = data.Unmarshal(agent)
	if err != nil {
		err := fmt.Errorf("heartbeat data unmarshal error: %s", err)
		return nil, err
	}
	return agent, nil
}

func (h *ApiHelper) doSyncApiData(ctx context.Context) bool {
	{
		stime := time.Now()
		defer func() {
			elapsed := time.Since(stime)
			log.Infof("sync api data done, elapsed: %s", elapsed.String())
		}()
	}

	s := h.adminClientSession(ctx)
	if h.corpus == nil {
		h.corpus = agentmodels.NewEmptyLoadbalancerCorpus()
	}
	r, err := h.corpus.SyncModelSets(s, h.opts.ApiListBatchSize)
	if err != nil {
		log.Errorf("sync models: %s", err)
		return false
	}
	if r.Changed {
		h.agentUpdateSeen(ctx)
	}
	if !r.Correct {
		log.Warningf("sync models: not correct")
		return false
	}
	if r.Changed {
		err := h.saveCorpus(ctx)
		if err != nil {
			log.Errorf("save corpus failed: %s", err)
			return false
		}
		if err := h.dataDirMan.Prune(h.opts.DataPreserveN); err != nil {
			log.Errorf("prune corpus data dir failed: %s", err)
			// continue
		}
		log.Infof("corpus changed")
		return true
	}
	return false
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
	if h.corpus == nil {
		log.Warningf("agent corpus nil")
		return
	}
	if h.agentParams == nil {
		log.Warningf("agent params nil")
		return
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
