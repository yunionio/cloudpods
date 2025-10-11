package llm

import (
	"context"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/pkg/errors"
)

type LLMCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(LLMCreateTask{})
}

func (task *LLMCreateTask) taskFailed(ctx context.Context, llm *models.SLLM, err error) {
	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
	db.OpsLog.LogEvent(llm, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_CREATE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *LLMCreateTask) taskComplete(ctx context.Context, llm *models.SLLM, status string) {
	llm.SetStatus(ctx, task.GetUserCred(), status, "create success")
	task.SetStageComplete(ctx, nil)
}

func (task *LLMCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	serverCreateInput := api.LLMCreateInput{}
	err := body.Unmarshal(&serverCreateInput)
	if err != nil {
		task.taskFailed(ctx, llm, err)
		return
	}

	serverCreateInput.Name = llm.Name
	serverId, err := llm.ServerCreate(ctx, task.UserCred, &serverCreateInput)
	if err != nil {
		task.taskFailed(ctx, llm, err)
		return
	}

	db.Update(llm, func() error {
		llm.SvrId = serverId
		return nil
	})
	llm.SvrId = serverId
	task.SetStage("OnLLMRefreshStatusComplete", nil)
	var expectStatus []string
	if serverCreateInput.AutoStart {
		expectStatus = []string{computeapi.VM_RUNNING}
	} else {
		expectStatus = []string{computeapi.VM_READY}
	}
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		server, err := llm.WaitServerStatus(ctx, task.UserCred, expectStatus, 7200)
		if err != nil {
			return nil, errors.Wrap(err, "WaitServerStatus")
		}
		return jsonutils.Marshal(server), nil
	})
}

func (task *LLMCreateTask) OnLLMRefreshStatusCompleteFailed(ctx context.Context, llm *models.SLLM, err jsonutils.JSONObject) {
	task.taskFailed(ctx, llm, errors.Error(err.String()))
}

func (task *LLMCreateTask) OnLLMRefreshStatusComplete(ctx context.Context, llm *models.SLLM, body jsonutils.JSONObject) {
	server := computeapi.ServerDetails{}
	err := body.Unmarshal(&server)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "Unmarshal"))
		return
	}

	// 创建磁盘
	for _, disk := range server.DisksInfo {
		volume := models.SVolume{}
		volume.SvrId = disk.Id
		volume.LLMId = llm.Id
		volume.SizeMB = disk.SizeMb
		volume.Name = disk.Name
		volume.StorageType = disk.StorageType
		volume.Status = computeapi.DISK_READY
		volume.DomainId = llm.DomainId
		volume.ProjectId = llm.ProjectId
		volume.ProjectSrc = llm.ProjectSrc
		// if len(input.TemplateId) > 0 {
		volume.TemplateId = disk.ImageId
		// }
		// volume.MountedApps = mountedApps

		err := models.GetVolumeManager().TableSpec().Insert(ctx, &volume)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "VolumeManager.TableSpec().Insert"))
			return
		}
	}

	// 创建访问信息、portmappings
	if len(server.Nics) > 0 {
		db.Update(llm, func() error {
			llm.LLMIp = server.Nics[0].IpAddr
			return nil
		})

		for _, portMapping := range server.Nics[0].PortMappings {
			access := models.SAccessInfo{}
			access.LLMId = llm.Id

			access.ListenPort = int(portMapping.Port)
			access.AccessPort = int(*portMapping.HostPort)
			access.Protocol = string(portMapping.Protocol)
			access.RemoteIps = portMapping.RemoteIps
			envs := make([]api.PortMappingEnv, 0)
			for _, env := range portMapping.Envs {
				envs = append(envs, api.PortMappingEnv{
					Key:       env.Key,
					ValueFrom: string(env.ValueFrom),
				})
			}
			access.PortMappingEnvs = envs

			models.GetAccessInfoManager().TableSpec().Insert(ctx, &access)
		}
	}

	// 创建应用容器记录
	if len(server.Containers) != 1 {
		task.taskFailed(ctx, llm, errors.Errorf("expected 1 containers, but got %d", len(server.Containers)))
		return
	}
	llmCtr := models.GetSvrLLMContainer(server.Containers)
	if llmCtr == nil {
		task.taskFailed(ctx, llm, errors.Errorf("cannot find app container"))
		return
	}
	if _, err := models.GetLLMContainerManager().CreateOnLLM(ctx, task.GetUserCred(), llm.GetOwnerId(), llm, llmCtr.Id, llmCtr.Name); nil != err {
		task.taskFailed(ctx, llm, errors.Wrap(err, "create llm container on llm"))
		return
	}

	// 调用子任务在容器中拉取模型

	task.taskComplete(ctx, llm, server.Status)
}
