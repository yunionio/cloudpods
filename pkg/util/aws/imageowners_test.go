package aws

import (
	"regexp"
	"testing"
)

func TestImageDistMatch(t *testing.T) {
	cases := []struct {
		pattern *regexp.Regexp
		match   string
	}{
		{SUSE_SLES, "suse-sles-11-v20161021-hvm-ssd-x86_64"},
		{SUSE_SLES_SP, "suse-sles-11-sp4-v20161021-hvm-ssd-x86_64"},
		{SUSE_SLES_RIGHTLINK, "suse-sles-11-rightscale-v20160804-hvm-ssd-x86_64"},
		{SUSE_SLES_RIGHTLINK_SP, "suse-sles-11-sp4-rightscale-v20160804-hvm-ssd-x86_64"},
		{SUSE_SLES_SAPCAL, "suse-sles-11-sapcal-20150729-hvm-mag-x86_64"},
		{SUSE_SLES_SAPCAL_SP, "suse-sles-11-sp4-sapcal-20150729-hvm-mag-x86_64"},
		{SUSE_SLES_BYOS, "suse-sles-15-byos-v20180806-hvm-ssd-x86_64"},
		{SUSE_SLES_BYOS_SP, "suse-sles-15-sp4-byos-v20180806-hvm-ssd-x86_64"},
		{SUSE_SLES_SAP, "suse-sles-sap-12-v20180706-hvm-ssd-x86_64"},
		{SUSE_SLES_SAP_SP, "suse-sles-sap-12-sp2-v20180706-hvm-ssd-x86_64"},
		{SUSE_SLES_SAP_BYOS, "suse-sles-sap-12-byos-v20180706-hvm-ssd-x86_64"},
		{SUSE_SLES_SAP_BYOS_SP, "suse-sles-sap-12-sp2-byos-v20180706-hvm-ssd-x86_64"},
		{SUSE_CAASP_CLUSTER_BYOS, "suse-caasp-2-1-cluster-byos-v20180815-hvm-ssd-x86_64"},
		{SUSE_CAASP_ADMIN_BYOS, "suse-caasp-2-1-admin-byos-v20180524-hvm-ssd-x86_64"},
		{SUSE_MANAGER_SERVER_BYOS, "suse-manager-3-1-server-byos-v20170627-hvm-ssd-x86_64"},
		{SUSE_MANAGER_PROXY_BYOS, "suse-manager-3-1-proxy-byos-v20180215-hvm-ssd-x86_64"},
	}
	for _, c := range cases {
		if !c.pattern.MatchString(c.match) {
			t.Errorf("not match %s %s", c.pattern, c.match)
		}
	}
}
