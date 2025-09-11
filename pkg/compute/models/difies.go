package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"
)

type SDifyManager struct {
	db.SVirtualResourceBaseManager
}

var DifyManager *SDifyManager

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

func (manager *SDifyManager) GetDifyContainersManager() *DifyContainersManager {
	return &DifyContainersManager{}
}

func (manager *SDifyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, _ jsonutils.JSONObject, input *api.DifyCreateInput) (*api.DifyCreateInput, error) {
	// check disk mount, can not create without disk
	if len(input.Disks) == 0 {
		return nil, httperrors.NewNotEmptyError("disk is required")
	}

	// first deploy redis containers
	redis, err := manager.GetDifyContainersManager().GetContainer(input.Name, api.DIFY_REDIS_KEY)
	if nil != err {
		return nil, err
	}
	input.Pod.Containers = []*api.PodContainerCreateInput{
		redis,
	}

	return input, nil
}

type SDify struct {
	db.SVirtualResourceBase

	// GuestId is also the pod id
	GuestId    string               `width:"36" charset:"ascii" list:"user" index:"true" create:"optional"`
	Containers jsonutils.JSONObject `length:"long" list:"user" update:"user" create:"optional"`
}

func (dify *SDify) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// unmarshal input
	input := &api.DifyCreateInput{}
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "Unmarshal ServerCreateInput")
	}

	// init task
	dify.Id = stringutils.UUID4()
	task, err := taskman.TaskManager.NewTask(ctx, "DifyCreateTask", dify, userCred, jsonutils.NewDict(), "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	input.ParentTaskId = task.GetId()

	// use data to create a pod
	handler := db.NewModelHandler(GuestManager)
	server, err := handler.Create(ctx, jsonutils.NewDict(), jsonutils.Marshal(input.ServerCreateInput), nil)
	if err != nil {
		return errors.Wrap(err, "CreateServer")
	}

	// set guest id
	guestId, err := server.GetString("id")
	if err != nil {
		return errors.Wrap(err, "GetGuestId")
	}
	dify.GuestId = guestId

	return dify.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
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
	params := jsonutils.NewDict()
	params.Set("auto_start", jsonutils.JSONTrue)
	if err := ctr.StartCreateTask(ctx, userCred, taskId, params); nil != err {
		return err
	}

	return nil
}

func (dify *SDify) CheckContainerHealth(ctx context.Context, userCred mcclient.TokenCredential, containerKey string, command ...string) error {
	container, err := dify.getContainerByContainerKey(containerKey)
	if nil != err {
		return err
	}

	// exec command
	result, err := container.PerformExecSync(ctx, userCred, nil, &api.ContainerExecSyncInput{
		Command: command,
	})
	if nil != err {
		return err
	}
	log.Infof("dify check container %s health, result: %s", container.GetName(), result.String())

	return nil
}

func (dify *SDify) CheckRedis(ctx context.Context, userCred mcclient.TokenCredential) error {
	// get redis container
	ctrs, err := GetContainerManager().GetContainersByPod(dify.GuestId)
	if err != nil {
		return err
	}

	// set redis container id
	if len(ctrs) != 1 {
		return errors.Errorf("Strange redis container")
	}
	ctr := ctrs[0]
	dify.setContainerId(api.DIFY_REDIS_KEY, ctr.GetId())

	// exec to check health
	if err := dify.CheckContainerHealth(ctx, userCred, api.DIFY_REDIS_KEY, "redis-cli", "-c", "ping"); nil != err {
		return err
	}
	return nil
}

func (dify *SDify) createContainerOnPod(ctx context.Context, userCred mcclient.TokenCredential, containerKey string, data *api.PodContainerCreateInput) (*SContainer, error) {
	// get guest
	guest, err := GuestManager.FetchById(dify.GuestId)
	if nil != err {
		return nil, err
	}
	pod := guest.(*SGuest)

	// create container on pod
	ctr, err := GetContainerManager().CreateOnPod(ctx, userCred, pod.GetOwnerId(), pod, data)
	if nil != err {
		return nil, err
	}

	// set container id
	err = dify.setContainerId(containerKey, ctr.GetId())

	return ctr, err
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

func (dify *SDify) getContainerByContainerKey(containerKey string) (*SContainer, error) {
	if dify.Containers == nil {
		return nil, errors.Error("containers not initialized")
	}
	containersDict, ok := dify.Containers.(*jsonutils.JSONDict)
	if !ok {
		return nil, errors.Error("containers is not a valid JSON dict")
	}

	// get ctrId
	ctrIdJson, err := containersDict.Get(containerKey)
	if nil != err {
		return nil, errors.Wrapf(err, "container key %s not found", containerKey)
	}
	ctrId, err := ctrIdJson.GetString()
	if nil != err {
		return nil, errors.Wrapf(err, "container id not string")
	}

	// get ctr
	imd, err := GetContainerManager().FetchById(ctrId)
	if nil != err {
		return nil, errors.Wrapf(err, "No container with id %s", ctrId)
	}

	return imd.(*SContainer), nil
}
