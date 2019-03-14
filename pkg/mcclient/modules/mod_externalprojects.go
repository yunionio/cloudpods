package modules

var (
	ExternalProjects ResourceManager
)

func init() {
	ExternalProjects = NewComputeManager(
		"externalproject",
		"externalprojects",
		[]string{"ID", "Name", "External_Id", "Tenant_id", "Tenant", "Manager_id", "Manager"},
		[]string{})

	registerComputeV2(&ExternalProjects)
}
