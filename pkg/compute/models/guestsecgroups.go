package models

import (
	"context"
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SGuestsecgroupManager struct {
	SGuestJointsManager
}

var GuestsecgroupManager *SGuestsecgroupManager

func init() {
	db.InitManager(func() {
		GuestsecgroupManager = &SGuestsecgroupManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestsecgroup{},
				"guestsecgroups_tbl",
				"guestsecgroup",
				"guestsecgroups",
				SecurityGroupManager,
			),
		}
	})
}

type SGuestsecgroup struct {
	SGuestJointsBase

	SecgroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" key_index:"true"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (self *SGuestsecgroup) getSecgroup() *SSecurityGroup {
	secgroup := SSecurityGroup{}
	secgroup.SetModelManager(SecurityGroupManager)
	q := SecurityGroupManager.Query()
	q = q.Equals("id", self.SecgroupId)
	if err := q.First(&secgroup); err != nil {
		log.Errorf("failed to find secgroup %s", self.SecgroupId)
		return nil
	}
	return &secgroup
}

func (manager *SGuestsecgroupManager) newGuestSecgroup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, secgroup *SSecurityGroup) (*SGuestsecgroup, error) {
	q := manager.Query()
	q = q.Equals("guest_id", guest.Id).Equals("secgroup_id", secgroup.Id)
	if count := q.Count(); count > 0 {
		return nil, fmt.Errorf("security group %s has assign guest %s", secgroup.Name, guest.Name)
	}

	gs := SGuestsecgroup{SecgroupId: secgroup.Id}
	gs.SetModelManager(manager)
	gs.GuestId = guest.Id

	lockman.LockObject(ctx, secgroup)
	defer lockman.ReleaseObject(ctx, secgroup)

	return &gs, manager.TableSpec().Insert(&gs)
}

func (manager *SGuestsecgroupManager) DeleteGuestSecgroup(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest, secgroup *SSecurityGroup) error {
	gss := []SGuestsecgroup{}
	q := manager.Query()
	q = q.Equals(guest.Id, q.Field("guest_id")).Equals(secgroup.Id, q.Field("secgroup_id"))
	if err := db.FetchModelObjects(manager, q, &gss); err != nil {
		return err
	}
	for _, gs := range gss {
		if err := gs.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (self *SGuestsecgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}
