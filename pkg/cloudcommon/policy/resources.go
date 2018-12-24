package policy

import "yunion.io/x/pkg/utils"

var (
	computeAdminResources = []string{
		"hosts",
		"zones",
		"storages",
		"wires",
		"vpcs",
		"cloudregions",
		"recyclebins",
		"schedtags",
		"serverskus",
		"cachedimages",
		"cloudaccounts",
		"dynamicschedtags",
		"baremetalagents",
		"schedpolicies",
	}
	notifyAdminResources = []string{
		"configs",
	}
	meterAdminResources = []string{
		"rates",
	}
	k8sAdminResources = []string{
		"clusters",
		"kube_nodes",
	}
	yunionagentAdminResources = []string{
		"notices",
		"readmarks",
	}
)

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
	default:
		return false
	}
}
