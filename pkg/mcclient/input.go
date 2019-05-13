package mcclient

type SAuthenticationInputV2 struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username,omitempty"`
			Password string `json:"password,omitempty"`
		} `json:"passwordCredentials,omitempty"`
		TenantName string `json:"tenantName,omitempty"`
		TenantId   string `json:"tenantId,omitempty"`
		Token      struct {
			Id string
		} `json:"token,omitempty"`
	} `json:"auth,omitempty"`
}

type SAuthenticationIdentity struct {
	Methods  []string `json:"methods,omitempty"`
	Password struct {
		User struct {
			Id       string `json:"id,omitempty"`
			Name     string `json:"name,omitempty"`
			Password string `json:"password,omitempty"`
			Domain   struct {
				Id   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			}
		} `json:"user,omitempty"`
	} `json:"password,omitempty"`
	Token struct {
		Id string `json:"id,omitempty"`
	} `json:"token,omitempty"`
}

type SAuthenticationInputV3 struct {
	Auth struct {
		Identity SAuthenticationIdentity `json:"identity,omitempty"`
		Scope    struct {
			Project struct {
				Id     string `json:"id,omitempty"`
				Name   string `json:"name,omitempty"`
				Domain struct {
					Id   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				} `json:"domain,omitempty"`
			} `json:"project,omitempty"`
			Domain struct {
				Id   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"domain,omitempty"`
		} `json:"scope,omitempty"`
	} `json:"auth,omitempty"`
}
