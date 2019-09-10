package loader

import (
	"yunion.io/x/log"

	_ "yunion.io/x/onecloud/pkg/util/redfish/generic"
	// _ "yunion.io/x/onecloud/pkg/util/redfish/hprest"
	_ "yunion.io/x/onecloud/pkg/util/redfish/idrac"
	_ "yunion.io/x/onecloud/pkg/util/redfish/ilo"
)

func init() {
	log.Infof("Redfish drivers loaded!")
}
