package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"


const (
	MODE_FIXED = "fixed"
	MODE_STANDALONE = "standalone"

	ASSOCIATE_TYPE_SERVER = "server"
)

type SElasticipManager struct {
	db.SVirtualResourceBaseManager
}

var ElasticipManager *SElasticipManager

func init() {
	ElasticipManager = &SElasticipManager{SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SElasticip{}, "elasticips_tbl", "eip", "eips")}
}

type SElasticip struct {
	db.SVirtualResourceBase

	Mode string `width:"32" charset:"ascii"`

	AssociateType string `width:"32" charset:"ascii"`
	AssociateId string `width:"128" charset:"ascii"`
}
