package service

import (
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apihelper"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

var errTestSync = errors.Error("sync failed")

func newTestService() *ModelSetsService {
	return &ModelSetsService{startupTS: 1000}
}

func modelSetsWithGuests(n int) *agentmodels.ModelSets {
	mss := agentmodels.NewModelSets()
	vpc := &agentmodels.Vpc{}
	vpc.Id = "vpc1"
	vpc.Name = "vpc1"
	mss.Vpcs[vpc.Id] = vpc
	for i := 0; i < n; i++ {
		guest := &agentmodels.Guest{}
		guest.Id = string(rune('a' + i))
		guest.Name = guest.Id
		guest.HostId = "host1"
		mss.Guests[guest.Id] = guest
	}
	return mss
}

// a platform that has nothing in it is a valid answer, not a service that is
// not ready: the agents have to be able to tell the two apart
func TestEmptyPlatformBecomesReady(t *testing.T) {
	s := newTestService()

	if _, ok := s.Timestamp(); ok {
		t.Fatal("timestamp is served before the first successful sync")
	}
	if _, ok := s.Payload(); ok {
		t.Fatal("model sets are served before the first successful sync")
	}

	s.onSyncDone(apihelper.SyncStatus{Correct: true})

	ts, ok := s.Timestamp()
	if !ok {
		t.Fatal("not ready after a successful sync")
	}
	if ts != 1000 {
		t.Fatalf("timestamp = %d, want the startup timestamp 1000", ts)
	}
	payload, ok := s.Payload()
	if !ok {
		t.Fatal("no model sets after a successful sync")
	}
	obj, err := jsonutils.Parse(payload)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if _, err := obj.Get("models"); err != nil {
		t.Fatalf("payload has no models: %s", payload)
	}
}

// a sync that failed, or that came back incomplete, must leave both the
// published model sets and the version alone, so that the agents keep what
// they have instead of picking up a half baked set
func TestFailedSyncKeepsThePublishedVersion(t *testing.T) {
	s := newTestService()
	s.onSyncDone(apihelper.SyncStatus{Correct: true})
	s.publish(modelSetsWithGuests(3))

	ts, _ := s.Timestamp()
	payload, _ := s.Payload()
	if len(payload) == 0 {
		t.Fatal("no payload")
	}

	s.onSyncDone(apihelper.SyncStatus{Err: errTestSync})
	tsErr, _ := s.Timestamp()
	payloadErr, _ := s.Payload()
	if tsErr != ts {
		t.Fatalf("timestamp moved to %d on a failed sync", tsErr)
	}
	if string(payloadErr) != string(payload) {
		t.Fatal("payload changed on a failed sync")
	}

	s.onSyncDone(apihelper.SyncStatus{Correct: false})
	tsBad, _ := s.Timestamp()
	if tsBad != ts {
		t.Fatalf("timestamp moved to %d on an incomplete sync", tsBad)
	}

	// and once a good sync comes back, the version moves on again
	s.publish(modelSetsWithGuests(4))
	tsOK, _ := s.Timestamp()
	if tsOK != ts+1 {
		t.Fatalf("timestamp = %d after a new version, want %d", tsOK, ts+1)
	}
}

// the model sets that come with the first successful sync are the initial
// version, not a change to it; every publish after that moves the version on
func TestVersionMovesOnChanges(t *testing.T) {
	s := newTestService()
	s.onSyncDone(apihelper.SyncStatus{Correct: true})

	s.publish(modelSetsWithGuests(1))
	ts, _ := s.Timestamp()
	if ts != 1000 {
		t.Fatalf("first data at version %d, want the startup version 1000", ts)
	}

	s.publish(modelSetsWithGuests(2))
	tsNext, _ := s.Timestamp()
	if tsNext != ts+1 {
		t.Fatalf("version %d after a change, want %d", tsNext, ts+1)
	}

	s.publish(modelSetsWithGuests(3))
	tsLast, _ := s.Timestamp()
	if tsLast != ts+2 {
		t.Fatalf("version %d after another change, want %d", tsLast, ts+2)
	}
}

// an empty platform is served as an empty payload, and the model sets arriving
// later have to move the version: otherwise the agents, which compare versions,
// would never come back for them
func TestVersionMovesWhenDataArrivesAfterAnEmptyStart(t *testing.T) {
	s := newTestService()
	s.onSyncDone(apihelper.SyncStatus{Correct: true})

	tsEmpty, ok := s.Timestamp()
	if !ok {
		t.Fatal("an empty platform is not ready")
	}
	if _, ok := s.Payload(); !ok {
		t.Fatal("an empty platform has no payload")
	}

	s.publish(modelSetsWithGuests(1))
	tsData, _ := s.Timestamp()
	if tsData == tsEmpty {
		t.Fatalf("version stayed at %d when the data arrived, agents would not notice", tsData)
	}
}

func TestStatusReportsSyncOutcome(t *testing.T) {
	s := newTestService()
	s.onSyncDone(apihelper.SyncStatus{Correct: true})
	s.publish(modelSetsWithGuests(2))

	st := s.Status()
	if !st.Ready || st.SyncCount != 1 || !st.LastSyncCorrect {
		t.Fatalf("unexpected status: %+v", st)
	}
	if st.Counts == nil || st.Counts.Guests != 2 || st.Counts.Vpcs != 1 {
		t.Fatalf("unexpected counts: %+v", st.Counts)
	}
	// the counts travel under the same keys as the model sets themselves
	dumped := jsonutils.Marshal(st.Counts)
	if _, err := dumped.Get("guests"); err != nil {
		t.Fatalf("counts have no guests key: %s", dumped.String())
	}
}

// the payload has to be readable by the agents the very way they read it off
// the wire: parse it, take the version out of it, and rebuild the model sets
// from the "models" key
func TestPayloadIsReadableByAgents(t *testing.T) {
	s := newTestService()
	s.onSyncDone(apihelper.SyncStatus{Correct: true})
	s.publish(modelSetsWithGuests(2))

	payload, ok := s.Payload()
	if !ok {
		t.Fatal("no payload")
	}
	obj, err := jsonutils.Parse(payload)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	ts, err := obj.Int("timestamp")
	if err != nil {
		t.Fatalf("payload has no timestamp: %v", err)
	}
	want, _ := s.Timestamp()
	if ts != want {
		t.Fatalf("payload carries version %d, serving %d", ts, want)
	}

	mssNews := agentmodels.NewModelSets()
	if err := obj.Unmarshal(mssNews, "models"); err != nil {
		t.Fatalf("unmarshal models: %v", err)
	}
	if len(mssNews.Guests) != 2 || len(mssNews.Vpcs) != 1 {
		t.Fatalf("agents would see %s", mssNews.Stats().Dump())
	}
}

func TestCompressPayload(t *testing.T) {
	payload := []byte(`{"timestamp":1,"models":{}}`)
	compressed, err := compressPayload(payload)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if len(compressed) == 0 {
		t.Fatal("empty compressed payload")
	}
	if string(compressed) == string(payload) {
		t.Fatal("payload was not compressed")
	}
}
