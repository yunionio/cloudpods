package session

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/golang-plus/uuid"

	"yunion.io/x/log"
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

func (man *SSessionManager) Save(data ISessionData) (session *SSession, err error) {
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
		Id:           idStr,
		ISessionData: data,
		AccessToken:  token,
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
	s := obj.(*SSession)
	if time.Since(s.AccessedAt) < AccessInterval {
		log.Warningf("Token: %s, Session: %s can't be accessed during %s, last accessed at: %s", accessToken, s.Id, AccessInterval, s.AccessedAt)
		return nil, false
	}
	s.AccessedAt = time.Now()
	return s, true
}

type ISessionData interface {
	command.ICommand
}

type SSession struct {
	ISessionData
	Id          string
	AccessToken string
	AccessedAt  time.Time
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
