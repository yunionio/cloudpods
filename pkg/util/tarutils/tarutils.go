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

package tarutils

import (
	"os"
	"path/filepath"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func TarSparseFile(origin, tar string) error {
	origin, _ = filepath.Abs(origin)
	tar, _ = filepath.Abs(tar)
	workDir := filepath.Dir(origin)
	originFile := filepath.Base(origin)
	if err := os.Chdir(workDir); err != nil {
		log.Errorln(err)
		return err
	}
	err := procutils.NewCommand("tar", "-Scf", tar, originFile).Run()
	if err != nil {
		log.Errorf("Tar sparse file error: %s", err)
	}
	return nil
}
