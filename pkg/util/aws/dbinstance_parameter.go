package aws

type SDBInstanceParameter struct {
	instance *SDBInstance

	AllowedValues  string `xml:"AllowedValues"`
	ApplyType      string `xml:"ApplyType"`
	DataType       string `xml:"DataType"`
	Description    string `xml:"Description"`
	ApplyMethod    string `xml:"ApplyMethod"`
	ParameterName  string `xml:"ParameterName"`
	Source         string `xml:"Source"`
	IsModifiable   bool   `xml:"IsModifiable"`
	ParameterValue string `xml:"ParameterValue"`
}

type SDBInstanceParameters struct {
	Parameters []SDBInstanceParameter `xml:"Parameters>Parameter"`
}

func (param *SDBInstanceParameter) GetGlobalId() string {
	return param.ParameterName
}

func (param *SDBInstanceParameter) GetKey() string {
	return param.ParameterName
}

func (param *SDBInstanceParameter) GetValue() string {
	return param.ParameterValue
}

func (param *SDBInstanceParameter) GetDescription() string {
	return param.Description
}

func (region *SRegion) GetDBInstanceParameters(name string) ([]SDBInstanceParameter, error) {
	param := map[string]string{"DBParameterGroupName": name}
	parameters := SDBInstanceParameters{}
	err := region.rdsRequest("DescribeDBParameters", param, &parameters)
	if err != nil {
		return nil, err
	}
	return parameters.Parameters, nil
}
