package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

type SDifyManager struct {
	db.SVirtualResourceBaseManager
}

var DifyManager *SDifyManager

var __difyContainersMngr__ *DifyContainersManager

func init() {
	DifyManager = &SDifyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SDify{},
			"difies_tbl",
			"dify",
			"difies",
		),
	}
	DifyManager.SetVirtualObject(DifyManager)
	DifyManager.SetAlias("dify", "difies")
	DifyManager.NameRequireAscii = false
}

func (manager *SDifyManager) DeleteByGuestId(ctx context.Context, userCred mcclient.TokenCredential, gstId string) error {
	q := manager.Query().Equals("guest_id", gstId)
	difies := make([]SDify, 0)
	if err := db.FetchModelObjects(manager, q, &difies); err != nil {
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	for _, dify := range difies {
		if err := dify.RealDelete(ctx, userCred); nil != err {
			return err
		}
	}
	return nil
}

func (manager *SDifyManager) InitDifyContainersManager(input *api.DifyCustomized) {
	var userCustomizedEnvs *DifyContainerEnv
	if input.CustomizedEnvs != nil {
		userCustomizedEnvs = new(DifyContainerEnv)
		for _, envs := range input.CustomizedEnvs {
			userCustomizedEnvs.SetContainerEnv(envs.Key, envs.Value)
		}
	}
	__difyContainersMngr__ = &DifyContainersManager{
		UserCustomizedEnvs: userCustomizedEnvs,
		ImageRegistry:      input.Registry,
	}
}

func (manager *SDifyManager) GetDifyContainersManager() *DifyContainersManager {
	if __difyContainersMngr__ == nil {
		__difyContainersMngr__ = &DifyContainersManager{}
	}
	return __difyContainersMngr__
}

func (manager *SDifyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.DifyCreateInput) (*api.DifyCreateInput, error) {
	// check disk mount, can not create without disk
	if len(input.Disks) == 0 {
		return nil, httperrors.NewNotEmptyError("disk is required")
	}

	// init dify containers manager
	manager.InitDifyContainersManager(&input.DifyCustomized)

	// first deploy redis containers
	redis, err := manager.GetDifyContainersManager().GetContainer(input.Name, api.DIFY_REDIS_KEY)
	if nil != err {
		return nil, err
	}
	input.Pod.Containers = []*computeapi.PodContainerCreateInput{
		redis,
	}

	return input, nil
}

type SDify struct {
	SLLMBase

	Containers jsonutils.JSONObject `length:"long" list:"user" update:"user" create:"optional"`
	Registry   string               `width:"64" charset:"ascii" list:"user"`
}

// func (dify *SDify) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
// 	// unmarshal input
// 	input := &api.DifyCreateInput{}
// 	if err := data.Unmarshal(input); err != nil {
// 		return errors.Wrap(err, "Unmarshal ServerCreateInput")
// 	}

// 	// init task
// 	dify.Id = stringutils.UUID4()
// 	task, err := taskman.TaskManager.NewTask(ctx, "DifyCreateTask", dify, userCred, jsonutils.NewDict(), "", "", nil)
// 	if err != nil {
// 		return errors.Wrap(err, "NewTask")
// 	}
// 	input.ParentTaskId = task.GetId()

// 	// use data to create a pod
// 	server, err := dify.GetPodDriver().RequestCreatePod(ctx, userCred, &input.ServerCreateInput)
// 	if err != nil {
// 		return errors.Wrap(err, "CreateServer")
// 	}

// 	// set guest id
// 	guestId, err := server.GetString("id")
// 	if err != nil {
// 		return errors.Wrap(err, "GetGuestId")
// 	}
// 	dify.GuestId = guestId

// 	return dify.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
// }

func (dify *SDify) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	input := new(api.DifyCreateInput)
	if err := data.Unmarshal(&input); nil != err {
		return
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DifyCreateTask", dify, userCred, jsonutils.NewDict(), "", "", nil)
	if nil != err {
		return
	}
	dify.StartCreatePodTask(ctx, userCred, &input.ServerCreateInput, task.GetId())
	dify.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
}

func (dify *SDify) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return dify.SVirtualResourceBase.Delete(ctx, userCred)
}

func (dify *SDify) CreateContainer(ctx context.Context, userCred mcclient.TokenCredential, containerKey string, taskId string) error {
	// get input
	input, err := DifyManager.GetDifyContainersManager().GetContainer(dify.GetName(), containerKey)
	if nil != err {
		return err
	}

	// create on pod
	ctr, err := dify.createContainerOnPod(ctx, userCred, containerKey, input)
	if nil != err {
		return err
	}

	// do create task
	if err := dify.GetPodDriver().RequestDoCreateContainer(ctx, userCred, ctr, taskId); nil != err {
		return err
	}

	return nil
}

func (dify *SDify) CheckContainerHealth(ctx context.Context, userCred mcclient.TokenCredential, containerKey string, command ...string) error {
	containerId, err := dify.getContainerIdByContainerKey(containerKey)
	if nil != err {
		return err
	}

	// exec command
	result, err := dify.GetPodDriver().RequestExecSyncContainer(ctx, userCred, containerId, &computeapi.ContainerExecSyncInput{
		Command: command,
	})
	if nil != err {
		return err
	}
	log.Infof("dify check container %s health, result: %s", containerId, result.String())

	return nil
}

func (dify *SDify) CheckRedis(ctx context.Context, userCred mcclient.TokenCredential) error {
	// get redis container
	ctrs, err := dify.GetPodDriver().RequestGetContainersByPodId(ctx, userCred, dify.GuestId)
	if err != nil {
		return err
	}

	// set redis container id
	if len(ctrs) != 1 {
		return errors.Errorf("Strange redis container")
	}
	ctr, err := ctrs[0].GetString("id")
	if nil != err {
		return err
	}
	dify.setContainerId(api.DIFY_REDIS_KEY, ctr)

	// exec to check health
	if err := dify.CheckContainerHealth(ctx, userCred, api.DIFY_REDIS_KEY, "redis-cli", "-c", "ping"); nil != err {
		return err
	}
	return nil
}

func (dify *SDify) GetLLMBase() *SLLMBase {
	return &dify.SLLMBase
}

func (dify *SDify) createContainerOnPod(ctx context.Context, userCred mcclient.TokenCredential, containerKey string, data *computeapi.PodContainerCreateInput) (string, error) {
	// create container on pod
	container, err := dify.GetPodDriver().RequestCreateContainerOnPod(ctx, userCred, dify.GuestId, data)
	if nil != err {
		return "", err
	}

	// set container id
	ctrId, err := container.GetString("id")
	if err != nil {
		return "", errors.Wrap(err, "GetContainerId")
	}
	err = dify.setContainerId(containerKey, ctrId)

	return ctrId, err
}

func (dify *SDify) setContainerId(containerKey, ctrId string) error {
	// if not set, init it
	if nil == dify.Containers {
		dify.Containers = jsonutils.NewDict()
	}
	containersDict, ok := dify.Containers.(*jsonutils.JSONDict)
	if !ok {
		containersDict = jsonutils.NewDict()
		dify.Containers = containersDict
	}

	// save change
	if _, err := db.Update(dify, func() error {
		// set container id
		containersDict.Set(containerKey, jsonutils.NewString(ctrId))
		return nil
	}); nil != err {
		return errors.Wrap(err, "db.Update")
	}

	return nil
}

func (dify *SDify) getContainerIdByContainerKey(containerKey string) (string, error) {
	if dify.Containers == nil {
		return "", errors.Error("containers not initialized")
	}
	containersDict, ok := dify.Containers.(*jsonutils.JSONDict)
	if !ok {
		return "", errors.Error("containers is not a valid JSON dict")
	}

	// get ctrId
	ctrIdJson, err := containersDict.Get(containerKey)
	if nil != err {
		return "", errors.Wrapf(err, "container key %s not found", containerKey)
	}
	ctrId, err := ctrIdJson.GetString()
	if nil != err {
		return "", errors.Wrapf(err, "container id not string")
	}

	return ctrId, nil
}
