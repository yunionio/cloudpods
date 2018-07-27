package modules

var (
	Tenants ResourceManager
)

func init() {
	Tenants = NewIdentityManager("tenant", "tenants",
		[]string{},
		[]string{"ID", "Name", "Enabled", "Description"})

	register(&Tenants)
}
