package snapshot_service

type IGuestManager interface {
	GetContainerManager(serverId string) (ISnapshotContainerManager, error)
}

type ISnapshotContainerManager interface {
	GetRootFsMountPath(containerId string) (string, error)
}
