package modules

var (
	EmailConfigs ResourceManager
)

func init() {
	EmailConfigs = NewNotifyManager("email_config", "email_configs",
		[]string{"username", "password", "hostname", "ssl_global", "hostport"},
		[]string{},
	)
	register(&EmailConfigs)
}
