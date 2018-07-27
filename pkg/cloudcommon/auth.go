package cloudcommon

import (
	"fmt"
	"os"

	"github.com/yunionio/mcclient/auth"
)

func InitAuth(options *Options, authComplete auth.AuthCompletedCallback) {
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

	auth.AsyncInit(a, false, true, authComplete)
}
