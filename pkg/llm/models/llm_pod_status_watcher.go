package models

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

func StartLLMPodStatusWatcher(ctx context.Context, region string) {
	session := auth.GetAdminSession(ctx, region)
	informer.NewWatchManagerBySessionBg(session, func(watchMan *informer.SWatchManager) error {
		handler := &llmPodStatusEventHandler{
			userCred: session.GetToken(),
		}
		if err := watchMan.For(compute.Servers).AddEventHandler(ctx, handler); err != nil {
			return errors.Wrap(err, "watch compute servers for llm pod status")
		}
		return nil
	})
}

type llmPodStatusEventHandler struct {
	userCred mcclient.TokenCredential
}

func (h *llmPodStatusEventHandler) OnAdd(obj *jsonutils.JSONDict) {
	if !serverWatchStatusChanged(nil, obj) {
		return
	}
	h.handleServerStatus(context.Background(), obj)
}

func (h *llmPodStatusEventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	if !serverWatchStatusChanged(oldObj, newObj) {
		return
	}
	h.handleServerStatus(context.Background(), newObj)
}

func (h *llmPodStatusEventHandler) OnDelete(obj *jsonutils.JSONDict) {
}

func (h *llmPodStatusEventHandler) handleServerStatus(ctx context.Context, obj *jsonutils.JSONDict) {
	serverId := watchEventStringField(obj, "id")
	if serverId == "" {
		log.Warningf("LLM pod status watcher: server event missing id: %s", obj.String())
		return
	}

	llm, err := fetchLLMByCmpId(serverId)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			log.Warningf("LLM pod status watcher: fetch llm by cmp_id %s: %s", serverId, err)
		}
		return
	}

	server, err := llm.GetServer(ctx)
	if err != nil {
		log.Warningf("LLM pod status watcher: fetch server %s for llm %s: %s", serverId, llm.Name, err)
		return
	}
	if server.Hypervisor != computeapi.HYPERVISOR_POD {
		return
	}

	resolved, err := ResolveLLMStatusFromServerDetails(ctx, llm, server)
	if err != nil {
		log.Warningf("LLM pod status watcher: resolve status for llm %s: %s", llm.Name, err)
		return
	}
	if !resolved.Update {
		return
	}
	if err := llm.SetStatus(ctx, h.userCred, resolved.Status, resolved.Reason); err != nil {
		log.Warningf("LLM pod status watcher: set llm %s status %s: %s", llm.Name, resolved.Status, err)
	}
}

func fetchLLMByCmpId(cmpId string) (*SLLM, error) {
	llm := &SLLM{}
	if err := GetLLMManager().Query().Equals("cmp_id", cmpId).First(llm); err != nil {
		return nil, err
	}
	llm.SetModelManager(GetLLMManager(), llm)
	return llm, nil
}

func getLLMPrimaryContainerStatus(ctx context.Context, llm *SLLM, containers []*computeapi.PodContainerDesc) (string, error) {
	if len(containers) == 0 {
		return "", nil
	}
	ctr, err := llm.GetLLMContainerDriver().GetPrimaryContainer(ctx, llm, containers)
	if err != nil {
		return "", err
	}
	if ctr == nil {
		return "", nil
	}
	return ctr.Status, nil
}

func serverWatchStatusChanged(oldObj *jsonutils.JSONDict, newObj *jsonutils.JSONDict) bool {
	if newObj == nil {
		return false
	}
	newStatus := watchEventStringField(newObj, "status")
	if newStatus == "" {
		return false
	}
	if oldObj == nil {
		return true
	}
	if watchEventStringField(oldObj, "status") != newStatus {
		return true
	}
	return watchEventContainerStatusSignature(oldObj) != watchEventContainerStatusSignature(newObj)
}

func watchEventStringField(obj *jsonutils.JSONDict, key string) string {
	if obj == nil {
		return ""
	}
	value, err := obj.GetString(key)
	if err != nil {
		return ""
	}
	return value
}

func watchEventContainerStatusSignature(obj *jsonutils.JSONDict) string {
	if obj == nil {
		return ""
	}
	containers, err := obj.GetArray("containers")
	if err != nil || len(containers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(containers))
	for idx, container := range containers {
		key := watchEventJSONObjectStringField(container, "id")
		if key == "" {
			key = watchEventJSONObjectStringField(container, "name")
		}
		if key == "" {
			key = fmt.Sprintf("%d", idx)
		}
		status := watchEventJSONObjectStringField(container, "status")
		parts = append(parts, fmt.Sprintf("%s=%s", key, status))
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func watchEventJSONObjectStringField(obj jsonutils.JSONObject, key string) string {
	if obj == nil {
		return ""
	}
	value, err := obj.GetString(key)
	if err != nil {
		return ""
	}
	return value
}
