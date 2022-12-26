package oscli

const (
	DEFAULT_DOMAIN_ID   = "default"
	DEFAULT_DOMAIN_NAME = "Default"

	AUTH_METHOD_PASSWORD = "password"
	AUTH_METHOD_TOKEN    = "token"

	AUTH_TOKEN_HEADER         = "X-Auth-Token"
	AUTH_SUBJECT_TOKEN_HEADER = "X-Subject-Token"
)

type SIdentityObject struct {
	// UUID
	Id string `json:"id"`
	// 名称
	Name string `json:"name"`
}
