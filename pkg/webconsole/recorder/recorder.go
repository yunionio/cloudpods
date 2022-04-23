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

package recorder

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/models"
)

type Recoder interface {
	Start()
	Write(data string)
}

type Object struct {
	Id        string
	Name      string
	Type      string
	LoginUser string
	Notes     jsonutils.JSONObject
}

func NewObject(id, name, oType, loginUser string, notes jsonutils.JSONObject) *Object {
	return &Object{
		Id:        id,
		Name:      name,
		Type:      oType,
		LoginUser: loginUser,
		Notes:     notes,
	}
}

type cmdRecoder struct {
	cs         *mcclient.ClientSession
	sessionId  string
	accessedAt time.Time
	buffer     []string
	curCmd     string
	cmdCh      chan string
	object     *Object
}

func NewCmdRecorder(s *mcclient.ClientSession, obj *Object, sessionId string, accessedAt time.Time) Recoder {
	return &cmdRecoder{
		cs:         s,
		sessionId:  sessionId,
		accessedAt: accessedAt,
		buffer:     make([]string, 0),
		curCmd:     "",
		cmdCh:      make(chan string),
		object:     obj,
	}
}

func (r *cmdRecoder) Write(data string) {
	if data == "\n" || data == "\r" {
		r.sendMessage(r.curCmd)
		return
	}
	r.curCmd += data
}

func (r *cmdRecoder) cleanCmd() *cmdRecoder {
	r.curCmd = ""
	return r
}

func (r *cmdRecoder) sendMessage(msg string) {
	r.cmdCh <- msg
	r.cleanCmd()
	log.Infof("Message %q sended", msg)
}

func (r *cmdRecoder) Start() {
	for {
		select {
		case cmd := <-r.cmdCh:
			if err := r.save(cmd); err != nil {
				log.Errorf("save comand %q error: %v", cmd, err)
			}
		}
	}
}

func (r *cmdRecoder) save(command string) error {
	if r.object == nil {
		return nil
	}
	userCred := r.cs.GetToken()
	input := r.newModelInput(userCred, command)
	_, err := models.GetCommandLogManager().Create(r.cs.GetContext(), userCred, input)
	if err != nil {
		return errors.Wrapf(err, "Create command log by input: %s", jsonutils.Marshal(input))
	}
	return nil
}

func (r *cmdRecoder) newModelInput(userCred mcclient.TokenCredential, command string) *models.CommandLogCreateInput {
	return &models.CommandLogCreateInput{
		ObjId:           r.object.Id,
		ObjName:         r.object.Name,
		ObjType:         r.object.Type,
		Notes:           r.object.Notes,
		Action:          "record",
		UserId:          userCred.GetUserId(),
		User:            userCred.GetUserName(),
		TenantId:        userCred.GetTenantId(),
		Tenant:          userCred.GetTenantName(),
		DomainId:        userCred.GetDomainId(),
		Domain:          userCred.GetDomainName(),
		ProjectDomainId: userCred.GetProjectDomainId(),
		ProjectDomain:   userCred.GetProjectDomain(),
		Roles:           strings.Join(userCred.GetRoles(), ","),
		SessionId:       r.sessionId,
		AccessedAt:      r.accessedAt,
		LoginUser:       r.object.LoginUser,
		Type:            models.CommandTypeSSH,
		StartTime:       time.Now(),
		Command:         command,
	}
}
