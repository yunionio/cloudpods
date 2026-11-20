package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/scheduler"
)

func TestSummarizeDistReplicaGroupsCountsLogicalReplicas(t *testing.T) {
	rows := []distReplicaMemberRow{
		{Id: "l1", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_LEADER, DistGroup: 0, DistNnodes: 2},
		{Id: "f1", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistGroup: 0, DistNnodes: 2, DistRank: 1},
		{Id: "s1", Status: api.LLM_STATUS_RUNNING},
	}
	summary := summarizeDistReplicaGroups(rows)
	if summary.Running != 2 {
		t.Fatalf("running = %d, want 2 (one distributed group + one single)", summary.Running)
	}
}

func TestSummarizeDistReplicaGroupsIncompleteNotReady(t *testing.T) {
	rows := []distReplicaMemberRow{
		{Id: "l1", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_LEADER, DistGroup: 0, DistNnodes: 2},
		{Id: "f1", Status: api.LLM_STATUS_START_FAIL, DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistGroup: 0, DistNnodes: 2, DistRank: 1},
	}
	summary := summarizeDistReplicaGroups(rows)
	if summary.Running != 0 {
		t.Fatalf("running = %d, want 0", summary.Running)
	}
	if !summary.HasStartFailure {
		t.Fatal("expected start failure")
	}
}

func TestShouldRegisterAiproxySkipsFollower(t *testing.T) {
	follower := &SLLM{DistRole: api.LLM_DIST_ROLE_FOLLOWER}
	if follower.ShouldRegisterAiproxy() {
		t.Fatal("follower must not register aiproxy")
	}
	leader := &SLLM{DistRole: api.LLM_DIST_ROLE_LEADER}
	if !leader.ShouldRegisterAiproxy() {
		t.Fatal("leader must register aiproxy")
	}
	single := &SLLM{}
	if !single.ShouldRegisterAiproxy() {
		t.Fatal("single pod must register aiproxy")
	}
}

func TestParseDistGroupFromInstanceName(t *testing.T) {
	g, ok := ParseDistGroupFromInstanceName("dep", "dep-3-f1")
	if !ok || g != 3 {
		t.Fatalf("got %d %v", g, ok)
	}
	g, ok = ParseDistGroupFromInstanceName("dep", "dep-2")
	if !ok || g != 2 {
		t.Fatalf("got %d %v", g, ok)
	}
}

func TestDistGroupCompleteRequiresAllRanks(t *testing.T) {
	complete := distReplicaGroup{Expected: 2, Members: []distReplicaMemberRow{
		{Id: "l1", DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0},
		{Id: "f1", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1},
	}}
	if !distGroupComplete(complete) {
		t.Fatal("expected complete group")
	}
	missing := distReplicaGroup{Expected: 3, Members: complete.Members}
	if distGroupComplete(missing) {
		t.Fatal("group with a missing follower must be incomplete")
	}
	duplicate := distReplicaGroup{Expected: 2, Members: []distReplicaMemberRow{
		{Id: "l1", DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0},
		{Id: "f1", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 0},
		{Id: "f2", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1},
	}}
	if distGroupComplete(duplicate) {
		t.Fatal("duplicate rank must be incomplete")
	}
}

func TestDistGroupIsReadyRequiresCompleteRunningGroup(t *testing.T) {
	group := distReplicaGroup{Expected: 2, Members: []distReplicaMemberRow{
		{Id: "l1", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0},
		{Id: "f1", Status: api.LLM_STATUS_PROBING, DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1},
	}}
	if distGroupIsReady(group) {
		t.Fatal("probing follower must not be ready")
	}
	group.Members[1].Status = api.LLM_STATUS_RUNNING
	if !distGroupIsReady(group) {
		t.Fatal("expected complete running group to be ready")
	}
}

func TestValidateDistributedMemberCreateInput(t *testing.T) {
	valid := []api.LLMCreateInput{
		{},
		{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2, DistGroup: 0},
		{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2, DistLeaderId: "l1", DistGroup: 0},
	}
	for i := range valid {
		if err := validateDistributedMemberCreateInput(&valid[i]); err != nil {
			t.Fatalf("valid input %d rejected: %v", i, err)
		}
	}

	invalid := []api.LLMCreateInput{
		{DistRank: 1},
		{DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 1, DistNnodes: 2},
		{DistRole: api.LLM_DIST_ROLE_LEADER, DistNnodes: 1},
		{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 0, DistNnodes: 2, DistLeaderId: "l1"},
		{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 2, DistNnodes: 2, DistLeaderId: "l1"},
		{DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2},
		{DistRole: "bogus"},
	}
	for i := range invalid {
		if err := validateDistributedMemberCreateInput(&invalid[i]); err == nil {
			t.Fatalf("invalid input %d was accepted: %+v", i, invalid[i])
		}
	}
}

func TestValidateDistributedMemberWritePermission(t *testing.T) {
	if err := validateDistributedMemberWritePermission(nil, &api.LLMCreateInput{}); err != nil {
		t.Fatalf("plain instance rejected: %v", err)
	}
	internal := &api.LLMCreateInput{
		DistRole:        api.LLM_DIST_ROLE_LEADER,
		DistNnodes:      2,
		LLMDeploymentId: "dep",
	}
	if err := validateDistributedMemberWritePermission(nil, internal); err != nil {
		t.Fatalf("deployment-scoped member rejected: %v", err)
	}
	standalone := &api.LLMCreateInput{
		DistRole:   api.LLM_DIST_ROLE_FOLLOWER,
		DistRank:   1,
		DistNnodes: 2,
	}
	if err := validateDistributedMemberWritePermission(nil, standalone); err == nil {
		t.Fatal("standalone dist member must require admin")
	}
}

func TestReservePlanGPUsAccumulatesPerHost(t *testing.T) {
	reserved := map[string]int{"h1": 1}
	plan := &scheduler.ReplicaPlan{Members: []scheduler.ReplicaMemberPlan{
		{HostId: "h1", Devices: make(api.Devices, 3)},
		{HostId: "h2", Devices: make(api.Devices, 2)},
	}}
	ReservePlanGPUs(reserved, plan)
	if reserved["h1"] != 4 || reserved["h2"] != 2 {
		t.Fatalf("reserved = %+v, want h1=4 h2=2", reserved)
	}
}

func TestBuildDistGroupInfosSortsMembersAndReportsHosts(t *testing.T) {
	rows := []distReplicaMemberRow{
		{Id: "f1", Name: "dep-0-f1", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2, DistGroup: 0, CmpId: "srv-f", LLMIp: "10.0.0.2"},
		{Id: "l1", Name: "dep-0", Status: api.LLM_STATUS_RUNNING, DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2, DistGroup: 0, CmpId: "srv-l", LLMIp: "10.0.0.1"},
	}
	servers := make(map[string]computeapi.ServerDetails, 2)
	srvL := computeapi.ServerDetails{}
	srvL.HostId = "h1"
	srvL.Host = "node-a"
	servers["srv-l"] = srvL
	srvF := computeapi.ServerDetails{}
	srvF.HostId = "h2"
	srvF.Host = "node-b"
	servers["srv-f"] = srvF
	groups := buildDistGroupInfos(rows, 8, servers)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	g := groups[0]
	if g.Group != 0 || g.Leader != "dep-0" || !g.Ready {
		t.Fatalf("unexpected group: %+v", g)
	}
	if len(g.Members) != 2 || g.Members[0].Rank != 0 || g.Members[1].Rank != 1 {
		t.Fatalf("members not sorted by rank: %+v", g.Members)
	}
	if g.Members[0].Host != "node-a" || g.Members[1].Host != "node-b" {
		t.Fatalf("host lookup failed: %+v", g.Members)
	}
	if g.Members[0].Gpus != 4 {
		t.Fatalf("gpus = %d, want 4", g.Members[0].Gpus)
	}
}
