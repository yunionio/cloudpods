package cos

const (
	defaultPartSize   = 80 * 1024 * 1024
	defaultRetryTimes = 3
	defaultUA         = "cos-go-sdk-v5.2.9"
	defaultDomain     = "myqcloud.com"
)

// Conf config struct
type Conf struct {
	AppID      string
	SecretID   string
	SecretKey  string
	Region     string
	PartSize   int64
	RetryTimes int
	UA         string
	Domain     string
	Bucket     string
}

func getDefaultConf() *Conf {
	conf := Conf{}
	conf.PartSize = defaultPartSize
	conf.RetryTimes = defaultRetryTimes
	conf.UA = defaultUA
	conf.Domain = defaultDomain

	return &conf
}
