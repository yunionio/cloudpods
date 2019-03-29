// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
