package qcloud

import "yunion.io/x/onecloud/pkg/multicloud"

type SElasticcacheSecgroup struct {
	multicloud.SElasticcacheBackupBase

	cacheDB *SElasticcache

	CreateTime          string      `json:"CreateTime"`
	InboundRule         []BoundRule `json:"InboundRule"`
	OutboundRule        []BoundRule `json:"OutboundRule"`
	ProjectID           int64       `json:"ProjectId"`
	SecurityGroupID     string      `json:"SecurityGroupId"`
	SecurityGroupName   string      `json:"SecurityGroupName"`
	SecurityGroupRemark string      `json:"SecurityGroupRemark"`
}

type BoundRule struct {
	Action string `json:"Action"`
	IP     string `json:"Ip"`
	Port   string `json:"Port"`
	Proto  string `json:"Proto"`
}
