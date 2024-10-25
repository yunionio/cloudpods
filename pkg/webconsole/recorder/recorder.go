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
	"sync"
	"time"

	"github.com/LeeEirc/terminalparser"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/models"
	"yunion.io/x/onecloud/pkg/webconsole/options"
)

type Recoder interface {
	Start()
	Write(userInput string, ptyOutput string)
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
	cs               *mcclient.ClientSession
	sessionId        string
	accessedAt       time.Time
	userInputStart   bool
	userInputBuff    string
	ptyInitialOutput string
	ptyOutputBuff    string
	ps1Parsed        bool
	ps1              string
	cmdCh            chan string
	object           *Object
	wLock            *sync.Mutex
}

func NewCmdRecorder(s *mcclient.ClientSession, obj *Object, sessionId string, accessedAt time.Time) Recoder {
	return &cmdRecoder{
		cs:             s,
		sessionId:      sessionId,
		accessedAt:     accessedAt,
		userInputStart: false,
		userInputBuff:  "",
		ptyOutputBuff:  "",
		cmdCh:          make(chan string),
		object:         obj,
		wLock:          new(sync.Mutex),
	}
}

func (r *cmdRecoder) Write(userInput string, ptyOutput string) {
	if !options.Options.EnableCommandRecording {
		return
	}

	r.wLock.Lock()
	defer r.wLock.Unlock()

	if len(userInput) != 0 {
		r.userInputStart = true
	}

	if !r.userInputStart && userInput == "" && len(ptyOutput) > 0 {
		r.ptyInitialOutput += ptyOutput
	}
	// try parse PS1
	if !r.ps1Parsed && r.userInputBuff != "" && userInput == "\r" && r.ptyInitialOutput != "" {
		outs := r.parsePtyOutputs(r.ptyInitialOutput)
		if len(outs) != 0 && len(outs) > 1 {
			r.ps1 = outs[len(outs)-1]
		}
		r.ps1Parsed = true
	}

	// user enter command
	if userInput == "\r" && r.userInputBuff != "" {
		r.sendMessage(r.ptyOutputBuff)
		return
	}

	r.userInputBuff += userInput
	r.ptyOutputBuff += ptyOutput
}

func (r *cmdRecoder) cleanCmd() *cmdRecoder {
	r.userInputBuff = ""
	r.ptyOutputBuff = ""
	return r
}

func (r *cmdRecoder) parsePtyOutputs(data string) []string {
	s := terminalparser.Screen{
		Rows:   make([]*terminalparser.Row, 0, 1024),
		Cursor: &terminalparser.Cursor{},
	}
	return s.Parse([]byte(data))
}

func (r *cmdRecoder) sendMessage(ptyOutputBuff string) {
	ptyOuts := r.parsePtyOutputs(ptyOutputBuff)
	if len(ptyOuts) == 0 {
		return
	}
	cmd := ptyOuts[len(ptyOuts)-1]
	if r.ps1 != "" {
		cmd = strings.TrimPrefix(cmd, r.ps1)
	}
	r.cleanCmd()
	r.cmdCh <- cmd
	log.Debugf("sendMessage ps1: %q, ptyOuts: %#v, cmd: %q", r.ps1, ptyOuts, cmd)
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
	if command == "" {
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
		Ps1:             r.ps1,
		Command:         command,
	}
}
