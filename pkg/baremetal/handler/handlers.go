// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	baremetalapi "yunion.io/x/onecloud/pkg/apis/baremetal"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/options"
	baremetalstatus "yunion.io/x/onecloud/pkg/baremetal/status"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

var registerWorkMan *appsrv.SWorkerManager

func init() {
	registerWorkMan = appsrv.NewWorkerManager("bm_register_worker", 20, 1024, false)
}

func InitHandlers(app *appsrv.Application) {
	initBaremetalsHandler(app)
}

func AddHandler(app *appsrv.Application, method string, prefix string,
	handler func(context.Context, http.ResponseWriter, *http.Request)) *appsrv.SHandlerInfo {
	return app.AddHandler(method, prefix, auth.Authenticate(handler))
}

func AddHandler2(app *appsrv.Application, method string, prefix string,
	handler func(context.Context, http.ResponseWriter, *http.Request),
	metadata map[string]interface{}, name string, tags map[string]string) *appsrv.SHandlerInfo {
	return app.AddHandler2(method, prefix, auth.Authenticate(handler), metadata, name, tags)
}

func customizeHandlerInfo(info *appsrv.SHandlerInfo) {
	if info.GetName(nil) == "baremetal-register" {
		info.SetProcessTimeout(time.Second * 300).SetWorkerManager(registerWorkMan)
	}
}

func initBaremetalsHandler(app *appsrv.Application) {
	app_common.ExportOptionsHandler(app, &options.Options)

	// baremetal actions handler
	AddHandler(app, "GET", bmActionPrefix("notify"), bmObjMiddleware(handleBaremetalNotify))
	AddHandler(app, "POST", bmActionPrefix("maintenance"), bmObjMiddleware(handleBaremetalMaintenance))
	AddHandler(app, "POST", bmActionPrefix("unmaintenance"), bmObjMiddleware(handleBaremetalUnmaintenance))
	AddHandler(app, "POST", bmActionPrefix("delete"), bmObjMiddlewareWithFetch(handleBaremetalDelete, false))
	AddHandler(app, "POST", bmActionPrefix("syncstatus"), bmObjMiddleware(handleBaremetalSyncStatus))
	AddHandler(app, "POST", bmActionPrefix("sync-config"), bmObjMiddleware(handleBaremetalSyncConfig))
	AddHandler(app, "POST", bmActionPrefix("sync-ipmi"), bmObjMiddleware(handleBaremetalSyncIPMI))
	AddHandler(app, "POST", bmActionPrefix("prepare"), bmObjMiddleware(handleBaremetalPrepare))
	AddHandler(app, "POST", bmActionPrefix("reset-bmc"), bmObjMiddleware(handleBaremetalResetBMC))
	AddHandler(app, "POST", bmActionPrefix("ipmi-probe"), bmObjMiddleware(handleBaremetalIpmiProbe))
	AddHandler(app, "POST", bmActionPrefix("cdrom"), bmObjMiddleware(handleBaremetalCdromTask))
	AddHandler(app, "POST", bmActionPrefix("jnlp"), bmObjMiddleware(handleBaremetalJnlpTask))
	AddHandler(app, "POST", "/baremetals/validate-ipmi", handleBaremetalValidateIPMI())

	// server actions handler
	AddHandler(app, "POST", srvActionPrefix("create"), srvClassMiddleware(handleServerCreate))
	AddHandler(app, "POST", srvActionPrefix("deploy"), srvObjMiddleware(handleServerDeploy))
	AddHandler(app, "POST", srvActionPrefix("rebuild"), srvObjMiddleware(handleServerRebuild))
	AddHandler(app, "POST", srvActionPrefix("start"), srvObjMiddleware(handleServerStart))
	AddHandler(app, "POST", srvActionPrefix("stop"), srvObjMiddleware(handleServerStop))
	AddHandler(app, "POST", srvActionPrefix("reset"), srvObjMiddleware(handleServerReset))
	AddHandler(app, "POST", srvActionPrefix("status"), srvObjMiddleware(handleServerStatus))
	AddHandler(app, "DELETE", srvIdPrefix(), srvObjMiddleware(handleServerDelete))

	handInfo := AddHandler2(app, "POST", bmRegisterPrefix(),
		bmRegisterMiddleware(handleBaremetalRegister), nil, "baremetal-register", nil)
	customizeHandlerInfo(handInfo)
}

func handleBaremetalNotify(ctx *Context, bm *baremetal.SBaremetalInstance) {
	key, err := ctx.Query().GetString("key")
	if err != nil {
		ctx.ResponseError(httperrors.NewInputParameterError("Not found key in query"))
		return
	}
	remoteAddr := ctx.RequestRemoteIP()
	err = bm.SaveSSHConfig(remoteAddr, key)
	if err != nil {
		log.Errorf("Save baremetal %s ssh config: %v", bm.GetId(), err)
	}

	// execute BaremetalServerPrepareTask
	task := bm.GetTask()
	if task == nil {
		task = tasks.NewBaremetalServerPrepareTask(bm)
		bm.SyncStatus(ctx, baremetalstatus.PREPARE, "")
	}
	log.Infof("Get notify from pxe rom os, start exec task: %s", task.GetName())
	task.SSHExecute(remoteAddr, key, nil)
	ctx.ResponseOk()
}

func handleBaremetalMaintenance(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalMaintenanceTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalUnmaintenance(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalUnmaintenanceTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalDelete(ctx *Context, bm *baremetal.SBaremetalInstance) {
	ctx.DelayProcess(bm.DelayedRemove, nil)
	ctx.ResponseOk()
}

func handleBaremetalSyncStatus(ctx *Context, bm *baremetal.SBaremetalInstance) {
	ctx.DelayProcess(bm.DelayedSyncStatus, nil)
	ctx.ResponseOk()
}

func handleBaremetalSyncConfig(ctx *Context, bm *baremetal.SBaremetalInstance) {
	ctx.DelayProcess(bm.DelayedSyncDesc, nil)
	ctx.ResponseOk()
}

func handleBaremetalSyncIPMI(ctx *Context, bm *baremetal.SBaremetalInstance) {
	ctx.DelayProcess(bm.DelayedSyncIPMIInfo, nil)
	ctx.ResponseOk()
}

func handleBaremetalPrepare(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalReprepareTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalResetBMC(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalResetBMCTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalIpmiProbe(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalIpmiProbeTask(ctx, ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalCdromTask(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalCdromTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}

func handleBaremetalJnlpTask(ctx *Context, bm *baremetal.SBaremetalInstance) {
	jnlp, err := bm.GetConsoleJNLP(ctx)
	if err != nil {
		ctx.ResponseError(errors.Wrap(err, "GetConsoleJNLP"))
		return
	}
	result := jsonutils.NewDict()
	result.Add(jsonutils.NewString(jnlp), "jnlp")
	ctx.ResponseJson(result)
}

func handleBaremetalValidateIPMI() appsrv.FilterHandler {
	return auth.Authenticate(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		_, _, body := appsrv.FetchEnv(ctx, w, r)
		if body == nil {
			httperrors.BadRequestError(ctx, w, "body is empty")
			return
		}
		validate := func() (*baremetalapi.ValidateIPMIResponse, error) {
			input := new(baremetalapi.ValidateIPMIRequest)
			if err := body.Unmarshal(input); err != nil {
				return nil, errors.Wrapf(err, "unmarshal validate ipmi request: %s", body)
			}
			redfishCli := redfish.NewRedfishDriver(ctx, "https://"+input.Ip, input.Username, input.Password, false)
			resp := &baremetalapi.ValidateIPMIResponse{}
			if redfishCli == nil {
				resp.IsRedfishSupported = false
				// use ipmitool to validate
				tool := ipmitool.NewLanPlusIPMI(input.Ip, input.Username, input.Password)
				info, err := ipmitool.GetSysInfo(tool)
				if err != nil {
					return nil, errors.Wrap(err, "GetSysInfo by ipmitool")
				}
				resp.IPMISystemInfo = info
			} else {
				path, info, err := redfishCli.GetSystemInfo(ctx)
				if err != nil {
					return nil, errors.Wrapf(err, "get system info")
				}
				resp.RedfishSystemInfo = &baremetalapi.RedfishSystemInfo{
					Path: path,
					Info: info,
				}
			}
			return resp, nil
		}
		resp, err := validate()
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		appsrv.SendStruct(w, resp)
	})
}

func handleServerCreate(ctx *Context, bm *baremetal.SBaremetalInstance) {
	err := bm.StartServerCreateTask(ctx, ctx.UserCred(), ctx.TaskId(), ctx.Data())
	if err != nil {
		ctx.ResponseError(httperrors.NewGeneralError(err))
		return
	}
	ctx.ResponseOk()
}

func handleServerDelete(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	bm.StartServerDestroyTask(ctx.UserCred(), ctx.TaskId(), nil)
	ctx.ResponseOk()
}

func handleServerDeploy(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	if err := bm.StartServerDeployTask(ctx.UserCred(), ctx.TaskId(), ctx.Data()); err != nil {
		ctx.ResponseError(httperrors.NewGeneralError(err))
		return
	}
	ctx.ResponseOk()
}

func handleServerRebuild(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	if err := bm.StartServerRebuildTask(ctx.UserCred(), ctx.TaskId(), ctx.Data()); err != nil {
		ctx.ResponseError(httperrors.NewGeneralError(err))
		return
	}
	ctx.ResponseOk()
}

func handleServerStart(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	if err := bm.StartServerStartTask(ctx.UserCred(), ctx.TaskId(), ctx.Data()); err != nil {
		ctx.ResponseError(httperrors.NewGeneralError(err))
		return
	}
	ctx.ResponseOk()
}

func handleServerStop(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	if bm.HasBMC() {
		if err := bm.StartServerStopTask(ctx.UserCred(), ctx.TaskId(), ctx.Data()); err != nil {
			ctx.ResponseError(httperrors.NewGeneralError(err))
			return
		}
	} else {
		bm.StartBaremetalMaintenanceTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	}
	ctx.ResponseOk()
}

func handleServerReset(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	ctx.DelayProcess(bm.DelayedServerReset, nil)
	ctx.ResponseOk()
}

func handleServerStatus(ctx *Context, bm *baremetal.SBaremetalInstance, _ baremetaltypes.IBaremetalServer) {
	ctx.DelayProcess(bm.DelayedServerStatus, nil)
	ctx.ResponseOk()
}

func handleBaremetalRegister(ctx *Context, input *baremetal.BmRegisterInput) {
	ctx.DelayProcess(func(_ context.Context, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
		baremetal.GetBaremetalManager().RegisterBaremetal(ctx, ctx.userCred, input)
		return nil, nil
	}, nil)
}
