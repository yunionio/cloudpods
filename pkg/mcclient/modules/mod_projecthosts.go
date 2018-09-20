package modules

import (

)

type ProjectNodeManager struct {
	ResourceManager
}

var (
	ProjectHosts ProjectNodeManager
)

func init() {
	ProjectHosts = ProjectNodeManager{NewMonitorManager("project_host", "project_hosts",
		[]string{"id", "host_id", "host_name", "ip", "vcpu_count", "vmem_size", "disk", "project", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&ProjectHosts)
}
