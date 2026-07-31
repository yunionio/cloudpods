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

package guestman

import (
	"os/user"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func parseGroupEntLine(line string) (int64, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, errors.Error("empty group entry")
	}
	parts := strings.Split(line, ":")
	if len(parts) < 3 {
		return 0, errors.Errorf("invalid group entry: %q", line)
	}
	gid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "parse gid from group entry %q", line)
	}
	return gid, nil
}

func lookupHostGroupGID(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.Error("group name is empty")
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("getent", "group", name).Output()
	if err == nil {
		if gid, parseErr := parseGroupEntLine(string(out)); parseErr == nil {
			return gid, nil
		}
	}
	grp, lookupErr := user.LookupGroup(name)
	if lookupErr != nil {
		return 0, errors.Wrapf(lookupErr, "lookup group %q", name)
	}
	gid, err := strconv.ParseInt(grp.Gid, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "parse gid for group %q", name)
	}
	return gid, nil
}

func resolveSupplementalGroups(gids []int64, names []string) ([]int64, error) {
	seen := make(map[int64]struct{}, len(gids)+len(names))
	out := make([]int64, 0, len(gids)+len(names))
	for _, gid := range gids {
		if _, ok := seen[gid]; ok {
			continue
		}
		seen[gid] = struct{}{}
		out = append(out, gid)
	}
	for _, name := range names {
		gid, err := lookupHostGroupGID(name)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve supplemental group %q", name)
		}
		if _, ok := seen[gid]; ok {
			continue
		}
		seen[gid] = struct{}{}
		out = append(out, gid)
	}
	return out, nil
}
