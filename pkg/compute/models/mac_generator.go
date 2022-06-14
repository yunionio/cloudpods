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

package models

import (
	"fmt"
	"math/rand"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type IMacGenerator interface {
	FilterByMac(mac string) *sqlchemy.SQuery
}

const maxMacTries = 10

func generateMac(suggestion string) (string, error) {
	for tried := 0; tried < maxMacTries; tried += 1 {
		var mac string
		if len(suggestion) > 0 && regutils.MatchMacAddr(suggestion) {
			mac = suggestion
			suggestion = ""
		} else {
			b := make([]byte, 4)
			_, err := rand.Read(b)
			if err != nil {
				log.Errorf("generate random mac failed: %s", err)
				continue
			}
			mac = fmt.Sprintf("%s:%02x:%02x:%02x:%02x", options.Options.GlobalMacPrefix, b[0], b[1], b[2], b[3])
		}
		found := false
		for _, man := range []IMacGenerator{
			GuestnetworkManager,
			NetTapServiceManager,
		} {
			q := man.FilterByMac(mac)
			cnt, err := q.CountWithError()
			if err != nil {
				log.Errorf("find mac %s error %s", mac, err)
				return "", err
			}
			if cnt > 0 {
				found = true
				break
			}
		}
		if !found {
			return mac, nil
		}
	}
	return "", errors.Wrap(httperrors.ErrTooManyAttempts, "maximal retry reached")
}
