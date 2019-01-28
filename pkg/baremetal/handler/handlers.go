package handler

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func InitHandlers(app *appsrv.Application) {
	initBaremetalsHandler(app)
}

func initBaremetalsHandler(app *appsrv.Application) {
	// baremetal actions handler
	app.AddHandler("GET", bmActionPrefix("notify"), bmObjMiddleware(handleBaremetalNotify))
	app.AddHandler("POST", bmActionPrefix("maintenance"), bmObjMiddleware(handleBaremetalMaintenance))
	app.AddHandler("POST", bmActionPrefix("unmaintenance"), bmObjMiddleware(handleBaremetalUnmaintenance))
	app.AddHandler("POST", bmActionPrefix("delete"), bmObjMiddleware(handleBaremetalDelete))
	app.AddHandler("POST", bmActionPrefix("syncstatus"), bmObjMiddleware(handleBaremetalSyncStatus))
	app.AddHandler("POST", bmActionPrefix("sync-config"), bmObjMiddleware(handleBaremetalSyncConfig))
	app.AddHandler("POST", bmActionPrefix("sync-ipmi"), bmObjMiddleware(handleBaremetalSyncIPMI))
	app.AddHandler("POST", bmActionPrefix("prepare"), bmObjMiddleware(handleBaremetalPrepare))
	app.AddHandler("POST", bmActionPrefix("reset-bmc"), bmObjMiddleware(handleBaremetalResetBMC))

	// server actions handler
	app.AddHandler("POST", srvActionPrefix("create"), srvClassMiddleware(handleServerCreate))
	app.AddHandler("POST", srvActionPrefix("deploy"), srvObjMiddleware(handleServerDeploy))
	app.AddHandler("POST", srvActionPrefix("rebuild"), srvObjMiddleware(handleServerRebuild))
	app.AddHandler("POST", srvActionPrefix("start"), srvObjMiddleware(handleServerStart))
	app.AddHandler("POST", srvActionPrefix("stop"), srvObjMiddleware(handleServerStop))
	app.AddHandler("POST", srvActionPrefix("reset"), srvObjMiddleware(handleServerReset))
	app.AddHandler("POST", srvActionPrefix("status"), srvObjMiddleware(handleServerStatus))
	app.AddHandler("DELETE", srvIdPrefix(), srvObjMiddleware(handleServerDelete))
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
	if task != nil {
		task.SSHExecute(task, remoteAddr, key, nil)
	}
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

func handleServerCreate(ctx *Context, bm *baremetal.SBaremetalInstance) {
	err := bm.StartServerCreateTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
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
	if err := bm.StartServerStopTask(ctx.UserCred(), ctx.TaskId(), ctx.Data()); err != nil {
		ctx.ResponseError(httperrors.NewGeneralError(err))
		return
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
