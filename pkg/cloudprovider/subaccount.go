package cloudprovider

type SSubAccount struct {
	Name         string
	State        string
	Account      string
	HealthStatus string // 云端服务健康状态。例如欠费、项目冻结都属于不健康状态。
}
