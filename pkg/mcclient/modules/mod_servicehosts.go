package modules

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type ServiceNodeManager struct {
	ResourceManager
}

func (this *ServiceNodeManager) DoDeleteServiceHost(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.KeywordPlural)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.Keyword)
	}

	return this._delete(s, path, body, this.Keyword)
}

var (
	ServiceHosts ServiceNodeManager
)

func init() {
	ServiceHosts = ServiceNodeManager{NewMonitorManager("service_host", "service_hosts",
		[]string{"id", "host_id", "host_name", "ip", "vcpu_count", "vmem_size", "disk", "project", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&ServiceHosts)
}
