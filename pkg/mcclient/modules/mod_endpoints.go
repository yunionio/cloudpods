package modules

var (
	Endpoints   ResourceManager
	EndpointsV3 ResourceManager
)

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
			"Service_ID",
			"URL", "Interface", "Enabled"})

	register(&EndpointsV3)
}
