package models

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

var (
	llmContainerManager *SLLMContainerManager
)

func GetLLMContainerManager() *SLLMContainerManager {
	if llmContainerManager != nil {
		return llmContainerManager
	}
	m := NewLLMContainerManager(SLLMContainer{}, "llm_container", "llm_containers")
	llmContainerManager = &m
	llmContainerManager.SetVirtualObject(llmContainerManager)
	return llmContainerManager
}

func NewLLMContainerManager(dt interface{}, keyword string, keywordPlural string) SLLMContainerManager {
	return SLLMContainerManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(dt, "llm_containers_tbl", keyword, keywordPlural),
	}
}

type SLLMContainerManager struct {
	db.SVirtualResourceBaseManager
}

type SLLMContainer struct {
	db.SVirtualResourceBase

	LLMId        string `width:"128" charset:"ascii" nullable:"false" list:"user" primary:"true" create:"required"`
	CmpId        string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"required"`
	Type         string `width:"16" charset:"ascii" list:"user" primary:"true" create:"required"`
	RunningAppId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
}

func (m *SLLMContainerManager) CreateOnLLM(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	llm *SLLM, cmpId string, svrName string,
) (*SLLMContainer, error) {
	input := &api.LLMContainerCreateInput{
		LLMId: llm.Id,
		Type:  string(llm.GetLLMContainerDriver().GetType()),
		CmpId: cmpId,
	}
	input.Name = svrName
	obj, err := db.DoCreate(m, ctx, userCred, nil, jsonutils.Marshal(input), ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "create llm container")
	}
	return obj.(*SLLMContainer), nil
}
