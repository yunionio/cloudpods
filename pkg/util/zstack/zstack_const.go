package zstack

import "time"

type ZStackTime struct {
	CreateDate time.Time `json:"createDate"`
	LastOpDate time.Time `json:"lastOpDate"`
}

type ZStackBasic struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

const (
	DEFAULT_VPC_NAME = "ZStackVPC"
)
