package modules

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
)

type RecipientsManager struct {
	ResourceManager
}

func (this *RecipientsManager) DoDeleteRecipient(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("/%s", this.KeywordPlural)

	body := jsonutils.NewDict()
	if params != nil {
		body.Add(params, this.Keyword)
	}

	return this._delete(s, path, body, this.Keyword)
}

var (
	Recipients RecipientsManager
)

func init() {

	Recipients = RecipientsManager{NewMonitorManager("recipient", "recipients",
		[]string{"id", "type", "details", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&Recipients)
}
