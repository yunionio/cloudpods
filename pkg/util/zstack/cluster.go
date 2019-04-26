package zstack

type SCluster struct {
	ZStackBasic
	Description    string `json:"description"`
	State          string `json:"State"`
	HypervisorType string `json:"hypervisorType"`
	ZStackTime
	ZoneUUID string `json:"zoneUuid"`
	Type     string `json:"type"`
}

func (region *SRegion) GetClusters() ([]SCluster, error) {
	clusters := []SCluster{}
	params := []string{}
	if SkipEsxi {
		params = append(params, "q=type!=vmware")
	}
	return clusters, region.client.listAll("clusters", params, &clusters)
}

func (region *SRegion) GetClusterIds() ([]string, error) {
	clusters, err := region.GetClusters()
	if err != nil {
		return nil, err
	}
	clusterIds := []string{}
	for i := 0; i < len(clusters); i++ {
		clusterIds = append(clusterIds, clusters[i].UUID)
	}
	return clusterIds, nil
}
