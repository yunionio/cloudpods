package service

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/handler"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models/base"
)

func initHandlers(app *appsrv.Application) {
	for _, manager := range []base.IEtcdModelManager{
		models.ServiceRegistryManager,
	} {
		handler := handler.NewEtcdModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler)
	}
}
