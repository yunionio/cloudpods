package compute

const (
	STRATEGY_REQUIRE = "require"
	STRATEGY_EXCLUDE = "exclude"
	STRATEGY_PREFER  = "prefer"
	STRATEGY_AVOID   = "avoid"

	// # container used aggregate
	CONTAINER_AGGREGATE = "container"
)

var STRATEGY_LIST = []string{STRATEGY_REQUIRE, STRATEGY_EXCLUDE, STRATEGY_PREFER, STRATEGY_AVOID}
