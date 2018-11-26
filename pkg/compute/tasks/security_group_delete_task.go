package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SecurityGroupDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SecurityGroupDeleteTask{})
}

func (self *SecurityGroupDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	secgroup := obj.(*models.SSecurityGroup)
	secgroupCache := secgroup.GetSecurityGroupCaches()
	for _, cache := range secgroupCache {
		cache.Delete(ctx, self.GetUserCred())
	}
	secgroup.RealDelete(ctx, self.GetUserCred())
	self.SetStageComplete(ctx, nil)
}
