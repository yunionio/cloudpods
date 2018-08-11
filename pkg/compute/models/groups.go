package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

const (
	REDIS_TYPE = "REDIS"
	RDS_TYPE   = "RDS"
)

type SGroupManager struct {
	db.SVirtualResourceBaseManager
}

var GroupManager *SGroupManager

func init() {
	GroupManager = &SGroupManager{SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SGroup{}, "groups_tbl", "group", "groups")}
}

type SGroup struct {
	db.SVirtualResourceBase

	ServiceType string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	ParentId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"optional"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	ZoneId string `width:"36" charset:"ascii" nullable:"true" list:"user" update:"user" create:"required"` // Column(VARCHAR(36, charset='ascii'), nullable=True)

	SchedStrategy string `width:"16" charset:"ascii" nullable:"true" default:"" list:"user" update:"user" create:"optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True, default='')
}
