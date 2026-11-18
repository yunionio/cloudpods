package llm

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestGroupInstanceInfosGroupsDistributedMembers(t *testing.T) {
	instances := []instanceInfo{
		{Id: "l0", DistRole: api.LLM_DIST_ROLE_LEADER, DistGroup: 0, DistNnodes: 2},
		{Id: "f0", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistGroup: 0, DistNnodes: 2, DistRank: 1},
		{Id: "s1"},
		{Id: "l1", DistRole: api.LLM_DIST_ROLE_LEADER, DistGroup: 1, DistNnodes: 3},
	}
	groups := groupInstanceInfos(instances)
	if len(groups) != 3 {
		t.Fatalf("groups = %d, want 3 (g0, single, g1)", len(groups))
	}
	byKey := make(map[string]replicaGroupInfo, len(groups))
	for _, g := range groups {
		byKey[g.Key] = g
	}
	if g, ok := byKey["g:0"]; !ok || len(g.Members) != 2 || g.Group != 0 {
		t.Fatalf("g:0 = %+v", g)
	}
	if _, ok := byKey["s:s1"]; !ok {
		t.Fatal("single instance should keep its own group key")
	}
	if g, ok := byKey["g:1"]; !ok || g.Group != 1 {
		t.Fatalf("g:1 = %+v", g)
	}
}

func TestReplicaGroupComplete(t *testing.T) {
	complete := replicaGroupInfo{Group: 0, Members: []instanceInfo{
		{Id: "l", DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2},
		{Id: "f", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2},
	}}
	if !replicaGroupComplete(complete) {
		t.Fatal("expected complete group")
	}

	missingLeader := replicaGroupInfo{Group: 0, Members: []instanceInfo{
		{Id: "f", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2},
	}}
	if replicaGroupComplete(missingLeader) {
		t.Fatal("group without leader must be incomplete")
	}

	duplicateRank := replicaGroupInfo{Group: 0, Members: []instanceInfo{
		{Id: "l", DistRole: api.LLM_DIST_ROLE_LEADER, DistRank: 0, DistNnodes: 2},
		{Id: "f1", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 0, DistNnodes: 2},
		{Id: "f2", DistRole: api.LLM_DIST_ROLE_FOLLOWER, DistRank: 1, DistNnodes: 2},
	}}
	if replicaGroupComplete(duplicateRank) {
		t.Fatal("duplicate rank must be incomplete")
	}

	single := replicaGroupInfo{Group: -1, Members: []instanceInfo{{Id: "s"}}}
	if !replicaGroupComplete(single) {
		t.Fatal("single instance group should be complete")
	}
}

func TestReplicaGroupHasFailure(t *testing.T) {
	group := replicaGroupInfo{Members: []instanceInfo{{Id: "l", Status: api.LLM_STATUS_RUNNING}}}
	if replicaGroupHasFailure(group) {
		t.Fatal("running group must not be failed")
	}
	group.Members = append(group.Members, instanceInfo{Id: "f", Status: api.LLM_STATUS_CREATE_FAIL})
	if !replicaGroupHasFailure(group) {
		t.Fatal("create_fail member must mark the group failed")
	}
}

func TestDistRoleDeletePriorityDeletesFollowersFirst(t *testing.T) {
	if !(distRoleDeletePriority(api.LLM_DIST_ROLE_FOLLOWER) <
		distRoleDeletePriority("") &&
		distRoleDeletePriority("") <
			distRoleDeletePriority(api.LLM_DIST_ROLE_LEADER)) {
		t.Fatal("expected follower < single < leader delete order")
	}
}
