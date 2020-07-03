package azure

import "yunion.io/x/pkg/errors"

type SSubscription struct {
	SubscriptionId string `json:"subscriptionId"`
	State          string
	DisplayName    string `json:"displayName"`
}

func (self *SAzureClient) GetSubscriptions() ([]SSubscription, error) {
	resp, err := self.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	subscriptions := []SSubscription{}
	err = resp.Unmarshal(&subscriptions, "value")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return subscriptions, nil
}
