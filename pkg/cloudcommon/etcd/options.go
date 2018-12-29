package etcd

type SEtcdOptions struct {
	EtcdEndpoint              []string `help:"etcd endpoints in format of addr:port"`
	EtcdTimeoutSeconds        int      `default:"5" help:"etcd dial timeout in seconds"`
	EtcdRequestTimeoutSeconds int      `default:"2" help:"etcd request timeout in seconds"`
	EtcdLeaseExpireSeconds    int      `default:"5" help:"etcd expire timeout in seconds"`

	EtcdNamspace string `help:"etcd namespace"`

	EtcdUsername string `help:"etcd username"`
	EtcdPassword string `help:"etcd password"`

	EtcdEnabldSsl   bool   `help:"enable SSL/TLS"`
	EtcdSslCertfile string `help:"ssl certification file"`
	EtcdSslKeyfile  string `help:"ssl certification private key file"`
}
