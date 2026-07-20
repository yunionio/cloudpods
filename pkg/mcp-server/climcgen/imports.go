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

package climcgen

// 导入 climc shell 子包以填充 shell.CommandTable
import (
	_ "yunion.io/x/onecloud/cmd/climc/shell/compute"
	_ "yunion.io/x/onecloud/cmd/climc/shell/identity"
	_ "yunion.io/x/onecloud/cmd/climc/shell/image"
	_ "yunion.io/x/onecloud/cmd/climc/shell/logger"
	_ "yunion.io/x/onecloud/cmd/climc/shell/monitor"
	_ "yunion.io/x/onecloud/cmd/climc/shell/scheduler"
)
