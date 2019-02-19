package policy

import "yunion.io/x/onecloud/pkg/util/rbacutils"

var (
	defaultRules = []rbacutils.SRbacRule{
		{
			Resource: "tasks",
			Action:   PolicyActionPerform,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "hosts",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "zones",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "zones",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "storages",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "storages",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "schedtags",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "schedtags",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cloudregions",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cloudregions",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cachedimages",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "cachedimages",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			// quotas for any services
			// Service:  "compute",
			Resource: "quotas",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			// usages for any services
			// Service:  "compute",
			Resource: "usages",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "compute",
			Resource: "serverskus",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "compute",
			Resource: "serverskus",
			Action:   PolicyActionGet,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "notices",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "readmarks",
			Action:   PolicyActionList,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionagent",
			Resource: "readmarks",
			Action:   PolicyActionCreate,
			Result:   rbacutils.UserAllow,
		},
		{
			Service:  "yunionconf",
			Resource: "parameters",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionList,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "image",
			Resource: "images",
			Action:   PolicyActionPerform,
			Extra:    []string{"update-torrent-status"},
			Result:   rbacutils.GuestAllow,
		},
		{
			Service:  "log",
			Resource: "actions",
			Action:   PolicyActionList,
			Result:   rbacutils.OwnerAllow,
		},
		{
			Service:  "log",
			Resource: "actions",
			Action:   PolicyActionGet,
			Result:   rbacutils.OwnerAllow,
		},
	}
)
