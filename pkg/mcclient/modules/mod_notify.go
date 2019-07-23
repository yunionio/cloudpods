package modules

type ConfigsManager struct {
	ResourceManager
}

var (
	Configs ConfigsManager
)

func init() {
	Configs = ConfigsManager{NewNotifyManager("config", "configs",
		[]string{},
		[]string{})}

	register(&Configs)
}

//type Contacts2Manager struct {
//	ResourceManager
//}
//
//func (this *Contacts2Manager) PerformActionWithArrayParams(s *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
//	path := fmt.Sprintf("/%s/%s/%s", this.KeywordPlural, id, action)
//
//	body := jsonutils.NewDict()
//	if params != nil {
//		body.Add(params, this.KeywordPlural)
//	}
//
//	return this._post(s, path, body, this.Keyword)
//}
//
//func (this *Contacts2Manager) DoBatchDeleteContacts(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
//	path := "/contacts/delete-contact"
//
//	return this._post(s, path, params, this.Keyword)
//}
//
//var (
//	Contacts2 Contacts2Manager
//)
//
//func init() {
//	Contacts2 = Contacts2Manager{NewNotifyManager("contact", "contacts",
//		[]string{"id", "uid", "contact_type", "contact", "status"},
//		[]string{})}
//
//	register(&Contacts2)
//}
//
//var (
//	Verifications2 ResourceManager
//)
//
//func init() {
//	Verifications2 = NewNotifyManager("verification", "verifications",
//		[]string{"id", "cid", "sent_at", "token", "expire_at", "status", "created_at", "updated_at", "deleted_at", "created_by", "updated_by", "deleted_by", "deleted", "remark"},
//		[]string{})
//
//	register(&Verifications2)
//}
//
//type TNotifyPriority string
//
//type TNotifyChannel string
//
//const (
//	NotifyPriorityImportant = TNotifyPriority("important")
//	NotifyPriorityCritical  = TNotifyPriority("fatal")
//	NotifyPriorityNormal    = TNotifyPriority("normal")
//
//	NotifyByEmail      = TNotifyChannel("email")
//	NotifyByMobile     = TNotifyChannel("mobile")
//	NotifyByDingTalk   = TNotifyChannel("dingtalk")
//	NotifyByWebConsole = TNotifyChannel("webconsole")
//)
//
//var (
//	Notifications2 Notification2Manager
//)
//
//type SNotifyMessage struct {
//	Uid         string          `json:"uid,omitempty"`
//	Gid         string          `json:"gid,omitempty"`
//	ContactType TNotifyChannel  `json:"contact_type,omitempty"`
//	Topic       string          `json:"topic,omitempty"`
//	Priority    TNotifyPriority `json:"priority,omitempty"`
//	Msg         string          `json:"msg,omitempty"`
//	Remark      string          `json:"remark,omitempty"`
//}
//
//type Notification2Manager struct {
//	ResourceManager
//}
//
//func (manager *Notification2Manager) Send(s *mcclient.ClientSession, msg SNotifyMessage) error {
//	_, err := manager.Create(s, jsonutils.Marshal(&msg))
//	return err
//}
//
//func init() {
//	Notifications2 = Notification2Manager{
//		NewNotifyManager("notification", "notifications",
//			[]string{"id", "uid", "contact_type", "topic", "priority", "msg", "received_at", "send_by", "status", "created_at", "updated_at", "deleted_at", "create_by", "updated_by", "deleted_by", "deleted", "remark"},
//			[]string{}),
//	}
//
//	Register(&Notifications2)
//}
