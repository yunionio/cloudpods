package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Apps        *AppManager
	AppFromFile *AppFromFileManager
)

type AppManager struct {
	*NamespaceResourceManager
}

type AppFromFileManager struct {
	*NamespaceResourceManager
}

func init() {
	Apps = &AppManager{
		NewNamespaceResourceManager("app", "apps",
			NewNamespaceCols(), NewColumns())}

	AppFromFile = &AppFromFileManager{
		NewNamespaceResourceManager("appfromfile", "appfromfiles",
			NewNamespaceCols(), NewColumns())}

	modules.Register(Apps)
	modules.Register(AppFromFile)
}
