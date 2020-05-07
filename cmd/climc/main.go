package main

import (
	"yunion.io/x/onecloud/cmd/climc/entry"
	_ "yunion.io/x/onecloud/cmd/climc/shell"
	_ "yunion.io/x/onecloud/cmd/climc/shell/ansible"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudevent"
	_ "yunion.io/x/onecloud/cmd/climc/shell/cloudnet"
	_ "yunion.io/x/onecloud/cmd/climc/shell/compute"
	_ "yunion.io/x/onecloud/cmd/climc/shell/etcd"
	_ "yunion.io/x/onecloud/cmd/climc/shell/events"
	_ "yunion.io/x/onecloud/cmd/climc/shell/identity"
	_ "yunion.io/x/onecloud/cmd/climc/shell/image"
	_ "yunion.io/x/onecloud/cmd/climc/shell/itsm"
	_ "yunion.io/x/onecloud/cmd/climc/shell/k8s"
	_ "yunion.io/x/onecloud/cmd/climc/shell/logger"
	_ "yunion.io/x/onecloud/cmd/climc/shell/meter"
	_ "yunion.io/x/onecloud/cmd/climc/shell/misc"
	_ "yunion.io/x/onecloud/cmd/climc/shell/monitor"
	_ "yunion.io/x/onecloud/cmd/climc/shell/notify"
	_ "yunion.io/x/onecloud/cmd/climc/shell/servicetree"
	_ "yunion.io/x/onecloud/cmd/climc/shell/yunionconf"
)

func main() {
	entry.ClimcMain()
}
