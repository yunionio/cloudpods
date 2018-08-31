package options

type AlarmEventListOptions struct {
	BaseListOptions
	NodeLabels     string `help:"Service tree node labels"`
	MetricName     string `help:"Metric name"`
	HostName       string `help:"Host name"`
	HostIp         string `help:"Host IP address"`
	AlarmLevel     string `help:"Alarm level"`
	AlarmCondition string `help:"Concrete alarm rule"`
	Template       string `help:"Template number of the alarm condition"`
	AckStatus      string `help:"Alarm event ack status"`
}
