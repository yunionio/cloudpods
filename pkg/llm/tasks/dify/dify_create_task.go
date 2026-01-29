package dify

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DifyCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DifyCreateTask{})
}

func (task *DifyCreateTask) taskFailed(ctx context.Context, dify *models.SDify, err error) {
	dify.SetStatus(ctx, task.UserCred, api.LLM_STATUS_CREATE_FAIL, err.Error())
	db.OpsLog.LogEvent(dify, db.ACT_CREATE, err, task.UserCred)
	logclient.AddActionLogWithStartable(task, dify, logclient.ACT_CREATE, err, task.UserCred, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (task *DifyCreateTask) taskComplete(ctx context.Context, dify *models.SDify, status string) {
	dify.SetStatus(ctx, task.GetUserCred(), status, "create success")
	task.SetStageComplete(ctx, nil)
}

func (task *DifyCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	dify := obj.(*models.SDify)
	serverCreateInput := api.DifyCreateInput{}
	err := body.Unmarshal(&serverCreateInput)
	if err != nil {
		task.taskFailed(ctx, dify, err)
		return
	}

	serverCreateInput.Name = dify.Name

	task.SetStage("OnDifyRefreshStatusComplete", nil)
	s := auth.GetSession(ctx, task.GetUserCred(), "")
	s.WithTaskCallback(task.GetId(), func() error {
		serverId, err := dify.ServerCreate(ctx, task.UserCred, s, &serverCreateInput)
		if err != nil {
			task.taskFailed(ctx, dify, err)
			return err
		}

		db.Update(dify, func() error {
			dify.CmpId = serverId
			return nil
		})
		dify.CmpId = serverId
		return nil
	})
	// var expectStatus []string
	// if serverCreateInput.AutoStart {
	// 	expectStatus = []string{computeapi.VM_RUNNING}
	// } else {
	// 	expectStatus = []string{computeapi.VM_READY}
	// }
	// taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
	// 	server, err := dify.WaitServerStatus(ctx, task.UserCred, expectStatus, 7200)
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "WaitServerStatus")
	// 	}
	// 	return jsonutils.Marshal(server), nil
	// })
}

func (task *DifyCreateTask) OnDifyRefreshStatusCompleteFailed(ctx context.Context, dify *models.SDify, err jsonutils.JSONObject) {
	task.taskFailed(ctx, dify, errors.Error(err.String()))
}

func (task *DifyCreateTask) OnDifyRefreshStatusComplete(ctx context.Context, dify *models.SDify, body jsonutils.JSONObject) {
	server, err := dify.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, dify, errors.Wrap(err, "GetServer"))
		return
	}

	// 创建磁盘
	for _, disk := range server.DisksInfo {
		volume := models.SVolume{}
		volume.CmpId = disk.Id
		volume.LLMId = dify.Id
		volume.SizeMB = disk.SizeMb
		volume.Name = disk.Name
		volume.StorageType = disk.StorageType
		volume.Status = computeapi.DISK_READY
		volume.DomainId = dify.DomainId
		volume.ProjectId = dify.ProjectId
		volume.ProjectSrc = dify.ProjectSrc
		// if len(input.TemplateId) > 0 {
		volume.TemplateId = disk.ImageId
		// }
		// volume.MountedApps = mountedApps

		err := models.GetVolumeManager().TableSpec().Insert(ctx, &volume)
		if err != nil {
			task.taskFailed(ctx, dify, errors.Wrap(err, "VolumeManager.TableSpec().Insert"))
			return
		}
	}

	// 创建访问信息、portmappings
	if len(server.Nics) > 0 {
		db.Update(dify, func() error {
			dify.LLMIp = server.Nics[0].IpAddr
			return nil
		})

		for _, portMapping := range server.Nics[0].PortMappings {
			access := models.SAccessInfo{}
			access.LLMId = dify.Id

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

	// // 创建应用容器记录
	// if len(server.Containers) != 1 {
	// 	task.taskFailed(ctx, dify, errors.Errorf("expected 1 containers, but got %d", len(server.Containers)))
	// 	return
	// }
	// llmCtr := models.GetSvrLLMContainer(server.Containers)
	// if llmCtr == nil {
	// 	task.taskFailed(ctx, dify, errors.Errorf("cannot find app container"))
	// 	return
	// }
	// if _, err := models.GetLLMContainerManager().CreateOnLLM(ctx, task.GetUserCred(), dify.GetOwnerId(), dify, llmCtr.Id, llmCtr.Name); nil != err {
	// 	task.taskFailed(ctx, dify, errors.Wrap(err, "create llm container on llm"))
	// 	return
	// }

	task.taskComplete(ctx, dify, server.Status)
}
