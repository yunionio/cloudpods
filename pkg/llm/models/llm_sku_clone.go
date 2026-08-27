package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func validateLLMSkuCloneable(sku *SLLMSku) error {
	if sku == nil {
		return httperrors.NewInputParameterError("empty llm_sku")
	}
	switch sku.Status {
	case api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL:
		return httperrors.NewInvalidStatusError("cannot clone sku while importing model")
	case api.LLM_DEPLOYMENT_STATUS_IMPORT_MODEL_FAILED:
		return httperrors.NewInvalidStatusError("cannot clone sku after import model failed")
	}
	return nil
}

func buildLLMSkuCloneCreateInput(sku *SLLMSku, input api.LLMSkuCloneInput) (*api.LLMSkuCreateInput, error) {
	if err := validateLLMSkuCloneable(sku); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	generateName := strings.TrimSpace(input.GenerateName)
	if name == "" {
		name = generateName
	}
	if name == "" {
		return nil, httperrors.NewMissingParameterError("name")
	}

	create := &api.LLMSkuCreateInput{
		LLMSKuBaseCreateInput: api.LLMSKuBaseCreateInput{
			Cpu:          sku.Cpu,
			Memory:       sku.Memory,
			Bandwidth:    sku.Bandwidth,
			Volumes:      cloneJSONPtr(sku.Volumes),
			HostPaths:    cloneJSONPtr(sku.HostPaths),
			PortMappings: cloneJSONPtr(sku.PortMappings),
			Devices:      cloneJSONPtr(sku.Devices),
			Envs:         cloneJSONPtr(sku.Envs),
			Properties:   cloneStringMap(sku.Properties),
		},
		LLMImageId:          sku.LLMImageId,
		LLMType:             sku.LLMType,
		LLMSpec:             cloneJSONPtr(sku.LLMSpec),
		Source:              sku.Source,
		HuggingfaceRepoId:   sku.HuggingfaceRepoId,
		HuggingfaceFilename: sku.HuggingfaceFilename,
		ModelScopeModelId:   sku.ModelScopeModelId,
		ModelScopeFilePath:  sku.ModelScopeFilePath,
		LocalPath:           sku.LocalPath,
		PreferHosts:         cloneStringSlice(sku.PreferHosts),
		Categories:          cloneStringSlice(sku.Categories),
		BackendVersion:      sku.BackendVersion,
		BackendParameters:   cloneStringSlice(sku.BackendParameters),
	}
	create.Name = name
	if generateName != "" {
		create.GenerateName = generateName
	} else {
		create.GenerateName = name
	}
	if desc := strings.TrimSpace(input.Description); desc != "" {
		create.Description = desc
	} else {
		create.Description = sku.Description
	}
	create.MountedModels = cloneStringSlice(sku.MountedModels)
	return create, nil
}

func (sku *SLLMSku) PerformClone(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMSkuCloneInput) (jsonutils.JSONObject, error) {
	createInput, err := buildLLMSkuCloneCreateInput(sku, input)
	if err != nil {
		return nil, err
	}
	cloned, err := GetLLMSkuManager().createFromClone(ctx, userCred, createInput)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(cloned), nil
}

func (man *SLLMSkuManager) createFromClone(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMSkuCreateInput) (*SLLMSku, error) {
	data := jsonutils.Marshal(input)
	obj, err := db.DoCreate(man, ctx, userCred, nil, data, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "DoCreate cloned llm_sku")
	}
	cloned := obj.(*SLLMSku)
	func() {
		lockman.LockObject(ctx, cloned)
		defer lockman.ReleaseObject(ctx, cloned)
		cloned.PostCreate(ctx, userCred, userCred, nil, data)
		if err := man.GetExtraHook().AfterPostCreate(ctx, userCred, userCred, cloned, nil, data); err != nil {
			logclient.AddActionLogWithContext(ctx, cloned, logclient.ACT_POST_CREATE_HOOK, err, userCred, false)
		}
	}()
	notes := cloned.GetShortDesc(ctx)
	db.OpsLog.LogEvent(cloned, db.ACT_CREATE, notes, userCred)
	logclient.AddActionLogWithContext(ctx, cloned, logclient.ACT_CLONE, notes, userCred, true)
	man.OnCreateComplete(ctx, []db.IModel{cloned}, userCred, userCred, nil, []jsonutils.JSONObject{data})
	return cloned, nil
}

func cloneJSONPtr[T any](src *T) *T {
	if src == nil {
		return nil
	}
	dst := new(T)
	if err := jsonutils.Marshal(src).Unmarshal(dst); err != nil {
		copied := *src
		return &copied
	}
	return dst
}

func cloneStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
