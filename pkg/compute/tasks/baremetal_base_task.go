package tasks

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SBaremetalBaseTask struct {
	taskman.STask
}

func (self *SBaremetalBaseTask) Action() string {
	actionMap := map[string]string{
		"start":         logclient.ACT_VM_START,
		"stop":          logclient.ACT_VM_STOP,
		"maintenance":   logclient.ACT_BM_MAINTENANCE,
		"unmaintenance": logclient.ACT_BM_UNMAINTENANCE,
	}
	if self.Params.Contains("action") {
		action, _ := self.Params.GetString("action")
		self.Params.Remove("action")
		if val, ok := actionMap[action]; len(action) > 0 && ok {
			return val
		}
	}
	return ""
}

func (self *SBaremetalBaseTask) GetBaremetal() *models.SHost {
	obj := self.GetObject()
	return obj.(*models.SHost)
}
