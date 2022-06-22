package incloudsphere

type SRegion struct {
	client *SphereClient
}

func (self *SRegion) GetClient() *SphereClient {
	return self.client
}
