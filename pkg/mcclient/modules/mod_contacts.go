package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type ContactsManager struct {
	ResourceManager
}

func (this *ContactsManager) PerformActionWithArrayParams(s *mcclient.ClientSession, id string, action string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s/%s/%s", this.KeywordPlural, id, action)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.KeywordPlural)
	}

	return this._post(s, path, body, this.Keyword)
}

func (this *ContactsManager) DoBatchDeleteContacts(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := "/contacts/delete-contact"

	return this._post(s, path, params, this.Keyword)
}

var (
	Contacts ContactsManager
)

func init() {
	Contacts = ContactsManager{NewNotifyManager("contact", "contacts",
		[]string{"id", "name", "display_name", "details", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&Contacts)
}
