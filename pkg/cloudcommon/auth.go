package cloudcommon

import (
	"context"
	"fmt"
	"os"
	"time"

	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func InitAuth(options *CommonOptions, authComplete auth.AuthCompletedCallback) {

	if len(options.AuthURL) == 0 {
		fmt.Println("Missing AuthURL")
		os.Exit(1)
	}

	if len(options.AdminUser) == 0 {
		fmt.Println("Mising AdminUser")
		os.Exit(1)
	}

	if len(options.AdminPassword) == 0 {
		fmt.Println("Missing AdminPasswd")
		os.Exit(1)
	}

	if len(options.AdminProject) == 0 {
		fmt.Println("Missing AdminProject")
		os.Exit(1)
	}

	a := auth.NewAuthInfo(options.AuthURL,
		options.AdminDomain,
		options.AdminUser,
		options.AdminPassword,
		options.AdminProject)

	// debug := options.LogLevel == "debug"

	auth.Init(a, options.DebugClient, true, options.SslCertfile, options.SslKeyfile) // , authComplete)

	users := options.NotifyAdminUsers
	groups := options.NotifyAdminGroups
	if len(users) == 0 && len(groups) == 0 {
		users = []string{"sysadmin"}
	}
	notifyclient.FetchNotifyAdminRecipients(context.Background(), options.Region, users, groups)

	authComplete()

	if options.EnableRbac {
		policy.EnableGlobalRbac(time.Duration(options.RbacPolicySyncPeriodSeconds)*time.Second,
			time.Duration(options.RbacPolicySyncFailedRetrySeconds)*time.Second,
			options.RbacDebug,
		)
	}
}
