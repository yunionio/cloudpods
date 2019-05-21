package zstack

type SSysTag struct {
	ZStackTime
	Inherent     bool   `json:"inherent"`
	ResourceType string `json:"resourceType"`
	ResourceUUID string `json:"resourceUuid"`
	Tag          string `json:"tag"`
	Type         string `json:"type"`
	UUID         string `json:"uuid"`
}

func (region *SRegion) GetSysTags(tagId string, resourceType string, resourceId string, tag string) ([]SSysTag, error) {
	tags := []SSysTag{}
	params := []string{}
	if len(tagId) > 0 {
		params = append(params, "q=uuid="+tagId)
	}
	if len(resourceType) > 0 {
		params = append(params, "q=resourceType="+resourceType)
	}
	if len(resourceId) > 0 {
		params = append(params, "q=resourceUuid="+resourceId)
	}
	if len(tag) > 0 {
		params = append(params, "q=tag="+tag)
	}
	return tags, region.client.listAll("system-tags", params, &tags)
}
