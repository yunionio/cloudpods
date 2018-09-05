package session

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"

	"github.com/golang-plus/uuid"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/webconsole/command"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
)

var (
	Manager *SSessionManager
	AES_KEY string
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

func (man *SSessionManager) Save(Command command.ICommand) (session *SSession, err error) {
	key, err := uuid.NewV4()
	if err != nil {
		return
	}
	idStr := key.String()
	token, err := utils.EncryptAESBase64Url(AES_KEY, idStr)
	if err != nil {
		return
	}
	session = &SSession{
		Id:          idStr,
		ICommand:    Command,
		AccessToken: token,
	}
	man.Store(idStr, session)
	return
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
	return obj.(*SSession), true
}

type SSession struct {
	command.ICommand
	Id          string
	AccessToken string
}

func (s SSession) GetConnectParams() (string, error) {
	params := url.Values{
		"api_server":   {o.Options.ApiServer},
		"access_token": {s.AccessToken},
		"protocol":     {s.GetProtocol()},
	}
	return params.Encode(), nil
}

func (s *SSession) Close() error {
	if err := s.ICommand.Cleanup(); err != nil {
		log.Errorf("Clean up command error: %v", err)
	}
	Manager.Delete(s.Id)
	return nil
}
