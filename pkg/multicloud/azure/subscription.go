package azure

import "net/url"

type SSubscription struct {
	SubscriptionId string `json:"subscriptionId"`
	State          string
	DisplayName    string `json:"displayName"`
}

func (self *SAzureClient) ListSubscriptions() ([]SSubscription, error) {
	result := []SSubscription{}
	err := self.list("subscriptions", url.Values{}, &result)
	return result, err
}
