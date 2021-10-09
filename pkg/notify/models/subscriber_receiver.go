// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

var SubscriberReceiverManager *SSubscriberReceiverManager

func init() {
	SubscriberReceiverManager = &SSubscriberReceiverManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(
			SSubscriberReceiver{},
			"subscriber_receiver_tbl",
			"subscriber_receiver",
			"subscriber_receivers",
			SubscriberManager,
			ReceiverManager,
		),
	}
	SubscriberReceiverManager.SetVirtualObject(SubscriberReceiverManager)
}

type SSubscriberReceiverManager struct {
	db.SJointResourceBaseManager
}

type SSubscriberReceiver struct {
	db.SJointResourceBase
	SubscriberId string `width:"36" charset:"ascii" nullable:"false" index:"true"`
	ReceiverId   string `width:"128" charset:"ascii" nullable:"false" index:"true"`
}

func (srm *SSubscriberReceiverManager) GetMasterFieldName() string {
	return "subscriber_id"
}

func (srm *SSubscriberReceiverManager) GetSlaveFieldName() string {
	return "receiver_id"
}

func (srm *SSubscriberReceiverManager) getBySubscriberId(sId string) ([]SSubscriberReceiver, error) {
	q := srm.Query().Equals("subscriber_id", sId)
	srs := make([]SSubscriberReceiver, 0, 2)
	err := db.FetchModelObjects(srm, q, &srs)
	if err != nil {
		return nil, err
	}
	return srs, nil
}

func (srm *SSubscriberReceiverManager) create(ctx context.Context, sId, receiverId string) (*SSubscriberReceiver, error) {
	sr := &SSubscriberReceiver{
		SubscriberId: sId,
		ReceiverId:   receiverId,
	}
	return sr, srm.TableSpec().Insert(ctx, sr)
}

func (srm *SSubscriberReceiverManager) delete(sId, receiverId string) error {
	q := srm.Query().Equals("receiver_id", receiverId).Equals("subscriber_id", sId)
	var sr SSubscriberReceiver
	err := q.First(&sr)
	if err != nil {
		return err
	}
	return sr.Delete(context.Background(), nil)
}
