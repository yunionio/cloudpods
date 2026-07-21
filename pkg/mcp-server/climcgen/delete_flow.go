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

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// handleServerDelete：删除前若 disable_delete/protected，先自动解锁（对齐控制台先关删除保护）。
func (t *ClimcTool) handleServerDelete(ctx context.Context, session *mcclient.ClientSession, args map[string]interface{}) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids := serverIDsFromArgs(args)
	unlocked := make([]string, 0)
	for _, id := range ids {
		if id == "" {
			continue
		}
		if err := unlockServerIfNeeded(session, id); err != nil {
			log.Warningf("auto-unlock server %s before delete: %s", id, err)
			continue
		}
		unlocked = append(unlocked, id)
	}
	out, err := invokeCommand(t.cmd, session, args)
	if len(unlocked) > 0 && out != "" {
		out = out + fmt.Sprintf("\n\n[MCP] auto-unlocked disable_delete for: %s", strings.Join(unlocked, ","))
	} else if len(unlocked) > 0 && out == "" {
		out = fmt.Sprintf(`{"command":"server-delete","success":true,"auto_unlocked":%q}`, strings.Join(unlocked, ","))
	}
	return out, err
}

func serverIDsFromArgs(args map[string]interface{}) []string {
	raw, ok := argLookup(args, "id", "ID", "server", "SERVER")
	if !ok {
		return nil
	}
	return valueToArgvParts(raw)
}

func unlockServerIfNeeded(session *mcclient.ClientSession, idOrName string) error {
	obj, err := modules.Servers.Get(session, idOrName, nil)
	if err != nil {
		return err
	}
	need := jsonutils.QueryBoolean(obj, "disable_delete", false) ||
		jsonutils.QueryBoolean(obj, "protected", false)
	if !need {
		return nil
	}
	params := jsonutils.NewDict()
	params.Set("disable_delete", jsonutils.JSONFalse)
	params.Set("protected", jsonutils.JSONFalse)
	sid, _ := obj.GetString("id")
	if sid == "" {
		sid = idOrName
	}
	_, err = modules.Servers.Update(session, sid, params)
	if err != nil {
		return err
	}
	log.Infof("auto-unlocked server %s (disable_delete/protected)", sid)
	return nil
}
