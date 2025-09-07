package dify

import (
	"context"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DifyBaseTask struct {
	taskman.STask
}

type DifyCreateTask struct {
	DifyBaseTask
}

func init() {
	taskman.RegisterTask(DifyCreateTask{})
}

func (t *DifyCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.checkRedisStatus(ctx, obj.(*models.SDify))
}

// check redis

func (t *DifyCreateTask) checkRedisStatus(ctx context.Context, dify *models.SDify) {
	if err := dify.CheckRedis(ctx, t.GetUserCred()); nil != err {
		t.OnDeployRedisFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
	t.requestCreatePostgres(ctx, dify)
}

func (t *DifyCreateTask) OnDeployRedisFailed(ctx context.Context, dify *models.SDify, reason jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_REDIS_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

// postgres

func (t *DifyCreateTask) requestCreatePostgres(ctx context.Context, dify *models.SDify) {
	t.SetStage("OnPostgresCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_POSTGRES_KEY, t.GetId()); nil != err {
		t.OnPostgresCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnPostgresCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_POSTGRES_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *DifyCreateTask) OnPostgresCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	// check status
	if err := dify.CheckContainerHealth(ctx, t.GetUserCred(), api.DIFY_POSTGRES_KEY,
		"pg_isready", "-h", "localhost", "-U", "$POSTGRES_USER", "-d", "$POSTGRES_DB"); nil != err {
		t.OnPostgresCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
	t.requestCreateApi(ctx, dify)
}

// api

func (t *DifyCreateTask) requestCreateApi(ctx context.Context, dify *models.SDify) {
	t.SetStage("OnApiCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_API_KEY, t.GetId()); nil != err {
		t.OnApiCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnApiCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_API_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// worker

func (t *DifyCreateTask) OnApiCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnWorkerCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_WORKER_KEY, t.GetId()); nil != err {
		t.OnWorkerCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnWorkerCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_WORKER_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// worker_beat

func (t *DifyCreateTask) OnWorkerCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnWorkerBeatCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_WORKER_BEAT_KEY, t.GetId()); nil != err {
		t.OnWorkerBeatCreate(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnWorkerBeatCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_WORKER_BEAT_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// web

func (t *DifyCreateTask) OnWorkerBeatCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnWebCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_WEB_KEY, t.GetId()); nil != err {
		t.OnWebCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnWebCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_WEB_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// plugin

func (t *DifyCreateTask) OnWebCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnPluginCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_PLUGIN_KEY, t.GetId()); nil != err {
		t.OnPluginCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnPluginCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_PLUGIN_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// sandbox

func (t *DifyCreateTask) OnPluginCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnSandboxCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_SANDBOX_KEY, t.GetId()); nil != err {
		t.OnSandboxCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnSandboxCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_SANDBOX_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *DifyCreateTask) OnSandboxCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	// check status
	if err := dify.CheckContainerHealth(ctx, t.GetUserCred(), api.DIFY_SANDBOX_KEY,
		"curl", "-f", "http://localhost:8194/health"); nil != err {
		t.OnSandboxCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
	t.requestCreateSsrf(ctx, dify)
}

// ssrf

func (t *DifyCreateTask) requestCreateSsrf(ctx context.Context, dify *models.SDify) {
	t.SetStage("OnSsrfCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_SSRF_KEY, t.GetId()); nil != err {
		t.OnSsrfCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnSsrfCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_SSRF_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// nginx

func (t *DifyCreateTask) OnSsrfCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnNginxCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_NGINX_KEY, t.GetId()); nil != err {
		t.OnNginxCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnNginxCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_NGINX_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

// weaviate

func (t *DifyCreateTask) OnNginxCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.SetStage("OnWeaviateCreate", nil)

	if err := dify.CreateContainer(ctx, t.GetUserCred(), api.DIFY_WEAVIATE_KEY, t.GetId()); nil != err {
		t.OnWeaviateCreateFailed(ctx, dify, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *DifyCreateTask) OnWeaviateCreateFailed(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_DEPLOY_WEAVIATE_FAILED, data.String())
	t.SetStageFailed(ctx, data)
}

func (t *DifyCreateTask) OnWeaviateCreate(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	t.OnCreateDify(ctx, dify, nil)
}

func (t *DifyCreateTask) OnCreateDify(ctx context.Context, dify *models.SDify, data jsonutils.JSONObject) {
	dify.SetStatus(ctx, t.GetUserCred(), api.DIFY_CREATED, "")
	t.SetStageComplete(ctx, nil)
}
