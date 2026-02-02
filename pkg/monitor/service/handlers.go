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

package service

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/options"
)

var (
	modelManagers      = []db.IModelManager{}
	jointModelManagers = []db.IJointModelManager{}
)

type GetRegisteredModelManagersF func() []db.IModelManager
type GetRegisteredJointModelManagersF func() []db.IJointModelManager

var (
	GetRegisteredModelManagers      GetRegisteredModelManagersF      = nil
	GetRegisteredJointModelManagers GetRegisteredJointModelManagersF = nil
)

func InitHandlers(app *appsrv.Application, isSlave bool) {
	db.InitAllManagers()

	db.RegisterModelManager(db.TenantCacheManager)
	db.RegisterModelManager(db.UserCacheManager)
	db.RegisterModelManager(db.RoleCacheManager)
	db.RegistUserCredCacheUpdater()

	taskman.AddTaskHandler("", app, isSlave)

	for _, manager := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		taskman.ArchivedTaskManager,
	} {
		db.RegisterModelManager(manager)
	}

	if GetRegisteredModelManagers == nil {
		GetRegisteredModelManagers = func() []db.IModelManager {
			return []db.IModelManager{
				db.OpsLog,
				db.Metadata,
				models.DataSourceManager,
				models.AlertManager,
				models.NodeAlertManager,
				models.MeterAlertManager,
				models.NotificationManager,
				models.CommonAlertManager,
				models.MetricMeasurementManager,
				models.MetricFieldManager,
				models.AlertRecordManager,
				models.AlertDashBoardManager,
				models.GetAlertResourceManager(),
				models.AlertPanelManager,
				models.MonitorResourceManager,
				models.AlertRecordShieldManager,
				models.GetMigrationAlertManager(),
			}
		}
	}

	for _, manager := range GetRegisteredModelManagers() {
		db.RegisterModelManager(manager)
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler, isSlave)
	}

	for _, manager := range []db.IModelManager{
		models.UnifiedMonitorManager,
	} {
		handler := db.NewModelHandler(manager)
		dispatcher.AddModelDispatcher("", app, handler, isSlave)
	}

	if GetRegisteredJointModelManagers == nil {
		GetRegisteredJointModelManagers = func() []db.IJointModelManager {
			return []db.IJointModelManager{
				models.AlertNotificationManager,
				models.MetricManager,
				models.GetAlertResourceAlertManager(),
				models.AlertDashBoardPanelManager,
				models.MonitorResourceAlertManager,
			}
		}
	}
	for _, manager := range GetRegisteredJointModelManagers() {
		db.RegisterModelManager(manager)
		handler := db.NewJointModelHandler(manager)
		dispatcher.AddJointModelDispatcher("", app, handler, isSlave)
	}

}

func InitInfluxDBSubscriptionHandlers(app *appsrv.Application, options *common_options.BaseOptions) {
	root := mux.NewRouter()
	root.UseEncodedPath()

	addCommonAlertDispatcher("", app)
	addMiscHandlers(app, root)
	root.PathPrefix("").Handler(app)

	addr := net.JoinHostPort(options.Address, strconv.Itoa(options.Port))
	if options.EnableSsl {
		srv := appsrv.InitHTTPServer(app, addr)
		srv.Handler = root
		err := srv.ListenAndServeTLS(
			options.SslCertfile,
			options.SslKeyfile,
		)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("%v", err)
		}
	} else {
		err := http.ListenAndServe(addr, root)
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
}

func addMiscHandlers(app *appsrv.Application, root *mux.Router) {
	adapterF := func(appHandleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			appHandleFunc(app.GetContext(), w, r)
		}
	}
	root.HandleFunc("/subscriptions/write", adapterF(performHandler))
	appsrv.AddMiscHandlersToMuxRouter(app, root, options.Options.EnableAppProfiling)
}
