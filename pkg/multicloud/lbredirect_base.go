package multicloud

type SLoadbalancerRedirectBase struct {
}

func (self *SLoadbalancerRedirectBase) GetRedirect() string {
	return ""
}

func (self *SLoadbalancerRedirectBase) GetRedirectCode() int64 {
	return 0
}

func (self *SLoadbalancerRedirectBase) GetRedirectScheme() string {
	return ""
}

func (self *SLoadbalancerRedirectBase) GetRedirectHost() string {
	return ""
}

func (self *SLoadbalancerRedirectBase) GetRedirectPath() string {
	return ""
}
