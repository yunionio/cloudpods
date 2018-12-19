package handler

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func InitHandlers(app *appsrv.Application) {
	initBaremetalsHandler(app)
}

func initBaremetalsHandler(app *appsrv.Application) {
	app.AddHandler("GET", getBaremetalPrefix("notify"), objectMiddleware(handleBaremetalNotify))
	app.AddHandler("POST", getBaremetalPrefix("maintenance"), objectMiddleware(handleBaremetalMaintenance))
	app.AddHandler("POST", getBaremetalPrefix("unmaintenance"), objectMiddleware(handleBaremetalUnmaintenance))
	app.AddHandler("POST", getBaremetalPrefix("delete"), objectMiddleware(handleBaremetalDelete))
	app.AddHandler("POST", getBaremetalPrefix("syncstatus"), objectMiddleware(handleBaremetalSyncStatus))
	app.AddHandler("POST", getBaremetalPrefix("sync-config"), objectMiddleware(handleBaremetalSyncConfig))
	app.AddHandler("POST", getBaremetalPrefix("sync-ipmi"), objectMiddleware(handleBaremetalSyncIPMI))
	app.AddHandler("POST", getBaremetalPrefix("prepare"), objectMiddleware(handleBaremetalPrepare))
	app.AddHandler("POST", getBaremetalPrefix("reset-bmc"), objectMiddleware(handleBaremetalResetBMC))
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
	log.Errorf("====== get task %#v", task)
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
