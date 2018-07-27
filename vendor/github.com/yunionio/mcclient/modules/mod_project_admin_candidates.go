package modules

var (
	ProjectAdminCandidate ResourceManager
)

func init() {
	ProjectAdminCandidate = NewMonitorManager("project_admin_candidate", "project_admin_candidates",
		[]string{"id", "name"},
		[]string{},
	)
	register(&ProjectAdminCandidate)
}
