package models

import (
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func usableCloudProviders() *sqlchemy.SQuery {
	return CloudproviderManager.Query("id").
		In("status", api.CLOUD_PROVIDER_VALID_STATUS).
		In("health_status", api.CLOUD_PROVIDER_VALID_HEALTH_STATUS).
		IsTrue("enabled")
}
