package huawei

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
type SElasticcacheParameter struct {
	multicloud.SElasticcacheParameterBase

	cacheDB *SElasticcache

	Description  string `json:"description"`
	ParamID      int64  `json:"param_id"`
	ParamName    string `json:"param_name"`
	ParamValue   string `json:"param_value"`
	DefaultValue string `json:"default_value"`
	ValueType    string `json:"value_type"`
	ValueRange   string `json:"value_range"`
}

func (self *SElasticcacheParameter) GetId() string {
	return fmt.Sprintf("%d", self.ParamID)
}

func (self *SElasticcacheParameter) GetName() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceID, self.GetId())
}

func (self *SElasticcacheParameter) GetStatus() string {
	return ""
}

func (self *SElasticcacheParameter) GetParameterKey() string {
	return self.ParamName
}

func (self *SElasticcacheParameter) GetParameterValue() string {
	return self.ParamValue
}

func (self *SElasticcacheParameter) GetParameterValueRange() string {
	return self.Description
}

func (self *SElasticcacheParameter) GetDescription() string {
	return self.ValueRange
}

func (self *SElasticcacheParameter) GetModifiable() bool {
	return true
}

func (self *SElasticcacheParameter) GetForceRestart() bool {
	return false
}
