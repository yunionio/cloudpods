package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ServerScreenDumpListOptions struct {
	options.BaseListOptions
	Server string `help:"Id or name of server"`
}

func (o *ServerScreenDumpListOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func init() {
	cmd := shell.NewResourceCmd(&modules.ServerScreenDumps)
	cmd.List(new(ServerScreenDumpListOptions))
}
