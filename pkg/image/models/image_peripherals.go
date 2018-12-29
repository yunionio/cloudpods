package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SImagePeripheral struct {
	db.SResourceBase

	Id      int    `primary:"true" auto_increment:"true" nullable:"false"`
	ImageId string `width:"36" index:"true" nullable:"false"`
}
