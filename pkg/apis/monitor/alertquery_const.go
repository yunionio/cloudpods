package monitor

var (
	ServerTags = map[string]string{
		"host":             "host",
		"host_id":          "host_id",
		"vm_id":            "id",
		"vm_ip":            "ips",
		"vm_name":          "name",
		"zone":             "zone",
		"zone_id":          "zone_id",
		"zone_ext_id":      "zone_ext_id",
		"os_type":          "os_type",
		"status":           "status",
		"cloudregion":      "cloudregion",
		"cloudregion_id":   "cloudregion_id",
		"region_ext_id":    "region_ext_id",
		"tenant":           "tenant",
		"tenant_id":        "tenant_id",
		"brand":            "brand",
		"scaling_group_id": "vm_scaling_group_id",
		"domain_id":        "domain_id",
		"project_domain":   "project_domain",
	}

	HostTags = map[string]string{
		"host_id":        "id",
		"host_ip":        "ips",
		"host":           "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	RdsTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"rds_id":         "id",
		"rds_ip":         "ips",
		"rds_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	RedisTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"redis_id":       "id",
		"redis_ip":       "ips",
		"redis_name":     "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	OssTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"oss_id":         "id",
		"oss_ip":         "ips",
		"oss_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"region_ext_id":  "region_ext_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	ElbTags = map[string]string{
		"host":           "host",
		"host_id":        "host_id",
		"elb_id":         "id",
		"elb_ip":         "ips",
		"elb_name":       "name",
		"zone":           "zone",
		"zone_id":        "zone_id",
		"zone_ext_id":    "zone_ext_id",
		"os_type":        "os_type",
		"status":         "status",
		"region":         "region",
		"cloudregion":    "cloudregion",
		"cloudregion_id": "cloudregion_id",
		"tenant":         "tenant",
		"tenant_id":      "tenant_id",
		"brand":          "brand",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}

	CloudAccountTags = map[string]string{
		"cloudaccount_id":   "id",
		"cloudaccount_name": "name",
		"brand":             "brand",
		"domain_id":         "domain_id",
		"project_domain":    "project_domain",
	}

	TenantTags = map[string]string{
		"tenant_id":      "id",
		"tenant":         "name",
		"domain_id":      "domain_id",
		"project_domain": "project_domain",
	}
	DomainTags = map[string]string{
		"domain_id":      "id",
		"project_domain": "name",
	}
)
