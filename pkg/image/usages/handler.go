package usages

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func AddUsageHandler(prefix string, app *appsrv.Application) {
	prefix = fmt.Sprintf("%s/usages", prefix)
	app.AddHandler2("GET", prefix, auth.Authenticate(ReportGeneralUsage), nil, "get_usage", nil)
}

func ReportGeneralUsage(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	projectName := jsonutils.GetAnyString(query, []string{"project", "tenant"})
	if projectName != "" {
		isAllow := false
		if consts.IsRbacEnabled() {
			result := policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
				policy.PolicyDelegation, policy.PolicyActionGet)
			isAllow = result == rbacutils.AdminAllow
		} else {
			isAllow = userCred.IsAdminAllow(consts.GetServiceType(), policy.PolicyDelegation, policy.PolicyActionGet)
		}
		if !isAllow {
			httperrors.ForbiddenError(w, "not allow to delegate query usage")
			return
		}
		var err error
		userCred, err = db.TenantCacheManager.GenerateProjectUserCred(ctx, projectName)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
	}

	isAdmin := false
	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
			"usages", policy.PolicyActionGet) == rbacutils.AdminAllow {
			isAdmin = true
		}
	} else {
		isAdmin = userCred.IsAdminAllow(consts.GetServiceType(), "usages", policy.PolicyActionGet)
	}

	var adminUsage map[string]int64
	var projectUsage map[string]int64
	if isAdmin {
		adminUsage = models.ImageManager.Usage(userCred.GetProjectId(), "all")
	}

	isProject := false
	if consts.IsRbacEnabled() {
		if policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
			"usages", policy.PolicyActionGet) == rbacutils.Deny {
			isProject = false
		} else {
			isProject = true
		}
	} else {
		isProject = true
	}

	if isProject {
		projectUsage = models.ImageManager.Usage(userCred.GetProjectId(), "")
	}

	if !isAdmin && !isProject {
		httperrors.ForbiddenError(w, "not allow to get usage")
		return
	}

	usages := jsonutils.NewDict()
	if isProject {
		usages.Update(jsonutils.Marshal(projectUsage))
	}
	if isAdmin {
		usages.Update(jsonutils.Marshal(adminUsage))
	}

	body := jsonutils.NewDict()
	body.Add(usages, "usage")
	appsrv.SendJSON(w, body)
}
