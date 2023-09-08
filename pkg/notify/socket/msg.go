package socket

type SMsgEntry struct {
	ObjType   string `json:"obj_type"`
	ObjId     string `json:"obj_id"`
	ObjName   string `json:"obj_name"`
	Success   bool   `json:"success"`
	Action    string `json:"action"`
	Notes     string `json:"notes"`
	UserId    string `json:"user_id"`
	User      string `json:"user"`
	TenantId  string `json:"tenant_id"`
	Tenant    string `json:"tenant"`
	Broadcast bool   `json:"broadcast"`
	//控制前端是否进行弹窗信息提示
	IgnoreAlert bool `json:"ignore_alert"`
}

type SMsg struct {
	WebSocket SMsgEntry `json:"notification"`
}
