package models

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

// IsDistFollower reports a multi-host replica worker that should not expose HTTP.
func (llm *SLLM) IsDistFollower() bool {
	return llm != nil && llm.DistRole == api.LLM_DIST_ROLE_FOLLOWER
}

// IsDistLeader reports the API-facing member of a multi-host replica.
func (llm *SLLM) IsDistLeader() bool {
	return llm != nil && llm.DistRole == api.LLM_DIST_ROLE_LEADER
}

// IsDistributedMember reports any multi-host replica membership.
func (llm *SLLM) IsDistributedMember() bool {
	return llm != nil && (llm.DistRole == api.LLM_DIST_ROLE_LEADER || llm.DistRole == api.LLM_DIST_ROLE_FOLLOWER)
}

// ShouldRegisterAiproxy is true for single-pod replicas and distributed leaders.
func (llm *SLLM) ShouldRegisterAiproxy() bool {
	if llm == nil {
		return false
	}
	return !llm.IsDistFollower()
}

// DistReplicaGroupKey identifies a logical replica under a deployment.
// Single-pod instances use their own id; distributed members share DistGroup.
func DistReplicaGroupKey(llm *SLLM) string {
	if llm == nil {
		return ""
	}
	if llm.IsDistributedMember() {
		return fmt.Sprintf("g:%d", llm.DistGroup)
	}
	return "s:" + llm.Id
}

type distReplicaMemberRow struct {
	Id              string
	Name            string
	Status          string
	DistRole        string
	DistRank        int
	DistNnodes      int
	DistLeaderId    string
	DistGroup       int
	CmpId           string
	LLMIp           string
	LLMDeploymentId string
}

func (r distReplicaMemberRow) isFollower() bool {
	return r.DistRole == api.LLM_DIST_ROLE_FOLLOWER
}

func (r distReplicaMemberRow) isLeader() bool {
	return r.DistRole == api.LLM_DIST_ROLE_LEADER
}

func (r distReplicaMemberRow) isDistributed() bool {
	return r.isLeader() || r.isFollower()
}

func (r distReplicaMemberRow) groupKey() string {
	if r.isDistributed() {
		return fmt.Sprintf("g:%d", r.DistGroup)
	}
	return "s:" + r.Id
}

type distReplicaGroup struct {
	Key      string
	Members  []distReplicaMemberRow
	Leader   *distReplicaMemberRow
	Expected int
}

func groupDistReplicaMembers(rows []distReplicaMemberRow) []distReplicaGroup {
	order := make([]string, 0)
	byKey := make(map[string]*distReplicaGroup)
	for i := range rows {
		row := rows[i]
		key := row.groupKey()
		g, ok := byKey[key]
		if !ok {
			g = &distReplicaGroup{Key: key, Expected: 1}
			byKey[key] = g
			order = append(order, key)
		}
		g.Members = append(g.Members, row)
		if row.isLeader() {
			cp := row
			g.Leader = &cp
			if row.DistNnodes > 0 {
				g.Expected = row.DistNnodes
			}
		} else if !row.isDistributed() {
			g.Expected = 1
		} else if row.DistNnodes > g.Expected {
			g.Expected = row.DistNnodes
		}
	}
	out := make([]distReplicaGroup, 0, len(order))
	for _, key := range order {
		out = append(out, *byKey[key])
	}
	return out
}

func distGroupIsReady(g distReplicaGroup) bool {
	if !distGroupComplete(g) {
		return false
	}
	for _, m := range g.Members {
		if m.Status != api.LLM_STATUS_RUNNING {
			return false
		}
	}
	return true
}

// distGroupComplete reports whether a group has exactly one leader at rank 0,
// a contiguous 0..Expected-1 rank set without duplicates, and all expected
// members. Single-pod groups are complete when they have one member.
func distGroupComplete(g distReplicaGroup) bool {
	expected := g.Expected
	if expected <= 0 {
		expected = 1
	}
	if len(g.Members) != expected {
		return false
	}
	leaders := 0
	ranks := make(map[int]struct{}, len(g.Members))
	for _, m := range g.Members {
		if !m.isDistributed() {
			return expected == 1
		}
		switch {
		case m.isLeader():
			leaders++
			if m.DistRank != 0 {
				return false
			}
		case m.isFollower():
			if m.DistRank <= 0 {
				return false
			}
		}
		if m.DistRank < 0 || m.DistRank >= expected {
			return false
		}
		if _, dup := ranks[m.DistRank]; dup {
			return false
		}
		ranks[m.DistRank] = struct{}{}
	}
	return expected >= 2 && leaders == 1
}

func distGroupHasCreateFailure(g distReplicaGroup) bool {
	for _, m := range g.Members {
		if m.Status == api.LLM_STATUS_CREATE_FAIL {
			return true
		}
	}
	return false
}

func distGroupHasStartFailure(g distReplicaGroup) bool {
	for _, m := range g.Members {
		if m.Status == api.LLM_STATUS_START_FAIL {
			return true
		}
	}
	return false
}

func distGroupHasProbing(g distReplicaGroup) bool {
	for _, m := range g.Members {
		if m.Status == api.LLM_STATUS_PROBING {
			return true
		}
	}
	return false
}

// buildDistGroupInfos converts one deployment's replica rows into API group
// details. Only multi-host groups are emitted; single-pod instances are
// represented by the deployment's replica counters instead.
// buildDistGroupInfos converts one deployment's replica rows into API group
// details. totalGpus is the GPU count of one full logical replica (the SKU's
// device count); each member reports totalGpus/nnodes. Only multi-host groups
// are emitted; single-pod instances are represented by the deployment's
// replica counters instead.
func buildDistGroupInfos(rows []distReplicaMemberRow, totalGpus int, servers map[string]computeapi.ServerDetails) []*api.LLMDistGroupInfo {
	groups := groupDistReplicaMembers(rows)
	out := make([]*api.LLMDistGroupInfo, 0, len(groups))
	for _, g := range groups {
		nnodes := g.Expected
		if nnodes <= 0 {
			nnodes = len(g.Members)
		}
		gpusPerNode := totalGpus
		if nnodes > 1 && totalGpus > 0 && totalGpus%nnodes == 0 {
			gpusPerNode = totalGpus / nnodes
		}
		members := make([]api.LLMDistMemberInfo, 0, len(g.Members))
		leaderName := ""
		distributed := false
		group := 0
		haveGroup := false
		for _, m := range g.Members {
			if !m.isDistributed() {
				continue
			}
			if !haveGroup {
				group = m.DistGroup
				haveGroup = true
			}
			distributed = true
			info := api.LLMDistMemberInfo{
				Id:     m.Id,
				Name:   m.Name,
				Role:   m.DistRole,
				Rank:   m.DistRank,
				Status: m.Status,
				LlmIp:  m.LLMIp,
				Gpus:   gpusPerNode,
			}
			if srv, ok := servers[m.CmpId]; ok {
				info.HostId = srv.HostId
				info.Host = srv.Host
			}
			if m.isLeader() {
				leaderName = m.Name
			}
			members = append(members, info)
		}
		if !distributed || len(members) == 0 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return members[i].Rank < members[j].Rank })
		out = append(out, &api.LLMDistGroupInfo{
			Group:   group,
			Leader:  leaderName,
			Ready:   distGroupIsReady(g),
			Members: members,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Group < out[j].Group })
	return out
}

// FetchLLMHostIdsByCmpIds resolves the compute host id behind each guest
// (cmp) id. Missing or inaccessible servers are simply absent from the result
// so callers can treat verification as best-effort.
func FetchLLMHostIdsByCmpIds(ctx context.Context, cmpIds []string) map[string]string {
	out := make(map[string]string, len(cmpIds))
	servers := fetchDeploymentServerMap(ctx, cmpIds)
	for id, srv := range servers {
		if srv.HostId != "" {
			out[id] = srv.HostId
		}
	}
	return out
}

func summarizeDistReplicaGroups(rows []distReplicaMemberRow) deploymentReplicaStatusSummary {
	groups := groupDistReplicaMembers(rows)
	summary := deploymentReplicaStatusSummary{}
	for _, g := range groups {
		if distGroupIsReady(g) {
			summary.Running++
			continue
		}
		if distGroupHasCreateFailure(g) {
			summary.HasCreateFailure = true
		}
		if distGroupHasStartFailure(g) {
			summary.HasStartFailure = true
		}
		if distGroupHasProbing(g) {
			summary.HasProbing = true
		}
	}
	return summary
}

// ParseDistGroupFromInstanceName extracts the replica index from names like
// "dep-3" (leader/single) or "dep-3-f1" (follower).
func ParseDistGroupFromInstanceName(deploymentName, instanceName string) (int, bool) {
	prefix := deploymentName + "-"
	if !strings.HasPrefix(instanceName, prefix) {
		return 0, false
	}
	rest := strings.TrimPrefix(instanceName, prefix)
	if idx := strings.Index(rest, "-f"); idx >= 0 {
		rest = rest[:idx]
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return n, true
}

func DistFollowerInstanceName(deploymentName string, group, followerRank int) string {
	return fmt.Sprintf("%s-%d-f%d", deploymentName, group, followerRank)
}

func DistLeaderInstanceName(deploymentName string, group int) string {
	return fmt.Sprintf("%s-%d", deploymentName, group)
}

// FetchLLMServerMap resolves the compute server details (host access IP, NIC
// port mappings, ...) behind the given guests.
func FetchLLMServerMap(ctx context.Context, cmpIds []string) map[string]computeapi.ServerDetails {
	return fetchDeploymentServerMap(ctx, cmpIds)
}

// IsHostLocalLLM reports whether the instance runs on a hostlocal network.
// A hostlocal pod IP only exists on its own host, so cross-host peers must use
// the host access IP plus a host-mapped port instead.
func IsHostLocalLLM(llm *SLLM) bool {
	return llm != nil && llm.NetworkType == string(computeapi.NETWORK_TYPE_HOSTLOCAL)
}

// ServerHostMappedPort returns the host port the host agent allocated for a
// container port of the server's first NIC.
func ServerHostMappedPort(server *computeapi.ServerDetails, containerPort int) (int, bool) {
	if server == nil || containerPort <= 0 {
		return 0, false
	}
	for _, nic := range server.Nics {
		for _, pm := range nic.PortMappings {
			if pm == nil || pm.Port != containerPort || pm.HostPort == nil {
				continue
			}
			if *pm.HostPort > 0 {
				return *pm.HostPort, true
			}
		}
	}
	return 0, false
}

// IsDistInternalPort reports whether the in-pod port belongs to the internal
// distributed inference plumbing (master / DP RPC) of a replica group.
func IsDistInternalPort(group, port int) bool {
	return port == api.LLMDistMasterPort(group) || port == api.LLMDistDPRPCPort(group)
}
