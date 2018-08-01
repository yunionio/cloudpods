package modules

type RegionManager struct {
	ResourceManager
}

var (
	Regions RegionManager
)

func init() {
	Regions = RegionManager{NewIdentityV3Manager("region", "regions",
		[]string{},
		[]string{"ID", "Name", "Description"})}

	register(&Regions)
}
