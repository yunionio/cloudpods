package huawei

type SElbHealthCheck struct {
	region *SRegion

	Name          string `json:"name"`
	AdminStateUp  bool   `json:"admin_state_up"`
	TenantID      string `json:"tenant_id"`
	ProjectID     string `json:"project_id"`
	DomainName    string `json:"domain_name"`
	Delay         int    `json:"delay"`
	ExpectedCodes string `json:"expected_codes"`
	MaxRetries    int    `json:"max_retries"`
	HTTPMethod    string `json:"http_method"`
	Timeout       int    `json:"timeout"`
	Pools         []Pool `json:"pools"`
	URLPath       string `json:"url_path"`
	Type          string `json:"type"`
	ID            string `json:"id"`
	MonitorPort   int    `json:"monitor_port"`
}
