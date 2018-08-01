package modules

var (
	SmsConfigs ResourceManager
)

func init() {
	SmsConfigs = NewNotifyManager("sms_config", "sms_configs",
		[]string{"type", "access_key_id", "access_key_secret", "signature",
			"sms_template_one", "sms_template_two", "sms_template_three", "sms_check_code"},
		[]string{},
	)
	register(&SmsConfigs)
}
