package aliyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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

func (acl *SLoadbalancerAcl) GetAclEntries() []cloudprovider.SLoadbalancerAccessControlListEntry {
	detail, err := acl.region.GetLoadbalancerAclDetail(acl.AclId)
	if err != nil {
		log.Errorf("GetLoadbalancerAclDetail %s failed: %v", acl.AclId, err)
		return nil
	}
	entrys := []cloudprovider.SLoadbalancerAccessControlListEntry{}
	for _, entry := range detail.AclEntrys.AclEntry {
		entrys = append(entrys, cloudprovider.SLoadbalancerAccessControlListEntry{CIDR: entry.AclEntryIP, Comment: entry.AclEntryComment})
	}
	return entrys
}

func (region *SRegion) UpdateAclName(aclId, name string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["AclId"] = aclId
	params["AclName"] = name
	_, err := region.lbRequest("SetAccessControlListAttribute", params)
	return err
}

func (region *SRegion) RemoveAccessControlListEntry(aclId string, data jsonutils.JSONObject) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["AclId"] = aclId
	params["AclEntrys"] = data.String()
	_, err := region.lbRequest("RemoveAccessControlListEntry", params)
	return err
}

func (acl *SLoadbalancerAcl) Delete() error {
	params := map[string]string{}
	params["RegionId"] = acl.region.RegionId
	params["AclId"] = acl.AclId
	_, err := acl.region.lbRequest("DeleteAccessControlList", params)
	return err
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

func (region *SRegion) GetLoadBalancerAcls() ([]SLoadbalancerAcl, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	body, err := region.lbRequest("DescribeAccessControlLists", params)
	if err != nil {
		return nil, err
	}
	acls := []SLoadbalancerAcl{}
	return acls, body.Unmarshal(&acls, "Acls", "Acl")
}

func (acl *SLoadbalancerAcl) Sync(_acl *cloudprovider.SLoadbalancerAccessControlList) error {
	if acl.AclName != _acl.Name {
		if err := acl.region.UpdateAclName(acl.AclId, _acl.Name); err != nil {
			return err
		}
	}
	entrys := jsonutils.NewArray()
	for _, entry := range acl.AclEntrys.AclEntry {
		entrys.Add(jsonutils.Marshal(map[string]string{"entry": entry.AclEntryIP, "comment": entry.AclEntryComment}))
	}
	if entrys.Length() > 0 {
		if err := acl.region.RemoveAccessControlListEntry(acl.AclId, entrys); err != nil && !isError(err, "Acl does not have any entry") {
			return err
		}
	}
	if len(_acl.Entrys) > 0 {
		return acl.region.AddAccessControlListEntry(acl.AclId, _acl.Entrys)
	}
	return nil
}

func (acl *SLoadbalancerAcl) GetProjectId() string {
	return ""
}
