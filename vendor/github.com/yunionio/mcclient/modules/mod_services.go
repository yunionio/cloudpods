package modules

var (
	Services   ResourceManager
	ServicesV3 ResourceManager
)

func init() {
	Services = NewIdentityManager("OS-KSADM:service",
		"OS-KSADM:services",
		[]string{},
		[]string{"ID", "Name", "Type", "Description"})

	register(&Services)

	ServicesV3 = NewIdentityV3Manager("service",
		"services",
		[]string{},
		[]string{"ID", "Name", "Type", "Description"})

	register(&ServicesV3)
}
