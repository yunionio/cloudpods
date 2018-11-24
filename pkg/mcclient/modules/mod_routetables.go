package modules

type RouteTableManager struct {
	ResourceManager
}

var (
	RouteTables RouteTableManager
)

func init() {
	RouteTables = RouteTableManager{
		NewComputeManager(
			"route_table",
			"route_tables",
			[]string{
				"id",
				"name",
				"type",
				"vpc",
				"vpc_id",
				"routes",
			},
			[]string{"tenant"},
		),
	}
	registerCompute(&RouteTables)
}
