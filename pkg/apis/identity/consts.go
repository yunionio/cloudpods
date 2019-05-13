package identity

const (
	SERVICE_TYPE = "identity"

	DEFAULT_DOMAIN_ID   = "default"
	DEFAULT_DOMAIN_NAME = "Default"

	AUTH_METHOD_PASSWORD = "password"
	AUTH_METHOD_TOKEN    = "token"

	AUTH_METHOD_ID_PASSWORD = 1
	AUTH_METHOD_ID_TOKEN    = 2

	AUTH_TOKEN_HEADER         = "X-Auth-Token"
	AUTH_SUBJECT_TOKEN_HEADER = "X-Subject-Token"

	AssignmentUserProject  = "UserProject"
	AssignmentGroupProject = "GroupProject"
	AssignmentUserDomain   = "UserDomain"
	AssignmentGroupDomain  = "GroupDomain"

	EndpointInterfacePublic   = "public"
	EndpointInterfaceInternal = "internal"
	EndpointInterfaceAdmin    = "admin"

	KeystoneDomainRoot = "<<keystone.domain.root>>"

	IdMappingEntityUser  = "user"
	IdMappingEntityGroup = "group"

	IdentityDriverSQL  = "sql"
	IdentityDriverLDAP = "ldap"
)

var (
	AUTH_METHODS = []string{AUTH_METHOD_PASSWORD, AUTH_METHOD_TOKEN}

	SensitiveDomainConfigMap = map[string]string{
		"ldap": "password",
	}
)
