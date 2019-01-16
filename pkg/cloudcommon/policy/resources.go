package policy

import "yunion.io/x/pkg/utils"

var (
	computeAdminResources = []string{
		"hosts",
		"zones",
		"storages",
		"wires",
		"vpcs",
		"networks",
		"cloudregions",
		"recyclebins",
		"schedtags",
		"serverskus",
		"cachedimages",
		"cloudaccounts",
		"dynamicschedtags",
		"baremetalagents",
		"schedpolicies",
		"isolated-devices",
		"reservedips",
		"dnsrecords",
	}
	notifyAdminResources = []string{
		"configs",
		"contacts",
	}
	meterAdminResources = []string{
		"rates",
		"res_results",
	}
	k8sAdminResources = []string{
		"kube_clusters",
		"kube_nodes",
	}
	yunionagentAdminResources = []string{
		"notices",
		"readmarks",
		"infos",
	}
	yunionconfAdminResources = []string{}
	logAdminResources        = []string{}
	identityAdminResources   = []string{
		"users",
		"groups",
		"domains",
		"projects",
		"roles",
		"policies",
		"services",
		"endpoints",
	}

	adminResources = map[string][]string{
		"compute":     computeAdminResources,
		"notify":      notifyAdminResources,
		"meter":       meterAdminResources,
		"k8s":         k8sAdminResources,
		"yunionagent": yunionagentAdminResources,
		"yunionconf":  yunionconfAdminResources,
		"log":         logAdminResources,
		"identity":    identityAdminResources,
	}
)

func GetAdminResources() map[string][]string {
	return adminResources
}

func isAdminResource(service string, resource string) bool {
	switch service {
	case "identity":
		return true
	case "compute":
		if utils.IsInStringArray(resource, computeAdminResources) {
			return true
		}
		return false
	case "notify":
		if utils.IsInStringArray(resource, notifyAdminResources) {
			return true
		}
		return false
	case "k8s":
		if utils.IsInStringArray(resource, k8sAdminResources) {
			return true
		}
		return false
	case "meters":
		if utils.IsInStringArray(resource, meterAdminResources) {
			return true
		}
		return false
	case "yunionagent":
		if utils.IsInStringArray(resource, yunionagentAdminResources) {
			return true
		}
		return false
	case "log":
		if utils.IsInStringArray(resource, logAdminResources) {
			return true
		}
	default:
		return false
	}
	return false
}
