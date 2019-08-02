package huawei

type SDBInstanceParameter struct {
	instance *SDBInstance

	Name            string
	Value           string
	RestartRequired bool
	Readonly        bool
	ValueRange      string
	Type            string
	Description     string
}

func (region *SRegion) GetDBInstanceParameters(dbinstanceId string) ([]SDBInstanceParameter, error) {
	params := map[string]string{
		"instance_id": dbinstanceId,
	}
	paramters := []SDBInstanceParameter{}
	err := doListAll(region.ecsClient.DBInstance.ListParameters, params, &paramters)
	if err != nil {
		return nil, err
	}
	return paramters, nil
}

func (param *SDBInstanceParameter) GetGlobalId() string {
	return param.Name
}

func (param *SDBInstanceParameter) GetKey() string {
	return param.Name
}

func (param *SDBInstanceParameter) GetValue() string {
	return param.Value
}

func (param *SDBInstanceParameter) GetDescription() string {
	return param.Description
}
