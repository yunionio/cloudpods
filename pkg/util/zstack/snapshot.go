package zstack

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SSnapshot struct {
	region *SRegion

	ZStackBasic
	PrimaryStorageUUID string `json:"primaryStorageUuid"`
	VolumeUUID         string `json:"volumeUuid"`
	VolumeType         string `json:"volumeType"`
	Format             string `json:"format"`
	Latest             bool   `json:"latest"`
	Size               int    `json:"size"`
	State              string `json:"state"`
	Status             string `json:"status"`
}

func (snapshot *SSnapshot) GetId() string {
	return snapshot.UUID
}

func (snapshot *SSnapshot) GetName() string {
	return snapshot.Name
}

func (snapshot *SSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "Ready":
		return api.SNAPSHOT_READY
	default:
		log.Errorf("unknown snapshot %s(%s) status %s", snapshot.Name, snapshot.UUID, snapshot.Status)
		return api.SNAPSHOT_UNKNOWN
	}
}

func (snapshot *SSnapshot) GetSize() int32 {
	return int32(snapshot.Size / 1024 / 1024)
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.VolumeUUID
}

func (snapshot *SSnapshot) GetDiskType() string {
	switch snapshot.VolumeType {
	case "Root":
		return api.DISK_TYPE_SYS
	default:
		return api.DISK_TYPE_DATA
	}
}

func (snapshot *SSnapshot) Refresh() error {
	new, err := snapshot.region.GetSnapshot(snapshot.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, new)
}

func (snapshot *SSnapshot) GetGlobalId() string {
	return snapshot.UUID
}

func (snapshot *SSnapshot) IsEmulated() bool {
	return false
}

func (region *SRegion) GetSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshots, err := region.GetSnapshots(snapshotId, "")
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 1 {
		if snapshots[0].UUID == snapshotId {
			return &snapshots[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(snapshots) == 0 || len(snapshotId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetSnapshots(snapshotId string, diskId string) ([]SSnapshot, error) {
	snapshots := []SSnapshot{}
	params := []string{}
	if len(snapshotId) > 0 {
		params = append(params, "q=uuid="+snapshotId)
	}
	if len(diskId) > 0 {
		params = append(params, "q=volumeUuid="+diskId)
	}
	if err := region.client.listAll("volume-snapshots", params, &snapshots); err != nil {
		return nil, err
	}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
	}
	return snapshots, nil
}

func (snapshot *SSnapshot) Delete() error {
	return snapshot.region.DeleteSnapshot(snapshot.UUID)
}

func (snapshot *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	return region.client.delete("volume-snapshots", snapshotId, "Enforcing")
}

func (snapshot *SSnapshot) GetProjectId() string {
	return ""
}

func (region *SRegion) CreateSnapshot(name, diskId, desc string) (*SSnapshot, error) {
	params := map[string]interface{}{
		"params": map[string]string{
			"name":        name,
			"description": desc,
		},
	}
	resource := fmt.Sprintf("volumes/%s/volume-snapshots", diskId)
	resp, err := region.client.post(resource, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	snapshot := &SSnapshot{region: region}
	return snapshot, resp.Unmarshal(snapshot, "inventory")
}
