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

package session

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/webconsole/command"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

var (
	Manager        *SSessionManager
	AES_KEY        string
	AccessInterval time.Duration = 5 * time.Minute
)

func init() {
	Manager = NewSessionManager()
	AES_KEY = fmt.Sprintf("webconsole-%f", rand.Float32())
}

type SSessionManager struct {
	*sync.Map
}

func NewSessionManager() *SSessionManager {
	s := &SSessionManager{
		Map: &sync.Map{},
	}
	return s
}

func (man *SSessionManager) Save(data ISessionData) (*SSession, error) {
	idStr := data.GetId()
	if os, ok := man.Load(idStr); ok {
		oldSession := os.(*SSession)
		if oldSession.duplicateHook != nil {
			log.Warningf("session %s already exists, execute dupliate hook", idStr)
			oldSession.duplicateHook()
		}
	}
	token, err := utils.EncryptAESBase64Url(AES_KEY, idStr)
	if err != nil {
		return nil, err
	}
	session := &SSession{
		Id:           idStr,
		ISessionData: data,
		AccessToken:  token,
	}
	man.Store(idStr, session)
	return session, nil
}

func (man *SSessionManager) Get(accessToken string) (*SSession, bool) {
	id, err := utils.DescryptAESBase64Url(AES_KEY, accessToken)
	if err != nil {
		log.Errorf("DescryptAESBase64Url error: %v", err)
		return nil, false
	}
	obj, ok := man.Load(id)
	if !ok {
		return nil, false
	}
	s := obj.(*SSession)
	protocol := s.GetProtocol()
	if protocol != SPICE && time.Since(s.AccessedAt) < AccessInterval {
		log.Warningf("Protol: %q, Token: %s, Session: %s can't be accessed during %s, last accessed at: %s", s.GetProtocol(), accessToken, s.Id, AccessInterval, s.AccessedAt)
		return nil, false
	}
	s.AccessedAt = time.Now()
	return s, true
}

type ISessionData interface {
	command.ICommand
	GetId() string
}

type RandomSessionData struct {
	command.ICommand
	id string
}

func WrapCommandSession(cmd command.ICommand) *RandomSessionData {
	return &RandomSessionData{
		ICommand: cmd,
		id:       stringutils.UUID4(),
	}
}

func (s *RandomSessionData) GetId() string {
	return s.id
}

type SSession struct {
	ISessionData
	Id            string
	AccessToken   string
	AccessedAt    time.Time
	duplicateHook func()
}

func (s SSession) GetConnectParams(params url.Values) (string, error) {
	if params == nil {
		params = url.Values(make(map[string][]string))
	}
	params.Set("api_server", o.Options.ApiServer)
	params.Set("access_token", s.AccessToken)
	params.Set("protocol", s.GetProtocol())
	return params.Encode(), nil
}

func (s *SSession) Close() error {
	if err := s.ISessionData.Cleanup(); err != nil {
		log.Errorf("Clean up command error: %v", err)
	}
	Manager.Delete(s.Id)
	return nil
}

func (s *SSession) RegisterDuplicateHook(f func()) {
	s.duplicateHook = f
}
