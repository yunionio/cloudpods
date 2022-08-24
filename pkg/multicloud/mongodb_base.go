package multicloud

type SMongodbBase struct {
	SVirtualResourceBase
	SBillingBase
}

func (instance *SMongodbBase) GetMaxConnections() int {
	return 0
}
