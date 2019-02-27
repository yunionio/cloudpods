package aliyun

import "time"

type SProject struct {
	Status      string
	AccountId   string
	DisplayName string
	Id          string
	CreateDate  time.Time
	Name        string
}
