package lbagent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"yunion.io/x/log"

	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ApiHelper struct {
	opts *Options

	dataDirMan  *agentutils.ConfigDirManager
	corpus      *agentmodels.LoadbalancerCorpus
	agentParams *agentmodels.AgentParams
}

func NewApiHelper(opts *Options) (*ApiHelper, error) {
	helper := &ApiHelper{
		opts:       opts,
		dataDirMan: agentutils.NewConfigDirManager(opts.apiDataStoreDir),
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
				h.doUseCorpus(ctx)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *ApiHelper) adminClientSession(ctx context.Context) *mcclient.ClientSession {
	region := h.opts.CommonOptions.Region
	apiVersion := "v2"
	s := auth.GetAdminSession(ctx, region, apiVersion)
	return s
}

func (h *ApiHelper) agentPeekOnce(ctx context.Context) (*models.LoadbalancerAgent, error) {
	s := h.adminClientSession(ctx)
	data, err := modules.LoadbalancerAgents.Get(s, h.opts.ApiLbagentId, nil)
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
	r := h.agentPeek(ctx)
	if r == nil {
		return
	}
	if !r.staleInFuture(h.opts.ApiLbagentHbTimeoutRelaxation) {
		log.Warningf("agent will stale in %d seconds, re-sync",
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
	if changed := h.doSyncApiData(ctx); changed {
		h.doUseCorpus(ctx)
	}
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

func (h *ApiHelper) doHb(ctx context.Context) (*models.LoadbalancerAgent, error) {
	// TODO check if things changed recently
	s := h.adminClientSession(ctx)
	data, err := modules.LoadbalancerAgents.PerformAction(s, h.opts.ApiLbagentId, "hb", nil)
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
	agentParams, err := agentmodels.NewAgentParams(agent)
	if err != nil {
		log.Errorf("agent params prepare failure: %s", err)
		return false
	}
	if !agentParams.Equal(h.agentParams) {
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
