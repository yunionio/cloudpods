package stats

// listContainerNetworkStats returns the network stats of all the running containers.
// It should return (nil, nil) for platforms other than Windows.
func (p *criStatsProvider) listContainerNetworkStats() (map[string]*NetworkStats, error) {
	return nil, nil
}
