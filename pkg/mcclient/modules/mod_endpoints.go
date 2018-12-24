package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	Endpoints   ResourceManager
	EndpointsV3 ResourceManager
)

func endpointsV3ReadFilter(s *mcclient.ClientSession, result jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	resultDict := result.(*jsonutils.JSONDict)
	serviceId, _ := result.GetString("service_id")
	service, err := cachedResourceManager.getById(&ServicesV3, s, serviceId)
	// service, err := ServicesV3.GetById(s, serviceId, nil)
	if err == nil {
		serviceType, _ := service.Get("type")
		if serviceType != nil {
			resultDict.Add(serviceType, "service_type")
		}
		serviceName, _ := service.Get("name")
		if serviceName != nil {
			resultDict.Add(serviceName, "service_name")
		}
	}
	return resultDict, nil
}

func init() {
	Endpoints = NewIdentityManager("endpoint", "endpoints",
		[]string{},
		[]string{"ID", "Region", "Zone",
			"Service_ID", "Service_name",
			"PublicURL", "AdminURL", "InternalURL"})

	register(&Endpoints)

	EndpointsV3 = NewIdentityV3Manager("endpoint", "endpoints",
		[]string{},
		[]string{"ID", "Region_ID",
			"Service", "Service_ID", "Service_Name", "Service_Type",
			"URL", "Interface", "Enabled"})

	EndpointsV3.SetReadFilter(endpointsV3ReadFilter)

	register(&EndpointsV3)
}
