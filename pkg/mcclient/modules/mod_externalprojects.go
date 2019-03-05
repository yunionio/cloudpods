package modules

var (
	ExternalProjects ResourceManager
)

func init() {
	ExternalProjects = NewComputeManager("externalproject", "externalprojects",
		[]string{"ID", "Name", "ExternalId", "Created_at", "TenantId", "Tenant"},
		[]string{})

	registerComputeV2(&ExternalProjects)
}
