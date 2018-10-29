package consts

var (
	globalRegion = ""

	globalServiceType = ""
)

func SetRegion(region string) {
	globalRegion = region
}

func GetRegion() string {
	return globalRegion
}

func SetServiceType(srvType string) {
	globalServiceType = srvType
}

func GetServiceType() string {
	return globalServiceType
}
