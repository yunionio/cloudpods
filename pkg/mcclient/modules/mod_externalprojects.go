package modules

var (
	ExternalProjects ResourceManager
)

func init() {
	ExternalProjects = NewComputeManager("externalproject", "externalprojects",
		[]string{"ID", "Name", "ExternalId", "Created_at", "CloudregionId", "ProjectId"},
		[]string{})

	registerComputeV2(&ExternalProjects)
}
