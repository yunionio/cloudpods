package tasks

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
)

type DiskCleanOverduedSnapshots struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(DiskCleanOverduedSnapshots{})
}

func (self *DiskCleanOverduedSnapshots) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	disk := obj.(*models.SDisk)

	count, err := models.SnapshotManager.Query().Equals("disk_id", disk.Id).
		Equals("created_by", compute.SNAPSHOT_AUTO).Equals("fake_deleted", false).CountWithError()
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}

	if count <= (options.Options.DefaultMaxSnapshotCount - options.Options.DefaultMaxManualSnapshotCount) {
		self.SetStageComplete(ctx, nil)
		return
	}

	snapshot := new(models.SSnapshot)
	err = models.SnapshotManager.Query().Equals("disk_id", disk.Id).
		Equals("created_by", compute.SNAPSHOT_AUTO).Equals("fake_deleted", false).Asc("created_at").First(snapshot)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
	snapshot.SetModelManager(models.SnapshotManager, snapshot)
	err = snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.Id)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
		return
	}
}
