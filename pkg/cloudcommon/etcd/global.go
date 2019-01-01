package etcd

var (
	defaultClient *SEtcdClient
)

func InitDefaultEtcdClient(opt *SEtcdOptions) error {
	if defaultClient != nil {
		return nil
	}

	var err error
	defaultClient, err = NewEtcdClient(opt)
	return err
}

func CloseDefaultEtcdClient() error {
	if defaultClient != nil {
		return defaultClient.Close()
	} else {
		return nil
	}
}

func Default() *SEtcdClient {
	return defaultClient
}
