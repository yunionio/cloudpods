package lbagent

import (
	"sync"

	agentmodels "yunion.io/x/onecloud/pkg/lbagent/models"
)

type LbagentCmdType uintptr

const (
	LbagentCmdUseCorpus LbagentCmdType = iota
)

type LbagentCmdUseCorpusData struct {
	Corpus      *agentmodels.LoadbalancerCorpus
	AgentParams *agentmodels.AgentParams
	Wg          sync.WaitGroup
}

type LbagentCmd struct {
	Type LbagentCmdType
	Data interface{}
}
