package aliyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type AclEntrys struct {
	AclEntry []AclEntry
}

type AclEntry struct {
	AclEntryComment string
	AclEntryIP      string
}

type SLoadbalancerAcl struct {
	region *SRegion

	AclId   string
	AclName string

	AclEntrys AclEntrys
}

func (acl *SLoadbalancerAcl) GetName() string {
	return acl.AclName
}

func (acl *SLoadbalancerAcl) GetId() string {
	return acl.AclId
}

func (acl *SLoadbalancerAcl) GetGlobalId() string {
	return acl.AclId
}

func (acl *SLoadbalancerAcl) GetStatus() string {
	return ""
}

func (acl *SLoadbalancerAcl) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (acl *SLoadbalancerAcl) IsEmulated() bool {
	return false
}

func (acl *SLoadbalancerAcl) Refresh() error {
	loadbalancerAcl, err := acl.region.GetLoadbalancerAclDetail(acl.AclId)
	if err != nil {
		return err
	}
	return jsonutils.Update(acl, loadbalancerAcl)
}

func (acl *SLoadbalancerAcl) GetAclEntries() *jsonutils.JSONArray {
	result := jsonutils.NewArray()
	detail, err := acl.region.GetLoadbalancerAclDetail(acl.AclId)
	if err != nil {
		log.Errorf("GetLoadbalancerAclDetail %s failed: %v", acl.AclId, err)
		return result
	}
	for _, entry := range detail.AclEntrys.AclEntry {
		result.Add(jsonutils.Marshal(map[string]string{"cidr": entry.AclEntryIP, "comment": entry.AclEntryComment}))
	}
	return result
}

func (region *SRegion) GetLoadbalancerAclDetail(aclId string) (*SLoadbalancerAcl, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["AclId"] = aclId
	body, err := region.lbRequest("DescribeAccessControlListAttribute", params)
	if err != nil {
		return nil, err
	}
	detail := SLoadbalancerAcl{region: region}
	return &detail, body.Unmarshal(&detail)
}

func (region *SRegion) GetLoadbalancerAcls() ([]SLoadbalancerAcl, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	body, err := region.lbRequest("DescribeAccessControlLists", params)
	if err != nil {
		return nil, err
	}
	acls := []SLoadbalancerAcl{}
	return acls, body.Unmarshal(&acls, "Acls", "Acl")
}
