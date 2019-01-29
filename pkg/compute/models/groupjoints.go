package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SGroupJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewGroupJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SGroupJointsManager {
	return SGroupJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			GroupManager,
			slave,
		),
	}
}

type SGroupJointsBase struct {
	db.SVirtualJointResourceBase

	SrvtagId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)
}

func (self *SGroupJointsBase) GetGroup() *SGroup {
	guest, _ := GroupManager.FetchById(self.SrvtagId)
	return guest.(*SGroup)
}
