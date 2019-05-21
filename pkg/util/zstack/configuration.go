package zstack

type SConfiguration struct {
	Name         string
	Category     string
	Description  string
	DefaultValue string
	Value        string
}

func (region *SRegion) GetConfigrations() ([]SConfiguration, error) {
	configrations := []SConfiguration{}
	return configrations, region.client.listAll("global-configurations", []string{}, &configrations)
}
