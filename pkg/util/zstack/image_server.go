package zstack

type ImageServers []SImageServer

type SImageServer struct {
	ZStackBasic

	Hostname          string   `json:"hostname"`
	Username          string   `json:"username"`
	SSHPort           int      `json:"sshPort"`
	URL               string   `json:"url"`
	TotalCapacity     int      `json:"totalCapacity"`
	AvailableCapacity int      `json:"availableCapacity"`
	Type              string   `json:"type"`
	State             string   `json:"state"`
	Status            string   `json:"status"`
	AttachedZoneUUIDs []string `json:"attachedZoneUuids"`
	ZStackTime
}

func (v ImageServers) Len() int {
	return len(v)
}

func (v ImageServers) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v ImageServers) Less(i, j int) bool {
	if v[i].AvailableCapacity < v[j].AvailableCapacity {
		return false
	}
	return true
}

func (region *SRegion) GetImageServers(zoneId string) ([]SImageServer, error) {
	servers := []SImageServer{}
	params := []string{"q=state=Enabled", "q=status=Connected", "q=type=ImageStoreBackupStorage"}
	if SkipEsxi {
		params = append(params, "q=type!=VCenter")
	}
	if len(zoneId) > 0 {
		params = append(params, "q=zone.uuid="+zoneId)
	}
	return servers, region.client.listAll("backup-storage", params, &servers)
}
