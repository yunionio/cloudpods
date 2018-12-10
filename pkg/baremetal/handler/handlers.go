package handler

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/baremetal/tasks"
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
	app.AddHandler("POST", getBaremetalPrefix("prepare"), objectMiddleware(handleBaremetalPrepare))
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
		task.(*tasks.SBaremetalServerPrepareTask).SSHExecute(task, remoteAddr, key, nil)
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
	ctx.DelayProcess(bm.DelayedRemove)
	ctx.ResponseOk()
}

func handleBaremetalSyncStatus(ctx *Context, bm *baremetal.SBaremetalInstance) {
	ctx.DelayProcess(bm.DelayedSyncStatus)
	ctx.ResponseOk()
}

func handleBaremetalPrepare(ctx *Context, bm *baremetal.SBaremetalInstance) {
	bm.StartBaremetalReprepareTask(ctx.UserCred(), ctx.TaskId(), ctx.Data())
	ctx.ResponseOk()
}
