package logclient

import (
	"fmt"
	"net/http"
	"context"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/log"
)

func fetchRegion(req *http.Request) string {
	r, e := req.Cookie("region")
	if e != nil {
		return ""
	}
	return r.Value
}

func AddActionLog(ctx context.Context, userCred mcclient.TokenCredential, action, notes string, e error) {
  // 记录企业信息变更操作日志

//  token := auth.FetchUserCredential(ctx)
	log.Errorf("\n.\n.\n.\n.\n.")
	log.Errorf("[AddActionLog]ctx: %s", ctx)
//	log.Errorf("[AddActionLog] token: %s", token)
	log.Errorf("[AddActionLog] token.userid: %s", userCred.GetUserId())
	log.Errorf("\n.\n.\n.\n.\n.")
	// userid := userCred.GetUserId()
	// username := userCred.GetUserName()
	// tenantid := userCred.GetTenantId()
  token := userCred
  // s := auth.GetAdminSession(fetchRegion(req), "")
  s := auth.GetSession(userCred, "", "")
	log.Errorf("session id: Fzu3qiEYUS9P %s", s)

  log := jsonutils.NewDict()
  log.Add(jsonutils.NewString("infos"), "obj_type")
  log.Add(jsonutils.NewString("-"), "obj_id")
  log.Add(jsonutils.NewString("-"), "obj_name")
  log.Add(jsonutils.NewString("更新"), "action")
  log.Add(jsonutils.NewString(token.GetUserId()), "user_id")
  log.Add(jsonutils.NewString(token.GetUserName()), "user")
  log.Add(jsonutils.NewString(token.GetTenantId()), "tenant_id")
  log.Add(jsonutils.NewString(token.GetTenantName()), "tenant")
  log.Add(jsonutils.NewString(notes), "notes")

  if e != nil {
    // 失败日志
    log.Add(jsonutils.JSONFalse, "success")
  } else {
    // 成功日志
    log.Add(jsonutils.JSONTrue, "success")
  }

  _, e = modules.Actions.Create(s, log)
  if e != nil {
    fmt.Printf("create action log failed %s", e)
  } else {
    fmt.Println("create action log sucess")
  }
}
