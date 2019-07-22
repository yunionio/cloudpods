package aliyun

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheParameter struct {
	multicloud.SElasticcacheParameterBase

	cacheDB *SElasticcache

	ParameterDescription string `json:"ParameterDescription"`
	ParameterValue       string `json:"ParameterValue"`
	ForceRestart         string `json:"ForceRestart"`
	CheckingCode         string `json:"CheckingCode"`
	ModifiableStatus     string `json:"ModifiableStatus"`
	ParameterName        string `json:"ParameterName"`
}

func (self *SElasticcacheParameter) GetId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceID, self.ParameterName)
}

func (self *SElasticcacheParameter) GetName() string {
	return self.ParameterName
}

func (self *SElasticcacheParameter) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheParameter) GetStatus() string {
	return ""
}

func (self *SElasticcacheParameter) GetParameterKey() string {
	return self.ParameterName
}

func (self *SElasticcacheParameter) GetParameterValue() string {
	return self.ParameterValue
}

func (self *SElasticcacheParameter) GetParameterValueRange() string {
	return self.CheckingCode
}

func (self *SElasticcacheParameter) GetDescription() string {
	return self.ParameterDescription
}

func (self *SElasticcacheParameter) GetModifiable() bool {
	switch self.ModifiableStatus {
	case "true":
		return true
	default:
		return false
	}
}

func (self *SElasticcacheParameter) GetForceRestart() bool {
	switch self.ForceRestart {
	case "true":
		return true
	default:
		return false
	}
}
